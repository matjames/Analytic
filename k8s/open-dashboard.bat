@echo off
REM StatGate Dashboard Browser Access
REM Opens all UIs in your default browser

echo ========================================
echo StatGate Dashboard - Browser Access
echo ========================================
echo.

REM Check if port-forwards are already running
echo Checking kubectl connection...
kubectl cluster-info >nul 2>&1
if errorlevel 1 (
    echo ERROR: kubectl not found or cluster not accessible
    pause
    exit /b 1
)

echo.
echo Starting port-forwards and opening browsers...
echo.
echo (Keep the Command Prompt windows open to maintain connections)
echo.

REM Start port-forwards in background and open browsers
start "" cmd /c "kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005" && timeout /t 2 && start http://localhost:3005
start "" cmd /c "kubectl port-forward -n statgate svc/statchat-frontend 3009:3009" && timeout /t 2 && start http://localhost:3009
start "" cmd /c "kubectl port-forward -n statgate svc/statgate-pms-ui 3010:3010" && timeout /t 2 && start http://localhost:3010
start "" cmd /c "kubectl port-forward -n statgate svc/statgate-rms-ui 3011:3011" && timeout /t 2 && start http://localhost:3011
start "" cmd /c "kubectl port-forward -n statgate svc/statgate-governance-ui 3012:3012" && timeout /t 2 && start http://localhost:3012
start "" cmd /c "kubectl port-forward -n statgate svc/statgate-analytics 5000:5000" && timeout /t 2 && start http://localhost:5000

echo.
echo ========================================
echo Browsers are opening...
echo ========================================
echo.
echo Dashboard URLs:
echo   http://localhost:3005  (Helpdesk)
echo   http://localhost:3009  (StaChat)
echo   http://localhost:3010  (PMS)
echo   http://localhost:3011  (RMS)
echo   http://localhost:3012  (Governance)
echo   http://localhost:5000  (Analytics)
echo.
echo Keep Command Prompt windows open to maintain connections.
echo.
pause
