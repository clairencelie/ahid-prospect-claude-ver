import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { MonitoringSummary } from "../types";

function Card({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="card" style={{ flex: 1, textAlign: "center" }}>
      <div style={{ fontSize: 28, fontWeight: 700 }}>{value}</div>
      <div style={{ fontSize: 13, color: "#666" }}>{label}</div>
    </div>
  );
}

export function Monitoring() {
  const [summary, setSummary] = useState<MonitoringSummary | null>(null);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    api.getMonitoringSummary().then(setSummary).catch((err) => setError(err instanceof ApiError ? err.message : String(err)));
  }, []);

  if (error) return <p className="error-text">{error}</p>;
  if (!summary) return <p>Loading...</p>;

  return (
    <div>
      <h2>Monitoring</h2>
      <h3>Decisions by band</h3>
      <div style={{ display: "flex", gap: 12 }}>
        <Card label="PASS" value={summary.band_counts.PASS ?? 0} />
        <Card label="REVIEW" value={summary.band_counts.REVIEW ?? 0} />
        <Card label="BLOCK" value={summary.band_counts.BLOCK ?? 0} />
      </div>

      <h3>Queues</h3>
      <div style={{ display: "flex", gap: 12 }}>
        <Card label="Open review items" value={summary.review_queue_size} />
        <Card label="Lock conflicts" value={summary.lock_conflict_queue_size} />
      </div>

      <h3>Locks</h3>
      <div style={{ display: "flex", gap: 12 }}>
        <Card label="Active" value={summary.locks_active} />
        <Card label="Expired" value={summary.locks_expired} />
      </div>

      <h3>Relationship edges</h3>
      <div style={{ display: "flex", gap: 12 }}>
        <Card label="Verified" value={summary.edges_verified} />
        <Card label="AI-suggested (unverified)" value={summary.edges_unverified} />
      </div>
    </div>
  );
}
