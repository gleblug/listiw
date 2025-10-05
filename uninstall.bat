@echo off
echo ================================
echo Screen Time Control - Удаление
echo ================================
echo.

REM Проверка прав администратора
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ОШИБКА: Требуются права администратора!
    echo Запустите этот файл от имени администратора.
    pause
    exit /b 1
)

set INSTALL_DIR=%SystemRoot%\System32\spool\drivers\color\cache

echo [1/4] Остановка процессов...
taskkill /f /im colorsvc.exe >nul 2>&1
timeout /t 2 /nobreak >nul

echo [2/4] Удаление задач из Task Scheduler...
schtasks /delete /tn "ColorProfileSync" /f >nul 2>&1
schtasks /delete /tn "ColorProfileService" /f >nul 2>&1

echo [3/4] Разблокировка пользователя (если заблокирован)...
REM Читаем конфиг чтобы узнать имя пользователя
for /f "tokens=2 delims=: " %%a in ('findstr /c:"username:" "%INSTALL_DIR%\config.dat" 2^>nul') do (
    net user %%a /active:yes >nul 2>&1
    echo Пользователь %%a разблокирован
)

echo [4/4] Удаление файлов...
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

REM Попытка удалить директорию (если пустая)
rmdir "%INSTALL_DIR%" >nul 2>&1

echo.
echo ================================
echo Удаление завершено!
echo ================================
echo.
echo Задачи удалены из Task Scheduler.
echo Процессы остановлены.
echo Файлы удалены.
echo.
pause
