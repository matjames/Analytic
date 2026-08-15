@echo off
cd /d "%~dp0"
REM StatCollect local launcher.
REM Secret values have been removed (SG-SEC-2026-08). Set them in the
REM process environment or via a git-ignored local .env before running.
set STATCOLLECT_PORT=:8080
set STATCOLLECT_DATA_DIR=./data
set STATCOLLECT_DB_HOST=localhost
set STATCOLLECT_DB_USER=statcollect
if "%STATCOLLECT_DB_PASSWORD%"=="" set STATCOLLECT_DB_PASSWORD=
set STATCOLLECT_DB_NAME=statcollect
if "%STATCOLLECT_API_KEY%"=="" set STATCOLLECT_API_KEY=
set STATCOLLECT_ADMIN_KEYS=
set STATCOLLECT_USE_S3=false

REM ── StatGate Platform Integration (injected, never committed) ──
set STATGATE_REGISTRY_JWT_SECRET=
set STATGATE_INTERNAL_API_KEY=
set STATGATE_TENANT_ID=default
set STATCOLLECT_ENABLE_EVENTS=true
set STATCOLLECT_ENABLE_STATCHAT=true
set REDIS_ADDR=localhost:6379
set STATCHAT_API_URL=http://localhost:4000
set STATGATE_REGISTRY_API_URL=http://localhost:9090/api

statcollect.exe