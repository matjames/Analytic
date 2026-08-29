# ═══════════════════════════════════════════════════════════════════════════════
# STATGATE — LOCAL DEV: PROVISION DBs & START ALL BACKENDS
#
#   powershell -ExecutionPolicy Bypass -File start-local.ps1 -DbPassword <pw>
#
#   -DbPassword : PostgreSQL superuser or app password (required unless the
#                 per-app platforms run their own DB roles)
#   -SkipBuild  : reuse previously built binaries in ./.runtime-bin
#   -NoRedis    : force in-memory event-bus fallback (default when Redis is down)
#
# Starts: knowledge-portal :8099, ai-autonomy :8101, learning-crm :8102,
#          geointel :8103, bpm-hub :8104, statfederation :8105,
#          statiot :8106, statdata :8107
# ═══════════════════════════════════════════════════════════════════════════════

param(
    [string]$DbPassword = $env:KAGGLE_DB_PASSWORD,
    [switch]$SkipBuild,
    [switch]$NoRedis
)

$ErrorActionPreference = "Continue"
$root = $PSScriptRoot
$binDir = Join-Path $root ".runtime-bin"
$logDir = Join-Path $root "data\runtime"
New-Item -ItemType Directory -Force -Path $binDir, $logDir | Out-Null

if ([string]::IsNullOrWhiteSpace($DbPassword)) {
    Write-Host "ERROR: provide -DbPassword <postgres password> (or set KAGGLE_DB_PASSWORD). The 5 database-backed apps cannot start without it." -ForegroundColor Red
    Write-Host "NOTE: statfederation/statiot/statdata will self-start even without DBs (in-memory fallback)."
}

# ── shared runtime environment ────────────────────────────────────────────────
$jwt = $env:STATGATE_REGISTRY_JWT_SECRET
if ([string]::IsNullOrWhiteSpace($jwt)) { $jwt = "statgate-local-dev-secret-key-2026" }

function Start-Backend {
    param(
        [string]$Name, [string]$ModuleDir, [int]$Port,
        [string]$DbName, [string]$BinPath
    )
    $exe = Join-Path $binDir $BinPath
    if (-not $SkipBuild -or -not (Test-Path $exe)) {
        Write-Host "  building $Name ..." -ForegroundColor Gray
        Push-Location (Join-Path $root $ModuleDir)
        try {
            & go build -o $exe ./cmd/server 2>&1 | Out-Null
            if ($LASTEXITCODE -ne 0) { throw "build failed for $Name" }
        } finally { Pop-Location }
    }

    # App-specific environment variable names (each backend has its own prefix).
    $envs = @{}
    switch ($Name) {
        "knowledge-portal" {
            $envs["KNOWLEDGE_PORT"]="8099"; $envs["KNOWLEDGE_DB_HOST"]="127.0.0.1"; $envs["KNOWLEDGE_DB_PORT"]="5432"
            $envs["KNOWLEDGE_DB_USER"]="Kaggle"; $envs["KNOWLEDGE_DB_PASSWORD"]=$DbPassword; $envs["KNOWLEDGE_DB_NAME"]=$DbName; $envs["KNOWLEDGE_DB_SSLMODE"]="disable"
        }
        "ai-autonomy" {
            $envs["AIENG_PORT"]="8101"; $envs["AIENG_DB_HOST"]="127.0.0.1"; $envs["AIENG_DB_PORT"]="5432"
            $envs["AIENG_DB_USER"]="Kaggle"; $envs["AIENG_DB_PASSWORD"]=$DbPassword; $envs["AIENG_DB_NAME"]=$DbName; $envs["AIENG_DB_SSLMODE"]="disable"
        }
        "learning-crm" {
            $envs["LMS_PORT"]="8102"; $envs["LMS_DB_HOST"]="127.0.0.1"; $envs["LMS_DB_PORT"]="5432"
            $envs["LMS_DB_USER"]="Kaggle"; $envs["LMS_DB_PASSWORD"]=$DbPassword; $envs["LMS_DB_NAME"]=$DbName; $envs["LMS_DB_SSLMODE"]="disable"
        }
        "geointel" {
            $envs["GIS_PORT"]="8103"; $envs["GIS_DB_HOST"]="127.0.0.1"; $envs["GIS_DB_PORT"]="5432"
            $envs["GIS_DB_USER"]="Kaggle"; $envs["GIS_DB_PASSWORD"]=$DbPassword; $envs["GIS_DB_NAME"]=$DbName; $envs["GIS_DB_SSLMODE"]="disable"
        }
        "bpm-hub" {
            $envs["BPM_PORT"]="8104"; $envs["BPM_DB_HOST"]="127.0.0.1"; $envs["BPM_DB_PORT"]="5432"
            $envs["BPM_DB_USER"]="Kaggle"; $envs["BPM_DB_PASSWORD"]=$DbPassword; $envs["BPM_DB_NAME"]=$DbName; $envs["BPM_DB_SSLMODE"]="disable"
        }
        "statfederation" {
            $envs["PORT"]="8105"; $envs["STATFEDERATION_DB_HOST"]="127.0.0.1"; $envs["STATFEDERATION_DB_PORT"]="5432"
            $envs["STATFEDERATION_DB_USER"]="Kaggle"; $envs["STATFEDERATION_DB_PASSWORD"]=$DbPassword; $envs["STATFEDERATION_DB_NAME"]=$DbName; $envs["STATFEDERATION_DB_SSLMODE"]="disable"
        }
        "statiot" {
            $envs["STATIOT_PORT"]="8106"; $envs["STATIOT_DB_HOST"]="127.0.0.1"; $envs["STATIOT_DB_PORT"]="5432"
            $envs["STATIOT_DB_USER"]="Kaggle"; $envs["STATIOT_DB_PASSWORD"]=$DbPassword; $envs["STATIOT_DB_NAME"]=$DbName; $envs["STATIOT_DB_SSLMODE"]="disable"
        }
        "statdata" {
            $envs["PORT"]="8107"; $envs["APP_ENV"]="development"; $envs["DB_HOST"]="127.0.0.1"; $envs["DB_PORT"]="5432"
            $envs["DB_USER"]="Kaggle"; $envs["DB_PASSWORD"]=$DbPassword; $envs["DB_NAME"]=$DbName; $envs["DB_SSLMODE"]="disable"
        }
    }
    # Common service env
    $envs["STATGATE_ENV"]="development"
    $envs["STATGATE_REGISTRY_JWT_SECRET"]=$jwt
    $envs["STATGATE_JWT_ISSUER"]="statgate-registry"
    $envs["STATGATE_JWT_AUDIENCE"]="statgate"
    $envs["CORS_ALLOWED_ORIGIN"]="*"
    if ($NoRedis) { $envs["REDIS_HOST"]=""; $envs["REDIS_ADDR"]="" } else { $envs["REDIS_HOST"]="127.0.0.1"; $envs["REDIS_ADDR"]="127.0.0.1:6379" }

    $inlineEnv = (($envs.GetEnumerator() | ForEach-Object { "`$env:{0}='{1}'" -f $_.Key, $_.Value }) -join "; ") + "; & '$exe'"
    $p = Start-Process -FilePath "powershell" -ArgumentList "-NoProfile -Command `"$inlineEnv`"" -WorkingDirectory $root -WindowStyle Hidden -PassThru
    Write-Host ("  started {0} (pid {1}) -> http://localhost:{2}/health" -f $Name, $p.Id, $Port) -ForegroundColor Green
}
# ── 0. Provision databases (requires the postgres password) ─────────────────
if (-not [string]::IsNullOrWhiteSpace($DbPassword)) {
    Write-Host "`n=== Provisioning databases ===" -ForegroundColor Yellow
    Push-Location (Join-Path $root "scripts\dbprovision")
    try { & go run . 2>&1 | Out-String | Write-Host } finally { Pop-Location }
} else {
    Write-Host "`n! Skipping DB provisioning (no -DbPassword). The 5 database-backed apps may exit; statfederation/statiot/statdata self-fallback to in-memory storage." -ForegroundColor Yellow
}

