# AHID Prospect Protection & Auto-Verification System (Demo)

A demo system replacing AHID's manual, single-support prospect verification
process with automated checks: existing-client lookup, a multi-signal
duplicate/group matching engine, a first-come protection lock, a
maker-checker review queue, an AI enrichment loop, and an audit/monitoring
layer. See the full PRD in [GitHub issue #1](../../issues/1).

**Stack:** React + Vite (frontend) · Go (backend) · PostgreSQL · Docker Compose.

## Running it

```bash
cp .env.example .env
docker compose up --build
```

This brings up, in order: Postgres → migrations → seed data → the Go API
(`http://localhost:8080`) → the React app (`http://localhost:5173`). An
Adminer instance for poking at the database directly is at
`http://localhost:8081` (server `db`, user/password from `.env`).

Demo auth has no passwords — the app's role switcher (top-right) lets you
act as any seeded user (`marketing`, `branch_co`, `underwriter`,
`compliance`, `audit`, `admin`). The backend trusts the `X-Demo-Role` /
`X-Demo-User-Id` headers the frontend sends; this is explicitly demo-only
and documented as such in `internal/http/middleware.go`.

### Optional: real AI enrichment via Gemini

By default the AI enrichment loop (see Scenario 6 below) resolves
suggestions from a static fixture file (`backend/internal/enrichment/fixtures/enrichment_fixtures.json`),
so the demo works fully offline. To have it actually search the web via the
Gemini API (Google Search grounding) instead, set in `.env`:

```
GEMINI_API_KEY=your-key-here
GEMINI_MODEL=gemini-2.5-flash   # default, optional
```

If the key is unset, or a Gemini call fails or finds nothing, the worker
falls back to the fixture file automatically.

## Local development without Docker

Backend:
```bash
cd backend
go test ./...
DATABASE_URL=postgres://ahid:ahid@localhost:5432/ahid_prospect?sslmode=disable go run ./cmd/api
```

Frontend:
```bash
cd frontend
npm install
VITE_API_BASE=http://localhost:8080/api/v1 npm run dev
```

(Run `docker compose up -d db adminer && docker compose run --rm migrate && docker compose run --rm seed`
first to get a local Postgres with schema + seed data without starting the
Go/Node services in Docker.)

## Demo scenarios (definition of done)

Each of these can be driven from the UI (role switcher → Intake/Review
Queue/Company Master) or via `curl` against `http://localhost:8080/api/v1`.

> **Quick try — protection lock:** run `./scripts/demo-protection.sh` once
> after the stack is up. It registers and locks a prospect as Branch
> Surabaya, then prints the exact company name + NPWP to type into the
> Intake form as `Budi Marketing (Jakarta)` so you can watch it get blocked
> by protection (no identity leaked) and land in the compliance Review
> Queue. This is scenario 4 below, pre-staged for you.

1. **Existing client** — Intake a prospect with NPWP `031234567801000`
   (`PT Cahaya Abadi Sejahtera`) → `BLOCK` / `existing_active_policy`.
2. **Brand vs legal name** — Intake `Logisly` (no NPWP) → matches the
   verified `brand_names` entry on `PT Logistik Canggih Indonesia` →
   `BLOCK` / `brand_alias_match`.
3. **Group subsidiary** — Intake a near-exact variant of an existing
   Dharma Wibawa Guna Group subsidiary name (e.g. "Bangun Sahabat Tani
   Indonesia") with matching occupation → `REVIEW` via the verified
   `same_group` edge (never auto-`BLOCK` on a soft signal alone).
4. **Protection lock** — As `Sari Marketing (Surabaya)`, create a prospect
   with a fresh NPWP and request a lock (Prospects page → Lock). Switch to
   `Budi Marketing (Jakarta)`, create a prospect with the *same* NPWP, and
   request a lock too: you get a "protected" response with no identity
   leaked, and the conflict shows up in the Review Queue (visible to
   `branch_co`/`compliance`/`audit`/`admin`).
5. **Over-block guard** — Intake a prospect at the same address as `PT
   Maju Bersama Logistik` (`Jl. Industri Raya No. 12, Surabaya`) but with a
   clearly different name and line of business → stays at `PASS`/`REVIEW`,
   never `BLOCK` — address and occupation alone are deliberately weak
   signals (FR3.3).
6. **AI enrichment** — Intake an unknown company present in the fixture
   file, e.g. `PT Teknologi Inovasi Mandiri` → immediate `PASS` response.
   Within a few seconds the worker creates an **unverified** suggested
   `same_group` edge (visible in Company Master), which an `admin` can then
   verify. It never auto-blocks anything on its own.
7. **Maker-checker** — Open the Review Queue as `branch_co` on a `REVIEW`
   item and propose a decision (`confirm_duplicate` / `mark_distinct` /
   `assign_owner`). Switch to `compliance` and approve (or reject) it. Two
   audit entries with distinct actors appear in the Audit Log.

## Project layout

```
db/migrations, db/seed.sql   schema + seed data (PRD §6, §14)
backend/                     Go API (cmd/api, internal/{http,store,matching,core,enrichment,config,domain})
frontend/                    React + Vite app (src/{pages,components,context,api})
docker-compose.yml           db, migrate, seed, backend, frontend, adminer
```

`backend/internal/matching` has unit tests covering every band/scenario
above (`go test ./...` from `backend/`). The matching engine, normalization
rules, and scoring weights are documented in code comments next to the
implementation (`internal/matching/score.go`, `normalize.go`,
`similarity.go`).

## Known demo-only shortcuts (documented, not hidden)

- **Auth** is header-based with no passwords (§10 of the PRD) — replace with
  real SSO/JWT before any production use.
- **Core system integration** (existing client/policy lookup) is a mock
  reading seeded `clients`/`policies` tables behind the `core.CoreClient`
  interface — swap in a real implementation against the production contract
  noted in the PRD §9.
- **AI enrichment** defaults to a static fixture and can optionally call
  Gemini with search grounding; a production version would need an
  authoritative-sources pipeline (AHU, OSS/NIB) with provenance and PDP/ToS
  compliance, as noted in the PRD §8.
- CORS is wide open (`Access-Control-Allow-Origin: *`) since frontend and
  backend run on different ports/containers in this demo.
