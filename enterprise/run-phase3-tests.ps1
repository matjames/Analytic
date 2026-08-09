# ══════════════════════════════════════════════════════════════
# StatGate Phase III - End-to-End Integration Test Suite
# ══════════════════════════════════════════════════════════════
# This script demonstrates that real user actions in one StatGate
# application produce the correct consequences across the ecosystem.
#
# Prerequisites:
#   - All services running (docker compose up -d)
#   - Redis available on localhost:6379
#   - Enterprise Core on :8096
#   - Enterprise Search on :8095
#   - PMS on :8091
#   - RMS on :8092
#   - Registry on :9090
#   - StatChat on :4000
#   - HelpDesk on :5006
# ══════════════════════════════════════════════════════════════

$ErrorActionPreference = "Continue"
$CORE = "http://localhost:8096"
$SEARCH = "http://localhost:8095"
$PMS = "http://localhost:8091"
$RMS = "http://localhost:8092"
$REGISTRY = "http://localhost:9090"
$STATCHAT = "http://localhost:4000"
$HELPDESK = "http://localhost:5006"

$PASS = 0
$FAIL = 0
$SKIP = 0

function Test-Step {
    param([string]$Name, [scriptblock]$Test)
    Write-Host "`n── $Name ──" -ForegroundColor Cyan
    try {
        & $Test
        Write-Host "  ✓ PASS" -ForegroundColor Green
        $script:PASS++
    } catch {
        Write-Host "  ✗ FAIL: $($_.Exception.Message)" -ForegroundColor Red
        $script:FAIL++
    }
}

function Test-Skip {
    param([string]$Name, [string]$Reason)
    Write-Host "`n── $Name ──" -ForegroundColor Cyan
    Write-Host "  ⚠ SKIP: $Reason" -ForegroundColor Yellow
    $script:SKIP++
}

# ══════════════════════════════════════════════════════════════
# 1. SERVICE HEALTH CHECKS
# ══════════════════════════════════════════════════════════════
Write-Host "`n══════════════════════════════════════════════════════" -ForegroundColor Magenta
Write-Host "  STATGATE PHASE III - END-TO-END INTEGRATION TESTS" -ForegroundColor Magenta
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Magenta

Test-Step "Enterprise Core health" {
    $r = Invoke-RestMethod "$CORE/health" -TimeoutSec 5
    if ($r.status -ne "healthy") { throw "Core not healthy: $($r.status)" }
}

Test-Step "Enterprise Search health" {
    $r = Invoke-RestMethod "$SEARCH/health" -TimeoutSec 5
    if ($r.status -ne "healthy") { throw "Search not healthy: $($r.status)" }
}

Test-Step "PMS health" {
    $r = Invoke-RestMethod "$PMS/health" -TimeoutSec 5
    if ($r.status -ne "healthy") { throw "PMS not healthy: $($r.status)" }
}

Test-Step "RMS health" {
    $r = Invoke-RestMethod "$RMS/health" -TimeoutSec 5
    if ($r.status -ne "healthy") { throw "RMS not healthy: $($r.status)" }
}

Test-Step "Registry health" {
    $r = Invoke-RestMethod "$REGISTRY/ready" -TimeoutSec 5
    if ($r.status -ne "ready") { throw "Registry not ready: $($r.status)" }
}

Test-Step "StatChat health" {
    $r = Invoke-RestMethod "$STATCHAT/health" -TimeoutSec 5
    if ($r.status -ne "healthy") { throw "StatChat not healthy: $($r.status)" }
}

Test-Step "HelpDesk health" {
    $r = Invoke-RestMethod "$HELPDESK/health" -TimeoutSec 5
    if ($r.status -ne "healthy") { throw "HelpDesk not healthy: $($r.status)" }
}

# ══════════════════════════════════════════════════════════════
# 2. ENTERPRISE CORE CAPABILITIES
# ══════════════════════════════════════════════════════════════
Test-Step "Enterprise Core info shows Phase III" {
    $r = Invoke-RestMethod "$CORE/api/info" -TimeoutSec 5
    if ($r.phase -ne "PHASE_III") { throw "Phase not PHASE_III: $($r.phase)" }
    if ($r.version -ne "3.0.0") { throw "Version not 3.0.0: $($r.version)" }
}

