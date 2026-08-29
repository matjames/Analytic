param(
    [ValidateRange(100001, 10000000)]
    [int]$RowsPerSurvey = 100001,
    [string]$TenantId = "default",
    [string]$Container = "statcollect-postgres-1",
    [string]$Database = "statcollect",
    [string]$User = "statcollect"
)

$ErrorActionPreference = "Stop"
$sqlFile = Join-Path $PSScriptRoot "odk_synthetic_surveys.sql"
if (-not (Test-Path -LiteralPath $sqlFile)) { throw "SQL fixture not found: $sqlFile" }

$running = docker.exe inspect -f '{{.State.Running}}' $Container 2>$null
if ($LASTEXITCODE -ne 0 -or $running -ne 'true') {
    throw "PostgreSQL container '$Container' is not running. Start StatCollect first or pass -Container."
}

Write-Host "Loading 10 surveys x $RowsPerSurvey rows ($($RowsPerSurvey * 10) total) into $Database..."
Get-Content -LiteralPath $sqlFile -Raw |
    docker.exe exec -i $Container psql -v ON_ERROR_STOP=1 -v "rows_per_survey=$RowsPerSurvey" -v "tenant_id=$TenantId" -U $User -d $Database
if ($LASTEXITCODE -ne 0) { throw "Synthetic survey load failed with exit code $LASTEXITCODE" }

Write-Host "Synthetic ODK survey load completed."
