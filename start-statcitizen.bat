@echo off
echo ===============================================================================
echo Starting StatCitizen — Sovereign Citizen Engagement & Evidence Platform
echo ===============================================================================
cd /d "%~dp0StatCitizen\backend"
set STATGATE_ENV=development
set STATCITIZEN_API_PORT=8097
set STATCITIZEN_UI_PORT=3015
go run .
pause
