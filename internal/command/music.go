package command

import (
	"context"
	"fmt"

	"github.com/Akvanvig/roboto-go/internal/bot"
	"github.com/Akvanvig/roboto-go/internal/player"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/json"
)

// -- BOOTSTRAP --

// SEE https://github.com/KittyBot-Org/KittyBotGo/blob/master/service/bot/commands/player.go
func musicCommands(bot *bot.RobotoBot, r *handler.Mux) discord.ApplicationCommandCreate {
	if bot.Player == nil {
		return nil
	}

	cmds := discord.SlashCommandCreate{
		Name:        "music",
		Description: "Shows the music ??.",
		Contexts: []discord.InteractionContextType{
			discord.InteractionContextTypeGuild,
		},
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "play",
				Description: "Play some music",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionString{
						Name:        "query",
						Description: "The search query",
						Required:    true,
					},
					discord.ApplicationCommandOptionString{
						Name:        "src",
						Description: "The search source, default is YouTube",
						Choices: []discord.ApplicationCommandOptionChoiceString{
							{
								Name:  "YouTube",
								Value: string(lavalink.SearchTypeYouTube),
							},
							{
								Name:  "YouTube Music",
								Value: string(lavalink.SearchTypeYouTubeMusic),
							},
							{
								Name:  "SoundCloud",
								Value: string(lavalink.SearchTypeSoundCloud),
							},
						},
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "volume",
				Description: "Adjust the music volume",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionInt{
						Name:        "number",
						Description: "The volume percentage",
						Required:    true,
						MinValue:    json.Ptr(0),
						MaxValue:    json.Ptr(100),
					},
				},
			},
		},
	}

	h := &MusicHandler{
		Player: bot.Player,
	}
	r.Route("/music", func(r handler.Router) {
		r.SlashCommand("/play", h.onPlay)
		r.Group(func(r handler.Router) {
			// Middleware to check for existence of player
			r.Use(func(next handler.Handler) handler.Handler {
				return func(e *handler.InteractionEvent) error {
					channelID := h.Player.ChannelID(*e.GuildID())
					if channelID == nil {
						return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
							Embeds: json.Ptr(Embeds("No music is currently playing", MessageColorError)),
							Flags:  json.Ptr(discord.MessageFlagEphemeral),
						})
					}
					if *channelID != e.Channel().ID() {
						channel, _ := e.Client().Caches().Channel(*channelID)
						return e.Respond(discord.InteractionResponseTypeCreateMessage, discord.MessageUpdate{
							Embeds: json.Ptr(Embeds(fmt.Sprintf("The bot is expecting music interactions in the %s channel", channel.Mention()), MessageColorError)),
							Flags:  json.Ptr(discord.MessageFlagEphemeral),
						})
					}

					return next(e)
				}
			})

			r.SlashCommand("/volume", h.onVolume)
			r.Component("/skip", h.onSkipButton)
			r.Component("/stop", h.onStopButton)
			r.Component("/queue", h.onQueueButton)
		})
	})

	return cmds
}

// -- HANDLERS --

type MusicHandler struct {
	Player *player.Player
}

// NOTE:
// This isn't as easy as you would first expect.
func (h *MusicHandler) onPlay(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	client := e.Client()

	vs, ok := client.Caches().VoiceState(*e.GuildID(), e.User().ID)
	if !ok {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Must be in a voice channel to queue music", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	src, _ := data.OptString("src")
	q := data.String("query")
	switch src {
	case "SoundCloud":
		q = lavalink.SearchTypeSoundCloud.Apply(q)
	case "YouTube Music":
		q = lavalink.SearchTypeYouTubeMusic.Apply(q)
	case "YouTube":
		// Search query as is
	default:
		q = lavalink.SearchTypeYouTube.Apply(q)
	}

	err := e.DeferCreateMessage(false)
	if err != nil {
		return err
	}

	err = h.Player.Search(e.Ctx, *e.GuildID(), q,
		func(tracks ...lavalink.Track) {
			if len(tracks) == 0 {
				e.UpdateInteractionResponse(discord.MessageUpdate{
					Embeds: json.Ptr(Embeds(fmt.Sprintf("No results found for %s", q), MessageColorDefault)),
				})
				return
			}

			_, ok := client.Caches().VoiceState(*e.GuildID(), e.ApplicationID())
			if !ok {
				err := client.UpdateVoiceState(context.Background(), *e.GuildID(), vs.ChannelID, false, false)
				if err != nil {
					e.UpdateInteractionResponse(discord.MessageUpdate{
						Embeds: json.Ptr(Embeds(err.Error(), MessageColorError)),
					})
					return
				}
			}

			err := h.Player.Add(e.Ctx, *e.GuildID(), e.Channel().ID(), e.User(), tracks...)
			if err != nil {
				e.UpdateInteractionResponse(discord.MessageUpdate{
					Embeds: json.Ptr(Embeds(err.Error(), MessageColorError)),
				})
				return
			}

			e.UpdateInteractionResponse(discord.MessageUpdate{
				Embeds: json.Ptr(player.Embeds("Added to queue", true, tracks...)),
			})
		},
		func(err error) {
			e.UpdateInteractionResponse(discord.MessageUpdate{
				Embeds: json.Ptr(Embeds(err.Error(), MessageColorError)),
			})
		},
	)
	if err != nil {
		_, err = e.UpdateInteractionResponse(discord.MessageUpdate{
			Embeds: json.Ptr(Embeds("Failed to search for song", MessageColorError)),
		})
		return err
	}

	return nil
}

func (h *MusicHandler) onVolume(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	volume := data.Int("number")
	err := h.Player.Volume(e.Ctx, *e.GuildID(), volume)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to adjust volume", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds(fmt.Sprintf("Set volume to %d%%.", volume), MessageColorDefault),
	})
}

func (h *MusicHandler) onSkipButton(e *handler.ComponentEvent) error {
	track, err := h.Player.Next(e.Ctx, *e.GuildID())
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to skip current song", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}
	if track == nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Queue is currently empty", MessageColorDefault),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	e.Acknowledge()
	return nil
}

func (h *MusicHandler) onStopButton(e *handler.ComponentEvent) error {
	client := e.Client()

	err := client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, false)
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to disconnect bot from the voice channel", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: Embeds(fmt.Sprintf("%s stopped the music", e.User().Mention()), MessageColorDefault),
	})
}

func (h *MusicHandler) onQueueButton(e *handler.ComponentEvent) error {
	tracks, err := h.Player.Queue(e.Ctx, *e.GuildID())
	if err != nil {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Failed to get the current queue", MessageColorError),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	if len(tracks) == 0 {
		return e.CreateMessage(discord.MessageCreate{
			Embeds: Embeds("Queue is currently empty", MessageColorDefault),
			Flags:  discord.MessageFlagEphemeral,
		})
	}

	return e.CreateMessage(discord.MessageCreate{
		Embeds: player.Embeds("Next up", true, tracks...),
		Flags:  discord.MessageFlagEphemeral,
	})
}
