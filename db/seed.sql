-- Demo seed data for AHID Prospect Protection & Auto-Verification System.
-- Hardcoded UUIDs throughout so the demo scenarios in PRD §14 are reproducible via curl/UI.

-- settings (thresholds; FR3.6)
INSERT INTO settings (key, value) VALUES
  ('block_threshold', '80'),
  ('review_threshold', '50'),
  ('lock_ttl_days', '30')
ON CONFLICT (key) DO NOTHING;

-- branches
INSERT INTO branches (id, name, region) VALUES
  ('11111111-1111-1111-1111-111111111101', 'Branch Surabaya', 'East Java'),
  ('11111111-1111-1111-1111-111111111102', 'Branch Jakarta', 'Jakarta')
ON CONFLICT (id) DO NOTHING;

-- users: one per role, plus a second marketer so each branch has its own
INSERT INTO users (id, name, role, branch_id) VALUES
  ('22222222-2222-2222-2222-222222222201', 'Sari Marketing (Surabaya)', 'marketing', '11111111-1111-1111-1111-111111111101'),
  ('22222222-2222-2222-2222-222222222202', 'Budi Marketing (Jakarta)', 'marketing', '11111111-1111-1111-1111-111111111102'),
  ('22222222-2222-2222-2222-222222222203', 'Citra Branch Coordinator', 'branch_co', '11111111-1111-1111-1111-111111111101'),
  ('22222222-2222-2222-2222-222222222204', 'Dewi Underwriter', 'underwriter', '11111111-1111-1111-1111-111111111102'),
  ('22222222-2222-2222-2222-222222222205', 'Eko Compliance', 'compliance', NULL),
  ('22222222-2222-2222-2222-222222222206', 'Fajar Audit', 'audit', NULL),
  ('22222222-2222-2222-2222-222222222207', 'Gita Admin', 'admin', NULL)
ON CONFLICT (id) DO NOTHING;

-- ===== Group case: Dharma Wibawa Guna Group =====
-- Group root + 3 subsidiaries share corporate domain/phone (realistic: shared
-- back-office contact), so "same_group" plus those soft signals stack into
-- REVIEW without any one signal alone crossing a threshold.
INSERT INTO companies (id, npwp, nib, legal_name, normalized_name, brand_names, domain, phone, address, occupation_lob, group_id, source, verification_status) VALUES
  ('33333333-3333-3333-3333-333333333301', NULL, NULL, 'Dharma Wibawa Guna Group', 'DHARMA WIBAWA GUNA GROUP', '{}', 'dharmawibawaguna.co.id', '0312345600', 'Jl. Rungkut Industri No. 1, Surabaya', 'holding', NULL, 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333302', '011234567801000', NULL, 'PT Alam Semesta Agro', 'ALAM SEMESTA AGRO', '{}', 'dharmawibawaguna.co.id', '0312345601', 'Jl. Rungkut Industri No. 3, Surabaya', 'agriculture', '33333333-3333-3333-3333-333333333301', 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333303', '011234567802000', NULL, 'PT Bangun Sahabat Tani', 'BANGUN SAHABAT TANI', '{}', 'dharmawibawaguna.co.id', '0312345602', 'Jl. Rungkut Industri No. 5, Surabaya', 'agriculture', '33333333-3333-3333-3333-333333333301', 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333304', '011234567803000', NULL, 'PT Delta Giri Wacana', 'DELTA GIRI WACANA', '{}', 'dharmawibawaguna.co.id', '0312345603', 'Jl. Rungkut Industri No. 7, Surabaya', 'manufacturing', '33333333-3333-3333-3333-333333333301', 'manual', 'verified')
ON CONFLICT (id) DO NOTHING;

INSERT INTO company_relationships (parent_company_id, child_company_id, relation_type, confidence, source, verification_status) VALUES
  ('33333333-3333-3333-3333-333333333301', '33333333-3333-3333-3333-333333333302', 'same_group', 1.0, 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333301', '33333333-3333-3333-3333-333333333303', 'same_group', 1.0, 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333301', '33333333-3333-3333-3333-333333333304', 'same_group', 1.0, 'manual', 'verified')
ON CONFLICT DO NOTHING;

-- ===== Brand-vs-legal case =====
INSERT INTO companies (id, npwp, nib, legal_name, normalized_name, brand_names, domain, phone, address, occupation_lob, source, verification_status) VALUES
  ('33333333-3333-3333-3333-333333333305', '021234567801000', NULL, 'PT Saripuri Permai Hotel', 'SARIPURI PERMAI HOTEL', '{"SHANGRI-LA HOTEL SURABAYA"}', 'shangri-la.com', '0315551001', 'Jl. Mayjen Sungkono No. 120, Surabaya', 'hospitality', 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333306', '021234567802000', NULL, 'PT Logistik Canggih Indonesia', 'LOGISTIK CANGGIH INDONESIA', '{"LOGISLY"}', 'logisly.com', '0215551002', 'Jl. Sudirman Kav. 25, Jakarta', 'logistics', 'manual', 'verified')
ON CONFLICT (id) DO NOTHING;

-- ===== Existing-client case =====
INSERT INTO companies (id, npwp, nib, legal_name, normalized_name, domain, phone, address, occupation_lob, source, verification_status) VALUES
  ('33333333-3333-3333-3333-333333333307', '031234567801000', NULL, 'PT Cahaya Abadi Sejahtera', 'CAHAYA ABADI SEJAHTERA', 'cahayaabadi.co.id', '0315551003', 'Jl. Basuki Rahmat No. 88, Surabaya', 'retail', 'manual', 'verified')
ON CONFLICT (id) DO NOTHING;

INSERT INTO clients (id, npwp, client_name, owning_branch_id) VALUES
  ('44444444-4444-4444-4444-444444444401', '031234567801000', 'PT Cahaya Abadi Sejahtera', '11111111-1111-1111-1111-111111111101')
ON CONFLICT (id) DO NOTHING;

INSERT INTO policies (id, client_id, npwp, policy_no, status, period_start, period_end, owning_branch_id) VALUES
  ('55555555-5555-5555-5555-555555555501', '44444444-4444-4444-4444-444444444401', '031234567801000', 'POL-2025-001234', 'active', '2025-09-01', '2026-08-31', '11111111-1111-1111-1111-111111111101')
ON CONFLICT (id) DO NOTHING;

-- ===== Over-block guard case: two unrelated companies, same office address =====
INSERT INTO companies (id, npwp, nib, legal_name, normalized_name, domain, phone, address, occupation_lob, source, verification_status) VALUES
  ('33333333-3333-3333-3333-333333333308', '041234567801000', NULL, 'PT Maju Bersama Logistik', 'MAJU BERSAMA LOGISTIK', 'majubersama.co.id', '0315551004', 'Jl. Industri Raya No. 12, Surabaya', 'logistics', 'manual', 'verified'),
  ('33333333-3333-3333-3333-333333333309', '041234567802000', NULL, 'PT Cipta Boga Nusantara', 'CIPTA BOGA NUSANTARA', 'ciptaboga.co.id', '0315551005', 'Jl. Industri Raya No. 12, Surabaya', 'food_beverage', 'manual', 'verified')
ON CONFLICT (id) DO NOTHING;
