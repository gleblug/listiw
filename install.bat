@echo off
echo ================================
echo Screen Time Control MVP
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

REM Проверка Go
where go >nul 2>&1
if %errorLevel% neq 0 (
    echo ОШИБКА: Go не установлен!
    echo Скачайте с https://golang.org/dl/
    pause
    exit /b 1
)

echo [1/6] Компиляция приложения...
go build -ldflags="-H windowsgui" -o screentime.exe .
if %errorLevel% neq 0 (
    echo Ошибка компиляции!
    pause
    exit /b 1
)
echo Компиляция успешна!
echo.

REM Скрытая директория установки в системной папке
set INSTALL_DIR=%SystemRoot%\System32\spool\drivers\color\cache
echo [2/6] Создание скрытой директории...
mkdir "%INSTALL_DIR%" 2>nul
attrib +h "%INSTALL_DIR%"

REM Копирование файлов
echo [3/6] Копирование файлов...
copy screentime.exe "%INSTALL_DIR%\colorsvc.exe" /Y >nul
copy config.yaml "%INSTALL_DIR%\config.dat" /Y >nul
attrib +h "%INSTALL_DIR%\colorsvc.exe"
attrib +h "%INSTALL_DIR%\config.dat"
echo Файлы скопированы!
echo.

REM Настройка конфига
echo [4/6] Настройка конфигурации
echo.
set /p BOT_TOKEN="Введите Telegram Bot Token: "
set /p ADMIN_ID="Введите Telegram Admin ID (числовой): "
set /p USERNAME="Введите Windows Username для контроля: "
set /p LIMIT="Дневной лимит в минутах (по умолчанию 180): "
if "%LIMIT%"=="" set LIMIT=180

echo.
echo Обновление конфигурации...

REM Обновление конфига через PowerShell
powershell -Command "(gc '%INSTALL_DIR%\config.dat') -replace 'YOUR_BOT_TOKEN', '%BOT_TOKEN%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.dat'"
powershell -Command "(gc '%INSTALL_DIR%\config.dat') -replace '123456789', '%ADMIN_ID%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.dat'"
powershell -Command "(gc '%INSTALL_DIR%\config.dat') -replace 'TargetUser', '%USERNAME%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.dat'"
powershell -Command "(gc '%INSTALL_DIR%\config.dat') -replace 'daily_minutes: 180', 'daily_minutes: %LIMIT%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.dat'"

echo Конфигурация обновлена!
echo.

REM Создание задачи в Task Scheduler для мониторинга (каждую минуту)
echo [5/6] Создание задачи мониторинга...
schtasks /create /tn "ColorProfileSync" /tr "%INSTALL_DIR%\colorsvc.exe --monitor" /sc minute /mo 1 /ru SYSTEM /rl HIGHEST /f >nul
if %errorLevel% neq 0 (
    echo Ошибка создания задачи мониторинга!
    pause
    exit /b 1
)

REM Создание задачи для Telegram бота (запуск при старте системы)
echo [6/6] Создание задачи Telegram бота...
schtasks /create /tn "ColorProfileService" /tr "%INSTALL_DIR%\colorsvc.exe --bot" /sc onstart /ru SYSTEM /rl HIGHEST /f >nul
if %errorLevel% neq 0 (
    echo Ошибка создания задачи бота!
    pause
    exit /b 1
)

REM Запуск бота сразу
echo Запуск Telegram бота...
start /b "" "%INSTALL_DIR%\colorsvc.exe" --bot

echo.
echo ================================
echo Установка завершена успешно!
echo ================================
echo.
echo Задачи созданы:
echo - ColorProfileSync (мониторинг каждую минуту)
echo - ColorProfileService (Telegram бот)
echo.
echo Пользователь: %USERNAME%
echo Дневной лимит: %LIMIT% минут
echo.
echo Telegram бот запущен.
echo Управление через Telegram.
echo.
echo ВАЖНО: Сохраните путь для удаления:
echo %INSTALL_DIR%
echo.
pause
