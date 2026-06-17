import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import { BandBadge } from "../components/BandBadge";
import { MatchedSignalsTable } from "../components/MatchedSignalsTable";
import { useRole } from "../context/RoleContext";
import type { MatchResult, ProspectLock } from "../types";

const MAKER_DECISIONS = ["confirm_duplicate", "mark_distinct", "assign_owner"] as const;

export function ReviewQueue() {
  const { currentUser } = useRole();
  const [matchReviews, setMatchReviews] = useState<MatchResult[]>([]);
  const [lockConflicts, setLockConflicts] = useState<ProspectLock[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  async function load() {
    try {
      const res = await api.getReviewQueue();
      setMatchReviews(res.match_reviews);
      setLockConflicts(res.lock_conflicts);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleMaker(id: string, decision: string) {
    setBusyId(id);
    setError(null);
    try {
      await api.makerDecision(id, decision);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleChecker(id: string, approve: boolean) {
    setBusyId(id);
    setError(null);
    try {
      await api.checkerDecision(id, approve);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  const isMaker = currentUser?.role === "branch_co";
  const isChecker = currentUser?.role === "compliance";

  return (
    <div>
      <h2>Review Queue</h2>
      {error && <p className="error-text">{error}</p>}

      <h3>REVIEW-band match results</h3>
      {matchReviews.length === 0 && <p style={{ color: "#888" }}>Nothing pending.</p>}
      {matchReviews.map((m) => (
        <div key={m.id} className="card">
          <div style={{ display: "flex", justifyContent: "space-between", alignItems: "center" }}>
            <div style={{ display: "flex", gap: 12, alignItems: "center" }}>
              <BandBadge band={m.band} />
              <span style={{ fontSize: 13, color: "#666" }}>score {m.composite_score} — {m.reason}</span>
            </div>
            <button className="btn secondary" onClick={() => setExpanded(expanded === m.id ? null : m.id)}>
              {expanded === m.id ? "Hide signals" : "Show signals"}
            </button>
          </div>
          <p style={{ fontSize: 13, color: "#888" }}>Prospect: {m.prospect_id}</p>
          {expanded === m.id && <MatchedSignalsTable signals={m.matched_signals} />}

          <div style={{ marginTop: 12, fontSize: 13 }}>
            Maker: {m.maker_id ? `${m.maker_id} → ${m.decision}` : "pending"} | Checker: {m.checker_id ?? "pending"}
          </div>

          {isMaker && !m.maker_id && (
            <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
              {MAKER_DECISIONS.map((d) => (
                <button key={d} className="btn secondary" disabled={busyId === m.id} onClick={() => handleMaker(m.id, d)}>
                  {d}
                </button>
              ))}
            </div>
          )}

          {isChecker && m.maker_id && !m.checker_id && (
            <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
              <button className="btn" disabled={busyId === m.id} onClick={() => handleChecker(m.id, true)}>
                Approve
              </button>
              <button className="btn secondary" disabled={busyId === m.id} onClick={() => handleChecker(m.id, false)}>
                Reject (send back to maker)
              </button>
            </div>
          )}
        </div>
      ))}

      <h3>Lock conflicts</h3>
      {lockConflicts.length === 0 && <p style={{ color: "#888" }}>No conflicts.</p>}
      {lockConflicts.map((l) => (
        <div key={l.id} className="card">
          <p style={{ fontSize: 13 }}>
            NPWP {l.npwp} — prospect {l.prospect_id} — branch {l.branch_id ?? "?"} — locked at{" "}
            {new Date(l.locked_at).toLocaleString()}
          </p>
        </div>
      ))}
    </div>
  );
}
