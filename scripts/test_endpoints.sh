#!/usr/bin/env bash
# Simple e2e smoke script to exercise reports filters/tags/compare endpoints.
# Usage: run the server locally (e.g., `go run ./cmd -addr=:8080`) then execute this script.
set -euo pipefail
BASE=${1:-http://localhost:8080}

echo "Listing saved filters (expect 401 if not authenticated):"
curl -sS -D - "$BASE/api/v1/reports/filters" | sed -n '1,5p'

echo "Attempting to create a saved filter (will likely 401 without auth):"
echo '{"name":"smoke","params":{"q":"","sort":"created_at","order":"desc"}}' | curl -sS -X POST -H 'Content-Type: application/json' -d @- "$BASE/api/v1/reports/filters" | sed -n '1,10p'

echo "Comparing two reports (example ids: replace with real IDs):"
curl -sS "$BASE/api/v1/reports/compare?ids=<REPORT_A_ID>,<REPORT_B_ID>" | sed -n '1,20p'

echo "Done. Replace REPORT IDs and run with an authenticated session for full results."
