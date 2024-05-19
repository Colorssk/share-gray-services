package log

import (
	"fmt"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

var logger *zap.Logger

// log instance init
func InitLog() {
	level := viper.GetString("log.level")
	logLevel := zap.DebugLevel
	if "debug" == level {
		logLevel = zap.DebugLevel
	}

	if "info" == level {
		logLevel = zap.InfoLevel
	}

	if "error" == level {
		logLevel = zap.ErrorLevel
	}

	if "warn" == level {
		logLevel = zap.WarnLevel
	}
	fmt.Println(logLevel)

	logger = NewLogger(
		SetAppName(viper.GetString("log.appName")),
		SetDevelopment(viper.GetBool("log.development")),
		SetDebugFileName(viper.GetString("log.debugFileName")),
		SetErrorFileName(viper.GetString("log.errorFileName")),
		SetInfoFileName(viper.GetString("log.infoFileName")),
		SetMaxAge(viper.GetInt("log.maxAge")),
		SetMaxBackups(viper.GetInt("log.maxBackups")),
		SetMaxSize(viper.GetInt("log.maxSize")),
		SetLevel(zap.DebugLevel),
	)
}

func GetLogger() *zap.Logger {
	return logger
}
