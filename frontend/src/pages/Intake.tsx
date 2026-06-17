import { useState } from "react";
import { api, ApiError } from "../api/client";
import { BandBadge } from "../components/BandBadge";
import { MatchedSignalsTable } from "../components/MatchedSignalsTable";
import type { CheckResult, Prospect } from "../types";

interface FormState {
  raw_company_name: string;
  npwp: string;
  nib: string;
  occupation_lob: string;
  address: string;
  headcount: string;
}

const initialForm: FormState = {
  raw_company_name: "",
  npwp: "",
  nib: "",
  occupation_lob: "",
  address: "",
  headcount: "",
};

export function Intake() {
  const [form, setForm] = useState<FormState>(initialForm);
  const [result, setResult] = useState<CheckResult | null>(null);
  const [prospect, setProspect] = useState<Prospect | null>(null);
  const [lockMessage, setLockMessage] = useState<string | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [busy, setBusy] = useState(false);
  const [expanded, setExpanded] = useState<string | null>(null);

  function buildBody() {
    return {
      raw_company_name: form.raw_company_name,
      npwp: form.npwp || undefined,
      nib: form.nib || undefined,
      occupation_lob: form.occupation_lob || undefined,
      address: form.address || undefined,
      headcount: form.headcount ? Number(form.headcount) : undefined,
    };
  }

  async function handleCheck() {
    setError(null);
    setLockMessage(null);
    setBusy(true);
    try {
      const r = await api.checkProspect(buildBody());
      setResult(r);
      setProspect(null);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleSaveIndicative() {
    setError(null);
    setLockMessage(null);
    setBusy(true);
    try {
      const { prospect, check } = await api.createProspect(buildBody());
      setResult(check);
      setProspect(prospect);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusy(false);
    }
  }

  async function handleRequestProtection() {
    if (!form.npwp && !form.nib) {
      setError("NPWP or NIB is required to request a protection lock.");
      return;
    }
    setError(null);
    setLockMessage(null);
    setBusy(true);
    try {
      const { prospect, check } = await api.createProspect(buildBody());
      setResult(check);
      setProspect(prospect);
      const lock = await api.lockProspect(prospect.id);
      setLockMessage(
        lock.protected
          ? `Protection granted. Expires at ${lock.expires_at ?? "?"}.`
          : "Lock request did not succeed."
      );
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setLockMessage("This company is already under protection by another branch (no identity revealed). A conflict has been flagged for compliance.");
      } else {
        setError(err instanceof ApiError ? err.message : String(err));
      }
    } finally {
      setBusy(false);
    }
  }

  return (
    <div>
      <h2>Prospect Intake</h2>
      <div className="card">
        <h3 style={{ marginTop: 0 }}>Indicative gate</h3>
        <p style={{ color: "#666", fontSize: 13 }}>
          Minimal data, no unique key required. No protection lock is granted from this gate alone.
        </p>
        <div className="field">
          <label>Company name *</label>
          <input
            value={form.raw_company_name}
            onChange={(e) => setForm({ ...form, raw_company_name: e.target.value })}
            placeholder="PT Contoh Sejahtera"
          />
        </div>
        <div className="field">
          <label>Occupation / Line of business</label>
          <input
            value={form.occupation_lob}
            onChange={(e) => setForm({ ...form, occupation_lob: e.target.value })}
          />
        </div>
        <div className="field">
          <label>Address</label>
          <input value={form.address} onChange={(e) => setForm({ ...form, address: e.target.value })} />
        </div>
        <div className="field">
          <label>Headcount</label>
          <input
            type="number"
            value={form.headcount}
            onChange={(e) => setForm({ ...form, headcount: e.target.value })}
          />
        </div>

        <h3>Protection gate</h3>
        <p style={{ color: "#666", fontSize: 13 }}>
          NPWP or NIB is required to request a protection lock.
        </p>
        <div className="field">
          <label>NPWP</label>
          <input value={form.npwp} onChange={(e) => setForm({ ...form, npwp: e.target.value })} placeholder="03.123.456.7-801.000" />
        </div>
        <div className="field">
          <label>NIB</label>
          <input value={form.nib} onChange={(e) => setForm({ ...form, nib: e.target.value })} />
        </div>

        <div style={{ display: "flex", gap: 8 }}>
          <button className="btn secondary" disabled={busy || !form.raw_company_name} onClick={handleCheck}>
            Check only
          </button>
          <button className="btn secondary" disabled={busy || !form.raw_company_name} onClick={handleSaveIndicative}>
            Save as indicative
          </button>
          <button className="btn" disabled={busy || !form.raw_company_name} onClick={handleRequestProtection}>
            Request protection lock
          </button>
        </div>
        {error && <p className="error-text" style={{ marginTop: 12 }}>{error}</p>}
        {lockMessage && <p style={{ marginTop: 12, color: "#2d6cdf" }}>{lockMessage}</p>}
      </div>

      {result && (
        <div className="card">
          <div style={{ display: "flex", alignItems: "center", gap: 12 }}>
            <h3 style={{ margin: 0 }}>Result</h3>
            <BandBadge band={result.band} />
            {result.reason && <span style={{ color: "#666", fontSize: 13 }}>{result.reason}</span>}
          </div>
          {prospect && <p style={{ fontSize: 13, color: "#666" }}>Prospect ID: {prospect.id}</p>}
          <p>Composite score: {result.composite_score}</p>

          {result.existing_client.found && (
            <div style={{ marginBottom: 12 }}>
              <strong>Existing client</strong>
              <p style={{ fontSize: 13 }}>
                Policy {result.existing_client.policy_no} ({result.existing_client.status}), ends{" "}
                {result.existing_client.period_end}
                {result.existing_client.owning_branch && ` — branch ${result.existing_client.owning_branch}`}
              </p>
            </div>
          )}

          <div style={{ marginBottom: 12 }}>
            <strong>Lock status:</strong>{" "}
            {result.lock.protected ? `Protected (expires ${result.lock.expires_at ?? "?"})` : "Not locked"}
          </div>

          <strong>Candidates</strong>
          {result.candidates.length === 0 && <p style={{ color: "#888", fontSize: 13 }}>No candidates found.</p>}
          {result.candidates.map((c) => (
            <div key={c.company_id} style={{ borderTop: "1px solid #eee", paddingTop: 8, marginTop: 8 }}>
              <div
                style={{ display: "flex", justifyContent: "space-between", cursor: "pointer" }}
                onClick={() => setExpanded(expanded === c.company_id ? null : c.company_id)}
              >
                <span>{c.legal_name}</span>
                <span>score {c.score.toFixed(2)} {expanded === c.company_id ? "▲" : "▼"}</span>
              </div>
              {expanded === c.company_id && (
                <div style={{ marginTop: 8 }}>
                  <MatchedSignalsTable signals={c.matched_signals} />
                </div>
              )}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
