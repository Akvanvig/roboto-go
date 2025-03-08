package main

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/mroctopus/bottie-bot/internal/bot"
	"github.com/mroctopus/bottie-bot/internal/command"
	"github.com/mroctopus/bottie-bot/internal/config"
	"github.com/mroctopus/bottie-bot/internal/util"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

func main() {
	if util.IsDev() {
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})
	}

	log.Info().Msg("Reading config...")

	cfg, err := config.Load()
	if err != nil {
		log.Panic().Err(err).Msg("Failed to read config")
	}

	log.Info().Msg("Initializing bot...")

	bot, err := bot.New(cfg)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to initialize bot")
	}

	cmds, r := command.New(bot)

	log.Info().Msg("Starting bot...")

	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGTERM, syscall.SIGINT)

	err = bot.Start(cmds, r)
	if err != nil {
		log.Panic().Err(err).Msg("Failed to start bot")
	}

	log.Info().Msg("Bot started, press Ctrl+C to exit")
	<-channel

	log.Info().Msg("Shutting down bot...")

	bot.Stop()

	log.Info().Msg("Finished shutting down bot")
}
