@echo off
REM Load DB credentials from the root .env (single source of truth)
for /f "usebackq tokens=1,* delims==" %%a in ("%~dp0.env") do (
    if "%%a"=="REGISTRY_DB_HOST" set REGISTRY_DB_HOST=%%b
    if "%%a"=="REGISTRY_DB_PORT" set REGISTRY_DB_PORT=%%b
    if "%%a"=="REGISTRY_DB_USER" set REGISTRY_DB_USER=%%b
    if "%%a"=="REGISTRY_DB_PASSWORD" set REGISTRY_DB_PASSWORD=%%b
    if "%%a"=="REGISTRY_DB_NAME" set REGISTRY_DB_NAME=%%b
    if "%%a"=="REGISTRY_DB_SSLMODE" set DB_SSLMODE=%%b
    if "%%a"=="STATGATE_REGISTRY_JWT_SECRET" set STATGATE_REGISTRY_JWT_SECRET=%%b
    if "%%a"=="REGISTRY_PORT" set PORT=%%b
)

REM Defaults for non-secret values only (SG-SEC-2026-08).
if not defined REGISTRY_DB_HOST set REGISTRY_DB_HOST=localhost
if not defined REGISTRY_DB_PORT set REGISTRY_DB_PORT=5432
if not defined REGISTRY_DB_USER set REGISTRY_DB_USER=Kaggle
if not defined REGISTRY_DB_NAME set REGISTRY_DB_NAME=kaggle
if not defined DB_SSLMODE set DB_SSLMODE=disable
if not defined PORT set PORT=9090

REM Secrets MUST come from the environment / .env. Fail closed when absent.
if not defined STATGATE_REGISTRY_JWT_SECRET (
    echo FATAL: STATGATE_REGISTRY_JWT_SECRET is required. Startup aborted. 1>&2
    exit /b 1
)
if not defined REGISTRY_DB_PASSWORD (
    echo FATAL: REGISTRY_DB_PASSWORD is required. Startup aborted. 1>&2
    exit /b 1
)
set JWT_SECRET=%STATGATE_REGISTRY_JWT_SECRET%

cd /d "%~dp0stage_register\go-backend"
app.exe