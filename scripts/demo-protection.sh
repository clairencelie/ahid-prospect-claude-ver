#!/usr/bin/env bash
#
# Demo setup: "Branch Surabaya already protects a prospect."
#
# Run this once after `docker compose up`. It registers a prospect and grants
# a protection lock as the Surabaya marketer, so you can then reproduce the
# protection / need-to-know flow yourself from the UI as the Jakarta marketer.
#
# Re-runnable: it clears any prior state for the demo NPWP first.
#
# Usage:  ./scripts/demo-protection.sh
#
set -euo pipefail

API="${API_BASE:-http://localhost:8080/api/v1}"
COMPOSE="${DOCKER_COMPOSE:-docker-compose}"

# Seeded demo users (see db/seed.sql)
SARI_SURABAYA="22222222-2222-2222-2222-222222222201"   # marketing, Branch Surabaya

# The prospect Branch Surabaya will protect.
DEMO_COMPANY="PT Demo Proteksi Nusantara"
DEMO_NPWP="09.876.543.2-101.000"        # how a user would type it in the form
DEMO_NPWP_DIGITS="098765432101000"      # normalized (digits only), as stored

say() { printf '\n\033[1m%s\033[0m\n' "$1"; }

say "1) Clearing any previous run for NPWP ${DEMO_NPWP_DIGITS}..."
$COMPOSE exec -T db psql -U "${POSTGRES_USER:-ahid}" -d "${POSTGRES_DB:-ahid_prospect}" >/dev/null <<SQL
DELETE FROM prospect_locks WHERE npwp = '${DEMO_NPWP_DIGITS}';
DELETE FROM match_results  WHERE prospect_id IN (SELECT id FROM prospects WHERE npwp = '${DEMO_NPWP_DIGITS}');
DELETE FROM prospects      WHERE npwp = '${DEMO_NPWP_DIGITS}';
SQL

say "2) Branch Surabaya (Sari) registers the prospect at the protection gate..."
CREATE_RESP=$(curl -s -X POST "$API/prospects" \
  -H "X-Demo-User-Id: $SARI_SURABAYA" \
  -H "Content-Type: application/json" \
  -d "{\"raw_company_name\":\"$DEMO_COMPANY\",\"npwp\":\"$DEMO_NPWP\",\"occupation_lob\":\"manufacturing\"}")
PROSPECT_ID=$(printf '%s' "$CREATE_RESP" | sed -n 's/.*"id":"\([0-9a-f-]*\)".*/\1/p' | head -1)

if [ -z "$PROSPECT_ID" ]; then
  echo "Failed to create prospect. Response was:"
  echo "$CREATE_RESP"
  exit 1
fi
echo "   prospect id: $PROSPECT_ID"

say "3) Branch Surabaya requests the protection lock..."
LOCK_RESP=$(curl -s -X POST "$API/prospects/$PROSPECT_ID/lock" -H "X-Demo-User-Id: $SARI_SURABAYA")
echo "   lock response: $LOCK_RESP"

cat <<INSTRUCTIONS

============================================================
 SETUP DONE — Branch Surabaya now protects "$DEMO_COMPANY".
============================================================

Now try the protected-input scheme yourself in the UI (http://localhost:5173):

  1. Use the role switcher (top-right) to become:
       "Budi Marketing (Jakarta)"   (a different branch)

  2. Go to the Intake page and fill the PROTECTION GATE:
       Company name : $DEMO_COMPANY
       NPWP         : $DEMO_NPWP
     then click  "Request protection lock".

  3. Expected result:
       - You see "already under protection by another branch
         (no identity revealed)" — Surabaya's identity is NOT leaked.
       - A conflict is filed for compliance.

  4. Switch to "Eko Compliance" (or Audit/Admin) and open the
     Review Queue -> the lock conflict appears there for resolution.

(You can also just run the Intake "Check only" with that NPWP as any
 role to see lock.protected = true without revealing who holds it.)
============================================================

INSTRUCTIONS
