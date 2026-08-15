Write-Host '==========================================================' -ForegroundColor Cyan
Write-Host '   STATGATE PHASE IX - DATA & KNOWLEDGE FABRIC TEST SUITE ' -ForegroundColor Cyan
Write-Host '==========================================================' -ForegroundColor Cyan

$root = Get-Location
$allPassed = $true

# 1. Run Enterprise Core Test Suite (including Phase IX Fabric tests)
Write-Host "`n[1/5] Running Enterprise Core Unit & Fabric Tests..." -ForegroundColor Yellow
Set-Location "$root\enterprise\core"
go test -v ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host 'Enterprise Core tests failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'Enterprise Core tests passed successfully!' -ForegroundColor Green
}

# 2. Run StatGovernance Backend Unit Tests
Write-Host "`n[2/5] Running StatGovernance Backend Tests..." -ForegroundColor Yellow
Set-Location "$root\StatGovernance\backend"
go test -v ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host 'StatGovernance backend tests failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'StatGovernance backend tests passed successfully!' -ForegroundColor Green
}

# 3. Run Enterprise Search Package Check
Write-Host "`n[3/5] Testing Enterprise Search Service..." -ForegroundColor Yellow
Set-Location "$root\enterprise\search"
go test ./...
if ($LASTEXITCODE -ne 0) {
    Write-Host 'Enterprise Search tests failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'Enterprise Search tests passed successfully!' -ForegroundColor Green
}

# 4. Build App Launcher & Command Centre Next.js Frontend
Write-Host "`n[4/5] Verifying App Launcher & Command Centre Production Build..." -ForegroundColor Yellow
Set-Location "$root\appluancher"
npm.cmd run build
if ($LASTEXITCODE -ne 0) {
    Write-Host 'App Launcher build failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'App Launcher build passed successfully!' -ForegroundColor Green
}

# 5. Build StatGovernance React Frontend (Vite)
Write-Host "`n[5/5] Verifying StatGovernance React Frontend Build..." -ForegroundColor Yellow
Set-Location "$root\StatGovernance\frontend"
npm.cmd run build
if ($LASTEXITCODE -ne 0) {
    Write-Host 'StatGovernance frontend build failed!' -ForegroundColor Red
    $allPassed = $false
} else {
    Write-Host 'StatGovernance frontend build passed successfully!' -ForegroundColor Green
}

Set-Location $root

Write-Host "`n==========================================================" -ForegroundColor Cyan
if ($allPassed) {
    Write-Host '   ALL PHASE IX FABRIC SUITES PASSED (100%)              ' -ForegroundColor Green
} else {
    Write-Host '   SOME SUITES FAILED - INSPECT OUTPUT ABOVE             ' -ForegroundColor Red
}
Write-Host '==========================================================' -ForegroundColor Cyan
