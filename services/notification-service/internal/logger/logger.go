package logger

import (
	"encoding/json"
	"os"
	"time"
)

type Fields map[string]any

var levelOrder = map[string]int{"debug": 0, "info": 1, "warn": 2, "error": 3}
var currentLevel = "info"

func init() {
	if l := os.Getenv("LOG_LEVEL"); l != "" {
		if _, ok := levelOrder[l]; ok {
			currentLevel = l
		}
	}
}

func log(level, message string, fields Fields) {
	if levelOrder[level] < levelOrder[currentLevel] {
		return
	}
	entry := map[string]any{"timestamp": time.Now().UTC().Format(time.RFC3339Nano), "level": level, "service": getenv("SERVICE_NAME", "notification-service"), "environment": getenv("APP_ENV", "development"), "message": message}
	for k, v := range fields {
		entry[k] = v
	}
	_ = json.NewEncoder(os.Stdout).Encode(entry)
}

func Debug(msg string, f ...Fields) { log("debug", msg, merge(f)) }
func Info(msg string, f ...Fields)  { log("info", msg, merge(f)) }
func Warn(msg string, f ...Fields)  { log("warn", msg, merge(f)) }
func Error(msg string, f ...Fields) { log("error", msg, merge(f)) }

func merge(fs []Fields) Fields {
	out := Fields{}
	for _, f := range fs {
		for k, v := range f {
			out[k] = v
		}
	}
	return out
}

func getenv(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}
