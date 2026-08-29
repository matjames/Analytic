$svc = @('StatChat','StatCollect','StatFederation','StatGovernance','StatOps','StatSpatial','StatTrust','StatData','StatIoT','helpdesk-master','knowledge-portal')
foreach ($s in $svc) {
  Write-Output ('===== ' + $s + ' =====')
  $fe = Join-Path $s 'frontend'
  if (Test-Path $fe) {
    $targets = Get-ChildItem $fe -Directory -ErrorAction SilentlyContinue | Where-Object { $_.Name -in @('src','admin','field','ai','public','components','pages') }
    if (-not $targets) { $targets = Get-ChildItem $fe -Directory -ErrorAction SilentlyContinue }
    foreach ($d in $targets) {
      Get-ChildItem $d.FullName -File -Recurse -ErrorAction SilentlyContinue |
        Where-Object { $_.Name -match '\.(css|tsx|jsx|ts|js|html)$' -and $_.Name -notmatch '\.map$' -and $_.FullName -notmatch 'node_modules|dist|build' } |
        Select-Object -First 30 -ExpandProperty FullName
    }
  } else { Write-Output 'NO frontend dir' }
}