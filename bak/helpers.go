package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/viper"
)

// validateEnv ensures that the required variables have been supplied
func validateEnv() error {

	viper.AutomaticEnv()

	if viper.IsSet("RUNNER_DEBUG") {
		slog.Info("setting debug logging")
		opts := &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}

		logger := slog.New(slog.NewTextHandler(os.Stdout, opts))
		slog.SetDefault(logger)
	}

	viper.SetEnvPrefix("GITHUB")

	for _, v := range []string{"TOKEN", "PROJECT_ID", "FIELD_ID"} {
		if !viper.IsSet(v) {
			return fmt.Errorf("missing required environment variable: GITHUB_%v", v)
		}
	}

	return nil
}
