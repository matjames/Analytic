# ═══════════════════════════════════════════════════════════════════════════════
# STATGATE SOVEREIGN PLATFORM — START ALL 12 APPS & SERVICES
# ═══════════════════════════════════════════════════════════════════════════════

$root = "c:\Users\PC\Desktop\Analytic"
$pw = "Statgate_kaggle"
$jwt = "statgate-secret-key-2026"

Write-Host "`n🚀 Starting all StatGate Backends and Frontends..." -ForegroundColor Cyan

# 1. Backends
$backends = @(
  @{ Name = "Analytics Core"; Dir = "$root"; Cmd = "`$env:PORT='8080'; `$env:STATGATE_ENV='development'; `$env:KAGGLE_DB_HOST='127.0.0.1'; `$env:KAGGLE_DB_PORT='5432'; `$env:KAGGLE_DB_USER='Kaggle'; `$env:KAGGLE_DB_PASSWORD='$pw'; `$env:KAGGLE_DB_NAME='statgate_ml_staging'; `$env:KAGGLE_DB_SSLMODE='disable'; `$env:STATGATE_INTERNAL_API_KEY='$jwt'; go run ./cmd/server/main.go" },
  @{ Name = "StatCollect"; Dir = "$root\StatCollect"; Cmd = "`$env:STATCOLLECT_PORT=':8084'; `$env:STATCOLLECT_DB_USER='statcollect'; `$env:STATCOLLECT_DB_PASSWORD='StatCollect2026'; `$env:STATCOLLECT_DB_NAME='statcollect'; `$env:STATCOLLECT_DB_HOST='127.0.0.1'; `$env:STATCOLLECT_API_KEY='dev-api-key-2026'; `$env:STATCOLLECT_ADMIN_KEYS='dev-admin-key-2026'; `$env:REDIS_ADDR='127.0.0.1:6379'; `$env:STATGATE_INTERNAL_API_KEY='$jwt'; go run ./cmd/statcollect/main.go" },
  @{ Name = "Registry Backend"; Dir = "$root\stage_register"; Cmd = "`$env:PORT='9090'; `$env:REGISTRY_DB_HOST='127.0.0.1'; `$env:REGISTRY_DB_PORT='5432'; `$env:REGISTRY_DB_USER='Kaggle'; `$env:REGISTRY_DB_PASSWORD='$pw'; `$env:REGISTRY_DB_NAME='kaggle'; `$env:DB_SSLMODE='disable'; `$env:STATGATE_REGISTRY_JWT_SECRET='$jwt'; `$env:JWT_SECRET='$jwt'; go run ./main.go" },
  @{ Name = "StatFederation Backend"; Dir = "$root\StatFederation\backend"; Cmd = "`$env:PORT='8105'; `$env:STATFEDERATION_DB_HOST='127.0.0.1'; `$env:STATFEDERATION_DB_PORT='5432'; `$env:STATFEDERATION_DB_USER='Kaggle'; `$env:STATFEDERATION_DB_PASSWORD='$pw'; `$env:STATFEDERATION_DB_NAME='statfederation'; `$env:STATFEDERATION_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; `$env:STATGATE_INTERNAL_API_KEY='$jwt'; go run ./cmd/server/main.go" },
  @{ Name = "Knowledge Portal"; Dir = "$root\knowledge-portal"; Cmd = "`$env:KNOWLEDGE_PORT='8099'; `$env:KNOWLEDGE_DB_HOST='127.0.0.1'; `$env:KNOWLEDGE_DB_PORT='5432'; `$env:KNOWLEDGE_DB_USER='Kaggle'; `$env:KNOWLEDGE_DB_PASSWORD='$pw'; `$env:KNOWLEDGE_DB_NAME='knowledge_portal'; `$env:KNOWLEDGE_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "AI Autonomy Engine"; Dir = "$root\ai-autonomy"; Cmd = "`$env:AIENG_PORT='8101'; `$env:AIENG_DB_HOST='127.0.0.1'; `$env:AIENG_DB_PORT='5432'; `$env:AIENG_DB_USER='Kaggle'; `$env:AIENG_DB_PASSWORD='$pw'; `$env:AIENG_DB_NAME='ai_intelligence'; `$env:AIENG_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "Learning CRM"; Dir = "$root\learning-crm"; Cmd = "`$env:LMS_PORT='8102'; `$env:LMS_DB_HOST='127.0.0.1'; `$env:LMS_DB_PORT='5432'; `$env:LMS_DB_USER='Kaggle'; `$env:LMS_DB_PASSWORD='$pw'; `$env:LMS_DB_NAME='learning_crm'; `$env:LMS_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "GeoIntel GIS"; Dir = "$root\geointel"; Cmd = "`$env:GIS_PORT='8103'; `$env:GIS_DB_HOST='127.0.0.1'; `$env:GIS_DB_PORT='5432'; `$env:GIS_DB_USER='Kaggle'; `$env:GIS_DB_PASSWORD='$pw'; `$env:GIS_DB_NAME='gis_intelligence'; `$env:GIS_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "BPM Hub"; Dir = "$root\bpm-hub"; Cmd = "`$env:BPM_PORT='8104'; `$env:BPM_DB_HOST='127.0.0.1'; `$env:BPM_DB_PORT='5432'; `$env:BPM_DB_USER='Kaggle'; `$env:BPM_DB_PASSWORD='$pw'; `$env:BPM_DB_NAME='bpm_hub'; `$env:BPM_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "StatIoT"; Dir = "$root\StatIoT\backend"; Cmd = "`$env:STATIOT_PORT='8106'; `$env:STATIOT_DB_HOST='127.0.0.1'; `$env:STATIOT_DB_PORT='5432'; `$env:STATIOT_DB_USER='Kaggle'; `$env:STATIOT_DB_PASSWORD='$pw'; `$env:STATIOT_DB_NAME='statiot'; `$env:STATIOT_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "StatData Lake"; Dir = "$root\StatData\backend"; Cmd = "`$env:PORT='8107'; `$env:APP_ENV='development'; `$env:DB_HOST='127.0.0.1'; `$env:DB_PORT='5432'; `$env:DB_USER='Kaggle'; `$env:DB_PASSWORD='$pw'; `$env:DB_NAME='statdata'; `$env:DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "StatChat Backend"; Dir = "$root\StatChat\backend"; Cmd = "`$env:PORT='8092'; `$env:STATCHAT_PORT='8092'; `$env:STATCHAT_DB_HOST='127.0.0.1'; `$env:STATCHAT_DB_PORT='5432'; `$env:STATCHAT_DB_USER='Kaggle'; `$env:STATCHAT_DB_PASSWORD='$pw'; `$env:STATCHAT_DB_NAME='statchat'; `$env:STATCHAT_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "PMS Backend"; Dir = "$root\PMS\backend"; Cmd = "`$env:PORT='8090'; `$env:PMS_DB_HOST='127.0.0.1'; `$env:PMS_DB_PORT='5432'; `$env:PMS_DB_USER='Kaggle'; `$env:PMS_DB_PASSWORD='$pw'; `$env:PMS_DB_NAME='pms'; `$env:PMS_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./main.go" },
  @{ Name = "RMS Backend"; Dir = "$root\RMS\backend"; Cmd = "`$env:PORT='8095'; `$env:RMS_DB_HOST='127.0.0.1'; `$env:RMS_DB_PORT='5432'; `$env:RMS_DB_USER='Kaggle'; `$env:RMS_DB_PASSWORD='$pw'; `$env:RMS_DB_NAME='rms'; `$env:RMS_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./main.go" },
  @{ Name = "StatGovernance Backend"; Dir = "$root\StatGovernance\backend"; Cmd = "`$env:PORT='8093'; `$env:GOVERNANCE_DB_HOST='127.0.0.1'; `$env:GOVERNANCE_DB_PORT='5432'; `$env:GOVERNANCE_DB_USER='Kaggle'; `$env:GOVERNANCE_DB_PASSWORD='$pw'; `$env:GOVERNANCE_DB_NAME='statgovernance'; `$env:GOVERNANCE_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./main.go" },
  @{ Name = "StatSpatial Backend"; Dir = "$root\StatSpatial\backend"; Cmd = "`$env:PORT='8094'; `$env:SPATIAL_DB_HOST='127.0.0.1'; `$env:SPATIAL_DB_PORT='5432'; `$env:SPATIAL_DB_USER='Kaggle'; `$env:SPATIAL_DB_PASSWORD='$pw'; `$env:SPATIAL_DB_NAME='statspatial'; `$env:SPATIAL_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./cmd/server/main.go" },
  @{ Name = "StatTrust Backend"; Dir = "$root\StatTrust\backend"; Cmd = "`$env:PORT='8097'; `$env:TRUST_DB_HOST='127.0.0.1'; `$env:TRUST_DB_PORT='5432'; `$env:TRUST_DB_USER='Kaggle'; `$env:TRUST_DB_PASSWORD='$pw'; `$env:TRUST_DB_NAME='stattrust'; `$env:TRUST_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./main.go" },
  @{ Name = "StatOps Backend"; Dir = "$root\StatOps\backend"; Cmd = "`$env:PORT='8098'; `$env:STATOPS_DB_HOST='127.0.0.1'; `$env:STATOPS_DB_PORT='5432'; `$env:STATOPS_DB_USER='Kaggle'; `$env:STATOPS_DB_PASSWORD='$pw'; `$env:STATOPS_DB_NAME='statgovernance'; `$env:STATOPS_DB_SSLMODE='disable'; `$env:REDIS_ADDR='127.0.0.1:6379'; go run ./main.go" }
)

