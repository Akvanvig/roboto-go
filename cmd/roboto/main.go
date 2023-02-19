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
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix

	// Parse arguments
	flag.Parse()

	// Note(Fredrico).
	// If we are running in dev mode, we automatically set the RootPath to be the same as go.mod's directory
	if *dev {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

		log.Warn().Msg("Dev mode is enabled, do not use this flag in production")

		_, filename, _, _ := runtime.Caller(0)
		globals.RootPath = filepath.Join(filepath.Dir(filename), "../..")
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
