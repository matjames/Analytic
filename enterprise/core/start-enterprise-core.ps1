# ══════════════════════════════════════════════════════════════
# STATGATE ENTERPRISE CORE - PHASE II
# Start the Enterprise Core service locally
# ══════════════════════════════════════════════════════════════

$ErrorActionPreference = "Stop"
$rootDir = Split-Path -Parent $PSScriptRoot
$coreDir = $PSScriptRoot

# Check if Redis is running
$redisRunning = $false
try {
    $test = Test-NetConnection -ComputerName localhost -Port 6379 -WarningAction SilentlyContinue
    $redisRunning = $test.TcpTestSucceeded
} catch {
    $redisRunning = $false
}

if (-not $redisRunning) {
    Write-Host "⚠️  Redis is not running. The event bus will be disabled." -ForegroundColor Yellow
    Write-Host "   Start Redis first:  docker compose up -d redis" -ForegroundColor Yellow
}

# Load environment
if (Test-Path "$rootDir\.env") {
    Write-Host "Loading environment from $rootDir\.env" -ForegroundColor Green
    Get-Content "$rootDir\.env" | ForEach-Object {
        if ($_ -match '^\s*([^#][^=]+)=(.*)$') {
            $key = $matches[1].Trim()
            $value = $matches[2].Trim()
            [System.Environment]::SetEnvironmentVariable($key, $value, "Process")
        }
    }
}

Write-Host ""
Write-Host "════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  StatGate Enterprise Core - Phase II" -ForegroundColor Cyan
Write-Host "  Event Bus • Notifications • Timeline • Dashboards" -ForegroundColor Cyan
Write-Host "  Widgets • Files • Permissions • Calendar • Reports" -ForegroundColor Cyan
Write-Host "  AI Prep • API Governance • Monitoring" -ForegroundColor Cyan
Write-Host "════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host ""

# Build and run
Push-Location $coreDir
try {
    Write-Host "Building enterprise core..." -ForegroundColor Green
    go build -o statgate-enterprise-core.exe .
    Write-Host "Starting on http://localhost:8096" -ForegroundColor Green
    Write-Host "Press Ctrl+C to stop" -ForegroundColor Gray
    Write-Host ""
    & .\statgate-enterprise-core.exe
} finally {
    Pop-Location
}