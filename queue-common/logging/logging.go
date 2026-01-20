package logging

import (
	"log"
	"os"
	"strings"
)

// Level controls verbosity.
// Supported values via LOG_LEVEL: "debug" | "info" (default).
type Level int

const (
	LevelInfo Level = iota
	LevelDebug
)

var level = parseLevel(os.Getenv("LOG_LEVEL"))

func parseLevel(v string) Level {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "debug":
		return LevelDebug
	default:
		return LevelInfo
	}
}

func IsDebug() bool { return level >= LevelDebug }

func Debugf(format string, args ...any) {
	if !IsDebug() {
		return
	}
	log.Printf("[DEBUG] "+format, args...)
}

func Infof(format string, args ...any) {
	log.Printf("[INFO] "+format, args...)
}

func Warnf(format string, args ...any) {
	log.Printf("[WARN] "+format, args...)
}

func Errorf(format string, args ...any) {
	log.Printf("[ERROR] "+format, args...)
}
