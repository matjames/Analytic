Start-Process powershell -ArgumentList "-NoProfile","-ExecutionPolicy","Bypass","-File","$PSScriptRoot\start-backend.ps1" -WindowStyle Minimized
Start-Process powershell -ArgumentList "-NoProfile","-ExecutionPolicy","Bypass","-File","$PSScriptRoot\start-frontend.ps1" -WindowStyle Minimized
Write-Host "Started backend and frontend in separate PowerShell windows."
