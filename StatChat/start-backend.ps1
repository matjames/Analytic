# Load DATABASE_URL and BACKEND_PORT from the root .env (single source of truth)
$envFile = Join-Path (Split-Path -Parent $PSScriptRoot) '.env'
if (Test-Path $envFile) {
    Get-Content $envFile | ForEach-Object {
        if ($_ -match '^\s*DATABASE_URL\s*=\s*(.+)$') {
            $env:DATABASE_URL = $Matches[1].Trim()
        }
        if ($_ -match '^\s*BACKEND_PORT\s*=\s*(.+)$') {
            $env:BACKEND_PORT = $Matches[1].Trim()
        }
    }
}

# Defaults if not found in .env
if (-not $env:DATABASE_URL) {
    $env:DATABASE_URL = "postgres://Statchat:Statgate@localhost:5432/statchat?sslmode=disable"
}
if (-not $env:BACKEND_PORT) {
    $env:BACKEND_PORT = "4000"
}

Push-Location "$PSScriptRoot\backend"
go run .\cmd\server
Pop-Location