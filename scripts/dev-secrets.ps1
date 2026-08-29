# Fills missing/empty mandatory secrets in .env with freshly generated dev values.
# Never prints the secret values — only key=SET/EMPTY indicators.
$ErrorActionPreference = 'Stop'
$path = 'c:\Users\PC\Desktop\Analytic\.env'
$text = Get-Content -Raw -Path $path

function New-Secret {
    $b = New-Object byte[] 32
    [Security.Cryptography.RandomNumberGenerator]::Create().GetBytes($b)
    ($b | ForEach-Object { $_.ToString('x2') }) -join ''
}

# user@... name -> secret
$assign = @{
    'HELPDESK_DB_PASSWORD'            = (New-Secret)
    'STATGATE_INTERNAL_API_KEY'       = (New-Secret)
    'FLASK_SECRET_KEY'                = (New-Secret)
    'STATGATE_REGISTRY_JWT_SECRET'    = (New-Secret)
    'HELPDESK_JWT_SECRET'             = (New-Secret)
    'STATGATE_DB_PASSWORD'            = (New-Secret)
    'BASIC_AUTH_PASSWORD'             = (New-Secret)
    'MINIO_ROOT_PASSWORD'             = (New-Secret)
    'GRAFANA_ADMIN_PASSWORD'          = (New-Secret)
    'AIRFLOW_PASSWORD'                = (New-Secret)
    'SUPERSET_ADMIN_PASSWORD'         = (New-Secret)
    'GOVERNANCE_DB_PASSWORD'          = (New-Secret)
    'STATSPATIAL_DB_PASSWORD'         = (New-Secret)
    'KNOWLEDGE_DB_PASSWORD'           = (New-Secret)
    'AIENG_DB_PASSWORD'               = (New-Secret)
    'LMS_DB_PASSWORD'                 = (New-Secret)
    'GIS_DB_PASSWORD'                 = (New-Secret)
    'BPM_DB_PASSWORD'                 = (New-Secret)
    'STATFEDERATION_DB_PASSWORD'      = (New-Secret)
    'STATIOT_DB_PASSWORD'             = (New-Secret)
    'STATDATA_DB_PASSWORD'            = (New-Secret)
    'REDIS_PASSWORD'                  = (New-Secret)
    'SUPERSET_SECRET_KEY'             = (New-Secret)
    'STATIOT_GATEWAY_SECRET'          = (New-Secret)
    'INTEGRATION_DB_PASSWORD'         = (New-Secret)
    'RUNOPS_DB_PASSWORD'              = (New-Secret)
}
$users = @{
    'BASIC_AUTH_USERNAME'    = 'statgate-admin'
    'MINIO_ROOT_USER'        = 'minioadmin'
    'GRAFANA_ADMIN_USER'     = 'admin'
    'AIRFLOW_USER'           = 'airflow'
    'SUPERSET_ADMIN_USER'    = 'admin'
}

# 1) Replace any existing "KEY=" (empty) line with "KEY=<value>"
foreach ($k in $assign.Keys) {
    $re = "(?m)^${k}=\r?\n"
    if ($text -match $re) {
        $text = [regex]::Replace($text, $re, "$k=$($assign[$k])`n")
    }
}
foreach ($k in $users.Keys) {
    $re = "(?m)^${k}=\r?$"
    if ($text -match $re) { $text = [regex]::Replace($text, $re, "$k=$($users[$k])`n") }
}

# 2) Append any that are still missing entirely (so docker compose gets them)
$appended = @()
foreach ($k in ($assign.Keys + $users.Keys)) {
    if ($text -notmatch "(?m)^\s*$k=") {
        $v = if ($assign.ContainsKey($k)) { $assign[$k] } else { $users[$k] }
        $text += "`n$k=$v"
        $appended += $k
    }
}

$text = $text.TrimEnd("`r","`n") + "`n"
Set-Content -Path $path -Value $text -NoNewline

# 3) Report status (never the values)
Write-Output '--- .env secret coverage (post-write) ---'
Get-Content $path | Where-Object { $_ -match '^[A-Za-z0-9_]+=' } | ForEach-Object {
    if ($_ -match '^([^=]+)=(.*)$') {
        $s = if ([string]::IsNullOrWhiteSpace($matches[2])) { 'EMPTY' } else { 'SET' }
        "[$s] $($matches[1])"
    }
}
Write-Output ('appended-new: ' + ($appended -join ', '))