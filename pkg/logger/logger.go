package logger

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	LevelInfo    = "INFO"
	LevelSuccess = "SUCCESS"
	LevelError   = "ERROR"
	LevelWarn    = "WARN"
	LevelDebug   = "DEBUG"
)

var logDir = "logs"

type Logger struct {
	logDir string
}

func New() *Logger {
	// Create logs directory if it doesn't exist
	if err := os.MkdirAll(logDir, 0755); err != nil {
		log.Printf("Failed to create log directory: %v", err)
	}

	// Clean old logs (older than 7 days)
	cleanOldLogs()

	return &Logger{logDir: logDir}
}

func cleanOldLogs() {
	files, err := os.ReadDir(logDir)
	if err != nil {
		return
	}

	now := time.Now()
	maxAge := 7 * 24 * time.Hour

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		age := now.Sub(info.ModTime())
		if age > maxAge {
			filePath := filepath.Join(logDir, file.Name())
			if err := os.Remove(filePath); err == nil {
				log.Printf("Deleted old log file: %s", file.Name())
			}
		}
	}
}

func getDateString() string {
	return time.Now().Format("2006-01-02")
}

func getLogFilePath(isError bool, appName string) string {
	dateStr := getDateString()
	var filename string

	if appName != "" {
		appSlug := strings.ToLower(strings.ReplaceAll(appName, " ", "-"))
		if isError {
			filename = fmt.Sprintf("%s-error-%s.log", appSlug, dateStr)
		} else {
			filename = fmt.Sprintf("%s-%s.log", appSlug, dateStr)
		}
	} else {
		if isError {
			filename = fmt.Sprintf("error-%s.log", dateStr)
		} else {
			filename = fmt.Sprintf("app-%s.log", dateStr)
		}
	}

	return filepath.Join(logDir, filename)
}

func writeToFile(message string, isError bool, appName string) {
	// Write to app-specific log
	if appName != "" {
		filePath := getLogFilePath(isError, appName)
		appendToFile(filePath, message)
	}

	// Always write to main log
	mainFilePath := getLogFilePath(isError, "")
	appendToFile(mainFilePath, message)
}

func appendToFile(filePath, message string) {
	f, err := os.OpenFile(filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Failed to open log file: %v", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(message + "\n"); err != nil {
		log.Printf("Failed to write to log file: %v", err)
	}
}

func formatLog(level, message, metadata string) string {
	timestamp := time.Now().Format(time.RFC3339)
	if metadata != "" {
		return fmt.Sprintf("[%s] [%s] %s %s", timestamp, level, message, metadata)
	}
	return fmt.Sprintf("[%s] [%s] %s", timestamp, level, message)
}

func (l *Logger) Info(message string, metadata map[string]interface{}) {
	metaStr := formatMetadata(metadata)
	logMsg := formatLog(LevelInfo, message, metaStr)
	log.Println(logMsg)
	writeToFile(logMsg, false, getAppName(metadata))
}

func (l *Logger) Success(message string, metadata map[string]interface{}) {
	metaStr := formatMetadata(metadata)
	logMsg := formatLog(LevelSuccess, "✅ "+message, metaStr)
	log.Println(logMsg)
	writeToFile(logMsg, false, getAppName(metadata))
}

func (l *Logger) Error(message string, err error, metadata map[string]interface{}) {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}
	if err != nil {
		metadata["error"] = err.Error()
	}
	metaStr := formatMetadata(metadata)
	logMsg := formatLog(LevelError, "❌ "+message, metaStr)
	log.Println(logMsg)
	appName := getAppName(metadata)
	writeToFile(logMsg, false, appName)
	writeToFile(logMsg, true, appName)
}

func (l *Logger) Warn(message string, metadata map[string]interface{}) {
	metaStr := formatMetadata(metadata)
	logMsg := formatLog(LevelWarn, "⚠️ "+message, metaStr)
	log.Println(logMsg)
	writeToFile(logMsg, false, getAppName(metadata))
}

func (l *Logger) Debug(message string, metadata map[string]interface{}) {
	metaStr := formatMetadata(metadata)
	logMsg := formatLog(LevelDebug, "🔍 "+message, metaStr)
	log.Println(logMsg)
	writeToFile(logMsg, false, getAppName(metadata))
}

func (l *Logger) Flow(flow, step string, metadata map[string]interface{}) {
	message := fmt.Sprintf("[%s] %s", flow, step)
	l.Info(message, metadata)
}

func formatMetadata(metadata map[string]interface{}) string {
	if len(metadata) == 0 {
		return ""
	}

	var parts []string
	for k, v := range metadata {
		if k == "appName" {
			continue // Skip appName from metadata display
		}
		parts = append(parts, fmt.Sprintf(`"%s":"%v"`, k, v))
	}

	if len(parts) == 0 {
		return ""
	}

	return "{" + strings.Join(parts, ",") + "}"
}

func getAppName(metadata map[string]interface{}) string {
	if metadata == nil {
		return ""
	}
	if appName, ok := metadata["appName"].(string); ok {
		return appName
	}
	return ""
}
