@echo off
REM StatGate System Access Script
REM Opens multiple services with port-forwarding

echo ========================================
echo StatGate System Access
echo ========================================
echo.
echo Starting port-forwards...
echo.

REM Open each service in a new Command Prompt window
echo Opening Helpdesk UI on http://localhost:3005
start "StatGate - Helpdesk" cmd /k kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005

echo Opening StaChat on http://localhost:3009
start "StatGate - StaChat" cmd /k kubectl port-forward -n statgate svc/statchat-frontend 3009:3009

echo Opening PMS on http://localhost:3010
start "StatGate - PMS" cmd /k kubectl port-forward -n statgate svc/statgate-pms-ui 3010:3010

echo Opening RMS on http://localhost:3011
start "StatGate - RMS" cmd /k kubectl port-forward -n statgate svc/statgate-rms-ui 3011:3011

echo Opening Governance on http://localhost:3012
start "StatGate - Governance" cmd /k kubectl port-forward -n statgate svc/statgate-governance-ui 3012:3012

echo Opening Analytics on http://localhost:5000
start "StatGate - Analytics" cmd /k kubectl port-forward -n statgate svc/statgate-analytics 5000:5000

echo.
echo ========================================
echo Port-forwards are starting...
echo ========================================
echo.
echo Access your services:
echo   Helpdesk:   http://localhost:3005
echo   StaChat:    http://localhost:3009
echo   PMS:        http://localhost:3010
echo   RMS:        http://localhost:3011
echo   Governance: http://localhost:3012
echo   Analytics:  http://localhost:5000
echo.
echo Keep these Command Prompt windows open to maintain connections.
echo Close any window to stop that port-forward.
echo.
pause
