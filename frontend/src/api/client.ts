import type {
  AuditLog,
  CheckResult,
  Company,
  CompanyRelationship,
  EnrichmentJob,
  MatchResult,
  MonitoringSummary,
  Prospect,
  ReviewQueueResponse,
  User,
} from "../types";

const API_BASE = import.meta.env.VITE_API_BASE || "http://localhost:8080/api/v1";

export class ApiError extends Error {
  status: number;
  constructor(status: number, message: string) {
    super(message);
    this.status = status;
  }
}

let currentUserId: string | null = null;

export function setCurrentUserId(id: string | null) {
  currentUserId = id;
}

async function request<T>(method: string, path: string, body?: unknown): Promise<T> {
  const headers: Record<string, string> = { "Content-Type": "application/json" };
  if (currentUserId) headers["X-Demo-User-Id"] = currentUserId;

  const res = await fetch(`${API_BASE}${path}`, {
    method,
    headers,
    body: body !== undefined ? JSON.stringify(body) : undefined,
  });

  if (!res.ok) {
    let message = res.statusText;
    try {
      const data = await res.json();
      if (data?.error) message = data.error;
    } catch {
      // body wasn't JSON; fall back to statusText
    }
    throw new ApiError(res.status, message);
  }
  if (res.status === 204) return undefined as T;
  return (await res.json()) as T;
}

export const api = {
  me: () => request<User>("GET", "/me"),
  listUsers: () => request<User[]>("GET", "/users"),

  checkProspect: (body: Record<string, unknown>) =>
    request<CheckResult>("POST", "/prospects/check", body),
  createProspect: (body: Record<string, unknown>) =>
    request<{ prospect: Prospect; check: CheckResult }>("POST", "/prospects", body),
  listProspects: () => request<Prospect[]>("GET", "/prospects"),
  getProspect: (id: string) => request<Prospect>("GET", `/prospects/${id}`),
  lockProspect: (id: string) =>
    request<{ protected: boolean; expires_at?: string }>("POST", `/prospects/${id}/lock`),
  routeToUW: (id: string) => request<Prospect>("POST", `/prospects/${id}/route-uw`),
  closeUW: (id: string, rationale: string) =>
    request<Prospect>("POST", `/prospects/${id}/close-uw`, { rationale }),

  searchCompanies: (q: string) => request<Company[]>("GET", `/companies?q=${encodeURIComponent(q)}`),
  getCompany: (id: string) => request<Company>("GET", `/companies/${id}`),
  createCompany: (body: Record<string, unknown>) => request<Company>("POST", "/companies", body),
  listCompanyRelationships: (id: string) =>
    request<CompanyRelationship[]>("GET", `/companies/${id}/relationships`),
  createRelationship: (body: Record<string, unknown>) =>
    request<CompanyRelationship>("POST", "/relationships", body),
  verifyRelationship: (id: string, verification_status: "verified" | "unverified") =>
    request<CompanyRelationship>("PATCH", `/relationships/${id}/verify`, { verification_status }),

  getReviewQueue: () => request<ReviewQueueResponse>("GET", "/review-queue"),
  makerDecision: (id: string, decision: string) =>
    request<MatchResult>("POST", `/match-results/${id}/maker`, { decision }),
  checkerDecision: (id: string, approve: boolean) =>
    request<MatchResult>("POST", `/match-results/${id}/checker`, { approve }),

  enqueueEnrichmentJob: (company_name: string) =>
    request<EnrichmentJob>("POST", "/enrichment/jobs", { company_name }),
  getEnrichmentJob: (id: string) => request<EnrichmentJob>("GET", `/enrichment/jobs/${id}`),

  listAuditLogs: () => request<AuditLog[]>("GET", "/audit-logs"),
  getMonitoringSummary: () => request<MonitoringSummary>("GET", "/monitoring/summary"),
};
