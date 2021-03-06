package main

import (
	"github.com/getlantern/systray"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"os"
	"time"
)

var logger zerolog.Logger

func init() {
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	logger = zlog.Logger.With().Str("module", "main").Logger()
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	config, err := LoadConfig()
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to load config file")
	}

	// Set log level
	if lvl, err := zerolog.ParseLevel(config.LogLevel); err != nil {
		logger.Warn().Err(err).Str("log_level", config.LogLevel).Msg("Unknown log level")
	} else {
		zerolog.SetGlobalLevel(lvl)
	}

	_ = NewOTECStarApp(&config)
	logger.Info().Msg("Ready")
}

func onExit() {
	logger.Info().Msg("Quit")
}
