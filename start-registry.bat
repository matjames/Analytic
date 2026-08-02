@echo off
REM Load DB credentials from the root .env (single source of truth)
for /f "usebackq tokens=1,* delims==" %%a in ("%~dp0.env") do (
    if "%%a"=="REGISTRY_DB_HOST" set REGISTRY_DB_HOST=%%b
    if "%%a"=="REGISTRY_DB_PORT" set REGISTRY_DB_PORT=%%b
    if "%%a"=="REGISTRY_DB_USER" set REGISTRY_DB_USER=%%b
    if "%%a"=="REGISTRY_DB_PASSWORD" set REGISTRY_DB_PASSWORD=%%b
    if "%%a"=="REGISTRY_DB_NAME" set REGISTRY_DB_NAME=%%b
    if "%%a"=="REGISTRY_DB_SSLMODE" set DB_SSLMODE=%%b
    if "%%a"=="STATGATE_REGISTRY_JWT_SECRET" set JWT_SECRET=%%b
    if "%%a"=="PORT" set PORT=%%b
)

REM Defaults if not found in .env
if not defined REGISTRY_DB_HOST set REGISTRY_DB_HOST=localhost
if not defined REGISTRY_DB_PORT set REGISTRY_DB_PORT=5432
if not defined REGISTRY_DB_USER set REGISTRY_DB_USER=Kaggle
if not defined REGISTRY_DB_PASSWORD set REGISTRY_DB_PASSWORD=Statgate_kaggle
if not defined REGISTRY_DB_NAME set REGISTRY_DB_NAME=kaggle
if not defined DB_SSLMODE set DB_SSLMODE=disable
if not defined JWT_SECRET set JWT_SECRET=statgate_field_secret_key_2026
if not defined PORT set PORT=9090

cd /d "%~dp0stage_register\go-backend"
app.exe