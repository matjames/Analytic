# ═══════════════════════════════════════════════════════════════════
# STATGATE PHASE X — Full Platform Test Suite
# ═══════════════════════════════════════════════════════════════════
# Runs all backend Go tests and verifies the frontend build.
# Usage: .\run-phase10-tests.ps1
# ═══════════════════════════════════════════════════════════════════

$ErrorActionPreference = "Stop"
$OverallStart = Get-Date
$Failures = @()

function Run-Step {
    param($Name, $Dir, [ScriptBlock]$Block)
    Write-Host ""
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    Write-Host "  $Name" -ForegroundColor White
    Write-Host "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━" -ForegroundColor Cyan
    $start = Get-Date
    Push-Location $Dir
    try {
        & $Block
        if ($LASTEXITCODE -ne 0 -and $null -ne $LASTEXITCODE) {
            throw "Exit code $LASTEXITCODE"
        }
        $elapsed = (Get-Date) - $start
        Write-Host "  ✅ PASSED ($([math]::Round($elapsed.TotalSeconds, 1))s)" -ForegroundColor Green
    } catch {
        $elapsed = (Get-Date) - $start
        Write-Host "  ❌ FAILED ($([math]::Round($elapsed.TotalSeconds, 1))s): $_" -ForegroundColor Red
        $script:Failures += $Name
    } finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════╗" -ForegroundColor Magenta
Write-Host "║    STATGATE PHASE X — SOVEREIGN PLATFORM TEST SUITE     ║" -ForegroundColor Magenta
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Magenta

# ── 1. Enterprise Core: Build ──────────────────────────────────────
Run-Step "Enterprise Core — Go Build" "enterprise\core" {
    go build ./... 2>&1 | Write-Host
}

# ── 2. Enterprise Core: Phase X Platform Tests ────────────────────
Run-Step "Enterprise Core — Phase X Security & Observability Tests" "enterprise\core" {
    go test -v -run "TestSecurity|TestJWT|TestTenant|TestRate|TestRequest|TestProduction|TestMarshal|TestBuildDSN|TestMetrics|TestIsProbe|TestLiveness|TestReadiness" ./...
}

# ── 3. Enterprise Core: Analytics Tests ───────────────────────────
Run-Step "Enterprise Core — Analytics Tests" "enterprise\core" {
    go test -v ./... -run "TestAnalytics|TestKPI|TestAlert|TestAnomaly|TestDashboard|TestExport|TestSearch|TestLineage" 2>&1 | Write-Host
}

# ── 4. Enterprise Core: AI & Investigation Tests ──────────────────
Run-Step "Enterprise Core — AI Phase VII Tests" "enterprise\core" {
    go test -v -run "TestAI" ./...
}

# ── 5. Enterprise Core: Workflow & Fabric Tests ───────────────────
Run-Step "Enterprise Core — Workflow & Fabric Tests" "enterprise\core" {
    go test -v -run "TestWorkflow|TestFabric|TestKnowledge" ./...
}

# ── 6. Enterprise Core: Full Test Suite ───────────────────────────
Run-Step "Enterprise Core — Full Test Suite" "enterprise\core" {
    go test ./... -timeout 120s
}

# ── 7. App Launcher Frontend: Build ───────────────────────────────
Run-Step "App Launcher — Next.js Build" "appluancher" {
    npm.cmd run build 2>&1 | Select-Object -Last 30 | Write-Host
}

# ── 8. StatGovernance Backend: Build ─────────────────────────────
Run-Step "StatGovernance — Go Build" "StatGovernance\backend" {
    go build ./... 2>&1 | Write-Host
}

# ── Summary ────────────────────────────────────────────────────────
$total = (Get-Date) - $OverallStart
Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════╗" -ForegroundColor Magenta
Write-Host "║                   PHASE X TEST SUMMARY                  ║" -ForegroundColor Magenta
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Magenta
Write-Host ""
Write-Host "  Total time: $([math]::Round($total.TotalSeconds, 1))s" -ForegroundColor White

if ($Failures.Count -eq 0) {
    Write-Host "  Result: ✅ ALL TESTS PASSED" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Phase X — Sovereign Platform Hardening: COMPLETE" -ForegroundColor Cyan
    exit 0
} else {
    Write-Host "  Result: ❌ $($Failures.Count) STEP(S) FAILED:" -ForegroundColor Red
    $Failures | ForEach-Object { Write-Host "    - $_" -ForegroundColor Red }
    exit 1
}
