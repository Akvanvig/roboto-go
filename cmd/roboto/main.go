package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/Akvanvig/roboto-go/internal/command"
	"github.com/Akvanvig/roboto-go/internal/config"
)

func main() {
	// TODO:
	// We should send this logger downwards to our other functions
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	logger.Info("Reading config...")
	cfg, err := config.Load()
	if err != nil {
		logger.Error("Failed to read config", slog.Any("error", err))
		panic(err)
	}

	logger.Info("Initializing bot...")
	bot, err := bot.New(cfg)
	if err != nil {
		logger.Error("Failed to initialize bot", slog.Any("error", err))
		panic(err)
	}

	logger.Info("Starting bot...")
	cmds, r := command.New(bot)
	err = bot.Start(cmds, r)
	if err != nil {
		logger.Error("Failed to start bot", slog.Any("error", err))
		panic(err)
	}

	logger.Info("Bot started, press Ctrl+C to exit")
	channel := make(chan os.Signal, 1)
	signal.Notify(channel, syscall.SIGTERM, syscall.SIGINT)
	<-channel

	logger.Info("Shutting down bot...")
	bot.Stop()

	logger.Info("Finished shutting down bot")
}
