import { Fragment, useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import type { AuditLog } from "../types";

export function AuditLogPage() {
  const [logs, setLogs] = useState<AuditLog[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [expanded, setExpanded] = useState<string | null>(null);

  useEffect(() => {
    api
      .listAuditLogs()
      .then(setLogs)
      .catch((err) => setError(err instanceof ApiError ? err.message : String(err)));
  }, []);

  if (error) return <p className="error-text">{error}</p>;

  return (
    <div>
      <h2>Audit Log</h2>
      <div className="card" style={{ padding: 0 }}>
        <table>
          <thead>
            <tr style={{ textAlign: "left", borderBottom: "1px solid #eee" }}>
              <th style={{ padding: 10 }}>When</th>
              <th style={{ padding: 10 }}>Actor</th>
              <th style={{ padding: 10 }}>Action</th>
              <th style={{ padding: 10 }}>Entity</th>
              <th style={{ padding: 10 }}></th>
            </tr>
          </thead>
          <tbody>
            {logs.map((l) => (
              <Fragment key={l.id}>
                <tr style={{ borderBottom: "1px solid #f3f3f3" }}>
                  <td style={{ padding: 10 }}>{new Date(l.created_at).toLocaleString()}</td>
                  <td style={{ padding: 10 }}>{l.actor_id ?? "system"}</td>
                  <td style={{ padding: 10 }}>{l.action}</td>
                  <td style={{ padding: 10 }}>
                    {l.entity_type} {l.entity_id ?? ""}
                  </td>
                  <td style={{ padding: 10 }}>
                    <button className="btn secondary" onClick={() => setExpanded(expanded === l.id ? null : l.id)}>
                      {expanded === l.id ? "Hide" : "Details"}
                    </button>
                  </td>
                </tr>
                {expanded === l.id && (
                  <tr>
                    <td colSpan={5} style={{ padding: 10, background: "#fafafa" }}>
                      <div style={{ display: "flex", gap: 16 }}>
                        <div style={{ flex: 1 }}>
                          <strong>Before</strong>
                          <pre style={{ fontSize: 12, whiteSpace: "pre-wrap" }}>
                            {l.before ? JSON.stringify(l.before, null, 2) : "—"}
                          </pre>
                        </div>
                        <div style={{ flex: 1 }}>
                          <strong>After</strong>
                          <pre style={{ fontSize: 12, whiteSpace: "pre-wrap" }}>
                            {l.after ? JSON.stringify(l.after, null, 2) : "—"}
                          </pre>
                        </div>
                      </div>
                    </td>
                  </tr>
                )}
              </Fragment>
            ))}
            {logs.length === 0 && (
              <tr>
                <td colSpan={5} style={{ padding: 16, color: "#888" }}>
                  No audit entries.
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </div>
  );
}
