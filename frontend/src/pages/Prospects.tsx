import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { Prospect } from "../types";

export function Prospects() {
  const [prospects, setProspects] = useState<Prospect[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);
  const [message, setMessage] = useState<string | null>(null);

  async function load() {
    try {
      setProspects(await api.listProspects());
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleRouteToUW(id: string) {
    setBusyId(id);
    setMessage(null);
    setError(null);
    try {
      await api.routeToUW(id);
      setMessage("Routed to UW.");
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  async function handleLock(id: string) {
    setBusyId(id);
    setMessage(null);
    setError(null);
    try {
      const lock = await api.lockProspect(id);
      setMessage(lock.protected ? `Locked, expires ${lock.expires_at ?? "?"}` : "Lock not granted.");
    } catch (err) {
      if (err instanceof ApiError && err.status === 409) {
        setMessage("Already protected by another branch — no identity leaked, conflict flagged for compliance.");
      } else {
        setError(err instanceof ApiError ? err.message : String(err));
      }
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div>
      <h2>Prospects</h2>
      {error && <p className="error-text">{error}</p>}
      {message && <p style={{ color: "#2d6cdf" }}>{message}</p>}
      <div className="card" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr style={{ textAlign: "left", borderBottom: "1px solid #eee" }}>
              <th style={{ padding: 10 }}>Company</th>
              <th style={{ padding: 10 }}>NPWP/NIB</th>
              <th style={{ padding: 10 }}>Stage</th>
              <th style={{ padding: 10 }}>Created</th>
              <th style={{ padding: 10 }}>Actions</th>
            </tr>
          </thead>
          <tbody>
            {prospects.map((p) => (
              <tr key={p.id} style={{ borderBottom: "1px solid #f3f3f3" }}>
                <td style={{ padding: 10 }}>{p.raw_company_name}</td>
                <td style={{ padding: 10 }}>{p.npwp || p.nib || "—"}</td>
                <td style={{ padding: 10 }}>{p.stage}</td>
                <td style={{ padding: 10 }}>{new Date(p.created_at).toLocaleString()}</td>
                <td style={{ padding: 10, display: "flex", gap: 6 }}>
                  <button
                    className="btn secondary"
                    disabled={busyId === p.id || !(p.npwp || p.nib)}
                    onClick={() => handleLock(p.id)}
                  >
                    Lock
                  </button>
                  <button
                    className="btn secondary"
                    disabled={busyId === p.id || p.stage === "uw"}
                    onClick={() => handleRouteToUW(p.id)}
                  >
                    Route to UW
                  </button>
                </td>
              </tr>
            ))}
            {prospects.length === 0 && (
              <tr>
                <td colSpan={5} style={{ padding: 16, color: "#888" }}>
                  No prospects yet.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
