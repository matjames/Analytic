# Add report-builder to the shared services registry.
$p = 'c:\Users\PC\Desktop\Analytic\frontend\config\services.json'
$t = [System.IO.File]::ReadAllText($p)
$anchor = '"health": "http://statgate-enterprise-core:8096/health"'
$newEntry = "`n    },`n    {`n      `"id`": `"report-builder`",`n      `"name`": `"StatGate Report Builder`",`n      `"ui`": `"http://localhost:8110/builder.html`",`n      `"health`": `"http://localhost:8110/health`"`n    }`n  ]`n}"
if ($t.Contains($anchor)) {
  $t = $t.Replace($anchor, $anchor + $newEntry)
  [System.IO.File]::WriteAllText($p, $t, (New-Object System.Text.UTF8Encoding($false)))
  echo 'services.json updated'
} else {
  echo 'anchor not found'
}