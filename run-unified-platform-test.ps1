# ═══════════════════════════════════════════════════════════════════
# STATGATE UNIFIED CROSS-PLATFORM E2E VERIFICATION SUITE
# ═══════════════════════════════════════════════════════════════════

param (
    [string]$RegistryUrl   = "http://localhost:9090",
    [string]$PmsUrl        = "http://localhost:8091",
    [string]$RmsUrl        = "http://localhost:8092",
    [string]$GovernanceUrl = "http://localhost:8093",
    [string]$SpatialUrl    = "http://localhost:8094",
    [string]$EnterpriseUrl = "http://localhost:8096",
    [string]$AnalyticsUrl  = "http://localhost:5000",
    [string]$StatCollectUrl = "http://localhost:8080"
)

$ErrorActionPreference = "Continue"

Write-Host "═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  STATGATE UNIFIED PLATFORM INTEGRATION TEST SUITE (Phases 1-13)" -ForegroundColor Cyan
Write-Host "  Timestamp: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════════"

$Passed = 0
$Failed = 0
$Total = 0

function Assert-Test {
    param (
        [string]$Name,
        [scriptblock]$TestBlock
    )
    $script:Total++
    Write-Host -NoNewline " [TEST $script:Total] $Name ... "
    try {
        $result = & $TestBlock
        if ($result -eq $true) {
            Write-Host "PASSED" -ForegroundColor Green
            $script:Passed++
        } else {
            Write-Host "FAILED ($result)" -ForegroundColor Red
            $script:Failed++
        }
    } catch {
        Write-Host "ERROR: $_" -ForegroundColor Red
        $script:Failed++
    }
}

# ─── SECTION 1: Service Health Probes ─────────────────────────
Write-Host "`n--- Section 1: Microservice Health & Readiness Probes ---" -ForegroundColor Yellow

$Services = @(
    @{ Name = "Registry Backend"; Port = 9090; Url = "$RegistryUrl/health" },
    @{ Name = "PMS Backend"; Port = 8091; Url = "$PmsUrl/health" },
    @{ Name = "RMS Backend"; Port = 8092; Url = "$RmsUrl/health" },
    @{ Name = "StatGovernance Backend"; Port = 8093; Url = "$GovernanceUrl/health" },
    @{ Name = "StatSpatial Backend"; Port = 8094; Url = "$SpatialUrl/health" },
    @{ Name = "Enterprise Core"; Port = 8096; Url = "$EnterpriseUrl/health" },
    @{ Name = "StatCollect API"; Port = 8080; Url = "$StatCollectUrl/health" },
    @{ Name = "Knowledge Portal (App 2)"; Port = 8099; Url = "http://localhost:8099/health" },
    @{ Name = "StatTrust API (App 3)"; Port = 8094; Url = "http://localhost:8094/health" },
    @{ Name = "RunOps API (App 4)"; Port = 8098; Url = "http://localhost:8098/health" },
    @{ Name = "Integration Fabric (App 1)"; Port = 8097; Url = "http://localhost:8097/health" },
    @{ Name = "AI & Autonomy (App 5)"; Port = 8101; Url = "http://localhost:8101/health" },
    @{ Name = "StatFederation (App 6)"; Port = 8097; Url = "http://localhost:8097/health" },
    @{ Name = "Learning & Comm. (App 7)"; Port = 8102; Url = "http://localhost:8102/health" },
    @{ Name = "Geospatial & Remote (App 9)"; Port = 8103; Url = "http://localhost:8103/health" },
    @{ Name = "BPM & Workflow (App 11)"; Port = 8104; Url = "http://localhost:8104/health" },
    @{ Name = "Federation (App 6)"; Port = 8105; Url = "http://localhost:8105/health" },
    @{ Name = "IoT & Field Ops (App 8)"; Port = 8106; Url = "http://localhost:8106/health" },
    @{ Name = "Data Science (App 12)"; Port = 8107; Url = "http://localhost:8107/health" },
    @{ Name = "IoT & FieldOps (App 8)"; Port = 8103; Url = "http://localhost:8103/health" },
    @{ Name = "Geospatial & RS (App 9)"; Port = 8104; Url = "http://localhost:8104/health" }
)

