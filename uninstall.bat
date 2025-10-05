@echo off
echo ================================
echo Screen Time Control - Uninstall
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

set INSTALL_DIR=%SystemRoot%\System32\spool\drivers\color\cache

echo [1/4] Stopping processes...
taskkill /f /im colorsvc.exe >nul 2>&1
timeout /t 2 /nobreak >nul

echo [2/4] Removing tasks from Task Scheduler...
schtasks /delete /tn "ColorProfileSync" /f >nul 2>&1
schtasks /delete /tn "ColorProfileService" /f >nul 2>&1

echo [3/4] Unlocking user (if blocked)...
REM Read config to get username
for /f "tokens=2 delims=: " %%a in ('findstr /c:"username:" "%INSTALL_DIR%\config.dat" 2^>nul') do (
    net user %%a /active:yes >nul 2>&1
    echo User %%a unlocked
)

echo [4/4] Removing files...
if exist "%INSTALL_DIR%\colorsvc.exe" (
    attrib -h "%INSTALL_DIR%\colorsvc.exe"
    del /f /q "%INSTALL_DIR%\colorsvc.exe" >nul 2>&1
)
if exist "%INSTALL_DIR%\config.dat" (
    attrib -h "%INSTALL_DIR%\config.dat"
    del /f /q "%INSTALL_DIR%\config.dat" >nul 2>&1
)
if exist "%INSTALL_DIR%\timedata.json" (
    del /f /q "%INSTALL_DIR%\timedata.json" >nul 2>&1
)

REM Try to remove directory (if empty)
rmdir "%INSTALL_DIR%" >nul 2>&1

echo.
echo ================================
echo Uninstallation completed!
echo ================================
echo.
echo Tasks removed from Task Scheduler.
echo Processes stopped.
echo Files removed.
echo.
pause
