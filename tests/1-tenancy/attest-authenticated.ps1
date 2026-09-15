# =====================================================================
# StatGate authenticated tenant/authorization attestation (plan item 1.5, part 2)
# Mints a Registry-contract HS256 JWT (statgate-lib validator: userId, role,
# tenant_id, iss=statgate-registry, aud=statgate) and replays it per service:
#   1. token only                  -> auth must be accepted (no 401)
#   2. token + malformed workspace -> shape guard must answer 400
#   3. token + unknown workspace   -> membership must DENY (400|403|404|503)
#                                      when membership_enforced is true
# Expectations live in manifest.json (calibrated live 2026-09-15).
#
# Usage:
#   powershell -NoProfile -ExecutionPolicy Bypass -File tests/1-tenancy/attest-authenticated.ps1
# =====================================================================
param(
    [string]$OutFile = ""
)

$ErrorActionPreference = 'Continue'
$repo = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$manifest = Get-Content (Join-Path $PSScriptRoot 'manifest.json') -Raw | ConvertFrom-Json
if (-not $OutFile) { $OutFile = Join-Path $PSScriptRoot 'TEST_RESULT_AUTH' }

# ---- read the Registry signing secret (env, else gitignored .env) ----
$secret = $env:STATGATE_REGISTRY_JWT_SECRET
if (-not $secret) {
    $envFile = Join-Path $repo '.env'
    if (Test-Path $envFile) {
        $m = Select-String -Path $envFile -Pattern '^STATGATE_REGISTRY_JWT_SECRET=(.+)$' | Select-Object -First 1
        if ($m) { $secret = $m.Matches[0].Groups[1].Value.Trim() }
    }
}
if (-not $secret) {
    Write-Output 'FATAL: STATGATE_REGISTRY_JWT_SECRET not set and not found in .env; cannot mint a token.'
    exit 2
}

function B64Url([byte[]]$b) {
    return [Convert]::ToBase64String($b).TrimEnd('=').Replace('+', '-').Replace('/', '_')
}

$now = [DateTimeOffset]::UtcNow.ToUnixTimeSeconds()
$headerJson = '{"alg":"HS256","typ":"JWT"}'
$payloadJson = '{"userId":"attest-user","role":"viewer","tenant_id":"tenant-alpha","' +
    'email":"attest@example.com","iss":"statgate-registry","aud":"statgate","iat":' +
    $now + ',"exp":' + ($now + 600) + '}'
$h = B64Url ([Text.Encoding]::UTF8.GetBytes($headerJson))
$p = B64Url ([Text.Encoding]::UTF8.GetBytes($payloadJson))
$sig = B64Url ([System.Security.Cryptography.HMACSHA256]::new(
    [Text.Encoding]::UTF8.GetBytes($secret)).ComputeHash([Text.Encoding]::UTF8.GetBytes("$h.$p")))
$token = "$h.$p.$sig"

# ---- probe helpers ----
function Get-Status {
    param([string]$Url, [string]$Method = 'GET', [hashtable]$Headers)
    try {
        $req = @{ Uri = $Url; UseBasicParsing = $true; TimeoutSec = 10; Headers = $Headers }
        if ($Method -ne 'GET') { $req.Method = $Method; $req.ContentType = 'application/json'; $req.Body = '{}' }
        $r = Invoke-WebRequest @req
        return [int]$r.StatusCode
    } catch {
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode.value__) {
            return [int]$_.Exception.Response.StatusCode.value__
        }
        return -1
    }
}

$denialSet = @(400, 403, 404, 503)
$lines = New-Object System.Collections.Generic.List[string]
$lines.Add('StatGate authenticated tenant/authorization attestation')
$lines.Add("Generated: $(Get-Date -Format o)")
$lines.Add("Principal: synthetic JWT (userId=attest-user, role=viewer, tenant_id=tenant-alpha)")
$lines.Add('')

$checks = 0; $failures = 0; $notes = 0
foreach ($probe in $manifest.probes) {
    $method = if ($probe.PSObject.Properties.Name -contains 'method') { [string]$probe.method } else { 'GET' }
    $url = "http://localhost:$($probe.port)$($probe.path)"
    $authHeaders = @{'Authorization' = "Bearer $token"}

    $gotAuth = Get-Status -Url $url -Method $method -Headers $authHeaders
    $gotBad = Get-Status -Url $url -Method $method -Headers ($authHeaders + @{'X-Workspace-ID' = 'BAD ID!!'})
    $gotUnknown = Get-Status -Url $url -Method $method -Headers ($authHeaders + @{'X-Workspace-ID' = 'ws-attest-nonexistent'})

    $authOk = ($gotAuth -eq [int]$probe.authed_auth)
    $malOk = ($gotBad -eq [int]$probe.authed_malformed)
    $checks += 2
    if (-not $authOk) { $failures++ }
    if (-not $malOk) { $failures++ }

    $memberVerdict = 'NOTE'
    if ([bool]$probe.membership_enforced) {
        $checks++
        if ($denialSet -contains $gotUnknown) { $memberVerdict = 'DENIED' }
        else { $memberVerdict = 'FAIL'; $failures++ }
    } else {
        $notes++
    }

    $verdict = if ($authOk -and $malOk -and $memberVerdict -ne 'FAIL') { 'PASS' } else { 'FAIL' }
    $lines.Add("[$verdict] $($probe.svc) $method auth=$gotAuth(want $($probe.authed_auth)) " +
        "malformed=$gotBad(want $($probe.authed_malformed)) " +
        "unknown-ws=$gotUnknown/$memberVerdict")
    if ($probe.PSObject.Properties.Name -contains 'note') {
        $lines.Add("         note: $($probe.note)")
    }
}

$lines.Add('')
$lines.Add("asserted checks: $checks, failed: $failures, informational notes: $notes")
$lines.Add('Membership assertion: a well-formed unknown workspace must produce a denial (400|403|404|503).')
$lines.Add('  503 = workspace_membership_unavailable: Enterprise Core answered 401 for this synthetic')
$lines.Add('  principal (it has no record of it), so the middleware degrades to 503 rather than 403.')
$lines.Add('  A genuine 403 requires a real provisioned Enterprise Core user; all codes above deny access.')
if ($failures -gt 0) {
    $lines.Add('RESULT: FAIL')
    $lines | Set-Content -Path $OutFile
    $lines | ForEach-Object { Write-Output $_ }
    exit 1
}
$lines.Add('RESULT: PASS')
$lines | Set-Content -Path $OutFile
$lines | ForEach-Object { Write-Output $_ }
