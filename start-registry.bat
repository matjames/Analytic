@echo off
set REGISTRY_DB_HOST=localhost
set REGISTRY_DB_PORT=5432
set REGISTRY_DB_USER=Kaggle
set REGISTRY_DB_PASSWORD=Statgate_kaggle
set REGISTRY_DB_NAME=kaggle
set PORT=9090
set JWT_SECRET=statgate_field_secret_key_2026
cd /d c:\Users\PC\Desktop\Analytic\stage_register\go-backend
app.exe
