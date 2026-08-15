<#
.SYNOPSIS
    StatGate secret scanner (SG-SEC-2026-08).
.DESCRIPTION
    Scans the repository (excluding vendor/heavy trees) for known-compromise
    credentials, default credentials, and insecure markers. Assumed wired
    into CI; exits non-zero when any finding is present.
.USAGE
    powershell -ExecutionPolicy Bypass -File scripts/secret-scan.ps1 [-RepoRoot <path>]
#>
param(
    [string]$RepoRoot = (Split-Path $PSScriptRoot -Parent)
)

$ErrorActionPreference = 'Continue'

$excludeDirs = @('node_modules', '\.git$', '\.next', 'Lib', '\.venv', 'venv',
    '\.gocache', 'dist', 'collect-master', '\.venv\.bak', '^data$', 'uploads',
    'server-log', 'build', 'bower_components', 'pip_cache', '^tests$',
    '^test$', 'docs')

# Known-compromise credential sentinels (must be rotated + removed).
$knownSecrets = @(
    'statgate_field_secret_key_2026',
    'Kb7Qx3pV9mL2rT8wY4nC6dF1hJ5sA0eR',
    'Statgate_kaggle',
    'Uganda2026',
    'biostat@2026',
    'F6nQ2rK9vT4xM8pW1dC7yH3sL0aE5jB',
    'statgate_helpdesk_secret_2026',
    'StatGate_Superset_Secret_Key_2026'
)

# Default credential / insecure markers.
$defaultPatterns = @(
    '[Pp]assword\s*[:=]\s*["'']password["'']',
    'minioadmin',
    'changeme'
)

$fileTypes = @('.go','.py','.js','.ts','.tsx','.jsx','.yaml','.yml','.json','.env',
    '.properties','.sh','.ps1','.bat','.conf','.toml','.ini','.example','.txt','Dockerfile')

function Test-Ignored {
    param([string]$Path, [string]$LeafName)
    if ($LeafName -in @('secret-scan.ps1', 'generate-security-docs.ps1')) { return $true }
    foreach ($d in $excludeDirs) {
        if ($Path -match $d) { return $true }
    }
    return $false
}

$hits = @()
$maxBytes = 2MB

# Breadth-first directory walk that PRUNES excluded trees before descending,
# so vendor/runtime directories are never enumerated.
$queue = [System.Collections.Generic.Queue[string]]::new()
$queue.Enqueue($RepoRoot)

while ($queue.Count -gt 0) {
    $dir = $queue.Dequeue()
    $leaf = [System.IO.Path]::GetFileName($dir.TrimEnd('\', '/'))
    foreach ($ex in $excludeDirs) {
        if ($leaf -match $ex -and $dir -ne $RepoRoot) { continue 2 }
    }

    foreach ($filePath in [System.IO.Directory]::EnumerateFiles($dir)) {
        $file = Get-Item -LiteralPath $filePath -Force
        if ($file.Name -in @('secret-scan.ps1', 'generate-security-docs.ps1')) { continue }
        if ($file.Length -gt $maxBytes) { continue }
        if ($file.Name -notin $fileTypes -and $file.Extension -notin $fileTypes) { continue }

        try { $content = [System.IO.File]::ReadAllText($file.FullName) } catch { continue }

        foreach ($secret in $knownSecrets) {
            # Documentation and audit reports (SECRET_MANAGEMENT.md,
            # STATGATE_AUDIT_REPORT.txt) legitimately name the removed
            # sentinels, so only scan code/config files for them.
            if ($file.Extension -in @('.md', '.txt')) { continue }
            if ($content.Contains($secret)) {
                $hits += [pscustomobject]@{ Severity='KNOWN-SECRET'; File=$file.FullName; Match=$secret }
            }
        }

        foreach ($p in $defaultPatterns) {
            if ($file.Extension -in @('.md', '.txt')) { continue }
            if ($content -match $p) {
                $hits += [pscustomobject]@{ Severity='default-cred'; File=$file.FullName; Match=$p }
            }
        }
    }

    foreach ($sub in [System.IO.Directory]::EnumerateDirectories($dir)) {
        $subLeaf = [System.IO.Path]::GetFileName($sub.TrimEnd('\', '/'))
        $excluded = $false
        foreach ($ex in $excludeDirs) {
            if ($subLeaf -match $ex) { $excluded = $true; break }
        }
        if (-not $excluded) { $queue.Enqueue($sub) }
    }
}

Write-Output ''
Write-Output 'STATGATE SECRET SCAN (SG-SEC-2026-08)'
Write-Output '======================================'

if ($hits.Count -eq 0) {
    Write-Output 'PASS: 0 active secrets, 0 default credentials.'
    exit 0
}

$hits | Select-Object Severity, File, Match | Format-Table -AutoSize -Wrap
Write-Output "FAIL: $($hits.Count) finding(s) - rotate and remove before merge."
exit 1