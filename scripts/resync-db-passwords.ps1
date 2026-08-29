# Resync existing Postgres role passwords to the current .env values (ALTER ROLE),
# so containerized apps authenticate after .env secret rotation. Preserves all data.
$ErrorActionPreference = 'Stop'
$root = 'c:\Users\PC\Desktop\Analytic'
$envFile = Join-Path $root '.env'
$env = @{}
Get-Content $envFile | Where-Object { $_ -match '^[A-Za-z0-9_]+=' } | ForEach-Object {
    $i = $_.IndexOf('='); $env[$_.Substring(0, $i)] = $_.Substring($i + 1)
}

# role -> .env key
$map = [ordered]@{
    'statgate'          = 'HELPDESK_DB_PASSWORD'
    'Statchat'          = 'STATCHAT_DB_PASSWORD'
    'PMS'               = 'PMS_DB_PASSWORD'
    'RMS'               = 'RMS_DB_PASSWORD'
    'StatGovernance'    = 'GOVERNANCE_DB_PASSWORD'
    'StatSpatial'       = 'STATSPATIAL_DB_PASSWORD'
    'KnowledgePortal'   = 'KNOWLEDGE_DB_PASSWORD'
    'AIEngine'          = 'AIENG_DB_PASSWORD'
    'StatFederation'    = 'STATFEDERATION_DB_PASSWORD'
    'LearningCRM'       = 'LMS_DB_PASSWORD'
    'GISEngine'         = 'GIS_DB_PASSWORD'
    'BPMEngine'         = 'BPM_DB_PASSWORD'
    'StatDataEngine'    = 'STATDATA_DB_PASSWORD'
}

# sanity: passwords must be present
foreach ($k in $map.Values) { if ([string]::IsNullOrWhiteSpace($env[$k])) { throw "missing .env value for $k" } }

# Pre-flight: does registry password match the Kaggle superuser password?
$kaggle = $env['KAGGLE_DB_PASSWORD']; $reg = $env['REGISTRY_DB_PASSWORD']
if ($kaggle -eq $reg) { Write-Output 'INFO: REGISTRY_DB_PASSWORD == KAGGLE_DB_PASSWORD (registry can log in as Kaggle)' }
else { Write-Output 'WARN: REGISTRY_DB_PASSWORD != KAGGLE_DB_PASSWORD; registry-api may fail auth as user Kaggle.' }

$sqlLines = @()
$sqlLines += 'BEGIN;'
foreach ($role in $map.Keys) {
    $pw = $env[$map[$role]]
    $lit = "'" + ($pw -replace "'", "''") + "'"
    $q = "'" + ($role -replace "'", "''") + "'"
    $sqlLines += "SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', $q, $lit) WHERE NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = $q) \gexec"
    $alt = if ($role -cmatch '[A-Z]') { '"' + ($role -replace '"', '""') + '"' } else { $role }
    $sqlLines += "ALTER ROLE $alt WITH LOGIN PASSWORD $lit;"
}
$sqlLines += 'COMMIT;'
$sql = $sqlLines -join "`n"
Write-Output "Executing password resync for $($map.Count) roles ..."
$sql | docker exec -i analytic-postgres-1 psql -v ON_ERROR_STOP=1 -U Kaggle -d postgres
if ($LASTEXITCODE -ne 0) { throw "psql resync failed (exit $LASTEXITCODE)" }
Write-Output 'Role password resync completed.'