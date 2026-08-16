# ═══════════════════════════════════════════════════════════════════
# STATGATE AUTOMATED DATABASE BACKUP & REPLICATION SCRIPT (PowerShell)
# ═══════════════════════════════════════════════════════════════════

param (
    [string]$BackupDir = ".\backups",
    [string]$DbHost = "localhost",
    [string]$DbPort = "5432",
    [string]$DbUser = "Kaggle",
    [switch]$Compress = $true
)

$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$TargetFolder = Join-Path $BackupDir $Timestamp

if (!(Test-Path $TargetFolder)) {
    New-Item -ItemType Directory -Path $TargetFolder -Force | Out-Null
}

Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  STATGATE MULTI-DATABASE BACKUP ENGINE" -ForegroundColor Cyan
Write-Host "  Timestamp: $Timestamp" -ForegroundColor Cyan
Write-Host "═══════════════════════════════════════════════════════════"

$Databases = @(
    "statgate_ml_staging",
    "statgate",
    "pms",
    "rms",
    "statchat",
    "statgate_enterprise"
)

foreach ($db in $Databases) {
    $DumpFile = Join-Path $TargetFolder "$($db)_$Timestamp.sql"
    Write-Host " [BACKUP] Dumping database: $db -> $DumpFile" -ForegroundColor Yellow
    
    $env:PGPASSWORD = $env:KAGGLE_DB_PASSWORD
    
    try {
        & pg_dump -h $DbHost -p $DbPort -U $DbUser -d $db -F p -f $DumpFile 2>&1 | Out-Null
        if (Test-Path $DumpFile) {
            $Size = (Get-Item $DumpFile).Length / 1KB
            Write-Host "   -> OK: $([Math]::Round($Size, 2)) KB" -ForegroundColor Green
        } else {
            Write-Host "   -> SKIPPED (pg_dump not available in PATH or DB not provisioned)" -ForegroundColor Gray
        }
    } catch {
        Write-Host "   -> Warning: $_" -ForegroundColor Red
    }
}

Write-Host "═══════════════════════════════════════════════════════════" -ForegroundColor Cyan
Write-Host "  BACKUP SEQUENCE COMPLETE: $TargetFolder" -ForegroundColor Green
Write-Host "═══════════════════════════════════════════════════════════"
