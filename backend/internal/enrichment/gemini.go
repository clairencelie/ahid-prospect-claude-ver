package enrichment

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GeminiClient calls the Gemini API with Google Search grounding enabled,
// so the model can actually look up the company on the web (the "real
// crawling" the PRD's fixture-mock was standing in for) instead of only
// answering from its training data.
type GeminiClient struct {
	APIKey     string
	Model      string
	HTTPClient *http.Client
}

func NewGeminiClient(apiKey, model string) *GeminiClient {
	if model == "" {
		model = "gemini-2.5-flash"
	}
	return &GeminiClient{
		APIKey:     apiKey,
		Model:      model,
		HTTPClient: &http.Client{Timeout: 30 * time.Second},
	}
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
	Tools    []geminiTool    `json:"tools"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text"`
}

type geminiTool struct {
	GoogleSearch map[string]any `json:"google_search"`
}

type geminiResponse struct {
	Candidates []struct {
		Content geminiContent `json:"content"`
	} `json:"candidates"`
}

const suggestionPrompt = `You are a company-group/brand research assistant for an Indonesian insurance company's prospect-deduplication system.

Use web search to research this company: %q

Determine ONE of the following:
1. It is part of a larger corporate group/holding, or has known subsidiary or sibling companies under the same group.
2. It is a brand/trade name, and you can identify its registered legal entity (e.g. "PT ...").
3. You cannot find reliable information about it.

Respond with ONLY a single raw JSON object (no markdown fences, no commentary), in exactly one of these shapes:
{"group": "<GROUP NAME>", "siblings": ["<SIBLING NAME>", ...], "confidence": <0.0-1.0>}
{"legal_name": "<LEGAL ENTITY NAME>", "relation": "brand_of", "confidence": <0.0-1.0>}
{}

"confidence" must reflect how certain you are based on the search results you found.`

// SuggestRelationships asks Gemini (with search grounding) to suggest a
// group/subsidiary or brand-alias relationship for companyName. Returns
// (nil, nil) when the model found nothing reliable.
func (c *GeminiClient) SuggestRelationships(ctx context.Context, companyName string) (*Fixture, error) {
	reqBody := geminiRequest{
		Contents: []geminiContent{{Parts: []geminiPart{{Text: fmt.Sprintf(suggestionPrompt, companyName)}}}},
		Tools:    []geminiTool{{GoogleSearch: map[string]any{}}},
	}
	bodyJSON, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.Model, c.APIKey)
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(bodyJSON))
	if err != nil {
		return nil, err
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("gemini API returned %d: %s", resp.StatusCode, string(respBody))
	}

	var parsed geminiResponse
	if err := json.Unmarshal(respBody, &parsed); err != nil {
		return nil, fmt.Errorf("failed to parse gemini response: %w", err)
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return nil, fmt.Errorf("gemini response had no candidates")
	}

	text := extractJSON(parsed.Candidates[0].Content.Parts[0].Text)

	var fixture Fixture
	if err := json.Unmarshal([]byte(text), &fixture); err != nil {
		return nil, fmt.Errorf("failed to parse gemini suggestion JSON %q: %w", text, err)
	}
	fixture.Source = "gemini"

	if fixture.Group == "" && fixture.LegalName == "" {
		return nil, nil
	}
	return &fixture, nil
}

// extractJSON strips markdown code fences the model sometimes adds despite
// being asked not to (e.g. "```json\n{...}\n```").
func extractJSON(text string) string {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	return strings.TrimSpace(text)
}