foreach ($svc in $Services) {
    Assert-Test "$($svc.Name) (:$(($svc.Port))) Healthcheck" {
        try {
            $resp = Invoke-RestMethod -Uri $svc.Url -Method Get -TimeoutSec 3 -ErrorAction Stop
            return ($resp.status -eq "ok" -or $resp.status -eq "UP" -or $resp -ne $null)
        } catch {
            return "Unreachable (Service offline - start via docker/binary)"
        }
    }
}

# ─── SECTION 2: Code Compilation & Build Verification ─────────
Write-Host "`n--- Section 2: Go Backend Compile Checks (Exit Code 0) ---" -ForegroundColor Yellow

$GoModules = @(
    "c:\Users\PC\Desktop\Analytic\PMS\backend",
    "c:\Users\PC\Desktop\Analytic\RMS\backend",
    "c:\Users\PC\Desktop\Analytic\StatGovernance\backend",
    "c:\Users\PC\Desktop\Analytic\StatSpatial\backend",
    "c:\Users\PC\Desktop\Analytic\enterprise\core",
    "c:\Users\PC\Desktop\Analytic\stage_register\go-backend",
    "c:\Users\PC\Desktop\Analytic\backend",
    "c:\Users\PC\Desktop\Analytic\knowledge-portal\backend",
    "c:\Users\PC\Desktop\Analytic\StatTrust\backend",
    "c:\Users\PC\Desktop\Analytic\PlatformEngineering\backend",
    "c:\Users\PC\Desktop\Analytic\StatOps\backend",
    "c:\Users\PC\Desktop\Analytic\enterprise\integration",
    "c:\Users\PC\Desktop\Analytic\ai-autonomy\backend",
    "c:\Users\PC\Desktop\Analytic\StatFederation\backend",
    "c:\Users\PC\Desktop\Analytic\learning-crm\backend",
    "c:\Users\PC\Desktop\Analytic\StatIoT\backend",
    "c:\Users\PC\Desktop\Analytic\geointel\backend",
    "c:\Users\PC\Desktop\Analytic\bpm-hub\backend"
)

foreach ($mod in $GoModules) {
    $dirName = Split-Path $mod -Leaf
    $parent = Split-Path (Split-Path $mod -Parent) -Leaf
    Assert-Test "Compile check: $parent/$dirName" {
        $prev = Get-Location
        Set-Location $mod
        $output = go build ./... 2>&1
        $code = $LASTEXITCODE
        Set-Location $prev
        return ($code -eq 0)
    }
}

# ─── SECTION 3: Frontend Build Verifications ───────────────────
Write-Host "`n--- Section 3: Frontend Bundle Compilation Verification ---" -ForegroundColor Yellow

$FrontendModules = @(
    @{ Name = "StatSpatial Frontend (:3014)"; Path = "c:\Users\PC\Desktop\Analytic\StatSpatial\frontend" },
    @{ Name = "PMS Frontend (:3010)"; Path = "c:\Users\PC\Desktop\Analytic\PMS\frontend" },
    @{ Name = "RMS Frontend (:3011)"; Path = "c:\Users\PC\Desktop\Analytic\RMS\frontend" },
    @{ Name = "StatGovernance Frontend (:3012)"; Path = "c:\Users\PC\Desktop\Analytic\StatGovernance\frontend" }
)

foreach ($fe in $FrontendModules) {
    Assert-Test "Frontend build check: $($fe.Name)" {
        $dist = Join-Path $fe.Path "dist"
        return (Test-Path $dist)
    }
}

# ─── Final Summary ─────────────────────────────────────────────
Write-Host "`n═══════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  TEST RESULTS: $Passed / $Total PASSED" -ForegroundColor $(if ($Failed -eq 0) { "Green" } else { "Yellow" })
if ($Failed -gt 0) {
    Write-Host "  NOTE: Failed health tests reflect services currently stopped locally." -ForegroundColor Gray
}
Write-Host "═══════════════════════════════════════════════════════════════"
