# Screen Time Control - MVP Architecture

## Цель MVP
Проверить работоспособность базовой идеи: отслеживание времени и блокировка пользователя через Telegram управление.

---

## Минимальный функционал

### Что ДОЛЖНО работать:
1. ✅ Отслеживание активного времени пользователя Windows
2. ✅ Блокировка пользователя при превышении лимита
3. ✅ Telegram бот для управления (добавить/убрать минуты)
4. ✅ Чтение настроек из конфига
5. ✅ BAT файл для установки

### Что НЕ включаем в MVP:
- ❌ Шифрование конфига (plain text пока достаточно)
- ❌ База данных (используем JSON файл)
- ❌ Предупреждения перед блокировкой
- ❌ Grace period
- ❌ Emergency unlock
- ❌ Защита от обхода
- ❌ Логирование и аудит
- ❌ Обработка sleep/hibernate
- ❌ Idle detection (считаем всё время)

---

## Упрощённая архитектура

```
┌─────────────────────────────────────────┐
│         Main Service (main.go)          │
│                                         │
│  ┌─────────────┐    ┌────────────────┐ │
│  │   Config    │    │  Time Tracker  │ │
│  │  (YAML)     │───▶│  (JSON file)   │ │
│  └─────────────┘    └────────────────┘ │
│                                         │
│  ┌─────────────┐    ┌────────────────┐ │
│  │  Session    │    │  User Blocker  │ │
│  │  Monitor    │───▶│  (net user)    │ │
│  └─────────────┘    └────────────────┘ │
│                                         │
│  ┌─────────────────────────────────────┤
│  │      Telegram Bot Handler           │
│  │  /status /add /remove               │
│  └─────────────────────────────────────┘
└─────────────────────────────────────────┘
```

---

## Простая структура проекта

```
screen-time-mvp/
├── main.go                 # Всё в одном файле!
├── config.yaml             # Конфигурация
├── timedata.json           # Хранение времени
├── install.bat             # Установщик
├── uninstall.bat           # Деинсталлятор
├── go.mod
└── README.md
```

---

## Конфигурация (config.yaml)

```yaml
# Простой конфиг без шифрования
telegram:
  bot_token: "YOUR_BOT_TOKEN"
  admin_id: 123456789

time_limit:
  daily_minutes: 180  # 3 часа по умолчанию

windows:
  username: "TargetUser"  # Имя пользователя Windows для контроля
```

---

## Хранилище данных (timedata.json)

```json
{
  "date": "2025-01-05",
  "used_minutes": 145,
  "daily_limit": 180,
  "is_blocked": false,
  "last_check": "2025-01-05T15:30:00Z"
}
```

---

## Логика работы

### 1. Мониторинг сессии (каждую минуту)
```go
// Проверяем, залогинен ли пользователь
if IsUserLoggedIn(username) {
    // Добавляем 1 минуту к счётчику
    usedMinutes++
    SaveToFile()
    
    // Проверяем лимит
    if usedMinutes >= dailyLimit {
        BlockUser(username)
    }
}
```

### 2. Блокировка пользователя
```go
// Простая блокировка через команду Windows
exec.Command("net", "user", username, "/active:no").Run()

// Принудительный logoff если залогинен
exec.Command("logoff", sessionID).Run()
```

### 3. Telegram команды

**`/status`** - Показать статус
```
📊 Статус экранного времени

Пользователь: JohnDoe
Использовано: 2ч 25м / 3ч 00м
Осталось: 35м

Статус: ✅ Активен
```

**`/add 30`** - Добавить 30 минут
```
✅ Добавлено 30 минут
Новый лимит на сегодня: 3ч 30м
```

**`/remove 15`** - Убрать 15 минут
```
✅ Убрано 15 минут
Новый лимит на сегодня: 2ч 45м
```

**`/unlock`** - Разблокировать пользователя
```
✅ Пользователь разблокирован
Можно войти в систему
```

---

