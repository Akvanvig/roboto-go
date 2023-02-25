package main

import (
	"flag"
	"os"
	"os/signal"

	// Note(Fredrico):
	// Ideally we want to import util here first, but the formatter
	// keeps auto fucking that up so welp.
	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/Akvanvig/roboto-go/internal/util"
	"github.com/rs/zerolog/log"
)

func main() {
	// Stub function to ensure util is imported and the init() func runs
	util.Init()

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
