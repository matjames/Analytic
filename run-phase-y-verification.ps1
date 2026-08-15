# STATGATE PHASE y - Verification & Acceptance Test Suite
$ErrorActionPreference = "Stop"

Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  STATGATE PHASE y - ENTERPRISE CONVERGENCE VERIFICATION    " -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan

$passed = 0
$total = 0

function Run-Step($name, $scriptBlock) {
    $global:total++
    Write-Host "`n[STEP $global:total] $name..." -ForegroundColor Yellow
    try {
        & $scriptBlock
        Write-Host "[PASSED] $name" -ForegroundColor Green
        $global:passed++
    } catch {
        Write-Host "[FAILED] $($name): $($_.Exception.Message)" -ForegroundColor Red
    }
}

# 1. Test statgate-lib
Run-Step "Build and Test Shared Platform Library (statgate-lib)" {
    Push-Location "$PSScriptRoot\statgate-lib"
    try {
        $out = go test -v ./... 2>&1
        if ($LASTEXITCODE -ne 0) { throw $out }
    } finally {
        Pop-Location
    }
}

# 2. Test StatSpatial Backend
Run-Step "Build and Test StatSpatial API Layer" {
    Push-Location "$PSScriptRoot\StatSpatial\backend"
    try {
        $out = go test -v ./... 2>&1
        if ($LASTEXITCODE -ne 0) { throw $out }
        $buildOut = go build ./... 2>&1
        if ($LASTEXITCODE -ne 0) { throw $buildOut }
    } finally {
        Pop-Location
    }
}

# 3. Test Enterprise Core
Run-Step "Build and Test Enterprise Core Engine" {
    Push-Location "$PSScriptRoot\enterprise\core"
    try {
        $out = go test -v ./... 2>&1
        if ($LASTEXITCODE -ne 0) { throw $out }
    } finally {
        Pop-Location
    }
}

# 4. Check Documentation Suite
Run-Step "Verify Phase y Authoritative Documentation Suite" {
    $requiredDocs = @(
        "ENTERPRISE_ARCHITECTURE.md",
        "APPLICATION_INTEGRATION_MATRIX.md",
        "DATABASE_CATALOG.md",
        "API_CATALOG.md",
        "EVENT_CATALOG.md",
        "OBJECT_CATALOG.md",
        "SECURITY_BASELINE.md",
        "STORAGE_ARCHITECTURE.md",
        "WORKFLOW_CATALOG.md",
        "AI_GOVERNANCE.md",
        "DEPLOYMENT_ARCHITECTURE.md",
        "PRODUCTION_READINESS.md"
    )

    foreach ($doc in $requiredDocs) {
        $path = "$PSScriptRoot\docs\$doc"
        if (-not (Test-Path $path)) {
            throw "Missing required documentation: docs/$doc"
        }
        $len = (Get-Item $path).Length
        if ($len -lt 500) {
            throw "Documentation file docs/$doc is suspiciously short ($len bytes)"
        }
        Write-Host "  -> Verified docs/$doc ($len bytes)" -ForegroundColor Gray
    }
}

# 5. Security Zero-Default Verification
Run-Step "Verify Security Zero-Default Enforcements" {
    # Test that statgate-lib rejects blank secrets in production
    $env:STATGATE_ENV = "production"
    $env:STATGATE_REGISTRY_JWT_SECRET = ""
    Push-Location "$PSScriptRoot\statgate-lib"
    try {
        $out = go test ./... -run TestAuthValidation 2>&1
        if ($LASTEXITCODE -ne 0) { throw $out }
    } finally {
        Pop-Location
        $env:STATGATE_ENV = ""
    }
}

Write-Host "`n============================================================" -ForegroundColor Cyan
Write-Host "  VERIFICATION COMPLETE: $passed / $total STEPS PASSED      " -ForegroundColor $(if ($passed -eq $total) { "Green" } else { "Red" })
Write-Host "============================================================" -ForegroundColor Cyan

if ($passed -ne $total) {
    exit 1
}
