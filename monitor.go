package main

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// isUserLoggedIn проверяет, залогинен ли пользователь
func isUserLoggedIn(username string) bool {
	psCommand := "(Get-Process explorer -IncludeUserName -ErrorAction SilentlyContinue | Where-Object {$_.UserName -like '*\\" + username + "'}).Count -gt 0"
	cmd := exec.Command("powershell.exe", "-NoProfile", "-NonInteractive", "-Command", psCommand)
	output, err := cmd.Output()
	if err != nil {
		log.Printf("Error checking user login status: %v", err)
		return false
	}

	result := strings.TrimSpace(string(output))
	isLoggedIn := strings.EqualFold(result, "True")

	if isLoggedIn {
		log.Printf("User %s is logged in", username)
	}

	return isLoggedIn
}

// blockUser выполняет logoff через PowerShell скрипт
func blockUser(scriptPath string) {
	log.Println("Executing logoff script...")

	cmd := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptPath)
	output, err := cmd.CombinedOutput()

	if err != nil {
		log.Printf("Error executing logoff script: %v, output: %s", err, string(output))
	} else {
		log.Printf("Logoff script executed: %s", strings.TrimSpace(string(output)))
	}
}

// unblockUser разблокирует пользователя
func unblockUser(username string) {
	log.Printf("Unblocking user: %s", username)
	log.Println("User unblocked (no action needed, just clearing blocked flag)")
}

// monitorLoop основной цикл мониторинга
func monitorLoop(config *Config, timeData *TimeData, dataPath string) {
	// Получаем путь к директории с исполняемым файлом
	exePath, err := os.Executable()
	if err != nil {
		log.Fatal("Failed to get executable path:", err)
	}
	exeDir := filepath.Dir(exePath)
	logoffScript := filepath.Join(exeDir, "logoff.ps1")

	log.Println("Starting monitoring loop (checking every minute)")
	log.Printf("Logoff script path: %s", logoffScript)

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
					blockUser(logoffScript)
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
