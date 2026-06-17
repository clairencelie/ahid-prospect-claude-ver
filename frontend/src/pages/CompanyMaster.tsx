import { useEffect, useState } from "react";
import { api, ApiError } from "../api/client";
import { useRole } from "../context/RoleContext";
import type { Company, CompanyRelationship } from "../types";

function VerificationBadge({ status }: { status: "verified" | "unverified" }) {
  const verified = status === "verified";
  return (
    <span
      style={{
        fontSize: 11,
        padding: "2px 8px",
        borderRadius: 10,
        background: verified ? "#e6f4ea" : "#fff4e0",
        color: verified ? "#1e7e34" : "#b8860b",
      }}
    >
      {verified ? "verified" : "unverified (suggested)"}
    </span>
  );
}

export function CompanyMaster() {
  const { currentUser } = useRole();
  const [query, setQuery] = useState("");
  const [companies, setCompanies] = useState<Company[]>([]);
  const [allCompanies, setAllCompanies] = useState<Company[]>([]);
  const [selected, setSelected] = useState<Company | null>(null);
  const [relationships, setRelationships] = useState<CompanyRelationship[]>([]);
  const [error, setError] = useState<string | null>(null);
  const [busyId, setBusyId] = useState<string | null>(null);

  const isAdmin = currentUser?.role === "admin";

  async function search(q: string) {
    try {
      setCompanies(await api.searchCompanies(q));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }

  useEffect(() => {
    search("");
    api.searchCompanies("").then(setAllCompanies).catch(() => {});
  }, []);

  function nameOf(companyId: string): string {
    return allCompanies.find((c) => c.id === companyId)?.legal_name ?? companyId;
  }

  async function selectCompany(c: Company) {
    setSelected(c);
    try {
      setRelationships(await api.listCompanyRelationships(c.id));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    }
  }

  async function handleVerify(id: string, status: "verified" | "unverified") {
    setBusyId(id);
    setError(null);
    try {
      await api.verifyRelationship(id, status);
      if (selected) setRelationships(await api.listCompanyRelationships(selected.id));
    } catch (err) {
      setError(err instanceof ApiError ? err.message : String(err));
    } finally {
      setBusyId(null);
    }
  }

  return (
    <div>
      <h2>Company Master</h2>
      {error && <p className="error-text">{error}</p>}

      <div className="card">
        <input
          placeholder="Search by name..."
          value={query}
          onChange={(e) => {
            setQuery(e.target.value);
            search(e.target.value);
          }}
          style={{ width: "100%", padding: 8, borderRadius: 6, border: "1px solid #ccc" }}
        />
      </div>

      <div style={{ display: "flex", gap: 16 }}>
        <div className="card" style={{ flex: 1, padding: 0 }}>
          {companies.map((c) => (
            <div
              key={c.id}
              onClick={() => selectCompany(c)}
              style={{
                padding: 10,
                borderBottom: "1px solid #f3f3f3",
                cursor: "pointer",
                background: selected?.id === c.id ? "#eef4ff" : "transparent",
              }}
            >
              <div>{c.legal_name}</div>
              <div style={{ fontSize: 12, color: "#888" }}>
                {c.npwp ?? "no NPWP"} · <VerificationBadge status={c.verification_status} />
              </div>
            </div>
          ))}
          {companies.length === 0 && <div style={{ padding: 16, color: "#888" }}>No companies found.</div>}
        </div>

        <div style={{ flex: 1.5 }}>
          {selected ? (
            <div className="card">
              <h3 style={{ marginTop: 0 }}>{selected.legal_name}</h3>
              <p style={{ fontSize: 13, color: "#666" }}>
                NPWP {selected.npwp ?? "—"} · NIB {selected.nib ?? "—"} · source {selected.source} ·{" "}
                <VerificationBadge status={selected.verification_status} />
              </p>
              {selected.brand_names.length > 0 && (
                <p style={{ fontSize: 13 }}>Brand names: {selected.brand_names.join(", ")}</p>
              )}
              <p style={{ fontSize: 13 }}>Domain: {selected.domain ?? "—"} · Phone: {selected.phone ?? "—"}</p>
              <p style={{ fontSize: 13 }}>Address: {selected.address ?? "—"}</p>

              <h4>Relationships</h4>
              {relationships.length === 0 && <p style={{ color: "#888", fontSize: 13 }}>No relationships.</p>}
              {relationships.map((r) => (
                <div key={r.id} style={{ borderTop: "1px solid #eee", padding: "8px 0", fontSize: 13 }}>
                  <div>
                    {nameOf(r.child_company_id)} —[{r.relation_type}]→ {nameOf(r.parent_company_id)}{" "}
                    <VerificationBadge status={r.verification_status} />
                  </div>
                  <div style={{ color: "#888" }}>
                    source {r.source}, confidence {r.confidence}
                  </div>
                  {isAdmin && r.verification_status === "unverified" && (
                    <div style={{ marginTop: 6, display: "flex", gap: 8 }}>
                      <button className="btn secondary" disabled={busyId === r.id} onClick={() => handleVerify(r.id, "verified")}>
                        Verify
                      </button>
                      <button className="btn secondary" disabled={busyId === r.id} onClick={() => handleVerify(r.id, "unverified")}>
                        Keep unverified
                      </button>
                    </div>
                  )}
                  {isAdmin && r.verification_status === "verified" && (
                    <div style={{ marginTop: 6 }}>
                      <button className="btn secondary" disabled={busyId === r.id} onClick={() => handleVerify(r.id, "unverified")}>
                        Revert to unverified
                      </button>
                    </div>
                  )}
                </div>
              ))}
            </div>
          ) : (
            <div className="card" style={{ color: "#888" }}>
              Select a company to see details.
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
