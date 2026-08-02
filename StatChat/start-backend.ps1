$env:BACKEND_PORT = "4000"
$env:DATABASE_URL = "postgres://Statchat:Statgate@localhost:5432/statchat?sslmode=disable"
Push-Location "$PSScriptRoot\backend"
go run .\cmd\server
Pop-Location
