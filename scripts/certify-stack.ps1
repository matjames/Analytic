# =====================================================================
# STATGATE FULL-STACK CERTIFICATION
# Brings up the complete docker-compose stack with placeholder secrets,
# waits for container health, then probes every service's health/ready
# endpoint from the host. Exits non-zero if ANY service fails.
#
# Usage:
#   pwsh scripts/certify-stack.ps1                    # build + up + probe
#   pwsh scripts/certify-stack.ps1 -SkipBuild         # reuse built images
#   pwsh scripts/certify-stack.ps1 -UpTimeoutMin 40   # extend startup wait
#
# Reference: docs/BASELINE_RUNBOOK.md (port map, known behaviors)
# =====================================================================
param(
    [switch]$SkipBuild,
    [int]$UpTimeoutMin = 45,
    [int]$ProbeRetries = 10
)

$ErrorActionPreference = 'Continue'
$repo = Split-Path -Parent $PSScriptRoot
Set-Location $repo

# ---- 1. Placeholder secrets (fail-closed vars must be non-empty) ----
$placeholder = 'ci-placeholder-0123456789abcdef'
$requiredVars = @(
    'KAGGLE_DB_PASSWORD','HELPDESK_DB_PASSWORD','REGISTRY_DB_PASSWORD','STATCHAT_DB_PASSWORD',
    'PMS_DB_PASSWORD','RMS_DB_PASSWORD','GOVERNANCE_DB_PASSWORD','STATSPATIAL_DB_PASSWORD',
    'KNOWLEDGE_DB_PASSWORD','AIENG_DB_PASSWORD','LMS_DB_PASSWORD','GIS_DB_PASSWORD',
    'BPM_DB_PASSWORD','STATFEDERATION_DB_PASSWORD','STATIOT_DB_PASSWORD','STATDATA_DB_PASSWORD',
    'STATCOLLECT_DB_PASSWORD','INTEGRATION_DB_PASSWORD','RUNOPS_DB_PASSWORD','STATCITIZEN_DB_PASSWORD',
    'REDIS_PASSWORD','STATGATE_REGISTRY_JWT_SECRET','STATGATE_INTERNAL_API_KEY','FLASK_SECRET_KEY',
    'STATCITIZEN_CITIZEN_SESSION_SECRET','STATIOT_GATEWAY_SECRET','STATCOLLECT_API_KEY',
    'STATCOLLECT_ADMIN_KEYS','MINIO_ROOT_USER','MINIO_ROOT_PASSWORD','BASIC_AUTH_USERNAME',
    'BASIC_AUTH_PASSWORD','GRAFANA_ADMIN_USER','GRAFANA_ADMIN_PASSWORD','AIRFLOW_USER',
    'AIRFLOW_PASSWORD','SUPERSET_SECRET_KEY','SUPERSET_ADMIN_USER','SUPERSET_ADMIN_PASSWORD',
    'TURN_PUBLIC_URL','TURN_REALM','TURN_USERNAME','TURN_CREDENTIAL'
)
foreach ($v in $requiredVars) {
    if (-not [Environment]::GetEnvironmentVariable($v)) {
        Set-Item -Path "Env:$v" -Value $placeholder
    }
}
Write-Host "[certify] $($requiredVars.Count) required vars ensured non-empty"

# ---- 2. Bring the stack up ----
$buildArgs = @('compose')
if ($SkipBuild) { $buildArgs += @('up','-d') } else { $buildArgs += @('up','-d','--build') }
Write-Host "[certify] docker $($buildArgs -join ' ') (this is the long pole on first run)"
& docker @buildArgs 2>&1 | Select-Object -Last 15
if ($LASTEXITCODE -ne 0) { Write-Host '[certify] FATAL: compose up failed' -ForegroundColor Red; exit 2 }

# ---- 3. Wait for container health ----
$deadline = (Get-Date).AddMinutes($UpTimeoutMin)
$composeSvcCount = (& docker compose config --services | Measure-Object).Count
Write-Host "[certify] waiting up to ${UpTimeoutMin}min for $composeSvcCount services to be healthy"
while ((Get-Date) -lt $deadline) {
    $ps = docker compose ps --format json | ConvertFrom-Json
    $unhealthy = @($ps | Where-Object { $_.State -ne 'running' -or ($_.Health -and $_.Health -ne 'healthy') })
    $running   = @($ps | Where-Object { $_.State -eq 'running' })
    Write-Host ("[certify] {0}/{1} running, {2} not-yet-healthy" -f $running.Count, $ps.Count, $unhealthy.Count)
    if ($running.Count -eq $ps.Count -and $unhealthy.Count -eq 0) { break }
    Start-Sleep -Seconds 30
}
# ---- 3b. Final compose health accounting (authoritative for non-HTTP services) ----
$ps = docker compose ps --format json | ConvertFrom-Json
$failed = @($ps | Where-Object { $_.State -ne 'running' -or ($_.Health -and $_.Health -ne 'healthy') })
if ($failed.Count -gt 0) {
    Write-Host "[certify] compose-level failures:" -ForegroundColor Red
    $failed | ForEach-Object { Write-Host ("  - {0}: state={1} health={2}" -f $_.Service, $_.State, $_.Health) }
    $failed | ForEach-Object { & docker compose logs --tail 30 $_.Service 2>&1 | Select-Object -Last 30 }
}