foreach ($b in $backends) {
  Write-Host "  Starting $($b.Name)..." -ForegroundColor Yellow
  Start-Process powershell -ArgumentList "-NoProfile -Command `"$($b.Cmd)`"" -WorkingDirectory $b.Dir -WindowStyle Hidden
}

# 2. Frontends
$frontends = @(
  @{ Name = "Analytics Flask UI"; Dir = "$root\frontend"; Cmd = "`$env:FLASK_PORT='5000'; `$env:GO_BACKEND_URL='http://localhost:8080'; `$env:STATGATE_INTERNAL_API_KEY='$jwt'; `$env:STATFEDERATION_API_URL='http://localhost:8105'; & '$root\frontend\.venv\Scripts\python.exe' app.py" },
  @{ Name = "StatFederation UI"; Dir = "$root\StatFederation\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3017" },
  @{ Name = "StatChat Frontend"; Dir = "$root\StatChat\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3009" },
  @{ Name = "PMS Frontend"; Dir = "$root\PMS\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3010" },
  @{ Name = "RMS Frontend"; Dir = "$root\RMS\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3011" },
  @{ Name = "StatGovernance Frontend"; Dir = "$root\StatGovernance\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3012" },
  @{ Name = "StatSpatial Frontend"; Dir = "$root\StatSpatial\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3014" },
  @{ Name = "StatTrust Frontend"; Dir = "$root\StatTrust\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3013" },
  @{ Name = "StatOps Frontend"; Dir = "$root\StatOps\frontend"; Cmd = "npm.cmd run dev -- --host 0.0.0.0 --port 3015" },
  @{ Name = "Registry Frontend"; Dir = "$root\stage_register\frontend"; Cmd = "npm.cmd start" }
)

foreach ($f in $frontends) {
  Write-Host "  Starting $($f.Name)..." -ForegroundColor Yellow
  Start-Process powershell -ArgumentList "-NoProfile -Command `"$($f.Cmd)`"" -WorkingDirectory $f.Dir -WindowStyle Hidden
}

Write-Host "`nAll services and UIs initiated in background." -ForegroundColor Green
