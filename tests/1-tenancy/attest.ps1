# =====================================================================
# StatGate cross-service tenant/authorization attestation (plan item 1.5)
# Drives unauthenticated + malformed-workspace probes per tests/1-tenancy/manifest.json.
# Fails (exit 1) if ANY service answers differently than the manifest expects.
#
# Usage:
#   powershell -NoProfile -ExecutionPolicy Bypass -File tests/1-tenancy/attest.ps1 [-OutFile <path>]
# =====================================================================
param(
    [string]$OutFile = ""
)

$ErrorActionPreference = 'Continue'
$root = Split-Path -Parent (Split-Path -Parent $PSScriptRoot)
$manifestPath = Join-Path $PSScriptRoot 'manifest.json'
$manifest = Get-Content $manifestPath -Raw | ConvertFrom-Json
if (-not $OutFile) { $OutFile = Join-Path $PSScriptRoot 'TEST_RESULT' }

$lines = New-Object System.Collections.Generic.List[string]
$lines.Add("StatGate tenant/authorization attestation ($($manifest.probes.Count) services)")
$lines.Add("Generated: $(Get-Date -Format o)")
$lines.Add('')

function Get-Status {
    param([string]$Url, [string]$Method = 'GET', [hashtable]$Headers = $null)
    try {
        $p = @{ Uri = $Url; UseBasicParsing = $true; TimeoutSec = 8 }
        if ($Method -ne 'GET') { $p.Method = $Method; $p.ContentType = 'application/json'; $p.Body = '{}' }
        if ($Headers) { $p.Headers = $Headers }
        $r = Invoke-WebRequest @p
        return [int]$r.StatusCode
    } catch {
        if ($_.Exception.Response -and $_.Exception.Response.StatusCode.value__) {
            return [int]$_.Exception.Response.StatusCode.value__
        }
        return -1
    }
}

$total = 0; $failed = 0
foreach ($probe in $manifest.probes) {
    $method = if ($probe.PSObject.Properties.Name -contains 'method') { [string]$probe.method } else { 'GET' }
    $url = "http://localhost:$($probe.port)$($probe.path)"

    $gotNoAuth = Get-Status -Url $url -Method $method
    $gotBad = Get-Status -Url $url -Method $method -Headers @{'X-Workspace-ID' = 'NOT A VALID ID!!!'}

    $ok1 = ($gotNoAuth -eq [int]$probe.unauthed)
    $ok2 = ($gotBad -eq [int]$probe.malformed_ws)
    $total += 2
    if (-not $ok1) { $failed++ }
    if (-not $ok2) { $failed++ }

    $m1 = if ($ok1) { 'PASS' } else { 'FAIL' }
    $m2 = if ($ok2) { 'PASS' } else { 'FAIL' }
    $detail = '[' + $m1 + '/' + $m2 + '] ' + $probe.svc + ' ' + $method +
        ' no-auth: got=' + $gotNoAuth + ' want=' + $probe.unauthed +
        ' | malformed-ws: got=' + $gotBad + ' want=' + $probe.malformed_ws
    $lines.Add($detail)
}

$lines.Add('')
$lines.Add("checks: $total, failed: $failed")
if ($failed -gt 0) {
    $lines.Add('RESULT: FAIL')
    $lines | Set-Content -Path $OutFile
    $lines | ForEach-Object { Write-Output $_ }
    exit 1
}
$lines.Add('RESULT: PASS')
$lines | Set-Content -Path $OutFile
$lines | ForEach-Object { Write-Output $_ }