## Основной код (main.go) - Структура

```go
package main

import (
    "encoding/json"
    "fmt"
    "os"
    "os/exec"
    "time"
    
    tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
    "gopkg.in/yaml.v3"
)

// Config структура
type Config struct {
    Telegram struct {
        BotToken string `yaml:"bot_token"`
        AdminID  int64  `yaml:"admin_id"`
    } `yaml:"telegram"`
    TimeLimit struct {
        DailyMinutes int `yaml:"daily_minutes"`
    } `yaml:"time_limit"`
    Windows struct {
        Username string `yaml:"username"`
    } `yaml:"windows"`
}

// TimeData структура
type TimeData struct {
    Date        string    `json:"date"`
    UsedMinutes int       `json:"used_minutes"`
    DailyLimit  int       `json:"daily_limit"`
    IsBlocked   bool      `json:"is_blocked"`
    LastCheck   time.Time `json:"last_check"`
}

var (
    config   Config
    timeData TimeData
)

func main() {
    // 1. Загрузить конфиг
    loadConfig()
    
    // 2. Загрузить данные времени
    loadTimeData()
    
    // 3. Запустить Telegram бота в горутине
    go startTelegramBot()
    
    // 4. Основной цикл мониторинга
    monitorLoop()
}

// Загрузка конфига
func loadConfig() {
    data, _ := os.ReadFile("config.yaml")
    yaml.Unmarshal(data, &config)
}

// Загрузка данных времени
func loadTimeData() {
    data, err := os.ReadFile("timedata.json")
    if err != nil {
        // Создать новый файл
        timeData = TimeData{
            Date:        time.Now().Format("2006-01-02"),
            UsedMinutes: 0,
            DailyLimit:  config.TimeLimit.DailyMinutes,
            IsBlocked:   false,
            LastCheck:   time.Now(),
        }
        saveTimeData()
        return
    }
    json.Unmarshal(data, &timeData)
    
    // Сброс на новый день
    today := time.Now().Format("2006-01-02")
    if timeData.Date != today {
        timeData.Date = today
        timeData.UsedMinutes = 0
        timeData.DailyLimit = config.TimeLimit.DailyMinutes
        timeData.IsBlocked = false
        saveTimeData()
    }
}

// Сохранение данных
func saveTimeData() {
    data, _ := json.MarshalIndent(timeData, "", "  ")
    os.WriteFile("timedata.json", data, 0644)
}

// Проверка, залогинен ли пользователь
func isUserLoggedIn(username string) bool {
    // Выполнить: query user
    cmd := exec.Command("query", "user")
    output, err := cmd.Output()
    if err != nil {
        return false
    }
    // Проверить, есть ли username в выводе
    return bytes.Contains(output, []byte(username))
}

// Блокировка пользователя
func blockUser(username string) {
    // Отключить аккаунт
    exec.Command("net", "user", username, "/active:no").Run()
    
    // Если залогинен - выкинуть
    if isUserLoggedIn(username) {
        // Найти session ID и сделать logoff
        // Упрощённо: shutdown /l (logoff текущего)
        exec.Command("shutdown", "/l").Run()
    }
    
    timeData.IsBlocked = true
    saveTimeData()
}

// Разблокировка пользователя
func unblockUser(username string) {
    exec.Command("net", "user", username, "/active:yes").Run()
    timeData.IsBlocked = false
    saveTimeData()
}

// Основной цикл мониторинга
func monitorLoop() {
    ticker := time.NewTicker(1 * time.Minute)
    defer ticker.Stop()
    
    for range ticker.C {
        // Проверить, залогинен ли пользователь
        if isUserLoggedIn(config.Windows.Username) && !timeData.IsBlocked {
            // Добавить минуту
            timeData.UsedMinutes++
            timeData.LastCheck = time.Now()
            saveTimeData()
            
            // Проверить лимит
            if timeData.UsedMinutes >= timeData.DailyLimit {
                blockUser(config.Windows.Username)
            }
        }
    }
}

// Telegram бот
func startTelegramBot() {
    bot, err := tgbotapi.NewBotAPI(config.Telegram.BotToken)
    if err != nil {
        panic(err)
    }
    
    u := tgbotapi.NewUpdate(0)
    u.Timeout = 60
    updates := bot.GetUpdatesChan(u)
    
    for update := range updates {
        if update.Message == nil {
            continue
        }
        
        // Проверка админа
        if update.Message.From.ID != config.Telegram.AdminID {
            bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Доступ запрещён"))
            continue
        }
        
        // Обработка команд
        handleCommand(bot, update.Message)
    }
}

// Обработчик команд
func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message) {
    switch msg.Command() {
    case "status":
        text := fmt.Sprintf(
            "📊 Статус\n\n"+
            "Использовано: %dм / %dм\n"+
            "Осталось: %dм\n"+
            "Статус: %s",
            timeData.UsedMinutes,
            timeData.DailyLimit,
            timeData.DailyLimit - timeData.UsedMinutes,
            map[bool]string{true: "🔒 Заблокирован", false: "✅ Активен"}[timeData.IsBlocked],
        )
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, text))
        
    case "add":
        // Парсинг аргумента
        var minutes int
        fmt.Sscanf(msg.CommandArguments(), "%d", &minutes)
        
        timeData.DailyLimit += minutes
        saveTimeData()
        
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, 
            fmt.Sprintf("✅ Добавлено %d минут\nНовый лимит: %d минут", minutes, timeData.DailyLimit)))
        
    case "remove":
        var minutes int
        fmt.Sscanf(msg.CommandArguments(), "%d", &minutes)
        
        timeData.DailyLimit -= minutes
        if timeData.DailyLimit < 0 {
            timeData.DailyLimit = 0
        }
        saveTimeData()
        
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, 
            fmt.Sprintf("✅ Убрано %d минут\nНовый лимит: %d минут", minutes, timeData.DailyLimit)))
        
    case "unlock":
        unblockUser(config.Windows.Username)
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, "✅ Пользователь разблокирован"))
        
    default:
        bot.Send(tgbotapi.NewMessage(msg.Chat.ID, 
            "Доступные команды:\n"+
            "/status - статус\n"+
            "/add <минуты> - добавить время\n"+
            "/remove <минуты> - убрать время\n"+
            "/unlock - разблокировать"))
    }
}
```

