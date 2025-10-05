package main

import (
	"log"
	"os/exec"
	"strings"
	"time"
)

// isUserLoggedIn проверяет, залогинен ли пользователь
func isUserLoggedIn(username string) bool {
	// Проверяем наличие процесса explorer.exe для данного пользователя
	// explorer.exe всегда запущен когда пользователь залогинен
	psCommand := "(Get-Process explorer -IncludeUserName -ErrorAction SilentlyContinue | Where-Object {$_.UserName -like '*\\" + username + "'}).Count -gt 0"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCommand)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error checking user login status: %v", err)
		return false
	}

	// PowerShell вернет "True" или "False"
	result := strings.TrimSpace(string(output))
	isLoggedIn := strings.EqualFold(result, "True")

	if isLoggedIn {
		log.Printf("User %s is logged in (explorer.exe found)", username)
	} else {
		log.Printf("User %s is not logged in (no explorer.exe)", username)
	}

	return isLoggedIn
}

// blockUser блокирует пользователя Windows - завершает сессию
func blockUser(username string) {
	log.Printf("Blocking user: %s", username)

	// Используем PowerShell для выполнения shutdown
	psCommand := "-Command \"shutdown.exe /l\""
	cmd := exec.Command("powershell.exe", psCommand)
	output, err := cmd.CombinedOutput()

	log.Printf("Shutdown output: %s", strings.TrimSpace(string(output)))

	if err != nil {
		log.Printf("Error during shutdown: %v", err)
	} else {
		log.Println("Shutdown command executed")
	}
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