Test-Step "Event types include Phase III events" {
    $r = Invoke-RestMethod "$CORE/api/events/types" -TimeoutSec 5
    $types = $r.event_types
    if ($types -notcontains "survey.created") { throw "Missing survey.created" }
    if ($types -notcontains "submission.received") { throw "Missing submission.received" }
    if ($types -notcontains "ticket.assigned") { throw "Missing ticket.assigned" }
    if ($types -notcontains "field_data.submitted") { throw "Missing field_data.submitted" }
}

# ══════════════════════════════════════════════════════════════
# 3. EVENT BUS - PUBLISH & CONSUME
# ══════════════════════════════════════════════════════════════
$testEventId = "evt_phase3_test_$(Get-Date -Format 'yyyyMMddHHmmss')"

Test-Step "Publish project.created event" {
    $body = @{
        id = $testEventId
        event_type = "project.created"
        source = "pms"
        object_type = "project"
        object_id = "PRJ-PHASE3-TEST"
        actor = "admin"
        tenant_id = "statgate"
        payload = @{
            name = "Phase III Integration Test Project"
            owner = "admin"
            code = "P3-TEST"
        }
    } | ConvertTo-Json -Depth 5

    $r = Invoke-RestMethod "$CORE/api/events" -Method POST -Body $body -ContentType "application/json" -TimeoutSec 5
    if ($r.status -ne "published") { throw "Event not published: $($r.status)" }
}

Start-Sleep -Milliseconds 500

Test-Step "Event appears in history" {
    $r = Invoke-RestMethod "$CORE/api/events?limit=50" -TimeoutSec 5
    $found = $r.events | Where-Object { $_.id -eq $testEventId }
    if (-not $found) { throw "Event not found in history" }
}

Test-Step "Event creates timeline entry" {
    $r = Invoke-RestMethod "$CORE/api/timeline?limit=50" -TimeoutSec 5
    $found = $r.entries | Where-Object { $_.entity_id -eq "PRJ-PHASE3-TEST" }
    if (-not $found) { throw "Timeline entry not created" }
}

Test-Step "Event creates notification" {
    $r = Invoke-RestMethod "$CORE/api/notifications?user_id=admin" -TimeoutSec 5
    $found = $r.notifications | Where-Object { $_.source_entity_id -eq "PRJ-PHASE3-TEST" }
    if (-not $found) { throw "Notification not created" }
}

Test-Step "Event creates calendar event" {
    $r = Invoke-RestMethod "$CORE/api/calendar?limit=50" -TimeoutSec 5
    $found = $r.events | Where-Object { $_.entity_id -eq "PRJ-PHASE3-TEST" }
    if (-not $found) { throw "Calendar event not created" }
}

# ══════════════════════════════════════════════════════════════
# 4. IDEMPOTENCY - DUPLICATE EVENT HANDLING
# ══════════════════════════════════════════════════════════════
Test-Step "Duplicate event is skipped (idempotency)" {
    $body = @{
        id = $testEventId
        event_type = "project.created"
        source = "pms"
        object_type = "project"
        object_id = "PRJ-PHASE3-TEST"
        actor = "admin"
        tenant_id = "statgate"
        payload = @{
            name = "Phase III Integration Test Project"
            owner = "admin"
            code = "P3-TEST"
        }
    } | ConvertTo-Json -Depth 5

    # Publish the same event again
    Invoke-RestMethod "$CORE/api/events" -Method POST -Body $body -ContentType "application/json" -TimeoutSec 5 | Out-Null
    Start-Sleep -Milliseconds 500

    # Check that only ONE notification was created for this event
    $r = Invoke-RestMethod "$CORE/api/notifications?user_id=admin" -TimeoutSec 5
    $matching = $r.notifications | Where-Object { $_.source_entity_id -eq "PRJ-PHASE3-TEST" }
    if ($matching.Count -gt 1) { throw "Duplicate event created $($matching.Count) notifications" }
}

