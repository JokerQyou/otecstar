package main

import (
	"github.com/getlantern/systray"
	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
	"os"
	"otecstar/icons"
	"time"
)

const VERSION = "v0.1.0"

var logger zerolog.Logger

func init() {
	zlog.Logger = zlog.Output(zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339})
	logger = zlog.Logger.With().Str("module", "main").Logger()
}

func main() {
	systray.Run(onReady, onExit)
}

func onReady() {
	logger.Debug().Msg("Start")
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

	systray.SetIcon(icons.OK)
	systray.SetTooltip("OTECStar network status")

	_ = NewOTECStarApp(config.AuthConfig)
	logger.Debug().Msg("Ready")
}

func onExit() {
	logger.Debug().Msg("Quit")
}
