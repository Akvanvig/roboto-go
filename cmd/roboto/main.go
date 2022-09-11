package main

import (
	"flag"
	"os"
	"os/signal"
	"path"
	"runtime"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/Akvanvig/roboto-go/internal/globals"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Arguments
var (
	token = flag.String("token", "", "Bot access token")
	dev   = flag.Bool("dev", false, "Enable dev mode")
)

func init() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Parse arguments
	flag.Parse()

	if *token == "" {
		log.Fatal().Msg("Token argument can not be empty")
	}

	// Note(Fredrico).
	// If we are running in dev mode, we automatically set the RootPath to be the same as go.mod's directory
	if *dev {
		_, filename, _, _ := runtime.Caller(0)
		globals.RootPath = path.Join(path.Dir(filename), "../..")
	}
}

func main() {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)

	go bot.Start(token)

	log.Info().Msg("Running the bot, press Ctrl+C to exit")
	<-channel

	bot.Stop()
}
