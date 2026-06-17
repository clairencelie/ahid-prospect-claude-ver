export type Role =
  | "marketing"
  | "branch_co"
  | "underwriter"
  | "compliance"
  | "audit"
  | "admin";

export type Band = "PASS" | "REVIEW" | "BLOCK";

export type VerificationStatus = "unverified" | "verified";

export interface Branch {
  id: string;
  name: string;
  region?: string;
}

export interface User {
  id: string;
  name: string;
  role: Role;
  branch_id?: string;
}

export interface Company {
  id: string;
  npwp?: string;
  nib?: string;
  legal_name: string;
  normalized_name: string;
  brand_names: string[];
  domain?: string;
  phone?: string;
  address?: string;
  occupation_lob?: string;
  group_id?: string;
  source: string;
  confidence: number;
  verification_status: VerificationStatus;
  created_at: string;
  updated_at: string;
}

export interface CompanyRelationship {
  id: string;
  parent_company_id: string;
  child_company_id: string;
  relation_type: "subsidiary_of" | "same_group" | "brand_of" | "dba";
  confidence: number;
  source: string;
  verification_status: VerificationStatus;
  created_at: string;
}

export interface Prospect {
  id: string;
  company_id?: string;
  raw_company_name: string;
  npwp?: string;
  nib?: string;
  occupation_lob?: string;
  address?: string;
  domain?: string;
  phone?: string;
  headcount?: number;
  source_business?: string;
  branch_id?: string;
  marketing_id?: string;
  stage: "indicative" | "protection" | "spq" | "uw" | "closed";
  status: string;
  tanggal_masuk_uw?: string;
  tanggal_keluar_uw?: string;
  uw_rationale?: string;
  created_at: string;
}

export interface MatchedSignal {
  signal: string;
  similarity: number;
  contribution: number;
}

export interface Candidate {
  company_id: string;
  legal_name: string;
  score: number;
  matched_signals: MatchedSignal[];
}

export interface MatchResult {
  id: string;
  prospect_id: string;
  candidate_company_id?: string;
  composite_score: number;
  band: Band;
  reason: string;
  matched_signals: MatchedSignal[];
  decided_by: string;
  maker_id?: string;
  checker_id?: string;
  decision?: string;
  created_at: string;
}

export interface ProspectLock {
  id: string;
  prospect_id: string;
  company_id?: string;
  npwp: string;
  branch_id?: string;
  marketing_id?: string;
  locked_at: string;
  expires_at: string;
  status: "active" | "released" | "in_conflict";
  release_reason?: string;
}

export interface ExistingClientInfo {
  found: boolean;
  policy_no?: string;
  status?: string;
  period_end?: string;
  owning_branch?: string;
}

export interface LockInfo {
  protected: boolean;
  expires_at?: string;
}

export interface CheckResult {
  band: Band;
  reason: string;
  composite_score: number;
  candidates: Candidate[];
  existing_client: ExistingClientInfo;
  lock: LockInfo;
}

export interface EnrichmentJob {
  id: string;
  company_name: string;
  normalized_name: string;
  status: "queued" | "running" | "done" | "failed";
  result?: unknown;
  requested_at: string;
  completed_at?: string;
}

export interface AuditLog {
  id: string;
  actor_id?: string;
  action: string;
  entity_type: string;
  entity_id?: string;
  before?: unknown;
  after?: unknown;
  created_at: string;
}

export interface MonitoringSummary {
  band_counts: Record<string, number>;
  review_queue_size: number;
  lock_conflict_queue_size: number;
  locks_active: number;
  locks_expired: number;
  edges_verified: number;
  edges_unverified: number;
}

export interface ReviewQueueResponse {
  match_reviews: MatchResult[];
  lock_conflicts: ProspectLock[];
}
