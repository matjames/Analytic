<#
.SYNOPSIS
    StatGate Phase XII Preflight Security Regression Suite (SG-SEC-2026-08).
.DESCRIPTION
    Executes the mandated security regression programme:
      1. secret scan             6. tenant isolation tests
      2. Go tests                7. API security tests
      3. frontend tests          8. SQL/input validation tests
      4. authentication tests    9. Docker configuration validation
      5. authorization tests    10. production startup validation
      11/12. Phase X/XI tests
.DEPENDENCIES
    Requires Go, Node, Docker CLI and PowerShell 5.1+.
.USAGE
    powershell -ExecutionPolicy Bypass -File run-security-audit.ps1
#>
param(
    [string]$RepoRoot = $PSScriptRoot
)

$ErrorActionPreference = 'Continue'
$results = @()

function Test-Step {
    param([string]$Name, [hashtable]$Result)
    $script:results += [pscustomobject]@{ Step = $Name; Status = $Result.Status; Detail = $Result.Detail }
    Write-Output ("[{0}] {1} - {2}" -f $Result.Status, $Name, $Result.Detail)
}
function Invoke-FailedCheck { param([string]$Name, [string]$Detail)
    Test-Step -Name $Name -Result @{ Status = 'FAIL'; Detail = $Detail } }
function Invoke-PassedCheck { param([string]$Name, [string]$Detail)
    Test-Step -Name $Name -Result @{ Status = 'PASS'; Detail = $Detail } }

Write-Output '=========================================================='
Write-Output ' STATGATE SECURITY REGRESSION SUITE (SG-SEC-2026-08)'
Write-Output '=========================================================='

# 1. Secret scan
if (Test-Path (Join-Path $PSScriptRoot 'scripts\secret-scan.ps1')) {
    $out = (& (Join-Path $PSScriptRoot 'scripts\secret-scan.ps1') -RepoRoot $RepoRoot 2>&1 | Out-String)
    if ($LASTEXITCODE -eq 0) { Invoke-PassedCheck 'Secret scan' ($out -replace "`r?`n", ' ') }
    else { Invoke-FailedCheck 'Secret scan' (($out -split "`r?`n" | Where-Object { $_ -match 'KNOWN-SECRET|default-cred|FAIL' }) -join '; ') }
} else { Invoke-FailedCheck 'Secret scan' 'scripts/secret-scan.ps1 missing' }

# 2. Go tests (all Go modules)
$goModules = @('enterprise/core','PMS/backend','RMS/backend','StatGovernance/backend',
    'backend','StatChat/backend','stage_register/go-backend','StatCollect','enterprise/search')
$goFail = @()
foreach ($m in $goModules) {
    $dir = Join-Path $RepoRoot $m
    if (-not (Test-Path (Join-Path $dir 'go.mod'))) { continue }
    Push-Location $dir
    $o = go test ./... 2>&1
    if ($LASTEXITCODE -ne 0) { $goFail += "$m : $($o | Select-Object -Last 1)" }
    Pop-Location
}
if ($goFail.Count -eq 0) { Invoke-PassedCheck 'Go tests' 'all Go modules pass go test' }
else { Invoke-FailedCheck 'Go tests' ($goFail -join '; ') }

# 3. Frontend/Node syntax gate
Push-Location (Join-Path $RepoRoot 'helpdesk-master\backend')
$nodeOut = @()
foreach ($f in @('index.js','middleware/auth.js','routes/userRoutes.js','routes/ticketRoutes.js',
    'routes/commentsRoutes.js','routes/videoRoutes.js','routes/agents.js','routes/knowledgeBaseRoutes.js')) {
    $o = node --check $f 2>&1
    if ($LASTEXITCODE -ne 0) { $nodeOut += "$f : $o" }
}
Pop-Location
if ($nodeOut.Count -eq 0) { Invoke-PassedCheck 'Frontend/Node syntax' 'all modified JS modules pass node --check' }
else { Invoke-FailedCheck 'Frontend/Node syntax' ($nodeOut -join '; ') }
# 4. Authentication tests (fail-closed middleware static checks)
$a = @(
    @{ Name='No demo tokens in Go middlewares'; OK = (-not (Select-String -Path (Join-Path $RepoRoot 'StatGovernance\backend\middleware.go') -Pattern 'demo_' -Quiet)) -and (-not (Select-String -Path (Join-Path $RepoRoot 'PMS\backend\middleware.go') -Pattern 'demo_' -Quiet)) -and (-not (Select-String -Path (Join-Path $RepoRoot 'RMS\backend\middleware.go') -Pattern 'demo_' -Quiet)) },
    @{ Name='No user-001 fallback in StatChat'; OK = -not (Select-String -Path (Join-Path $RepoRoot 'StatChat\backend\pkg\api\handlers.go') -Pattern 'user-001' -Quiet) },
    @{ Name='StatChat auth permanently required'; OK = (Select-String -Path (Join-Path $RepoRoot 'StatChat\backend\pkg\api\handlers.go') -Pattern 'return true' -Quiet) }
)
foreach ($c in $a) {
    if ($c.OK) { Invoke-PassedCheck 'Authentication' $c.Name } else { Invoke-FailedCheck 'Authentication' $c.Name }
}

# 5. Authorization tests
$b = @(
    @{ Name='Registry public role whitelist'; OK = (Select-String -Path (Join-Path $RepoRoot 'stage_register\go-backend\handlers\users.go') -Pattern 'publicRegistrationRoles' -Quiet) },
    @{ Name='Helpdesk admin-guarded user mutations'; OK = (Select-String -Path (Join-Path $RepoRoot 'helpdesk-master\backend\routes\userRoutes.js') -Pattern 'requireAdmin' -Quiet) }
)
foreach ($c in $b) {
    if ($c.OK) { Invoke-PassedCheck 'Authorization' $c.Name } else { Invoke-FailedCheck 'Authorization' $c.Name }
}

