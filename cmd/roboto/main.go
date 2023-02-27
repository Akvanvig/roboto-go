package main

import (
	"flag"
	"os"
	"os/signal"

	// This import has to be top for init to setup logger state properly
	_ "github.com/Akvanvig/roboto-go/internal/_setup"
	// Rest of the imports
	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/rs/zerolog/log"
)

func main() {
	// Arguments
	var token = flag.String("token", "", "Bot access token")
	flag.Parse()

	if *token == "" {
		log.Fatal().Msg("Token argument can not be empty")
	}

	// Run
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, os.Interrupt)

	go bot.Start(token)

	log.Info().Msg("Running the bot, press Ctrl+C to exit")
	<-channel

	bot.Stop()
}
