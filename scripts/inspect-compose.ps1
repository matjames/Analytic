$ErrorActionPreference = 'Stop'
$cfgFile = 'c:\Users\PC\Desktop\Analytic\data\compose-config.txt'
$cfg = Get-Content $cfgFile

# Extract every "published: <port>" from the rendered compose config
$published = @()
foreach ($line in $cfg) {
    if ($line -match 'published:\s*"?(\d+)"?') { $published += [int]$matches[1] }
}
$published = $published | Sort-Object -Unique
Write-Output ("published host ports: " + ($published -join ', '))

# Snapshot current listeners once
$listeners = netstat -ano | Select-String -Pattern 'LISTENING'
foreach ($p in $published) {
    $hit = $listeners | Select-String -SimpleMatch (":$p ")
    if ($hit) {
        $procId = (($hit | Select-Object -First 1).ToString() -split '\s+')[-1]
        $procName = (Get-Process -Id $procId -ErrorAction SilentlyContinue).ProcessName
        "CONFLICT host:{0}  <- {1} (pid {2})" -f $p, $procName, $procId
    }
}
Write-Output '--- done ---'