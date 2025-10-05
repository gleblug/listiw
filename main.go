package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"
	"time"
)

var (
	config     *Config
	timeData   *TimeData
	configPath string
	dataPath   string
)

func main() {
	// Флаги командной строки
	monitorMode := flag.Bool("monitor", false, "Run in monitor mode (check once and exit)")
	botMode := flag.Bool("bot", false, "Run in bot mode (Telegram bot)")
	flag.Parse()

	// Определяем пути к файлам
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	exeDir := filepath.Dir(exePath)

	// Ищем config.dat (скрытое имя) или config.yaml
	configPath = filepath.Join(exeDir, "config.dat")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		configPath = filepath.Join(exeDir, "config.yaml")
	}

	dataPath = filepath.Join(exeDir, "timedata.json")

	// Настройка логирования в файл
	logPath := filepath.Join(exeDir, "service.log")
	logFile, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err == nil {
		log.SetOutput(logFile)
		defer logFile.Close()
	}

	log.Println("=== Screen Time Control Starting ===")
	log.Printf("Mode: monitor=%v, bot=%v", *monitorMode, *botMode)
	log.Printf("Config path: %s", configPath)
	log.Printf("Data path: %s", dataPath)

	// Загрузить конфигурацию
	config, err = loadConfig(configPath)
	if err != nil {
		log.Fatal("Failed to load config:", err)
	}
	log.Printf("Config loaded. Monitoring user: %s", config.Windows.Username)

	// Загрузить данные времени
	timeData, err = loadTimeData(dataPath, config.TimeLimit.DailyMinutes)
	if err != nil {
		log.Fatal("Failed to load time data:", err)
	}
	log.Printf("Time data loaded. Used: %d/%d minutes", timeData.UsedMinutes, timeData.DailyLimit)

	// Сохранить начальное состояние
	if err := saveTimeData(dataPath, timeData); err != nil {
		log.Printf("Warning: failed to save initial time data: %v", err)
	}

	// Выбор режима работы
	if *botMode {
		// Режим Telegram бота - работает постоянно
		log.Println("Starting in BOT mode")
		startTelegramBot(config, timeData, dataPath)
	} else if *monitorMode {
		// Режим мониторинга - проверяет один раз и выходит
		log.Println("Starting in MONITOR mode")
		runMonitorCheck(config, timeData, dataPath)
	} else {
		// Режим по умолчанию - для локального тестирования
		log.Println("Starting in DEFAULT mode (local testing)")
		log.Println("Starting Telegram bot in background...")
		go startTelegramBot(config, timeData, dataPath)

		log.Println("Starting monitoring loop...")
		monitorLoop(config, timeData, dataPath)
	}
}

// runMonitorCheck выполняет одну проверку и выходит (для Task Scheduler)
func runMonitorCheck(config *Config, timeData *TimeData, dataPath string) {
	log.Println("Running single monitor check...")

	// Проверяем, залогинен ли пользователь и не заблокирован ли он
	if isUserLoggedIn(config.Windows.Username) && !timeData.IsBlocked {
		// Добавляем минуту
		timeData.UsedMinutes++
		timeData.LastCheck = time.Now()

		if err := saveTimeData(dataPath, timeData); err != nil {
			log.Printf("Error saving time data: %v", err)
		}

		remaining := timeData.DailyLimit - timeData.UsedMinutes
		log.Printf("User active. Used: %d/%d minutes, Remaining: %d minutes",
			timeData.UsedMinutes, timeData.DailyLimit, remaining)

		// Проверяем лимит
		if timeData.UsedMinutes >= timeData.DailyLimit {
			log.Println("Time limit exceeded! Blocking user...")
			blockUser(config.Windows.Username)
			timeData.IsBlocked = true
			saveTimeData(dataPath, timeData)
		}
	} else if timeData.IsBlocked {
		log.Println("User is blocked, skipping monitoring")
	} else {
		log.Println("User not logged in, skipping monitoring")
	}

	log.Println("Monitor check completed")
}
