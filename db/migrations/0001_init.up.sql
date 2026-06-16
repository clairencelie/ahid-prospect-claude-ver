CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- branches & users
CREATE TABLE branches (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  region text
);

CREATE TABLE users (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  name text NOT NULL,
  role text NOT NULL CHECK (role IN ('marketing','branch_co','underwriter','compliance','audit','admin')),
  branch_id uuid REFERENCES branches(id)
);

-- company master + graph
CREATE TABLE companies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  npwp text,                 -- nullable, unique when present (normalized, digits only)
  nib text,
  legal_name text NOT NULL,
  normalized_name text NOT NULL,
  brand_names text[] DEFAULT '{}',
  domain text,
  phone text,                -- normalized
  address text,
  occupation_lob text,
  group_id uuid,             -- nullable; convenience pointer to a resolved group root
  source text DEFAULT 'manual',          -- manual | core | ai_mock
  confidence numeric DEFAULT 1.0,
  verification_status text DEFAULT 'verified' CHECK (verification_status IN ('unverified','verified')),
  created_at timestamptz DEFAULT now(),
  updated_at timestamptz DEFAULT now()
);
CREATE UNIQUE INDEX ux_companies_npwp ON companies(npwp) WHERE npwp IS NOT NULL;
CREATE INDEX ix_companies_normname ON companies(normalized_name);

CREATE TABLE company_relationships (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  parent_company_id uuid NOT NULL REFERENCES companies(id),
  child_company_id uuid NOT NULL REFERENCES companies(id),
  relation_type text NOT NULL CHECK (relation_type IN ('subsidiary_of','same_group','brand_of','dba')),
  confidence numeric DEFAULT 1.0,
  source text DEFAULT 'manual',          -- manual | ai_mock
  verification_status text DEFAULT 'verified' CHECK (verification_status IN ('unverified','verified')),
  created_at timestamptz DEFAULT now(),
  UNIQUE (parent_company_id, child_company_id, relation_type)
);

-- prospects & locks
CREATE TABLE prospects (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  company_id uuid REFERENCES companies(id),   -- null until resolved
  raw_company_name text NOT NULL,
  npwp text,
  nib text,
  occupation_lob text,
  address text,
  domain text,
  phone text,
  headcount int,
  source_business text,                       -- direct | banking | smg | agency | broker
  branch_id uuid REFERENCES branches(id),
  marketing_id uuid REFERENCES users(id),
  stage text DEFAULT 'indicative' CHECK (stage IN ('indicative','protection','spq','uw','closed')),
  status text DEFAULT 'open',
  tanggal_masuk_uw timestamptz,
  tanggal_keluar_uw timestamptz,
  uw_rationale text,
  created_at timestamptz DEFAULT now()
);

CREATE TABLE prospect_locks (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  prospect_id uuid NOT NULL REFERENCES prospects(id),
  company_id uuid REFERENCES companies(id),
  npwp text,
  branch_id uuid REFERENCES branches(id),
  marketing_id uuid REFERENCES users(id),
  locked_at timestamptz DEFAULT now(),
  expires_at timestamptz NOT NULL,
  status text DEFAULT 'active' CHECK (status IN ('active','released','in_conflict')),
  release_reason text
);
CREATE UNIQUE INDEX ux_active_lock_npwp ON prospect_locks(npwp) WHERE status = 'active';

CREATE TABLE match_results (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  prospect_id uuid REFERENCES prospects(id),
  candidate_company_id uuid REFERENCES companies(id),
  composite_score numeric NOT NULL,
  band text NOT NULL CHECK (band IN ('PASS','REVIEW','BLOCK')),
  reason text,                                -- e.g. existing_active_policy, hard_key_npwp, same_group
  matched_signals jsonb NOT NULL DEFAULT '[]',
  decided_by text DEFAULT 'system',           -- system | user
  maker_id uuid REFERENCES users(id),
  checker_id uuid REFERENCES users(id),
  decision text,                              -- confirm_duplicate | mark_distinct | assign_owner
  created_at timestamptz DEFAULT now()
);

CREATE TABLE enrichment_jobs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  company_name text NOT NULL,
  normalized_name text NOT NULL,
  status text DEFAULT 'queued' CHECK (status IN ('queued','running','done','failed')),
  result jsonb,
  requested_at timestamptz DEFAULT now(),
  completed_at timestamptz
);

-- core-system stand-in (mock); replaced by adapter to real API in production
CREATE TABLE clients (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  npwp text,
  client_name text NOT NULL,
  owning_branch_id uuid REFERENCES branches(id)
);
CREATE TABLE policies (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  client_id uuid REFERENCES clients(id),
  npwp text,
  policy_no text NOT NULL,
  status text NOT NULL CHECK (status IN ('active','expired')),
  period_start date,
  period_end date,
  owning_branch_id uuid REFERENCES branches(id)
);

CREATE TABLE audit_logs (
  id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  actor_id uuid REFERENCES users(id),
  action text NOT NULL,
  entity_type text NOT NULL,
  entity_id uuid,
  before jsonb,
  after jsonb,
  created_at timestamptz DEFAULT now()
);

CREATE TABLE settings (
  key text PRIMARY KEY,
  value text NOT NULL
);