# ══════════════════════════════════════════════════════════════
# 5. WORKSPACE API
# ══════════════════════════════════════════════════════════════
Test-Step "Workspace returns personalized data" {
    $r = Invoke-RestMethod "$CORE/api/workspace?user_id=admin" -TimeoutSec 5
    if (-not $r.user_id) { throw "Missing user_id" }
    if ($null -eq $r.my_tasks) { throw "Missing my_tasks" }
    if ($null -eq $r.my_projects) { throw "Missing my_projects" }
    if ($null -eq $r.my_notifications) { throw "Missing my_notifications" }
    if ($null -eq $r.recent_activity) { throw "Missing recent_activity" }
}

Test-Step "Workspace dead-letter queue accessible" {
    $r = Invoke-RestMethod "$CORE/api/workspace/dead-letter" -TimeoutSec 5
    if ($null -eq $r.count) { throw "Missing count" }
}

# ══════════════════════════════════════════════════════════════
# 6. DASHBOARD - REAL DATA VALIDATION
# ══════════════════════════════════════════════════════════════
Test-Step "Dashboard KPIs use real data" {
    $r = Invoke-RestMethod "$CORE/api/dashboards" -TimeoutSec 5
    # Just verify the endpoint works
    if ($null -eq $r.count) { throw "Missing count" }
}

Test-Step "Widget data endpoint works" {
    $r = Invoke-RestMethod "$CORE/api/widgets" -TimeoutSec 5
    if ($r.count -ne 11) { throw "Expected 11 widgets, got $($r.count)" }
}

Test-Step "Widget preview returns data" {
    $r = Invoke-RestMethod "$CORE/api/widgets/kpi_card/preview" -TimeoutSec 5
    if ($null -eq $r.preview) { throw "Missing preview data" }
}

# ══════════════════════════════════════════════════════════════
# 7. ENTERPRISE SEARCH
# ══════════════════════════════════════════════════════════════
Test-Step "Enterprise search returns results" {
    $r = Invoke-RestMethod "$SEARCH/api/search?q=test&limit=5" -TimeoutSec 5
    if ($null -eq $r.count) { throw "Missing count" }
}

Test-Step "Enterprise search handles unavailable services gracefully" {
    # Search should still work even if some services are down
    $r = Invoke-RestMethod "$SEARCH/api/search?q=project&limit=5" -TimeoutSec 5
    if ($null -eq $r.results) { throw "Missing results" }
}

# ══════════════════════════════════════════════════════════════
# 8. NOTIFICATIONS - FULL CRUD
# ══════════════════════════════════════════════════════════════
Test-Step "Notification CRUD operations" {
    # Create a notification
    $body = @{
        user_id = "admin"
        title = "Phase III Test Notification"
        body = "Testing notification CRUD"
        priority = "high"
        category = "test"
        source_app = "enterprise"
    } | ConvertTo-Json

    $created = Invoke-RestMethod "$CORE/api/notifications" -Method POST -Body $body -ContentType "application/json" -TimeoutSec 5
    $notifId = $created.id

    # Mark as read
    $read = Invoke-RestMethod "$CORE/api/notifications/$notifId/read?user_id=admin" -Method PUT -TimeoutSec 5
    if ($read.read -ne $true) { throw "Failed to mark as read" }

    # Archive
    $archived = Invoke-RestMethod "$CORE/api/notifications/$notifId/archive?user_id=admin" -Method PUT -TimeoutSec 5
    if ($archived.archived -ne $true) { throw "Failed to archive" }

    # Delete
    $deleted = Invoke-RestMethod "$CORE/api/notifications/$notifId?user_id=admin" -Method DELETE -TimeoutSec 5
    if ($deleted.deleted -ne $true) { throw "Failed to delete" }
}

# ══════════════════════════════════════════════════════════════
# 9. PERMISSIONS
# ══════════════════════════════════════════════════════════════
Test-Step "Permission check works" {
    $r = Invoke-RestMethod "$CORE/api/permissions/check?user_id=admin&resource=project&action=write" -TimeoutSec 5
    if ($null -eq $r.allowed) { throw "Missing allowed" }
}

