@echo off
REM Single Service Access - Choose One

setlocal enabledelayedexpansion

echo ========================================
echo StatGate Service Access Menu
echo ========================================
echo.
echo Choose a service to access:
echo.
echo 1. Helpdesk UI (Operations Support)
echo 2. StaChat (Real-time Collaboration)
echo 3. PMS (Projects Management)
echo 4. RMS (Research Management)
echo 5. Governance (Institutional Governance)
echo 6. Analytics API (Data Analytics)
echo 7. PostgreSQL Database
echo 8. Redis Cache
echo 9. Exit
echo.

set /p choice="Enter your choice (1-9): "

if "%choice%"=="1" (
    echo.
    echo Opening Helpdesk UI on port 3005...
    kubectl port-forward -n statgate svc/statgate-helpdesk-ui 3005:3005
) else if "%choice%"=="2" (
    echo.
    echo Opening StaChat on port 3009...
    kubectl port-forward -n statgate svc/statchat-frontend 3009:3009
) else if "%choice%"=="3" (
    echo.
    echo Opening PMS on port 3010...
    kubectl port-forward -n statgate svc/statgate-pms-ui 3010:3010
) else if "%choice%"=="4" (
    echo.
    echo Opening RMS on port 3011...
    kubectl port-forward -n statgate svc/statgate-rms-ui 3011:3011
) else if "%choice%"=="5" (
    echo.
    echo Opening Governance on port 3012...
    kubectl port-forward -n statgate svc/statgate-governance-ui 3012:3012
) else if "%choice%"=="6" (
    echo.
    echo Opening Analytics API on port 5000...
    kubectl port-forward -n statgate svc/statgate-analytics 5000:5000
) else if "%choice%"=="7" (
    echo.
    echo Opening PostgreSQL on port 5432...
    echo Connect with:
    echo   Host: localhost
    echo   Port: 5432
    echo   User: postgres
    echo   Password: statgate_k8s_password_prod
    echo.
    kubectl port-forward -n statgate svc/postgres 5432:5432
) else if "%choice%"=="8" (
    echo.
    echo Opening Redis on port 6379...
    echo Connect with: redis-cli -h localhost -p 6379
    echo.
    kubectl port-forward -n statgate svc/redis 6379:6379
) else if "%choice%"=="9" (
    echo Exiting...
    exit /b 0
) else (
    echo Invalid choice. Please try again.
    timeout /t 2
    cls
    goto start
)

echo.
echo Port-forward established. Use Ctrl+C to stop.
echo.
pause
