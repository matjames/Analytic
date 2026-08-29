# StatGate Registry (stage_register) local launcher.
# Loads registry connection settings from the repo-root .env so the Registry
# serves users from the registry DB (kaggle) on :9090, and signs tokens with
# the shared StatGate secret that StatChat validates.

$root = Split-Path -Parent $PSScriptRoot | Split-Path -Parent
$rootEnv = Join-Path $root '.env'
$vars = @{}
if (Test-Path $rootEnv) {
    Get-Content $rootEnv | ForEach-Object {
        if ($_ -match '^\s*([A-Z0-9_]+)\s*=\s*(.+)\s*$') {
            $vars[$Matches[1]] = $Matches[2]
        }
    }
}

function GetEnv($key, $def) {
    if ($vars.ContainsKey($key) -and $vars[$key]) { return $vars[$key] }
    return $def
}

$env:PORT = '9090'
$env:REGISTRY_DB_HOST  = GetEnv 'REGISTRY_DB_HOST'  'localhost'
$env:REGISTRY_DB_PORT  = GetEnv 'REGISTRY_DB_PORT'  '5432'
$env:REGISTRY_DB_USER  = GetEnv 'REGISTRY_DB_USER'  'Kaggle'
$env:REGISTRY_DB_PASSWORD = GetEnv 'REGISTRY_DB_PASSWORD' ''
$env:REGISTRY_DB_NAME  = GetEnv 'REGISTRY_DB_NAME'  'kaggle'
$env:DB_SSLMODE        = GetEnv 'REGISTRY_DB_SSLMODE' 'disable'
$env:STATGATE_REGISTRY_JWT_SECRET = GetEnv 'STATGATE_REGISTRY_JWT_SECRET' ''
$env:JWT_SECRET        = $env:STATGATE_REGISTRY_JWT_SECRET
$env:STATGATE_INTERNAL_API_KEY = GetEnv 'STATGATE_INTERNAL_API_KEY' ''
$env:BASIC_AUTH_USERNAME = GetEnv 'BASIC_AUTH_USERNAME' ''
$env:BASIC_AUTH_PASSWORD = GetEnv 'BASIC_AUTH_PASSWORD' ''
$env:STATGATE_ENV      = GetEnv 'STATGATE_ENV' 'development'

if (-not $env:REGISTRY_DB_PASSWORD) {
    Write-Host 'FATAL: REGISTRY_DB_PASSWORD not found in repo-root .env'
    exit 1
}

Push-Location $PSScriptRoot
& '.\app.exe'
Pop-Location