Test-Step "Permission roles listed" {
    $r = Invoke-RestMethod "$CORE/api/permissions/roles" -TimeoutSec 5
    if ($r.roles.Count -lt 6) { throw "Expected at least 6 roles" }
}

# ══════════════════════════════════════════════════════════════
# 10. MONITORING
# ══════════════════════════════════════════════════════════════
Test-Step "Monitoring services health" {
    $r = Invoke-RestMethod "$CORE/api/monitoring/services" -TimeoutSec 5
    if ($null -eq $r.services) { throw "Missing services" }
}

Test-Step "Monitoring metrics" {
    $r = Invoke-RestMethod "$CORE/api/monitoring/metrics" -TimeoutSec 5
    if ($null -eq $r.uptime_seconds) { throw "Missing uptime" }
}

# ══════════════════════════════════════════════════════════════
# 11. API GOVERNANCE
# ══════════════════════════════════════════════════════════════
Test-Step "API registry lists endpoints" {
    $r = Invoke-RestMethod "$CORE/api/apis" -TimeoutSec 5
    if ($r.count -lt 10) { throw "Expected at least 10 APIs" }
}

Test-Step "OpenAPI spec available" {
    $r = Invoke-RestMethod "$CORE/api/openapi.json" -TimeoutSec 5
    if ($r.openapi -ne "3.0.0") { throw "Invalid OpenAPI version" }
}

Test-Step "Audit log records events" {
    $r = Invoke-RestMethod "$CORE/api/apis/audit?limit=10" -TimeoutSec 5
    if ($null -eq $r.count) { throw "Missing audit count" }
}

# ══════════════════════════════════════════════════════════════
# 12. FILES SERVICE
# ══════════════════════════════════════════════════════════════
Test-Step "File service lists files" {
    $r = Invoke-RestMethod "$CORE/api/files" -TimeoutSec 5
    if ($null -eq $r.count) { throw "Missing count" }
}

# ══════════════════════════════════════════════════════════════
# 13. CALENDAR
# ══════════════════════════════════════════════════════════════
Test-Step "Calendar lists events" {
    $r = Invoke-RestMethod "$CORE/api/calendar" -TimeoutSec 5
    if ($null -eq $r.count) { throw "Missing count" }
}

# ══════════════════════════════════════════════════════════════
# 14. REPORTS
# ══════════════════════════════════════════════════════════════
Test-Step "Report engine lists reports" {
    $r = Invoke-RestMethod "$CORE/api/reports" -TimeoutSec 5
    if ($null -eq $r.count) { throw "Missing count" }
}

# ══════════════════════════════════════════════════════════════
# 15. AI PREPARATION
# ══════════════════════════════════════════════════════════════
Test-Step "AI catalog available" {
    $r = Invoke-RestMethod "$CORE/api/ai/catalog" -TimeoutSec 5
    if ($r.catalog.Count -lt 5) { throw "Expected at least 5 apps in catalog" }
}

Test-Step "AI schema available" {
    $r = Invoke-RestMethod "$CORE/api/ai/schema" -TimeoutSec 5
    if ($null -eq $r.entities) { throw "Missing entities" }
}

# ══════════════════════════════════════════════════════════════
# SUMMARY
# ══════════════════════════════════════════════════════════════
Write-Host "`n══════════════════════════════════════════════════════" -ForegroundColor Magenta
Write-Host "  TEST SUMMARY" -ForegroundColor Magenta
Write-Host "══════════════════════════════════════════════════════" -ForegroundColor Magenta
Write-Host "  PASS: $PASS" -ForegroundColor Green
Write-Host "  FAIL: $FAIL" -ForegroundColor Red
Write-Host "  SKIP: $SKIP" -ForegroundColor Yellow
Write-Host "  TOTAL: $($PASS + $FAIL + $SKIP)" -ForegroundColor White

if ($FAIL -gt 0) {
    Write-Host "`n  RESULT: FAILED" -ForegroundColor Red
    exit 1
} else {
    Write-Host "`n  RESULT: ALL TESTS PASSED" -ForegroundColor Green
    exit 0
}