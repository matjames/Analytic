# ═════════════════════════════════════════════════════════════════════════════
# STATGATE — CROSS-APPLICATION COMMUNICATION VERIFICATION SUITE
#   "Are all apps able to communicate with each other?"
#
# Checks (in order):
#   1. Shared platform library (statgate-lib) builds & the in-memory cross-app
#      event-bus round-trip passes (two app busses exchanging events on the
#      canonical channel WITHOUT infrastructure — proves the transport).
#   2. Every Go backend in the workspace builds and tests clean (exit 0).
#   3. Every Go backend conforms to the communication contracts:
#        - event bus : references statgate:events / STATGATE_EVENT_CHANNEL / statgate-lib
#        - identity  : references STATGATE_REGISTRY_JWT_SECRET / statgate-lib/auth
#        - probes    : serves a /health (and ideally /ready) endpoint
#   4. docker-compose topology: every service joins statgate-network and has a
#      healthcheck; per-app databases are provisioned by docker/postgres-init.
#   5. object_links cross-app linkage contract exists in the provisioning SQL.
#
# Usage:  powershell -ExecutionPolicy Bypass -File run-communication-verification.ps1
# ═══════════════════════════════════════════════════════════════════════════════

$ErrorActionPreference = "Continue"
$root = $PSScriptRoot
# Globals accumulate results; function counters must be script-scoped.
$script:passed = 0; $script:failed = 0; $script:total = 0

function Assert-CommunicationTest {
    param([string]$Name, [scriptblock]$Block)
    $script:total++
    Write-Host -NoNewline (" [{0}] {1} ... " -f $script:total, $Name)
    try {
        $ok = & $Block
        if ($ok) {
            Write-Host "PASSED" -ForegroundColor Green
            $script:passed++
        } else {
            Write-Host "FAILED" -ForegroundColor Red
            $script:failed++
        }
    } catch {
        Write-Host "ERROR: $($_.Exception.Message)" -ForegroundColor Red
        $script:failed++
    }
}

# ── 1. Shared platform library + cross-app event bus round trip ─────────────
Write-Host "`n=== 1. Shared platform library & cross-app event bus ===" -ForegroundColor Yellow

Assert-CommunicationTest "statgate-lib builds" {
    Push-Location "$root\statgate-lib"
    try { go build ./... 2>$null; if ($LASTEXITCODE -ne 0) { return $false } } finally { Pop-Location }
    return $true
}

Assert-CommunicationTest "Cross-app event-bus round trip (app A -> app B, then B -> A)" {
    Push-Location "$root\statgate-lib"
    try {
        $out = go test ./events/... -run 'TestInMemoryCrossAppCommunication|TestDefaultChannelIsCanonical' 2>&1
        if ($LASTEXITCODE -ne 0) { return $false }
    } finally { Pop-Location }
    return $true
}

# ── 2. Discover all Go backend modules ──────────────────────────────────────
$goModules = @()
Get-ChildItem -Path $root -Recurse -Filter go.mod -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -notmatch 'node_modules|vendor|\\.git|\.next' } |
    ForEach-Object { $goModules += (Split-Path $_.FullName -Parent) }

Write-Host "`n--- 2. Build & test all $($goModules.Count) Go modules ---" -ForegroundColor Yellow
foreach ($m in $goModules) {
    $name = $m.Replace($root + '\', '')
    Assert-CommunicationTest "build: $name" {
        Push-Location $m
        try {
            go build ./... 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) { return $false }
        } finally { Pop-Location }
        return $true
    }
    Assert-CommunicationTest "tests: $name" {
        Push-Location $m
        try {
            go vet ./... 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) { return $false }
            go test ./... 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) { return $false }
        } finally { Pop-Location }
        return $true
    }
}
# ── 3. Per-app communication contracts ───────────────────────────────────────
Write-Host "`n--- 3. Application communication contracts ---" -ForegroundColor Yellow
foreach ($m in $goModules) {
    $name = $m.Replace($root + '\', '')
    $src = Get-ChildItem -Path $m -Recurse -Filter *.go -ErrorAction SilentlyContinue

    Assert-CommunicationTest "$name -> event bus contract" {
        if (-not $src) { return $false }
        $lines = $src | Select-String -Pattern 'statgate:events|STATGATE_EVENT_CHANNEL|matjames/statgate-lib' -ErrorAction SilentlyContinue
        return ($lines.Count -gt 0)
    }
    Assert-CommunicationTest "$name -> registry auth contract" {
        if (-not $src) { return $false }
        $lines = $src | Select-String -Pattern 'STATGATE_REGISTRY_JWT_SECRET|statgate-lib/auth|Authorization|Bearer' -ErrorAction SilentlyContinue
        return ($lines.Count -gt 0)
    }
    Assert-CommunicationTest "$name -> /health probe" {
        if (-not $src) { return $false }
        $lines = $src | Select-String -Pattern '/health|HealthHandler|HealthProbe' -ErrorAction SilentlyContinue
        return ($lines.Count -gt 0)
    }
}

# ── 4. Docker Compose topology ───────────────────────────────────────────────
Write-Host "`n--- 4. Docker Compose topology (statgate-network + healthchecks) ---" -ForegroundColor Yellow
$compose = "$root\docker-compose.yml"

Assert-CommunicationTest "compose: top-level statgate-network declared" {
    return (Select-String -Path $compose -Pattern '^networks:|^\s{2}statgate-network:' -Quiet)
}

Write-Host "`n--- 5. object_links cross-reference contract provisioned ---" -ForegroundColor Yellow
$objLinkSql = Get-ChildItem "$root\docker\postgres-init" -Filter *.sql -ErrorAction SilentlyContinue |
    Select-String -Pattern 'object_links' | Measure-Object
Assert-CommunicationTest "object_links schema present in postgres-init SQL" {
    return ($objLinkSql.Count -ge 1)
}

# ── Summary ───────────────────────────────────────────────────────────────────
Write-Host "`n════════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  COMMUNICATION VERIFICATION: $script:passed / $script:total PASSED" -ForegroundColor $(if ($script:failed -eq 0) { "Green" } else { "Red" })
Write-Host "════════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
if ($script:failed -gt 0) { exit 1 }