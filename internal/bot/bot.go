package bot

import (
	"context"
	"sync"
	"time"

	"github.com/disgoorg/disgo"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/cache"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/gateway"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/lavaqueue-plugin"
	"github.com/mroctopus/bottie-bot/internal/config"
	"github.com/mroctopus/bottie-bot/internal/player"
)

type RobotoBot struct {
	// Config
	Config *config.RobotoConfig
	// Clients
	Discord bot.Client
	Player  *player.Player
}

func (b *RobotoBot) Start(cmds []discord.ApplicationCommandCreate, r *handler.Mux) error {
	var wgBot sync.WaitGroup

	if b.Player != nil {
		wgBot.Add(1)
		go func() {
			defer wgBot.Done()
			var wgLavalink sync.WaitGroup

			nodes := b.Config.Lavalink.Nodes
			for i := range nodes {
				wgLavalink.Add(1)
				node := nodes[i]
				go func() {
					defer wgLavalink.Done()

					ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
					defer cancel()

					_, err := b.Player.Lavalink.AddNode(ctx, disgolink.NodeConfig{
						Name:     node.Name,
						Address:  node.Address,
						Password: node.Password,
						Secure:   node.Secure,
					})
					if err != nil {
						// TODO
					}

				}()
			}

			wgLavalink.Wait()
		}()
	}

	wgBot.Add(1)
	go func() {
		defer wgBot.Done()
		b.Discord.AddEventListeners(r)
		handler.SyncCommands(b.Discord, cmds, nil)
	}()

	wgBot.Wait()

	err := b.Discord.OpenGateway(context.Background())
	if err != nil {
		return err
	}

	return nil
}

func (b *RobotoBot) Stop() {
	b.Discord.Close(context.Background())
	if b.Player != nil {
		b.Player.Lavalink.Close()
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
