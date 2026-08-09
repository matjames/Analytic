# StatGate Enterprise Services Startup Script
# Starts the enterprise search/aggregation service

$ErrorActionPreference = "Stop"

Write-Host ""
Write-Host "=======================================" -ForegroundColor Cyan
Write-Host "  StatGate Enterprise Services" -ForegroundColor Cyan
Write-Host "=======================================" -ForegroundColor Cyan
Write-Host ""

# Check if Go is available
if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
    Write-Host "ERROR: Go is not installed or not in PATH." -ForegroundColor Red
    exit 1
}

# Build and start enterprise search service
Write-Host "Building Enterprise Search Service..." -ForegroundColor Yellow
Push-Location "$PSScriptRoot\search"
try {
    go build -o bin/enterprise-search.exe ./cmd/server/
    if ($LASTEXITCODE -ne 0) {
        Write-Host "ERROR: Failed to build enterprise search service." -ForegroundColor Red
        exit 1
    }
    Write-Host "Enterprise Search Service built successfully." -ForegroundColor Green
} finally {
    Pop-Location
}

# Write sample environment file if missing
$envFile = "$PSScriptRoot\search\.env"
if (-not (Test-Path $envFile)) {
    @"
# Enterprise Search Service Configuration
ENTERPRISE_SEARCH_PORT=8095
PMS_API_URL=http://localhost:8091
RMS_API_URL=http://localhost:8092
REGISTRY_API_URL=http://localhost:9090
STATCHAT_API_URL=http://localhost:4000
HELPDESK_API_URL=http://localhost:5006
"@ | Set-Content $envFile
    Write-Host "Created sample .env file for Enterprise Search." -ForegroundColor Green
}

# Run the enterprise search service
Write-Host ""
Write-Host "Starting Enterprise Search Service on port 8095..." -ForegroundColor Yellow
Write-Host "Press Ctrl+C to stop." -ForegroundColor Gray
Write-Host ""

& "$PSScriptRoot\search\bin\enterprise-search.exe"