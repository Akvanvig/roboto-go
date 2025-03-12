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
						return e.Respond(discord.InteractionResponseTypeCreateMessage, *message(&discord.MessageUpdate{}, "No music is currently playing", MessageTypeError, discord.MessageFlagEphemeral))
					}
					if *channelID != e.Channel().ID() {
						channel, _ := e.Client().Caches().Channel(*channelID)
						return e.Respond(discord.InteractionResponseTypeCreateMessage, *message(&discord.MessageUpdate{}, fmt.Sprintf("The bot is expecting music interactions in the %s channel", channel.Mention()), MessageTypeError, discord.MessageFlagEphemeral))
					}

					return next(e)
				}
			})

			/*
				r.SlashCommand("/status", h.OnPlayerStatus)
				r.SlashCommand("/pause", h.OnPlayerPause)
				r.SlashCommand("/resume", h.OnPlayerResume)
				r.SlashCommand("/stop", h.OnPlayerStop)
				r.SlashCommand("/previous", h.OnPlayerPrevious)
				r.SlashCommand("/volume", h.OnPlayerVolume)
				r.SlashCommand("/bass-boost", h.OnPlayerBassBoost)
				r.SlashCommand("/seek", h.OnPlayerSeek)
				r.Component("/previous", h.OnPlayerPreviousButton)
				r.Component("/pause_play", h.OnPlayerPlayPauseButton)
			*/
			r.SlashCommand("/queue", h.onQueue)
			r.SlashCommand("/volume", h.onVolume)
			r.Component("/skip", h.onSkipButton)
			r.Component("/stop", h.onStopButton)
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
		return e.CreateMessage(*message(&discord.MessageCreate{}, "You must be in a voice channel to queue music", MessageTypeError, discord.MessageFlagEphemeral))
	}

	src, _ := data.OptString("src")
	q := data.String("query")
	switch src {
	case "SoundCloud":
		q = lavalink.SearchTypeSoundCloud.Apply(q)
	case "YouTube Music":
		q = lavalink.SearchTypeYouTubeMusic.Apply(q)
	case "YouTube":
		fallthrough
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
				e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, fmt.Sprintf("No results found for %s", q), MessageTypeDefault, 0))
				return
			}

			_, ok := client.Caches().VoiceState(*e.GuildID(), e.ApplicationID())
			if !ok {
				err := client.UpdateVoiceState(context.Background(), *e.GuildID(), vs.ChannelID, false, false)
				if err != nil {
					e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
					return
				}
			}

			err := h.Player.Add(e.Ctx, *e.GuildID(), e.Channel().ID(), e.User(), tracks...)
			if err != nil {
				e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
				return
			}

			e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, fmt.Sprintf("Added %d songs to the queue", len(tracks)), MessageTypeDefault, 0))
		},
		func(err error) {
			e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		},
	)
	if err != nil {
		_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, "Failed to search for song", MessageTypeError, 0))
		return err
	}

	return nil
}

func (h *MusicHandler) onQueue(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	// TODO
	return nil
}

func (h *MusicHandler) onVolume(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	volume := data.Int("volume")
	err := h.Player.Volume(e.Ctx, *e.GuildID(), volume)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to adjust the volume", MessageTypeError, 0))
	}

	return e.CreateMessage(*message(&discord.MessageCreate{}, fmt.Sprintf("Set the volume to %d%%.", volume), MessageTypeDefault, 0))
}

func (h *MusicHandler) onSkipButton(e *handler.ComponentEvent) error {
	track, err := h.Player.Next(e.Ctx, *e.GuildID())
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to skip the current song", MessageTypeError, 0))
	}
	if track == nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "No more songs in the queue", MessageTypeDefault, 0))
	}

	e.Acknowledge()
	return nil
}

func (h *MusicHandler) onStopButton(e *handler.ComponentEvent) error {
	client := e.Client()
	err := h.Player.Stop(e.Ctx, *e.GuildID())
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to stop the music player", MessageTypeError, 0))
	}

	err = client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, false)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to disconnect bot from the voice channel", MessageTypeError, 0))
	}

	// TODO:
	// Look into this
	return e.CreateMessage(*message(&discord.MessageCreate{}, "Stopped playing music", MessageTypeDefault, 0))
}