# ---- 4. HTTP health probes (host -> mapped port) ----
# API services: /health; Registry: /health + /ready; UIs: /; monitoring: own paths.
$probes = @(
    @{ Svc='statgate-core';              Port=8082; Path='/health' },
    @{ Svc='statgate-analytics';         Port=5000; Path='/health' },
    @{ Svc='statgate-registry-api';      Port=9090; Path='/health' },
    @{ Svc='statgate-registry-api';      Port=9090; Path='/ready'   },
    @{ Svc='statgate-launcher';          Port=3006; Path='/' },
    @{ Svc='statgate-registry-ui';       Port=3007; Path='/' },
    @{ Svc='statgate-helpdesk-api';      Port=5006; Path='/health' },
    @{ Svc='statgate-helpdesk-ui';       Port=3005; Path='/' },
    @{ Svc='statchat-backend';           Port=4000; Path='/health' },
    @{ Svc='statchat-frontend';          Port=3009; Path='/' },
    @{ Svc='statgate-pms-api';           Port=8091; Path='/health' },
    @{ Svc='statgate-pms-ui';            Port=3010; Path='/' },
    @{ Svc='statgate-rms-api';           Port=8092; Path='/health' },
    @{ Svc='statgate-rms-ui';            Port=3011; Path='/' },
    @{ Svc='statgate-governance-api';    Port=8093; Path='/health' },
    @{ Svc='statgate-governance-ui';     Port=3012; Path='/' },
    @{ Svc='statgate-trust-api';         Port=8094; Path='/health' },
    @{ Svc='statgate-trust-ui';          Port=3013; Path='/' },
    @{ Svc='statgate-ops-api';           Port=8098; Path='/health' },
    @{ Svc='statgate-ops-ui';            Port=3015; Path='/' },
    @{ Svc='statgate-runops-api';        Port=8100; Path='/health' },
    @{ Svc='statgate-integration-fabric';Port=8097; Path='/health' },
    @{ Svc='statgate-enterprise';        Port=8095; Path='/health' },
    @{ Svc='statgate-enterprise-core';   Port=8096; Path='/health' },
    @{ Svc='knowledge-portal';           Port=8099; Path='/health' },
    @{ Svc='ai-autonomy';                Port=8101; Path='/health' },
    @{ Svc='learning-crm';               Port=8102; Path='/health' },
    @{ Svc='geointel';                   Port=8103; Path='/health' },
    @{ Svc='bpm-hub';                    Port=8104; Path='/health' },
    @{ Svc='statfederation';             Port=8105; Path='/health' },
    @{ Svc='statfederation-ui';          Port=3016; Path='/' },
    @{ Svc='statiot';                    Port=8106; Path='/health' },
    @{ Svc='statdata';                   Port=8107; Path='/health' },
    @{ Svc='statgate-spatial-api';       Port=8108; Path='/health' },
    @{ Svc='statgate-spatial-ui';        Port=3017; Path='/' },
    @{ Svc='statcollect';                Port=8081; Path='/health' },
    @{ Svc='statcitizen-api';            Port=8115; Path='/health' },
    @{ Svc='prometheus';                 Port=9095; Path='/-/healthy' },
    @{ Svc='grafana';                    Port=3003; Path='/api/health' },
    @{ Svc='alertmanager';               Port=9093; Path='/-/healthy' },
    @{ Svc='minio';                      Port=9000; Path='/minio/health/live' }
)

$probeFailures = New-Object System.Collections.Generic.List[string]
foreach ($p in $probes) {
    $url = "http://localhost:$($p.Port)$($p.Path)"
    $ok = $false
    for ($i = 0; $i -lt $ProbeRetries -and -not $ok; $i++) {
        try {
            $resp = Invoke-WebRequest -Uri $url -UseBasicParsing -TimeoutSec 10 -MaximumRedirection 2
            $ok = ($resp.StatusCode -ge 200 -and $resp.StatusCode -lt 400)
        } catch { Start-Sleep -Seconds 5 }
    }
    $mark = if ($ok) { 'OK  ' } else { 'FAIL' }
    $color = if ($ok) { 'Green' } else { 'Red' }
    Write-Host ("[probe] {0} {1} -> {2}" -f $mark, $url, $p.Svc) -ForegroundColor $color
    if (-not $ok) { $probeFailures.Add("$($p.Svc) $url") }
}

# ---- 5. Verdict ----
Write-Host ''
Write-Host ('[certify] HTTP probes: {0}; compose services: {1}' -f $probes.Count, $composeSvcCount)
if ($failed.Count -gt 0 -or $probeFailures.Count -gt 0) {
    Write-Host "[certify] CERTIFICATION FAILED: $($failed.Count) compose-level, $($probeFailures.Count) probe-level" -ForegroundColor Red
    $probeFailures | ForEach-Object { Write-Host "  probe-failure: $_" }
    exit 1
}
Write-Host '[certify] CERTIFICATION PASSED: full stack healthy' -ForegroundColor Green