# ── 1. Start all backends ────────────────────────────────────────────────────
Write-Host "`n=== Starting backends ===" -ForegroundColor Yellow
$apps = @(
    @{Name="knowledge-portal"; Module="knowledge-portal\backend"; Path="knowledge-portal.exe"; Db="knowledge_portal"; Port=8099},
    @{Name="ai-autonomy";      Module="ai-autonomy\backend";      Path="ai-autonomy.exe";      Db="ai_intelligence"; Port=8101},
    @{Name="learning-crm";     Module="learning-crm\backend";     Path="learning-crm.exe";     Db="learning_crm";     Port=8102},
    @{Name="geointel";         Module="geointel\backend";         Path="geointel.exe";         Db="gis_intelligence"; Port=8103},
    @{Name="bpm-hub";          Module="bpm-hub\backend";          Path="bpm-hub.exe";          Db="bpm_hub";          Port=8104},
    @{Name="statfederation";   Module="StatFederation\backend";   Path="statfederation.exe";   Db="statfederation";   Port=8105},
    @{Name="statiot";          Module="StatIoT\backend";          Path="statiot.exe";          Db="statiot";          Port=8106},
    @{Name="statdata";         Module="StatData\backend";         Path="statdata.exe";         Db="statdata";         Port=8107}
)
foreach ($a in $apps) {
    try { Start-Backend -Name $a.Name -ModuleDir $a.Module -Port $a.Port -DbName $a.Db -BinPath $a.Path } catch { Write-Host ("  ERROR " + $a.Name + ": " + $_.Exception.Message) -ForegroundColor Red }
}

# ── 2. Health summary ────────────────────────────────────────────────────────
Write-Host "`n=== Health summary ===" -ForegroundColor Yellow
Start-Sleep -Seconds 8
$appPorts = @("knowledge-portal=8099","ai-autonomy=8101","learning-crm=8102","geointel=8103","bpm-hub=8104","statfederation=8105","statiot=8106","statdata=8107")
foreach ($entry in $appPorts) {
    $n, $p = $entry.Split("=")
    try {
        $r = Invoke-RestMethod -Uri "http://localhost:$p/health" -TimeoutSec 3
        Write-Host ("  {0,-16} :{1}  {2}" -f $n, $p, $r.status)
    } catch {
        Write-Host ("  {0,-16} :{1}  UNREACHABLE ({2})" -f $n, $p, $_.Exception.Message)
    }
}
Write-Host "`nLogs & binaries: $logDir ; $binDir" -ForegroundColor Gray
Write-Host "Run with a real Redis to enable cross-process events; without it the event bus uses per-process in-memory fallback." -ForegroundColor Gray