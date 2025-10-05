package main

import (
	"bytes"
	"log"
	"os/exec"
	"time"
)

// isUserLoggedIn проверяет, залогинен ли пользователь
func isUserLoggedIn(username string) bool {
	// Выполняем команду query user для получения списка активных сессий
	cmd := exec.Command("query", "user")
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error querying users: %v", err)
		return false
	}

	// Проверяем, есть ли username в выводе
	return bytes.Contains(bytes.ToLower(output), bytes.ToLower([]byte(username)))
}

// blockUser блокирует пользователя Windows
func blockUser(username string) {
	log.Printf("Blocking user: %s", username)

	// Отключаем аккаунт
	cmd := exec.Command("net", "user", username, "/active:no")
	if err := cmd.Run(); err != nil {
		log.Printf("Error disabling user account: %v", err)
	}

	// Если пользователь залогинен - выполняем logoff
	if isUserLoggedIn(username) {
		log.Println("User is logged in, performing logoff")
		// Используем shutdown /l для logoff текущей сессии
		cmd := exec.Command("shutdown", "/l", "/f")
		if err := cmd.Run(); err != nil {
			log.Printf("Error logging off user: %v", err)
		}
	}

	log.Println("User blocked successfully")
}

// unblockUser разблокирует пользователя Windows
func unblockUser(username string) {
	log.Printf("Unblocking user: %s", username)

	cmd := exec.Command("net", "user", username, "/active:yes")
	if err := cmd.Run(); err != nil {
		log.Printf("Error enabling user account: %v", err)
		return
	}

	log.Println("User unblocked successfully")
}

// monitorLoop основной цикл мониторинга
func monitorLoop(config *Config, timeData *TimeData, dataPath string) {
	log.Println("Starting monitoring loop (checking every minute)")
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
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
					log.Println("Time limit exceeded!")
					blockUser(config.Windows.Username)
					timeData.IsBlocked = true
					saveTimeData(dataPath, timeData)
				}
			} else if timeData.IsBlocked {
				log.Println("User is blocked, skipping monitoring")
			} else {
				log.Println("User not logged in, skipping monitoring")
			}
		}
	}
}
