package bot

import (
	"context"
	"sync"
	"time"

	"github.com/Akvanvig/roboto-go/internal/config"
	"github.com/Akvanvig/roboto-go/internal/player"
	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/lavaqueue-plugin"
)

type RobotoBot struct {
	// Config
	Config *config.RobotoConfig
	// Clients
	Discord bot.Client
	Player  *player.Player
}

func (b *RobotoBot) Start(cmds []discord.ApplicationCommandCreate, r *handler.Mux) error {
	var wg sync.WaitGroup

	if b.Player != nil {
		wg.Add(1)
		go func() {
			defer wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			b.Player.AddNodes(ctx, b.Config.Lavalink.Nodes...)
		}()
	}

	wg.Add(1)
	go func() {
		defer wg.Done()
		b.Discord.AddEventListeners(r)
		handler.SyncCommands(b.Discord, cmds, nil)
	}()

	// TODO:
	// Proper error handling for sync commands
	// and adding nodes
	wg.Wait()

	err := b.Discord.OpenGateway(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (b *RobotoBot) Stop() {
	b.Discord.Close(context.Background())
	if b.Player != nil {
		b.Player.Close()
	}
}

func New(cfg *config.RobotoConfig) (*RobotoBot, error) {
	roboto := &RobotoBot{
		Config: cfg,
	}

	discord, err := disgo.New(cfg.Discord.Token,
		bot.WithGatewayConfigOpts(
			gateway.WithIntents(
				gateway.IntentGuilds,
				gateway.IntentGuildVoiceStates,
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

	if cfg.Lavalink != nil {
		lavalink := disgolink.New(discord.ApplicationID(), disgolink.WithPlugins(lavaqueue.New()))
		roboto.Player = player.New(discord, lavalink)
	}
	roboto.Discord = discord

	return roboto, nil
}
