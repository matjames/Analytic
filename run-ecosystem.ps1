# ═══════════════════════════════════════════════════════════════════════════════
# STATGATE ECOSYSTEM UNIFIED RUNNER
# ═══════════════════════════════════════════════════════════════════════════════

$root = "c:\Users\PC\Desktop\Analytic"
$binDir = "$root\.runtime-bin"
# Credentials come from the environment or the gitignored .env — never hardcode.
$pw = $env:KAGGLE_DB_PASSWORD
if (-not $pw) {
    $envFile = Join-Path $root '.env'
    if (Test-Path $envFile) {
        $line = Select-String -Path $envFile -Pattern '^KAGGLE_DB_PASSWORD=(.*)$' | Select-Object -First 1
        if ($line) { $pw = $line.Matches[0].Groups[1].Value.Trim() }
    }
}
if (-not $pw) { Write-Error 'KAGGLE_DB_PASSWORD is not set and was not found in .env'; exit 1 }
$jwt = "statgate-secret-key-2026"

function Start-ServiceExe {
    param([string]$Name, [string]$ExeName, [hashtable]$Envs, [string]$WorkDir = $root)
    $exePath = Join-Path $binDir $ExeName
    if (-not (Test-Path $exePath)) {
        Write-Host "  [MISSING] $exePath" -ForegroundColor Red
        return
    }

    $Envs["STATGATE_ENV"] = "development"
    $Envs["STATGATE_REGISTRY_JWT_SECRET"] = $jwt
    $Envs["JWT_SECRET"] = $jwt
    $Envs["REDIS_ADDR"] = "127.0.0.1:6379"

    $inline = ($Envs.GetEnumerator() | ForEach-Object { "`$env:{0}='{1}'" -f $_.Key, $_.Value }) -join "; "
    $cmd = "$inline; & '$exePath'"
    Start-Process powershell -ArgumentList "-NoProfile -Command `"$cmd`"" -WorkingDirectory $WorkDir -WindowStyle Hidden
    Write-Host "  [LAUNCHED] $Name -> $ExeName" -ForegroundColor Green
}

Write-Host "`n🚀 Launching StatGate Microservice Backends..." -ForegroundColor Cyan

# 1. Analytics Core (:8080)
Start-ServiceExe -Name "Analytics Core (:8080)" -ExeName "analytics-core.exe" -Envs @{
    PORT = "8080"; KAGGLE_DB_HOST = "127.0.0.1"; KAGGLE_DB_PORT = "5432"; KAGGLE_DB_USER = "Kaggle"; KAGGLE_DB_PASSWORD = $pw; KAGGLE_DB_NAME = "statgate_ml_staging"; KAGGLE_DB_SSLMODE = "disable"; STATGATE_INTERNAL_API_KEY = $jwt
} -WorkDir "$root\backend"

# 2. StatCollect (:8084)
Start-ServiceExe -Name "StatCollect (:8084)" -ExeName "statcollect.exe" -Envs @{
    STATCOLLECT_PORT = ":8084"; STATCOLLECT_DB_HOST = "127.0.0.1"; STATCOLLECT_DB_PORT = "5432"; STATCOLLECT_DB_USER = "statcollect"; STATCOLLECT_DB_PASSWORD = "StatCollect2026"; STATCOLLECT_DB_NAME = "statcollect"; STATCOLLECT_API_KEY = "dev-api-key-2026"; STATCOLLECT_ADMIN_KEYS = "dev-admin-key-2026"; STATGATE_INTERNAL_API_KEY = $jwt
} -WorkDir "$root\StatCollect"

# 3. Registry (:9090)
Start-ServiceExe -Name "Registry Backend (:9090)" -ExeName "stage_register.exe" -Envs @{
    PORT = "9090"; REGISTRY_DB_HOST = "127.0.0.1"; REGISTRY_DB_PORT = "5432"; REGISTRY_DB_USER = "Kaggle"; REGISTRY_DB_PASSWORD = $pw; REGISTRY_DB_NAME = "kaggle"; DB_SSLMODE = "disable"
} -WorkDir "$root\stage_register"

# 4. StatFederation (:8105)
Start-ServiceExe -Name "StatFederation (:8105)" -ExeName "statfederation.exe" -Envs @{
    PORT = "8105"; STATFEDERATION_DB_HOST = "127.0.0.1"; STATFEDERATION_DB_PORT = "5432"; STATFEDERATION_DB_USER = "Kaggle"; STATFEDERATION_DB_PASSWORD = $pw; STATFEDERATION_DB_NAME = "statfederation"; STATFEDERATION_DB_SSLMODE = "disable"; STATGATE_INTERNAL_API_KEY = $jwt
}

# 5. Knowledge Portal (:8099)
Start-ServiceExe -Name "Knowledge Portal (:8099)" -ExeName "knowledge-portal.exe" -Envs @{
    KNOWLEDGE_PORT = "8099"; KNOWLEDGE_DB_HOST = "127.0.0.1"; KNOWLEDGE_DB_PORT = "5432"; KNOWLEDGE_DB_USER = "Kaggle"; KNOWLEDGE_DB_PASSWORD = $pw; KNOWLEDGE_DB_NAME = "knowledge_portal"; KNOWLEDGE_DB_SSLMODE = "disable"
}

# 6. AI Autonomy (:8101)
Start-ServiceExe -Name "AI Autonomy (:8101)" -ExeName "ai-autonomy.exe" -Envs @{
    AIENG_PORT = "8101"; AIENG_DB_HOST = "127.0.0.1"; AIENG_DB_PORT = "5432"; AIENG_DB_USER = "Kaggle"; AIENG_DB_PASSWORD = $pw; AIENG_DB_NAME = "ai_intelligence"; AIENG_DB_SSLMODE = "disable"
}

# 7. Learning CRM (:8102)
Start-ServiceExe -Name "Learning CRM (:8102)" -ExeName "learning-crm.exe" -Envs @{
    LMS_PORT = "8102"; LMS_DB_HOST = "127.0.0.1"; LMS_DB_PORT = "5432"; LMS_DB_USER = "Kaggle"; LMS_DB_PASSWORD = $pw; LMS_DB_NAME = "learning_crm"; LMS_DB_SSLMODE = "disable"
}

# 8. GeoIntel GIS (:8103)
Start-ServiceExe -Name "GeoIntel GIS (:8103)" -ExeName "geointel.exe" -Envs @{
    GIS_PORT = "8103"; GIS_DB_HOST = "127.0.0.1"; GIS_DB_PORT = "5432"; GIS_DB_USER = "Kaggle"; GIS_DB_PASSWORD = $pw; GIS_DB_NAME = "gis_intelligence"; GIS_DB_SSLMODE = "disable"
}

# 9. BPM Hub (:8104)
Start-ServiceExe -Name "BPM Hub (:8104)" -ExeName "bpm-hub.exe" -Envs @{
    BPM_PORT = "8104"; BPM_DB_HOST = "127.0.0.1"; BPM_DB_PORT = "5432"; BPM_DB_USER = "Kaggle"; BPM_DB_PASSWORD = $pw; BPM_DB_NAME = "bpm_hub"; BPM_DB_SSLMODE = "disable"
}

# 10. StatIoT (:8106)
Start-ServiceExe -Name "StatIoT (:8106)" -ExeName "statiot.exe" -Envs @{
    STATIOT_PORT = "8106"; STATIOT_DB_HOST = "127.0.0.1"; STATIOT_DB_PORT = "5432"; STATIOT_DB_USER = "Kaggle"; STATIOT_DB_PASSWORD = $pw; STATIOT_DB_NAME = "statiot"; STATIOT_DB_SSLMODE = "disable"
}

# 11. StatData (:8107)
Start-ServiceExe -Name "StatData (:8107)" -ExeName "statdata.exe" -Envs @{
    PORT = "8107"; APP_ENV = "development"; DB_HOST = "127.0.0.1"; DB_PORT = "5432"; DB_USER = "Kaggle"; DB_PASSWORD = $pw; DB_NAME = "statdata"; DB_SSLMODE = "disable"
}

# 12. StatChat (:8092)
Start-ServiceExe -Name "StatChat (:8092)" -ExeName "statchat.exe" -Envs @{
    PORT = "8092"; STATCHAT_PORT = "8092"; STATCHAT_DB_HOST = "127.0.0.1"; STATCHAT_DB_PORT = "5432"; STATCHAT_DB_USER = "Kaggle"; STATCHAT_DB_PASSWORD = $pw; STATCHAT_DB_NAME = "statchat"; STATCHAT_DB_SSLMODE = "disable"
} -WorkDir "$root\StatChat\backend"

# 13. PMS (:8090)
Start-ServiceExe -Name "PMS (:8090)" -ExeName "pms.exe" -Envs @{
    PORT = "8090"; PMS_DB_HOST = "127.0.0.1"; PMS_DB_PORT = "5432"; PMS_DB_USER = "Kaggle"; PMS_DB_PASSWORD = $pw; PMS_DB_NAME = "pms"; PMS_DB_SSLMODE = "disable"
} -WorkDir "$root\PMS\backend"

# 14. RMS (:8095)
Start-ServiceExe -Name "RMS (:8095)" -ExeName "rms.exe" -Envs @{
    PORT = "8095"; RMS_DB_HOST = "127.0.0.1"; RMS_DB_PORT = "5432"; RMS_DB_USER = "Kaggle"; RMS_DB_PASSWORD = $pw; RMS_DB_NAME = "rms"; RMS_DB_SSLMODE = "disable"
} -WorkDir "$root\RMS\backend"

# 15. StatGovernance (:8093)
Start-ServiceExe -Name "StatGovernance (:8093)" -ExeName "statgovernance.exe" -Envs @{
    PORT = "8093"; GOVERNANCE_DB_HOST = "127.0.0.1"; GOVERNANCE_DB_PORT = "5432"; GOVERNANCE_DB_USER = "Kaggle"; GOVERNANCE_DB_PASSWORD = $pw; GOVERNANCE_DB_NAME = "statgovernance"; GOVERNANCE_DB_SSLMODE = "disable"
} -WorkDir "$root\StatGovernance\backend"

# 16. StatSpatial (:8094)
Start-ServiceExe -Name "StatSpatial (:8094)" -ExeName "statspatial.exe" -Envs @{
    PORT = "8094"; SPATIAL_DB_HOST = "127.0.0.1"; SPATIAL_DB_PORT = "5432"; SPATIAL_DB_USER = "Kaggle"; SPATIAL_DB_PASSWORD = $pw; SPATIAL_DB_NAME = "statspatial"; SPATIAL_DB_SSLMODE = "disable"
} -WorkDir "$root\StatSpatial\backend"

# 17. StatTrust (:8097)
Start-ServiceExe -Name "StatTrust (:8097)" -ExeName "stattrust.exe" -Envs @{
    PORT = "8097"; TRUST_DB_HOST = "127.0.0.1"; TRUST_DB_PORT = "5432"; TRUST_DB_USER = "Kaggle"; TRUST_DB_PASSWORD = $pw; TRUST_DB_NAME = "stattrust"; TRUST_DB_SSLMODE = "disable"
} -WorkDir "$root\StatTrust\backend"

# 18. StatOps (:8098)
Start-ServiceExe -Name "StatOps (:8098)" -ExeName "statops.exe" -Envs @{
    PORT = "8098"; STATOPS_DB_HOST = "127.0.0.1"; STATOPS_DB_PORT = "5432"; STATOPS_DB_USER = "Kaggle"; STATOPS_DB_PASSWORD = $pw; STATOPS_DB_NAME = "statgovernance"; STATOPS_DB_SSLMODE = "disable"
} -WorkDir "$root\StatOps\backend"

Write-Host "`n🚀 Launching Frontend Applications..." -ForegroundColor Cyan

# Frontends
$frontends = @(
  @{ Name = "Analytics Flask UI (:5000)"; Dir = "$root\frontend"; Cmd = "`$env:FLASK_PORT='5000'; `$env:GO_BACKEND_URL='http://localhost:8080'; `$env:STATGATE_INTERNAL_API_KEY='$jwt'; `$env:STATFEDERATION_API_URL='http://localhost:8105'; & '$root\frontend\.venv\Scripts\python.exe' app.py" },
  @{ Name = "StatFederation UI (:3017)"; Dir = "$root\StatFederation\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3017" },
  @{ Name = "StatChat UI (:3009)"; Dir = "$root\StatChat\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3009" },
  @{ Name = "PMS UI (:3010)"; Dir = "$root\PMS\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3010" },
  @{ Name = "RMS UI (:3011)"; Dir = "$root\RMS\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3011" },
  @{ Name = "StatGovernance UI (:3012)"; Dir = "$root\StatGovernance\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3012" },
  @{ Name = "StatSpatial UI (:3014)"; Dir = "$root\StatSpatial\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3014" },
  @{ Name = "StatTrust UI (:3013)"; Dir = "$root\StatTrust\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3013" },
  @{ Name = "StatOps UI (:3015)"; Dir = "$root\StatOps\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3015" },
  @{ Name = "Registry UI (:3007)"; Dir = "$root\stage_register\frontend"; Cmd = "npm.cmd start" }
)

foreach ($f in $frontends) {
  Start-Process powershell -ArgumentList "-NoProfile -Command `"$($f.Cmd)`"" -WorkingDirectory $f.Dir -WindowStyle Hidden
  Write-Host "  [LAUNCHED] $($f.Name)" -ForegroundColor Green
}

Write-Host "`nAll 18 backends and 10 frontends are now active!" -ForegroundColor Cyan
