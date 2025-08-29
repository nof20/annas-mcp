package logger

import (
	"log"
	"os"
	"strings"

	"go.uber.org/zap"
)

var logger *zap.Logger

func init() {
	var err error

	// Check if we're running the MCP server
	isMCPMode := false
	for _, arg := range os.Args[1:] {
		if arg == "mcp" {
			isMCPMode = true
			break
		}
	}

	logLevel := os.Getenv("ANNAS_LOG_LEVEL")

	if isMCPMode {
		logger, err = zap.NewProduction()
	} else {
		config := zap.NewDevelopmentConfig()
		level := zap.WarnLevel
		switch strings.ToLower(logLevel) {
		case "debug":
			level = zap.DebugLevel
		case "info":
			level = zap.InfoLevel
		}
		config.Level = zap.NewAtomicLevelAt(level)
		logger, err = config.Build()
	}

	if err != nil {
		log.Fatalf("Failed to initialize zap logger: %v", err)
	}
}

func GetLogger() *zap.Logger {
	return logger
}