---

## Установщик (install.bat)

```batch
@echo off
echo ================================
echo Screen Time Control MVP
echo ================================
echo.

REM Проверка прав администратора
net session >nul 2>&1
if %errorLevel% neq 0 (
    echo ОШИБКА: Требуются права администратора!
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

REM Компиляция
echo Компиляция...
go build -o screentime.exe main.go
if %errorLevel% neq 0 (
    echo Ошибка компиляции!
    pause
    exit /b 1
)

REM Создание директории
set INSTALL_DIR=%ProgramFiles%\ScreenTime
mkdir "%INSTALL_DIR%" 2>nul

REM Копирование файлов
copy screentime.exe "%INSTALL_DIR%\" /Y
copy config.yaml "%INSTALL_DIR%\" /Y

REM Настройка конфига
echo.
echo === Настройка ===
set /p BOT_TOKEN="Telegram Bot Token: "
set /p ADMIN_ID="Telegram Admin ID: "
set /p USERNAME="Windows Username для контроля: "
set /p LIMIT="Дневной лимит (минуты, по умолчанию 180): "
if "%LIMIT%"=="" set LIMIT=180

REM Обновление конфига через PowerShell
powershell -Command "(gc '%INSTALL_DIR%\config.yaml') -replace 'YOUR_BOT_TOKEN', '%BOT_TOKEN%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.yaml'"
powershell -Command "(gc '%INSTALL_DIR%\config.yaml') -replace '123456789', '%ADMIN_ID%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.yaml'"
powershell -Command "(gc '%INSTALL_DIR%\config.yaml') -replace 'TargetUser', '%USERNAME%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.yaml'"
powershell -Command "(gc '%INSTALL_DIR%\config.yaml') -replace 'daily_minutes: 180', 'daily_minutes: %LIMIT%' | Out-File -encoding ASCII '%INSTALL_DIR%\config.yaml'"

REM Создание службы Windows
echo.
echo Установка службы...
sc create ScreenTimeControl binPath= "%INSTALL_DIR%\screentime.exe" start= auto DisplayName= "Screen Time Control"
sc description ScreenTimeControl "Контроль экранного времени"
sc start ScreenTimeControl

echo.
echo ================================
echo Установка завершена!
echo ================================
echo Служба запущена.
echo Управление через Telegram бота.
echo.
pause
```

