<#
.SYNOPSIS
    StatGate Sovereign Analytical Hub — Automated Turnkey Bootstrap Script
.DESCRIPTION
    Configures environment variables, generates cryptographic keys, verifies Docker prerequisites,
    and initializes the designated sector analytical hub profile.
.EXAMPLE
    .\setup-hub.ps1 -Sector "nso" -HubName "National Bureau of Statistics" -Domain "nso.gov.local"
#>

param(
    [ValidateSet("nso", "health", "development", "environment", "research", "full")]
    [string]$Sector = "nso",

    [string]$HubName = "StatGate Sovereign Analytical Hub",
    [string]$Domain = "localhost",
    [string]$CountryCode = "UG"
)

Write-Host "══════════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  STATGATE SOVEREIGN ANALYTICAL HUB — BOOTSTRAP INITIALIZATION" -ForegroundColor Cyan
Write-Host "  Sector Profile: $Sector | Hub: $HubName | Domain: $Domain" -ForegroundColor Yellow
Write-Host "══════════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan

# 1. Verify Prerequisites
Write-Host "`n[1/4] Checking prerequisites..." -ForegroundColor White
$dockerInstalled = Get-Command "docker" -ErrorAction SilentlyContinue
if (-not $dockerInstalled) {
    Write-Error "Docker is not installed or not in PATH. Please install Docker Desktop or Engine first."
    exit 1
}
Write-Host "  ✓ Docker Engine detected: $(docker --version)" -ForegroundColor Green

# 2. Generate Cryptographic Secrets if .env doesn't exist
Write-Host "`n[2/4] Generating cryptographic tokens and secrets..." -ForegroundColor White
$envPath = Join-Path $PSScriptRoot ".env"

function Generate-SecureSecret([int]$length = 32) {
    $bytes = New-Object byte[] $length
    $rng = [System.Security.Cryptography.RandomNumberGenerator]::Create()
    $rng.GetBytes($bytes)
    return [Convert]::ToBase64String($bytes).Replace('+', '').Replace('/', '').Replace('=', '').Substring(0, $length)
}

if (-not (Test-Path $envPath)) {
    Write-Host "  Generating fresh secure .env from template..." -ForegroundColor Yellow
    $registrySecret = Generate-SecureSecret 48
    $internalKey = Generate-SecureSecret 32
    $flaskSecret = Generate-SecureSecret 32
    $dbPassword = Generate-SecureSecret 24

    $envContent = @"
# ─── StatGate Analytical Hub Configuration ───
HUB_NAME=$HubName
HUB_SECTOR=$Sector
HUB_DOMAIN=$Domain
HUB_COUNTRY_CODE=$CountryCode

# ─── Security Secrets ───
STATGATE_REGISTRY_JWT_SECRET=$registrySecret
STATGATE_INTERNAL_API_KEY=$internalKey
FLASK_SECRET_KEY=$flaskSecret
KAGGLE_DB_PASSWORD=$dbPassword
KAGGLE_DB_USER=Kaggle
KAGGLE_DB_NAME=statgate_ml_staging

# ─── Service Ports ───
PORT=8080
FLASK_PORT=5000
STATFEDERATION_PORT=8105
STATFEDERATION_UI_PORT=3016
"@
    Set-Content -Path $envPath -Value $envContent -Encoding UTF8
    Write-Host "  ✓ .env created successfully with high-entropy cryptographic keys." -ForegroundColor Green
} else {
    Write-Host "  ✓ Existing .env file detected." -ForegroundColor Green
}

# 3. Apply Sector Profile
Write-Host "`n[3/4] Validating sector profile '$Sector'..." -ForegroundColor White
Write-Host "  Selected profile '$Sector' will orchestrate the following services:" -ForegroundColor Yellow
switch ($Sector) {
    "nso" {
        Write-Host "    • Analytics Core (Port 8080)" -ForegroundColor Cyan
        Write-Host "    • Analytics UI & Executive Hub (Port 5000)" -ForegroundColor Cyan
        Write-Host "    • Tabulation & SDMX Generator (/tabulate)" -ForegroundColor Cyan
        Write-Host "    • Sampling Framework Designer (/sampling)" -ForegroundColor Cyan
        Write-Host "    • StatFederation Sovereign Push (Port 8105 / 3016)" -ForegroundColor Cyan
        Write-Host "    • Knowledge Portal & CKAN 3.0 Open Data (Port 8099)" -ForegroundColor Cyan
    }
    "health" {
        Write-Host "    • Analytics Core & Anomaly Lakehouse (Port 8080)" -ForegroundColor Cyan
        Write-Host "    • StatIoT Telemetry Bridge (Port 8080)" -ForegroundColor Cyan
        Write-Host "    • GeoIntel Spatial Analysis (Port 8103)" -ForegroundColor Cyan
        Write-Host "    • Health Evidence Executive Dashboard (Port 5000)" -ForegroundColor Cyan
    }
    "development" {
        Write-Host "    • PMS Project Management & IATI Exporter (Port 8087)" -ForegroundColor Cyan
        Write-Host "    • RMS Resource Management (Port 8088)" -ForegroundColor Cyan
        Write-Host "    • StatFederation Multilateral Reports (Port 8105)" -ForegroundColor Cyan
    }
    default {
        Write-Host "    • Full Enterprise Unified Suite (All 15+ Microservices)" -ForegroundColor Cyan
    }
}

# 4. Ready to Launch Command
Write-Host "`n[4/4] Setup complete!" -ForegroundColor Green
Write-Host "══════════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "To start your sovereign hub, run:" -ForegroundColor White
Write-Host "  docker compose up -d" -ForegroundColor Yellow -NoNewline
Write-Host " (or docker compose --profile $Sector up -d)" -ForegroundColor White
Write-Host "`nTo check running hub status:" -ForegroundColor White
Write-Host "  curl http://localhost:8080/ready" -ForegroundColor Yellow
Write-Host "  curl http://localhost:5000/health" -ForegroundColor Yellow
Write-Host "══════════════════════════════════════════════════════════════════════════" -ForegroundColor Cyan
