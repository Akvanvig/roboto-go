package command

import (
	"context"
	"fmt"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/snowflake/v2"
	"github.com/mroctopus/bottie-bot/internal/bot"
	"github.com/mroctopus/bottie-bot/internal/player"
)

// -- INITIALIZER --

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
					if h.Player.ChannelID(*e.GuildID()) != nil {
						return e.Respond(discord.InteractionResponseTypeCreateMessage, *message(&discord.MessageUpdate{}, "No music is currently playing", MessageTypeDefault, discord.MessageFlagEphemeral))
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

func (h *MusicHandler) musicPlay(id snowflake.ID, tracks []lavalink.Track, e *handler.CommandEvent) error {
	client := e.Client()
	_, ok := client.Caches().VoiceState(*e.GuildID(), e.ApplicationID())
	if !ok {
		err := client.UpdateVoiceState(context.Background(), *e.GuildID(), &id, false, false)
		if err != nil {
			_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
			return err
		}
	}

	err := h.Player.Add(e.Ctx, *e.GuildID(), e.Channel().ID(), e.User(), tracks...)
	if err != nil {
		_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		return err
	}

	_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, fmt.Sprintf("Added %d songs to the queue", len(tracks)), MessageTypeDefault, 0))
	if err != nil {
		return err
	}

	return err
}

// NOTE:
// This isn't as easy as you would first expect.
func (h *MusicHandler) onPlay(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	client := e.Client()

	vs, ok := client.Caches().VoiceState(*e.GuildID(), e.User().ID)
	if !ok {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "You must be in a voice channel to queue music", MessageTypeDefault, discord.MessageFlagEphemeral))
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

	h.Player.Search(e.Ctx, *e.GuildID(), q, disgolink.NewResultHandler(
		func(track lavalink.Track) {
			err = h.musicPlay(*vs.ChannelID, []lavalink.Track{track}, e)
		},
		func(playlist lavalink.Playlist) {
			err = h.musicPlay(*vs.ChannelID, playlist.Tracks, e)
		},
		func(tracks []lavalink.Track) {
			err = h.musicPlay(*vs.ChannelID, []lavalink.Track{tracks[0]}, e)
		},
		func() {
			_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, fmt.Sprintf("No results found for %s", q), MessageTypeDefault, 0))
		},
		func(err error) {
			_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		},
	))

	return err
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
	return e.UpdateMessage(*message(&discord.MessageUpdate{}, "Stopped playing music", MessageTypeDefault, 0))
}
