package main

import (
	"flag"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Arguments
var (
	dev   = flag.Bool("dev", false, "Enable dev mode")
	token = flag.String("token", "", "Bot access token")
)

func main() {
	// Setup
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	flag.Parse()

	if *dev {
		// Note(Fredrico).
		// If we are running in dev mode, we automatically set the RootPath to be the same as go.mod's directory
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		log.Warn().Msg("Dev mode is enabled, do not use this flag in production")

		_, mainPath, _, _ := runtime.Caller(0)
		globals.RootPath = filepath.Join(filepath.Dir(mainPath), "../..")
	} else {
		// Note(Fredrico):
		// Else, set RootPath to executable path
		execPath, err := os.Executable()

		if err != nil {
			log.Fatal().Err(err).Msg("Failed to find running executable path")
		}

		globals.RootPath = filepath.Dir(execPath)
	}

	if *token == "" {
		log.Fatal().Msg("Token argument can not be empty")
	}

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)

	go bot.Start(token)

	log.Info().Msg("Running the bot, press Ctrl+C to exit")
	<-channel

	bot.Stop()
}
