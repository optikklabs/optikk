#!/bin/bash
# Smoke test for Optikk data CLI commands.
# Runs against a local cluster provisioned with `optikk up`.

set -e

echo "--- Optikk CLI Smoke Test ---"

export OPTIKK_API_URL="http://localhost:8080"
export OPTIKK_OUTPUT="json"
export FORCE_AGENT_MODE="1"

# 1. Auth
echo "Testing auth login..."
go run ./main.go auth login --email admin@optikk.dev --password 'Password123!'
echo "Testing auth status..."
go run ./main.go auth status

# 2. Traces
echo "Testing traces search..."
go run ./main.go traces search --query "has_error:true" --from 1h --limit 5 > /dev/null
echo "Testing traces trend..."
go run ./main.go traces trend --from 1h > /dev/null

# 3. Logs
echo "Testing logs search..."
go run ./main.go logs search --query "severity_text:ERROR" --from 1h --limit 5 > /dev/null

# 4. Metrics
echo "Testing metrics list..."
go run ./main.go metrics list --from 1h > /dev/null

# 5. Dashboards
echo "Testing dashboards create..."
PAGE_JSON=$(go run ./main.go dashboards create --name "Smoke Test Dashboard" --tags smoke,test)
PAGE_ID=$(echo $PAGE_JSON | jq -r '.id')
echo "Created dashboard $PAGE_ID"

echo "Testing dashboards update..."
go run ./main.go dashboards update $PAGE_ID --name "Updated Smoke Test Dashboard" > /dev/null

echo "Testing dashboards export..."
go run ./main.go dashboards export $PAGE_ID -f smoke-dashboard.json
echo "Testing dashboards import..."
go run ./main.go dashboards import -f smoke-dashboard.json --name "Imported Dashboard" > /dev/null

echo "Testing dashboards delete..."
go run ./main.go dashboards delete $PAGE_ID --yes

rm -f smoke-dashboard.json

# 6. Monitors
echo "Testing monitors list..."
go run ./main.go monitors list > /dev/null

# 7. Agent schema
echo "Testing agent schema..."
go run ./main.go agent schema > /dev/null

echo "--- All tests passed! ---"
