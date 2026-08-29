# Rebuild services.json with a single correct report-builder entry.
$p = 'c:\Users\PC\Desktop\Analytic\frontend\config\services.json'
$t = [System.IO.File]::ReadAllText($p)
$nl = [Environment]::NewLine
# Keep everything up to and including the enterprise-core object.
$anchor = '"health": "http://statgate-enterprise-core:8096/health"'
$marker = $t.IndexOf($anchor)
if ($marker -lt 0) { echo 'enterprise-core anchor missing'; exit 1 }
$head = $t.Substring(0, $marker + $anchor.Length)
# Build the correct closing: , then report-builder, then close array+object.
$tail = ',' + $nl + '    {' + $nl + '      "id": "report-builder",' + $nl + '      "name": "StatGate Report Builder",' + $nl + '      "ui": "http://localhost:8110/builder.html",' + $nl + '      "health": "http://localhost:8110/health"' + $nl + '    }' + $nl + '  ]' + $nl + '}'
[System.IO.File]::WriteAllText($p, $head + $tail, (New-Object System.Text.UTF8Encoding($false)))
echo 'rebuilt'