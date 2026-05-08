package logger

import (
	"encoding/json"
	"os"
	"time"
)

type Level string

const (
	DEBUG Level = "debug"
	INFO  Level = "info"
	WARN  Level = "warn"
	ERROR Level = "error"
)

var levelOrder = map[Level]int{DEBUG: 0, INFO: 1, WARN: 2, ERROR: 3}

var currentLevel Level

func init() {
	l := Level(os.Getenv("LOG_LEVEL"))
	if _, ok := levelOrder[l]; !ok {
		l = INFO
	}
	currentLevel = l
}

type Fields map[string]any

func log(level Level, message string, fields Fields) {
	if levelOrder[level] < levelOrder[currentLevel] {
		return
	}
	entry := map[string]any{
		"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
		"level":       string(level),
		"service":     getenv("SERVICE_NAME", "unknown"),
		"environment": getenv("APP_ENV", "development"),
		"message":     message,
	}
	for k, v := range fields {
		entry[k] = v
	}
	_ = json.NewEncoder(os.Stdout).Encode(entry)
}

func Debug(msg string, fields ...Fields) { log(DEBUG, msg, merge(fields)) }
func Info(msg string, fields ...Fields)  { log(INFO, msg, merge(fields)) }
func Warn(msg string, fields ...Fields)  { log(WARN, msg, merge(fields)) }
func Error(msg string, fields ...Fields) { log(ERROR, msg, merge(fields)) }

func merge(fs []Fields) Fields {
	out := Fields{}
	for _, f := range fs {
		for k, v := range f {
			out[k] = v
		}
	}
	return out
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
