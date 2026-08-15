$root = 'c:\Users\PC\Desktop\Analytic'
$exclude = @('node_modules', '.venv', '.venv.bak', '.git', '.gocache', '.pytest_cache', 'build', 'dist', '.next', '__pycache__', 'collect-master', 'helpdesk-master', '.vscode', '.gradle', 'gradle')
$exts = @('.go', '.py', '.js', '.jsx', '.ts', '.tsx', '.sql', '.yaml', '.yml', '.json', '.sh', '.ps1')

$allFiles = Get-ChildItem -Path $root -Recurse -File | Where-Object {
    $ext = $_.Extension.ToLower()
    if ($exts -notcontains $ext) { return $false }
    $p = $_.FullName
    foreach ($ex in $exclude) {
        if ($p.Contains("\$ex\") -or $p.Contains("/$ex/")) { return $false }
    }
    return $true
}

Write-Output "Scanned relevant source files: $($allFiles.Count)"

$patterns = @('TODO', 'FIXME', 'MOCK', 'STUB', 'PLACEHOLDER', 'DUMMY', 'NOT IMPLEMENTED', 'in-memory', 'hardcoded', 'fake_', 'sample_data')

$flagged = @()
foreach ($file in $allFiles) {
    $rel = $file.FullName.Substring($root.Length + 1)
    $lines = Get-Content -Path $file.FullName -ErrorAction SilentlyContinue
    $lineNum = 1
    foreach ($line in $lines) {
        foreach ($pat in $patterns) {
            if ($line -like "*$pat*") {
                $flagged += [PSCustomObject]@{
                    File = $rel
                    Line = $lineNum
                    Pattern = $pat
                    Text = if ($line.Trim().Length -gt 120) { $line.Trim().Substring(0, 120) + "..." } else { $line.Trim() }
                }
                break
            }
        }
        $lineNum++
    }
}

Write-Output "Total flagged lines: $($flagged.Count)"
$flagged | Export-Csv -Path "$root\.gemini\audit_matches.csv" -NoTypeInformation -Encoding UTF8
$flagged | Group-Object File | Select-Object Name, Count | Sort-Object Count -Descending | Format-Table -AutoSize
