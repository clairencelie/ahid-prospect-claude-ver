import type { MatchedSignal } from "../types";

export function MatchedSignalsTable({ signals }: { signals: MatchedSignal[] }) {
  if (signals.length === 0) {
    return <p style={{ color: "#888", fontSize: 13 }}>No signals matched.</p>;
  }
  return (
    <table style={{ width: "100%", fontSize: 13, borderCollapse: "collapse" }}>
      <thead>
        <tr style={{ textAlign: "left", color: "#666" }}>
          <th style={{ padding: "4px 8px" }}>Signal</th>
          <th style={{ padding: "4px 8px" }}>Similarity</th>
          <th style={{ padding: "4px 8px" }}>Contribution</th>
        </tr>
      </thead>
      <tbody>
        {signals.map((s, i) => (
          <tr key={i} style={{ borderTop: "1px solid #eee" }}>
            <td style={{ padding: "4px 8px" }}>{s.signal}</td>
            <td style={{ padding: "4px 8px" }}>{s.similarity.toFixed(2)}</td>
            <td style={{ padding: "4px 8px" }}>{s.contribution.toFixed(2)}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
