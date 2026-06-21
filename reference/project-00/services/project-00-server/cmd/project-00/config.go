package main

import (
	"strings"
	"log/slog"

	"github.com/joho/godotenv"
	"github.com/lwlee2608/adder"
    "github.com/lwlee2608/project-00/internal/api/http"
)

type Config struct {
	Log  LogConfig
	Http http.Config
}

var config Config

func InitConfig() error {
	_ = godotenv.Overload()

	adder.SetConfigName("application")
	adder.AddConfigPath(".")
	adder.SetConfigType("yaml")
	adder.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	adder.AutomaticEnv()

	if err := adder.ReadInConfig(); err != nil {
		return err
	}

	if err := adder.Unmarshal(&config); err != nil {
		return err
	}

	initLogger(config.Log.Level)

	if strings.ToUpper(config.Log.Level) == LOG_LEVEL_DEBUG {
		configJSON, err := adder.PrettyJSON(config)
		if err == nil {
			slog.Debug("Config loaded:")
			slog.Debug(configJSON)
		}
	}

	return nil
}
