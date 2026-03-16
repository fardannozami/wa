package logger

import (
	"log"
	"os"
)

type Logger struct {
	level string
}

func New(level string) *Logger {
	return &Logger{level: level}
}

func (l *Logger) Debug(msg string, keysAndValues ...interface{}) {
	if l.level == "debug" {
		log.Printf("[DEBUG] "+msg+" %v", keysAndValues...)
	}
}

func (l *Logger) Info(msg string, keysAndValues ...interface{}) {
	log.Printf("[INFO] "+msg+" %v", keysAndValues...)
}

func (l *Logger) Warn(msg string, keysAndValues ...interface{}) {
	log.Printf("[WARN] "+msg+" %v", keysAndValues...)
}

func (l *Logger) Error(msg string, keysAndValues ...interface{}) {
	log.Printf("[ERROR] "+msg+" %v", keysAndValues...)
}

func (l *Logger) Fatal(msg string, keysAndValues ...interface{}) {
	log.Printf("[FATAL] "+msg+" %v", keysAndValues...)
	os.Exit(1)
}
