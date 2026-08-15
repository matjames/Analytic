Write-Host '==========================================================' -ForegroundColor Cyan
Write-Host '   STATGATE PHASE VIII - STATGOVERNANCE TEST SUITE        ' -ForegroundColor Cyan
Write-Host '==========================================================' -ForegroundColor Cyan

$root = Get-Location
$allPassed = $true

# 1. Run StatGovernance Backend Unit Tests
Write-Host "`n[1/4] Running StatGovernance Backend Tests..." -ForegroundColor Yellow
Set-Location "$root\StatGovernance\backend"
go test -v ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host 'StatGovernance backend tests failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'StatGovernance backend tests passed successfully!' -ForegroundColor Green
}

# 2. Run Enterprise Core Test Suite
Write-Host "`n[2/4] Running Enterprise Core Unit Tests..." -ForegroundColor Yellow
Set-Location "$root\enterprise\core"
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host 'Enterprise Core tests failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'Enterprise Core tests passed successfully!' -ForegroundColor Green
}

# 3. Run Enterprise Search Build Check
Write-Host "`n[3/4] Testing Enterprise Search Package..." -ForegroundColor Yellow
Set-Location "$root\enterprise\search"
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host 'Enterprise Search tests failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'Enterprise Search tests passed successfully!' -ForegroundColor Green
}

# 4. Build StatGovernance Frontend Bundle (Vite)
Write-Host "`n[4/4] Verifying StatGovernance React Frontend Build..." -ForegroundColor Yellow
Set-Location "$root\StatGovernance\frontend"
if (-not (Test-Path "node_modules")) {
    Write-Host 'Installing frontend dependencies...' -ForegroundColor Cyan
    npm install --silent
}
npm run build
if ($LASTEXITCODE -ne 0) {
    Write-Host 'StatGovernance frontend build failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'StatGovernance frontend build passed successfully!' -ForegroundColor Green
}

Set-Location $root

Write-Host "`n==========================================================" -ForegroundColor Cyan
if ($allPassed) {
    Write-Host '   ALL PHASE VIII STATGOVERNANCE SUITES PASSED (100%)    ' -ForegroundColor Green
} else {
    Write-Host '   SOME SUITES FAILED - INSPECT OUTPUT ABOVE             ' -ForegroundColor Red
}
Write-Host '==========================================================' -ForegroundColor Cyan
