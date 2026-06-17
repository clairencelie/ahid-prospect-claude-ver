package enrichment

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"ahid-prospect/backend/internal/matching"
	"ahid-prospect/backend/internal/store"
)

// Worker is the FR7.2 "goroutine/ticker in demo" async enrichment loop. It
// never runs inline with prospects/check (FR7.4) — the check handler only
// ever does a fast enqueue; this ticker is what actually does the work, on
// its own schedule.
//
// When Gemini is configured, it's tried first (real web search via
// grounding); the static Fixtures map is the fallback when Gemini is unset,
// errors, or finds nothing — so the demo still works offline / without an
// API key.
type Worker struct {
	Store    *store.Store
	Fixtures map[string]Fixture
	Gemini   *GeminiClient // nil disables Gemini, falls straight back to Fixtures
	Interval time.Duration
}

// resolve tries Gemini first (if configured), falling back to the static
// fixture file. Returns (fixture, true) if a suggestion was found.
func (w *Worker) resolve(ctx context.Context, companyName, normalizedName string) (Fixture, bool) {
	if w.Gemini != nil {
		fixture, err := w.Gemini.SuggestRelationships(ctx, companyName)
		if err != nil {
			log.Printf("enrichment: gemini lookup for %q failed, falling back to fixtures: %v", companyName, err)
		} else if fixture != nil {
			return *fixture, true
		}
		// err == nil && fixture == nil means Gemini explicitly found nothing
		// reliable — still worth checking the fixture file as a backstop.
	}
	fixture, found := w.Fixtures[normalizedName]
	return fixture, found
}

func (w *Worker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.Interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.processQueued(ctx)
		}
	}
}

func (w *Worker) processQueued(ctx context.Context) {
	jobs, err := w.Store.ListQueuedEnrichmentJobs(ctx, 10)
	if err != nil {
		log.Printf("enrichment: failed to list queued jobs: %v", err)
		return
	}
	for _, job := range jobs {
		w.process(ctx, job.ID, job.CompanyName, job.NormalizedName)
	}
}

func (w *Worker) process(ctx context.Context, jobID, companyName, normalizedName string) {
	if err := w.Store.MarkEnrichmentJobRunning(ctx, jobID); err != nil {
		log.Printf("enrichment: failed to mark job %s running: %v", jobID, err)
		return
	}

	fixture, found := w.resolve(ctx, companyName, normalizedName)
	if !found {
		w.complete(ctx, jobID, "done", map[string]any{"suggestion": nil})
		return
	}

	createdEdges := []string{}

	subject, err := w.Store.FindOrCreateCompany(ctx, normalizedName, companyName, fixture.Source, fixture.Confidence)
	if err != nil {
		w.fail(ctx, jobID, err)
		return
	}

	if fixture.Group != "" {
		groupNorm := matching.NormalizeName(fixture.Group)
		groupCompany, err := w.Store.FindOrCreateCompany(ctx, groupNorm, fixture.Group, fixture.Source, fixture.Confidence)
		if err != nil {
			w.fail(ctx, jobID, err)
			return
		}
		if _, err := w.Store.CreateRelationshipUnverified(ctx, groupCompany.ID, subject.ID, "same_group", fixture.Source, fixture.Confidence); err != nil {
			w.fail(ctx, jobID, err)
			return
		}
		createdEdges = append(createdEdges, subject.LegalName+" -> same_group -> "+groupCompany.LegalName)

		for _, sibling := range fixture.Siblings {
			siblingNorm := matching.NormalizeName(sibling)
			siblingCompany, err := w.Store.FindOrCreateCompany(ctx, siblingNorm, sibling, fixture.Source, fixture.Confidence)
			if err != nil {
				w.fail(ctx, jobID, err)
				return
			}
			if _, err := w.Store.CreateRelationshipUnverified(ctx, groupCompany.ID, siblingCompany.ID, "same_group", fixture.Source, fixture.Confidence); err != nil {
				w.fail(ctx, jobID, err)
				return
			}
			createdEdges = append(createdEdges, siblingCompany.LegalName+" -> same_group -> "+groupCompany.LegalName)
		}
	}

	if fixture.LegalName != "" && fixture.Relation != "" {
		legalNorm := matching.NormalizeName(fixture.LegalName)
		legalCompany, err := w.Store.FindOrCreateCompany(ctx, legalNorm, fixture.LegalName, fixture.Source, fixture.Confidence)
		if err != nil {
			w.fail(ctx, jobID, err)
			return
		}
		if _, err := w.Store.CreateRelationshipUnverified(ctx, legalCompany.ID, subject.ID, fixture.Relation, fixture.Source, fixture.Confidence); err != nil {
			w.fail(ctx, jobID, err)
			return
		}
		createdEdges = append(createdEdges, subject.LegalName+" -> "+fixture.Relation+" -> "+legalCompany.LegalName)
	}

	w.complete(ctx, jobID, "done", map[string]any{"suggested_edges": createdEdges, "confidence": fixture.Confidence})
}

func (w *Worker) complete(ctx context.Context, jobID, status string, result map[string]any) {
	resultJSON, err := json.Marshal(result)
	if err != nil {
		log.Printf("enrichment: failed to marshal result for job %s: %v", jobID, err)
		return
	}
	if err := w.Store.CompleteEnrichmentJob(ctx, jobID, status, resultJSON); err != nil {
		log.Printf("enrichment: failed to complete job %s: %v", jobID, err)
	}
}

func (w *Worker) fail(ctx context.Context, jobID string, cause error) {
	log.Printf("enrichment: job %s failed: %v", jobID, cause)
	w.complete(ctx, jobID, "failed", map[string]any{"error": cause.Error()})
}
