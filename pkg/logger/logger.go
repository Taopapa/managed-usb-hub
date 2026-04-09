package logger

import (
	"context"
	"io"
	"os"
	"path/filepath"

	"github.com/sirupsen/logrus"
	"github.com/wailsapp/wails/v2/pkg/runtime"
	"gopkg.in/natefinch/lumberjack.v2"
)

var Log = logrus.New()
var appContext context.Context

// InitLogger initializes logrus with lumberjack for log rotation
func InitLogger(logDir string, isCLI bool) {
	if isCLI {
		// Keep CLI output clean; command handlers print user-facing results directly.
		Log.SetOutput(io.Discard)
		return
	}

	lumberjackLogger := &lumberjack.Logger{
		Filename:   filepath.Join(logDir, "app.log"),
		MaxSize:    10, // MB
		MaxBackups: 5,
		MaxAge:     30, // days
		Compress:   true,
	}

	multiWriter := io.MultiWriter(os.Stdout, lumberjackLogger)
	Log.SetOutput(multiWriter)
	Log.SetFormatter(&logrus.JSONFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
	})
}

// SetContext saves the Wails context so we can emit events to frontend
func SetContext(ctx context.Context) {
	appContext = ctx
}

// Error logs an error and optionally emits it to the frontend
func Error(message string, deviceID string, event string) {
	if event == "" {
		event = "Error"
	}
	Log.WithFields(logrus.Fields{"device": deviceID, "event": event}).Error(message)
	if appContext != nil {
		runtime.EventsEmit(appContext, "backend-error", map[string]interface{}{
			"message":  message,
			"deviceID": deviceID,
		})
	}
}

// Info logs an informational message
func Info(message string, deviceID string, event string) {
	if event == "" {
		event = "Info"
	}
	Log.WithFields(logrus.Fields{"device": deviceID, "event": event}).Info(message)
}

// Warn logs a warning message
func Warn(message string, deviceID string, event string) {
	if event == "" {
		event = "Warn"
	}
	Log.WithFields(logrus.Fields{"device": deviceID, "event": event}).Warn(message)
}

// Debug logs a debug message
func Debug(message string, deviceID string, event string) {
	if event == "" {
		event = "Debug"
	}
	Log.WithFields(logrus.Fields{"device": deviceID, "event": event}).Debug(message)
}

// GetLogPath gets the path to the current log file (if needed by Wails)
func GetLogPath(logDir string) string {
	return filepath.Join(logDir, "app.log")
}
