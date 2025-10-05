package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// startTelegramBot запускает Telegram бота
func startTelegramBot(config *Config, timeData *TimeData, dataPath string) {
	log.Println("Starting Telegram bot...")

	bot, err := tgbotapi.NewBotAPI(config.Telegram.BotToken)
	if err != nil {
		log.Printf("Failed to create bot: %v", err)
		return
	}

	bot.Debug = false
	log.Printf("Authorized on account @%s", bot.Self.UserName)

	// Отправляем уведомление админу о запуске
	startMsg := tgbotapi.NewMessage(config.Telegram.AdminID,
		"🟢 Screen Time Control запущен\n\n"+
			"Бот активен и готов к работе.\n"+
			"Используйте /status для проверки.")
	if _, err := bot.Send(startMsg); err != nil {
		log.Printf("Failed to send startup notification: %v", err)
	}

	// Настройка graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Обработка сигналов завершения
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-sigChan
		log.Println("Received shutdown signal")
		cancel()
	}()

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	log.Println("Telegram bot is ready to receive commands")

	// Обработка graceful shutdown при выходе
	defer func() {
		log.Println("Telegram bot shutting down...")
		stopMsg := tgbotapi.NewMessage(config.Telegram.AdminID,
			"🔴 Screen Time Control остановлен\n\n"+
				"Бот больше не активен.\n"+
				"Мониторинг времени приостановлен.")
		bot.Send(stopMsg)
		bot.StopReceivingUpdates()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Context cancelled, stopping bot")
			return
		case update := <-updates:
			if update.Message == nil {
				continue
			}

			// Проверка прав администратора
			if update.Message.From.ID != config.Telegram.AdminID {
				msg := tgbotapi.NewMessage(update.Message.Chat.ID, "❌ Доступ запрещён. Вы не являетесь администратором.")
				bot.Send(msg)
				log.Printf("Unauthorized access attempt from user ID: %d", update.Message.From.ID)
				continue
			}

			// Обработка команд
			handleCommand(bot, update.Message, config, dataPath)
		}
	}
}

// handleCommand обрабатывает команды Telegram бота
func handleCommand(bot *tgbotapi.BotAPI, msg *tgbotapi.Message, config *Config, dataPath string) {
	log.Printf("Received command from admin: %s", msg.Text)

	// Перечитываем актуальные данные из файла перед каждой командой
	timeData, err := loadTimeData(dataPath, config.TimeLimit.DailyMinutes)
	if err != nil {
		log.Printf("Error loading time data: %v", err)
		reply := tgbotapi.NewMessage(msg.Chat.ID, "❌ Ошибка загрузки данных")
		bot.Send(reply)
		return
	}

	var responseText string

	switch msg.Command() {
	case "start", "help":
		responseText = "🤖 Screen Time Control Bot\n\n" +
			"Доступные команды:\n" +
			"/status - показать текущий статус\n" +
			"/add <минуты> - добавить минуты к лимиту\n" +
			"/remove <минуты> - убрать минуты из лимита\n" +
			"/unlock - разблокировать пользователя\n" +
			"/help - показать эту справку"

	case "status":
		remaining := timeData.DailyLimit - timeData.UsedMinutes
		if remaining < 0 {
			remaining = 0
		}

		statusEmoji := "✅ Активен"
		if timeData.IsBlocked {
			statusEmoji = "🔒 Заблокирован"
		}

		responseText = fmt.Sprintf(
			"📊 Статус экранного времени\n\n"+
				"👤 Пользователь: %s\n"+
				"📅 Дата: %s\n\n"+
				"⏱ Использовано: %d мин / %d мин\n"+
				"⏳ Осталось: %d мин\n"+
				"📈 Использовано: %.1f%%\n\n"+
				"Статус: %s\n"+
				"🕐 Последняя проверка: %s",
			config.Windows.Username,
			timeData.Date,
			timeData.UsedMinutes,
			timeData.DailyLimit,
			remaining,
			float64(timeData.UsedMinutes)/float64(timeData.DailyLimit)*100,
			statusEmoji,
			timeData.LastCheck.Format("15:04:05"),
		)

	case "add":
		args := msg.CommandArguments()
		if args == "" {
			responseText = "❌ Укажите количество минут.\nПример: /add 30"
			break
		}

		minutes, err := strconv.Atoi(strings.TrimSpace(args))
		if err != nil || minutes <= 0 {
			responseText = "❌ Неверное количество минут. Укажите положительное число."
			break
		}

		timeData.DailyLimit += minutes
		if err := saveTimeData(dataPath, timeData); err != nil {
			log.Printf("Error saving time data: %v", err)
		}

		responseText = fmt.Sprintf(
			"✅ Добавлено %d минут\n\n"+
				"Новый лимит на сегодня: %d минут\n"+
				"Использовано: %d минут\n"+
				"Осталось: %d минут",
			minutes,
			timeData.DailyLimit,
			timeData.UsedMinutes,
			timeData.DailyLimit-timeData.UsedMinutes,
		)

		log.Printf("Admin added %d minutes. New limit: %d", minutes, timeData.DailyLimit)

	case "remove":
		args := msg.CommandArguments()
		if args == "" {
			responseText = "❌ Укажите количество минут.\nПример: /remove 15"
			break
		}

		minutes, err := strconv.Atoi(strings.TrimSpace(args))
		if err != nil || minutes <= 0 {
			responseText = "❌ Неверное количество минут. Укажите положительное число."
			break
		}

		timeData.DailyLimit -= minutes
		if timeData.DailyLimit < 0 {
			timeData.DailyLimit = 0
		}
		if err := saveTimeData(dataPath, timeData); err != nil {
			log.Printf("Error saving time data: %v", err)
		}

		responseText = fmt.Sprintf(
			"✅ Убрано %d минут\n\n"+
				"Новый лимит на сегодня: %d минут\n"+
				"Использовано: %d минут\n"+
				"Осталось: %d минут",
			minutes,
			timeData.DailyLimit,
			timeData.UsedMinutes,
			timeData.DailyLimit-timeData.UsedMinutes,
		)

		log.Printf("Admin removed %d minutes. New limit: %d", minutes, timeData.DailyLimit)

		// Проверяем, не превышен ли лимит после уменьшения
		if timeData.UsedMinutes >= timeData.DailyLimit && !timeData.IsBlocked {
			blockUser(config.Windows.Username)
			timeData.IsBlocked = true
			saveTimeData(dataPath, timeData)
			responseText += "\n\n⚠️ Лимит превышен! Пользователь заблокирован."
		}

	case "unlock":
		if !timeData.IsBlocked {
			responseText = "ℹ️ Пользователь не заблокирован"
			break
		}

		unblockUser(config.Windows.Username)
		timeData.IsBlocked = false
		if err := saveTimeData(dataPath, timeData); err != nil {
			log.Printf("Error saving time data: %v", err)
		}

		responseText = fmt.Sprintf(
			"✅ Пользователь %s разблокирован\n\n"+
				"Можно войти в систему.\n"+
				"Использовано: %d/%d минут",
			config.Windows.Username,
			timeData.UsedMinutes,
			timeData.DailyLimit,
		)

	default:
		responseText = "❌ Неизвестная команда. Используйте /help для списка команд."
	}

	// Отправляем ответ
	reply := tgbotapi.NewMessage(msg.Chat.ID, responseText)
	if _, err := bot.Send(reply); err != nil {
		log.Printf("Error sending message: %v", err)
	}
}
