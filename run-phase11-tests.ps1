# ═══════════════════════════════════════════════════════════════════
# STATGATE PHASE XI — Full Institutional Resilience Test Suite
# ═══════════════════════════════════════════════════════════════════
# Runs all backend Go resilience tests, failure injection simulations,
# and verifies the Next.js frontend Command Centre build.
# Usage: .\run-phase11-tests.ps1
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
Write-Host "║    STATGATE PHASE XI — RESILIENCE & ASSURANCE SUITE     ║" -ForegroundColor Magenta
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Magenta

# ── 1. Enterprise Core: Go Build ──────────────────────────────────
Run-Step "Enterprise Core — Go Build" "enterprise\core" {
    go build ./... 2>&1 | Write-Host
}

# ── 2. Enterprise Core: Phase XI Resilience Tests ─────────────────
Run-Step "Enterprise Core — Phase XI Resilience & DR Tests" "enterprise\core" {
    go test -v -run "TestResilience|TestRecovery|TestIncident|TestBackup|TestIntegrity|TestRunbook|TestFailure" ./...
}

# ── 3. Enterprise Core: Phase X Platform Security Tests ───────────
Run-Step "Enterprise Core — Phase X Platform Security Tests" "enterprise\core" {
    go test -v -run "TestSecurity|TestJWT|TestTenant|TestRate|TestRequest|TestProduction|TestMarshal|TestBuildDSN|TestMetrics|TestIsProbe|TestLiveness|TestReadiness" ./...
}

# ── 4. App Launcher Frontend: Next.js Build ───────────────────────
Run-Step "App Launcher Command Centre — Next.js Build" "appluancher" {
    npm.cmd run build 2>&1 | Select-Object -Last 25 | Write-Host
}

# ── 5. Summary ────────────────────────────────────────────────────
$total = (Get-Date) - $OverallStart
Write-Host ""
Write-Host "╔══════════════════════════════════════════════════════════╗" -ForegroundColor Magenta
Write-Host "║                  PHASE XI TEST SUMMARY                  ║" -ForegroundColor Magenta
Write-Host "╚══════════════════════════════════════════════════════════╝" -ForegroundColor Magenta
Write-Host ""
Write-Host "  Total execution time: $([math]::Round($total.TotalSeconds, 1))s" -ForegroundColor White

if ($Failures.Count -eq 0) {
    Write-Host "  Result: ✅ ALL TESTS & BUILDS PASSED" -ForegroundColor Green
    Write-Host ""
    Write-Host "  Phase XI — Institutional Resilience & Assurance: CERTIFIED" -ForegroundColor Cyan
    exit 0
} else {
    Write-Host "  Result: ❌ $($Failures.Count) STEP(S) FAILED:" -ForegroundColor Red
    $Failures | ForEach-Object { Write-Host "    - $_" -ForegroundColor Red }
    exit 1
}
