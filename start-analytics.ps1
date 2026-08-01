param()

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
$envFile = Join-Path $repoRoot '.env'

if (-not (Test-Path $envFile)) {
    throw "Missing .env at $envFile. Copy .env.example to .env and set the required keys before starting Analytics."
}

$requiredKeys = @(
    'STATGATE_INTERNAL_API_KEY',
    'FLASK_SECRET_KEY',
    'STATGATE_REGISTRY_JWT_SECRET'
)

$envLines = Get-Content $envFile
$missing = @()
foreach ($name in $requiredKeys) {
    if (-not ($envLines | Where-Object { $_ -match "^\s*$([regex]::Escape($name))\s*=" })) {
        $missing += $name
    }
}

if ($missing.Count -gt 0) {
    throw "Missing required values in .env: $($missing -join ', ')"
}

Write-Host "Starting Analytics services from $repoRoot..."
docker compose --project-directory $repoRoot up --build -d

$urls = @(
    'http://localhost:5001/health',
    'http://localhost:8081/health'
)

foreach ($url in $urls) {
    $healthy = $false
    for ($attempt = 1; $attempt -le 30; $attempt++) {
        try {
            $response = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 5
            if ($response.StatusCode -eq 200) {
                $healthy = $true
                Write-Host "Healthy: $url"
                break
            }
        }
        catch {
            Start-Sleep -Seconds 2
        }
    }

    if (-not $healthy) {
        throw "Service health check failed for $url"
    }
}

Write-Host "Analytics is running. UI: http://localhost:5001"
Write-Host "Core: http://localhost:8081/health"