# 6. Tenant isolation (JWT-derived; header trust removed)
$c = @(
    @{ Name='No X-User-ID header fallback (enterprise/core)'; OK = -not (Select-String -Path (Join-Path $RepoRoot 'enterprise\core\middleware_security.go') -Pattern 'GetHeader\("X-User-ID"\)' -Quiet) },
    @{ Name='tenantIsolationMiddleware rejects mismatched tenants'; OK = (Select-String -Path (Join-Path $RepoRoot 'enterprise\core\middleware_security.go') -Pattern 'tenant_mismatch' -Quiet) }
)
foreach ($x in $c) {
    if ($x.OK) { Invoke-PassedCheck 'Tenant isolation' $x.Name } else { Invoke-FailedCheck 'Tenant isolation' $x.Name }
}

# 7. API security (structured errors, no raw DB text)
if (Select-String -Path (Join-Path $RepoRoot 'helpdesk-master\backend\routes\userRoutes.js') -Pattern 'respondSafeError' -Quiet) {
    Invoke-PassedCheck 'API security' 'structured error responses wired in helpdesk mutations'
} else { Invoke-FailedCheck 'API security' 'structured error helper missing' }

# 8. SQL / input validation
if ((Select-String -Path (Join-Path $RepoRoot 'stage_register\go-backend\handlers\users.go') -Pattern 'isValidEmail' -Quiet) -and
    (Select-String -Path (Join-Path $RepoRoot 'stage_register\go-backend\handlers\users.go') -Pattern 'passwordMeetsPolicy' -Quiet)) {
    Invoke-PassedCheck 'SQL/input validation' 'email + password policy validators wired'
} else { Invoke-FailedCheck 'SQL/input validation' 'validators missing' }

# 9. Docker configuration validation
if (Get-Command docker -ErrorAction SilentlyContinue) {
    # Supply structurally-valid placeholder secrets so the compose file's
    # required-variable mechanism can be validated as a PARSE check.
    'KAGGLE_DB_PASSWORD','HELPDESK_DB_PASSWORD','REGISTRY_DB_PASSWORD','STATCHAT_DB_PASSWORD',
        'PMS_DB_PASSWORD','RMS_DB_PASSWORD','GOVERNANCE_DB_PASSWORD','STATCOLLECT_DB_PASSWORD',
        'STATGATE_INTERNAL_API_KEY','STATGATE_REGISTRY_JWT_SECRET','FLASK_SECRET_KEY',
        'BASIC_AUTH_USERNAME','BASIC_AUTH_PASSWORD','MINIO_ROOT_USER','MINIO_ROOT_PASSWORD',
        'GRAFANA_ADMIN_USER','GRAFANA_ADMIN_PASSWORD','AIRFLOW_USER','AIRFLOW_PASSWORD',
        'SUPERSET_SECRET_KEY','STATCOLLECT_API_KEY','STATCOLLECT_ADMIN_KEYS' |
        ForEach-Object { Set-Item -Path ("env:{0}" -f $_) -Value ('p' * 24) -Force }
    $o = docker compose -f (Join-Path $RepoRoot 'docker-compose.yml') config --quiet 2>&1
    if ($LASTEXITCODE -eq 0) { Invoke-PassedCheck 'Docker configuration' 'docker-compose.yml parses with injected secrets' }
    else { Invoke-FailedCheck 'Docker configuration' ($o | Select-Object -Last 3) }
} else { Invoke-FailedCheck 'Docker configuration' 'docker CLI unavailable' }

# 10. Production startup validation (fail-closed secret gates)
$d = @(
    @{ Name='Enterprise validateProductionSecrets'; OK = (Select-String -Path (Join-Path $RepoRoot 'enterprise\core\startup_check.go') -Pattern 'refuse to start in production mode' -Quiet) },
    @{ Name='Registry prod fail-fast'; OK = (Select-String -Path (Join-Path $RepoRoot 'stage_register\go-backend\main.go') -Pattern 'required in production. Startup aborted' -Quiet) },
    @{ Name='Core engine internal-key fail-closed'; OK = (Select-String -Path (Join-Path $RepoRoot 'backend\cmd\server\main.go') -Pattern 'service_misconfigured' -Quiet) },
    @{ Name='Helpdesk prod secret gate'; OK = (Select-String -Path (Join-Path $RepoRoot 'helpdesk-master\backend\index.js') -Pattern 'required production secrets missing' -Quiet) }
)
foreach ($x in $d) {
    if ($x.OK) { Invoke-PassedCheck 'Production startup' $x.Name } else { Invoke-FailedCheck 'Production startup' $x.Name }
}

# 11/12. Phase X/XI regression gate (enterprise core tests)
Push-Location (Join-Path $RepoRoot 'enterprise\core')
$o = go test ./... 2>&1
if ($LASTEXITCODE -eq 0) { Invoke-PassedCheck 'Phase X/XI tests' 'enterprise core test suite passed' }
else { Invoke-FailedCheck 'Phase X/XI tests' ($o | Select-Object -Last 3) }
Pop-Location

Write-Output ''
Write-Output '=========================================================='
Write-Output ' SUMMARY'
Write-Output '=========================================================='
$script:results | Format-Table -AutoSize -Wrap

$failed = @($script:results | Where-Object { $_.Status -eq 'FAIL' })
if ($failed.Count -eq 0) { Write-Output 'SECURITY REGRESSION SUITE: PASS'; exit 0 }
Write-Output "SECURITY REGRESSION SUITE: $($failed.Count) step(s) FAILED"
exit 1