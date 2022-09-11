package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// Arguments
var (
	token = flag.String("token", "", "Bot access token")
)

func init() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Parse arguments
	flag.Parse()
}

func main() {
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)

	go bot.Start(token)

	log.Info().Msg("Running the bot, press Ctrl+C to exit")
	<-channel

	bot.Stop()
}
