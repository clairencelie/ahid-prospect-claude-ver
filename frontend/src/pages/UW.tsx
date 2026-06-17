import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import { useRole } from "../context/RoleContext";
import type { Prospect } from "../types";

function formatDuration(startISO: string, endISO?: string): string {
  const start = new Date(startISO).getTime();
  const end = endISO ? new Date(endISO).getTime() : Date.now();
  const ms = Math.max(0, end - start);
  const minutes = Math.floor(ms / 60000);
  if (minutes < 60) return `${minutes}m`;
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return `${hours}h ${minutes % 60}m`;
  const days = Math.floor(hours / 24);
  return `${days}d ${hours % 24}h`;
}

export function UW() {
  const { currentUser } = useRole();
  const [prospects, setProspects] = useState<Prospect[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [rationales, setRationales] = useState<Record<string, string>>({});
  const [busyId, setBusyId] = useState<string | null>(null);

  async function load() {
    try {
      const all = await api.listProspects();
      setProspects(all.filter((p) => p.tanggal_masuk_uw));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }

  useEffect(() => {
    load();
  }, []);

  async function handleClose(id: string) {
    const rationale = rationales[id]?.trim();
    if (!rationale) {
      setError("Rationale is required to close a UW item.");
      return;
    }
    setBusyId(id);
    setError(null);
    try {
      await api.closeUW(id, rationale);
      await load();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  const isUW = currentUser?.role === "underwriter";

  return (
    <div>
      <h2>Underwriting (stub)</h2>
      {error && <p className="error-text">{error}</p>}
      {prospects.length === 0 && <p style={{ color: "#888" }}>No prospects routed to UW.</p>}
      {prospects.map((p) => (
        <div key={p.id} className="card">
          <div style={{ display: "flex", justifyContent: "space-between" }}>
            <strong>{p.raw_company_name}</strong>
            <span style={{ fontSize: 13, color: "#666" }}>
              SLA: {formatDuration(p.tanggal_masuk_uw!, p.tanggal_keluar_uw)}
            </span>
          </div>
          <p style={{ fontSize: 13, color: "#888" }}>
            In: {new Date(p.tanggal_masuk_uw!).toLocaleString()}
            {p.tanggal_keluar_uw && ` — Out: ${new Date(p.tanggal_keluar_uw).toLocaleString()}`}
          </p>
          {p.uw_rationale && <p style={{ fontSize: 13 }}>Rationale: {p.uw_rationale}</p>}

          {isUW && !p.tanggal_keluar_uw && (
            <div style={{ display: "flex", gap: 8, marginTop: 8 }}>
              <input
                placeholder="Rationale"
                style={{ flex: 1, padding: 8, borderRadius: 6, border: "1px solid #ccc" }}
                value={rationales[p.id] ?? ""}
                onChange={(e) => setRationales({ ...rationales, [p.id]: e.target.value })}
              />
              <button className="btn" disabled={busyId === p.id} onClick={() => handleClose(p.id)}>
                Close
              </button>
            </div>
          )}
        </div>
      ))}
    </div>
  );
}
