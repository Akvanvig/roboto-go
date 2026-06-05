package bot

import (
	"context"
	"log/slog"
	"time"

	"github.com/Akvanvig/roboto-go/internal/config"
	"github.com/Akvanvig/roboto-go/internal/ollama"
	"github.com/Akvanvig/roboto-go/internal/player"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"golang.org/x/sync/errgroup"
)

type RobotoBot struct {
	// Config
	Config *config.RobotoConfig
	// Clients
	Discord *bot.Client
	Player  *player.Player
	Ollama  *ollama.Ollama
}

func (b *RobotoBot) Start(cmds []discord.ApplicationCommandCreate, r *handler.Mux) error {
	var g errgroup.Group

	if b.Player != nil {
		g.Go(func() error {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			err := b.Player.Connect(ctx)
			return err
		})
	}

	g.Go(func() error {
		b.Discord.AddEventListeners(r)
		err := handler.SyncCommands(b.Discord, cmds, nil)
		return err
	})

	err := g.Wait()
	if err != nil {
		return err
	}

	err = b.Discord.OpenGateway(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (b *RobotoBot) Stop() {
	b.Discord.Close(context.Background())
	if b.Player != nil {
		b.Player.Disconnect()
	}
}

func New(logger *slog.Logger, cfg *config.RobotoConfig) (*RobotoBot, error) {
	roboto := &RobotoBot{
		Config: cfg,
	}

	discord, err := disgo.New(cfg.Discord.Token,
		bot.WithLogger(logger),
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
				gateway.IntentGuildMessages,
				gateway.IntentMessageContent,
				// gateway.IntentsDirectMessage,
			),
		),
		bot.WithCacheConfigOpts(
			cache.WithCaches(
				cache.FlagGuilds,
				cache.FlagVoiceStates,
			),
		),
	)
	if err != nil {
		return nil, err
	}

	roboto.Discord = discord
	if cfg.Lavalink != nil {
		logger.Info("lavalink integrations enabled")
		roboto.Player = player.New(*discord, cfg.Lavalink)
	} else {
		logger.Info("lavalink integrations disabled")
	}
	if cfg.Ollama != nil {
		logger.Info("ollama integrations enabled")
		roboto.Ollama = ollama.New(*discord, cfg.Ollama)
	} else {
		logger.Info("ollama integrations disabled")
	}

	return roboto, nil
}