---

## Деинсталлятор (uninstall.bat)

```batch
@echo off
echo Удаление Screen Time Control...

net session >nul 2>&1
if %errorLevel% neq 0 (
    echo Требуются права администратора!
    pause
    exit /b 1
)

REM Остановка и удаление службы
sc stop ScreenTimeControl
sc delete ScreenTimeControl

REM Удаление файлов
set INSTALL_DIR=%ProgramFiles%\ScreenTime
rmdir /s /q "%INSTALL_DIR%"

echo Удаление завершено!
pause
```

---

## Зависимости (go.mod)

```go
module screentime

go 1.21

require (
    github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
    gopkg.in/yaml.v3 v3.0.1
)
```

---

## Что можно улучшить после MVP

После проверки работоспособности базовой версии:

1. **Фаза 2**: Добавить предупреждения за 15 минут
2. **Фаза 3**: Idle detection (не считать неактивное время)
3. **Фаза 4**: База данных вместо JSON
4. **Фаза 5**: Защита от обхода
5. **Фаза 6**: Шифрование конфига

---

## Тестирование MVP

### Ручное тестирование:
1. ✅ Установка через install.bat
2. ✅ Служба запускается автоматически
3. ✅ Telegram бот отвечает на команды
4. ✅ Время отслеживается корректно
5. ✅ Блокировка срабатывает при превышении лимита
6. ✅ Разблокировка через /unlock работает
7. ✅ Добавление/удаление минут работает
8. ✅ Данные сохраняются между перезапусками
9. ✅ Сброс на новый день работает

---

## Ограничения MVP

⚠️ **Известные проблемы (приемлемые для MVP):**
- Можно обойти через Safe Mode
- Можно остановить службу вручную
- Нет защиты конфига
- Считается всё время (даже idle)
- Нет предупреждений
- Нет логов
- Один пользователь только

**Это нормально для MVP!** Цель - проверить базовую работоспособность.

---

## Итого: MVP vs Full Version

| Функция | MVP | Full |
|---------|-----|------|
| Отслеживание времени | ✅ | ✅ |
| Блокировка | ✅ (простая) | ✅ (надёжная) |
| Telegram управление | ✅ | ✅ |
| Конфиг | ✅ (plain) | ✅ (encrypted) |
| Установка | ✅ (BAT) | ✅ (installer) |
| База данных | ❌ (JSON) | ✅ (SQLite) |
| Предупреждения | ❌ | ✅ |
| Idle detection | ❌ | ✅ |
| Защита от обхода | ❌ | ✅ |
| Логирование | ❌ | ✅ |
| Мульти-юзер | ❌ | ✅ |

**Размер кода:** ~300 строк vs ~2000+ строк

---

## Заключение

MVP фокусируется на **проверке гипотезы**: можем ли мы отслеживать время и блокировать пользователя через Telegram?

Если MVP работает - добавляем функции постепенно.
Если не работает - не тратим время на сложную архитектуру.

**Время разработки MVP: 4-6 часов**
**Время разработки Full: 2-3 недели**

Начнём с MVP! 🚀
