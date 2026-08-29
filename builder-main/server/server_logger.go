package main

import (
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

// ServerLogger handles all server logging to both stdout and file
type ServerLogger struct {
	writer *lumberjack.Logger
}

var serverLogger *ServerLogger

// InitServerLogger initializes the server logger with rotation
// Logs are written to both stdout and logs/report_server.log
func InitServerLogger() error {
	// Ensure logs directory exists
	logsDir := "logs"
	if err := os.MkdirAll(logsDir, 0755); err != nil {
		return err
	}

	// Create log file path
	logFilePath := filepath.Join(logsDir, "report_server.log")
	absPath, err := filepath.Abs(logFilePath)
	if err != nil {
		return err
	}

	// Configure lumberjack for log rotation
	writer := &lumberjack.Logger{
		Filename:   logFilePath,
		MaxSize:    20,   // MB - rotate when file reaches 20MB
		MaxAge:     30,   // days - delete old logs after 30 days
		MaxBackups: 5,    // keep at most 5 old log files
		Compress:   true, // compress rotated files
	}

	serverLogger = &ServerLogger{
		writer: writer,
	}

	// Create a multi-writer that writes to both stdout and the rotating file
	multiWriter := io.MultiWriter(os.Stdout, writer)

	// Set the default logger to use the multi-writer
	log.SetOutput(multiWriter)
	log.SetFlags(log.Ldate | log.Ltime)

	// Write session start marker
	log.Printf("========================================")
	log.Printf("Server logger initialized: %s", absPath)
	log.Printf("Log rotation: 20MB/30days/5backups")
	log.Printf("Session started at: %s", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf("========================================")

	return nil
}

// CloseServerLogger closes the server logger
func CloseServerLogger() error {
	if serverLogger != nil && serverLogger.writer != nil {
		log.Printf("========================================")
		log.Printf("Server shutting down at: %s", time.Now().Format("2006-01-02 15:04:05"))
		log.Printf("========================================")
		return serverLogger.writer.Close()
	}
	return nil
}

// Log level functions - prefix messages for easy filtering with grep
// Usage: logError("Failed to connect: %v", err)
// Output: [ERROR] Failed to connect: connection refused

// logInfo logs informational messages (normal operations)
func logInfo(format string, v ...interface{}) {
	log.Printf("[INFO] "+format, v...)
}

// logWarn logs warning messages (non-critical issues, fallbacks)
func logWarn(format string, v ...interface{}) {
	log.Printf("[WARN] "+format, v...)
}

// logError logs error messages (failures that need attention)
func logError(format string, v ...interface{}) {
	log.Printf("[ERROR] "+format, v...)
}

// logDebug logs debug messages (detailed internal info)
func logDebug(format string, v ...interface{}) {
	log.Printf("[DEBUG] "+format, v...)
}
