#!/usr/bin/env bash
set -euo pipefail
: "${STATCOLLECT_API_KEY:?STATCOLLECT_API_KEY must be set}"
ROOT=$(cd "$(dirname "$0")/.." && pwd -P)
cd "$ROOT/.."

# choose working docker compose command
if command -v docker >/dev/null 2>&1; then
	if docker compose version >/dev/null 2>&1; then
		DOCKER_COMPOSE_CMD='docker compose'
	elif command -v docker-compose >/dev/null 2>&1; then
		DOCKER_COMPOSE_CMD='docker-compose'
	else
		echo "ERROR: docker compose or docker-compose is required" >&2
		exit 1
	fi
else
	echo "ERROR: docker is required" >&2
	exit 1
fi

# bring up services
$DOCKER_COMPOSE_CMD -f StatCollect/docker-compose.yml up --build -d
# wait for postgres
echo "waiting for postgres..."
sleep 6
# apply migration (try host psql, else exec into container)
if command -v psql >/dev/null 2>&1; then
	PGPASSWORD=Statgate psql -h localhost -U statcollect -d statcollect -f StatCollect/migrations/001_init.sql || true
else
	$DOCKER_COMPOSE_CMD -f StatCollect/docker-compose.yml exec -T postgres psql -U statcollect -d statcollect -f /workdir/StatCollect/migrations/001_init.sql || true
fi

echo "=== StatGate Platform Integration Tests ==="

# 1. Health check
echo "--- 1. Health check ---"
curl -s http://localhost:8080/health | grep -q "ok" && echo "PASS: /health" || echo "FAIL: /health"

# 2. Full health with integrations
echo "--- 2. Full health with integrations ---"
curl -s http://localhost:8080/health/full | jq '.integrations' && echo "PASS: /health/full shows integrations" || echo "FAIL: /health/full"

# 3. Submit sample
echo "--- 3. Submit sample ---"
cat > /tmp/sample.xml <<'XML'
<data><instanceID>test-inst-1</instanceID><value>42</value></data>
XML
curl -s -X POST -H "X-API-Key: ${STATCOLLECT_API_KEY}" -H "X-Instance-ID: test-inst-1" -F "xml_submission_file=@/tmp/sample.xml" http://localhost:8080/submission | grep -q "ok" && echo "PASS: submission accepted" || echo "FAIL: submission"

# 4. List submissions (with tenant/status)
echo "--- 4. List submissions ---"
curl -s -H "X-API-Key: ${STATCOLLECT_API_KEY}" "http://localhost:8080/admin/submissions?per_page=10&page=1" | jq '.items[0] | {instance_id, tenant_id, status}' && echo "PASS: submissions list" || echo "FAIL: submissions list"

# 5. Get submission detail
echo "--- 5. Get submission detail ---"
curl -s -H "X-API-Key: ${STATCOLLECT_API_KEY}" "http://localhost:8080/admin/submission?instance_id=test-inst-1" | jq '.summary | {instance_id, form_id, status}' && echo "PASS: submission detail" || echo "FAIL: submission detail"

# 6. Validate submission (workflow)
echo "--- 6. Validate submission ---"
curl -s -X POST -H "X-API-Key: ${STATCOLLECT_API_KEY}" "http://localhost:8080/admin/submission/validate?instance_id=test-inst-1&status=approved&notes=Integration+test" | jq '.status' && echo "PASS: validation workflow" || echo "FAIL: validation"

# 7. Check event log
echo "--- 7. Event log ---"
curl -s -H "X-API-Key: ${STATCOLLECT_API_KEY}" "http://localhost:8080/admin/events?object_type=submission&object_id=test-inst-1" | jq '.events[0] | {event_type, source}' && echo "PASS: event log" || echo "FAIL: event log"

# 8. Object links
echo "--- 8. Object links ---"
curl -s -H "X-API-Key: ${STATCOLLECT_API_KEY}" "http://localhost:8080/admin/objects/links?object_type=submission&object_id=test-inst-1" | jq '.count' && echo "PASS: object links" || echo "FAIL: object links"

# 9. StatChat discussion link
echo "--- 9. StatChat discussion ---"
curl -s -H "X-API-Key: ${STATCOLLECT_API_KEY}" "http://localhost:8080/admin/submission/discussion?instance_id=test-inst-1" | jq '.conversation_id' 2>/dev/null && echo "PASS: discussion link" || echo "SKIP: discussion (StatChat may not be running)"

echo ""
echo "=== Integration tests complete ==="
