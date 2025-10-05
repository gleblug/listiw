package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config структура для конфигурации
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

// TimeData структура для хранения данных о времени
type TimeData struct {
	Date        string    `json:"date"`
	UsedMinutes int       `json:"used_minutes"`
	DailyLimit  int       `json:"daily_limit"`
	IsBlocked   bool      `json:"is_blocked"`
	LastCheck   time.Time `json:"last_check"`
}

// loadConfig загружает конфигурацию из YAML файла
func loadConfig(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("reading config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Валидация
	if config.Telegram.BotToken == "" || config.Telegram.BotToken == "YOUR_BOT_TOKEN" {
		return nil, fmt.Errorf("telegram bot_token not configured")
	}
	if config.Telegram.AdminID == 0 || config.Telegram.AdminID == 123456789 {
		return nil, fmt.Errorf("telegram admin_id not configured")
	}
	if config.Windows.Username == "" || config.Windows.Username == "TargetUser" {
		return nil, fmt.Errorf("windows username not configured")
	}
	if config.TimeLimit.DailyMinutes <= 0 {
		return nil, fmt.Errorf("daily_minutes must be positive")
	}

	return &config, nil
}

// loadTimeData загружает данные о времени из JSON файла
func loadTimeData(dataPath string, defaultLimit int) (*TimeData, error) {
	data, err := os.ReadFile(dataPath)
	if err != nil {
		// Создать новый файл
		timeData := &TimeData{
			Date:        time.Now().Format("2006-01-02"),
			UsedMinutes: 0,
			DailyLimit:  defaultLimit,
			IsBlocked:   false,
			LastCheck:   time.Now(),
		}
		return timeData, nil
	}

	var timeData TimeData
	if err := json.Unmarshal(data, &timeData); err != nil {
		// Создать новый при ошибке парсинга
		timeData = TimeData{
			Date:        time.Now().Format("2006-01-02"),
			UsedMinutes: 0,
			DailyLimit:  defaultLimit,
			IsBlocked:   false,
			LastCheck:   time.Now(),
		}
		return &timeData, nil
	}

	// Проверка на новый день - сброс счётчика
	today := time.Now().Format("2006-01-02")
	if timeData.Date != today {
		timeData.Date = today
		timeData.UsedMinutes = 0
		timeData.DailyLimit = defaultLimit
		timeData.IsBlocked = false
	}

	return &timeData, nil
}

// saveTimeData сохраняет данные о времени в JSON файл
func saveTimeData(dataPath string, timeData *TimeData) error {
	data, err := json.MarshalIndent(timeData, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling time data: %w", err)
	}

	if err := os.WriteFile(dataPath, data, 0644); err != nil {
		return fmt.Errorf("writing time data: %w", err)
	}

	return nil
}
