# Runs the full StatGate stack build+up in the background, logging to data/compose-up.log
$ErrorActionPreference = 'Continue'
Set-Location 'c:\Users\PC\Desktop\Analytic'
& docker compose up -d --build --quiet-pull 2>&1 | Out-File -FilePath 'c:\Users\PC\Desktop\Analytic\data\compose-up.log' -Encoding utf8
"EXITCODE=$LASTEXITCODE" | Out-File -Append -FilePath 'c:\Users\PC\Desktop\Analytic\data\compose-up.log' -Encoding utf8