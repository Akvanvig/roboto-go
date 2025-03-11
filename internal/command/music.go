package command

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/handler"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavaqueue-plugin"
	"github.com/disgoorg/snowflake/v2"
	"github.com/mroctopus/bottie-bot/internal/bot"
	"github.com/mroctopus/bottie-bot/internal/player"
)

// -- INITIALIZER --

// SEE https://github.com/KittyBot-Org/KittyBotGo/blob/master/service/bot/commands/player.go
func musicCommands(bot *bot.RobotoBot, r *handler.Mux) discord.ApplicationCommandCreate {
	if bot.Lavalink == nil {
		return nil
	}

	cmds := discord.SlashCommandCreate{
		Name:        "music",
		Description: "Shows the music ??.",
		Options: []discord.ApplicationCommandOption{
			discord.ApplicationCommandOptionSubCommand{
				Name:        "connect",
				Description: "Connect to a voice channel",
				Options: []discord.ApplicationCommandOption{
					discord.ApplicationCommandOptionChannel{
						Name:        "channel",
						Description: "The voice channel",
						ChannelTypes: []discord.ChannelType{
							discord.ChannelTypeGuildVoice,
						},
						Required: true,
					},
				},
			},
			discord.ApplicationCommandOptionSubCommand{
				Name:        "disconnect",
				Description: "Disconnect the bot from its current voice channel",
			},
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
		Lavalink: bot.Lavalink,
		Messages: bot.LavalinkTrackMessages,
	}
	r.Route("/music", func(r handler.Router) {
		r.SlashCommand("/connect", h.onConnect)
		r.SlashCommand("/disconnect", h.onDisconnect)
		r.SlashCommand("/play", h.onPlay)
		r.Group(func(r handler.Router) {
			// Middleware to check for existence of player
			r.Use(func(next handler.Handler) handler.Handler {
				return func(e *handler.InteractionEvent) error {
					player := h.Lavalink.ExistingPlayer(*e.GuildID())
					if player == nil {
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
	Lavalink disgolink.Client
	Messages map[snowflake.ID]player.TrackMessageData
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

	// NOTE:
	// We must change the snowflake ID to have a specific node
	// if we are going to be using multiple deployments of the bot.
	user := e.User()
	time := time.Now()
	data := player.TrackUserData{
		ID:          snowflake.New(time),
		User:        user.Username,
		UserIconURL: *user.AvatarURL(),
		Timestamp:   time,
	}
	dataEnc, err := json.Marshal(data)
	if err != nil {
		_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		return err
	}

	p := h.Lavalink.Player(*e.GuildID())
	queueTracks := make([]lavaqueue.QueueTrack, len(tracks))
	for i := range tracks {
		track := tracks[i]
		queueTracks[i] = lavaqueue.QueueTrack{
			Encoded:  track.Encoded,
			UserData: dataEnc,
		}
	}

	track, err := lavaqueue.AddQueueTracks(e.Ctx, p.Node(), *e.GuildID(), queueTracks)
	if err != nil {
		_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		return err
	}

	_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, fmt.Sprintf("Added %d songs to the queue", len(tracks)), MessageTypeDefault, 0))
	if err != nil {
		return err
	}

	if track != nil {
		msg, err := e.CreateFollowupMessage(*player.Message(&discord.MessageCreate{}, "Now playing", *track, true))
		if err == nil {
			h.Messages[data.ID] = player.TrackMessageData{
				ChannelID:        msg.ChannelID,
				AppID:            *msg.ApplicationID,
				InteractionToken: e.Token(),
				MessageID:        msg.ID,
			}
		}
	}

	return err
}

func (h *MusicHandler) onConnect(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	client := e.Client()
	channel, _ := data.OptChannel("channel")

	vs, ok := client.Caches().VoiceState(*e.GuildID(), e.User().ID)
	if ok {
		if vs.ChannelID == &channel.ID {
			return e.CreateMessage(*message(&discord.MessageCreate{}, "You can't connect to a voice channel the bot is already in", MessageTypeDefault, discord.MessageFlagEphemeral))
		} else {
			client.UpdateVoiceState(e.Ctx, vs.GuildID, nil, false, false)
		}
	}

	err := client.UpdateVoiceState(e.Ctx, *e.GuildID(), &channel.ID, false, false)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to connect bot to the voice channel", MessageTypeError, 0))
	}

	return e.CreateMessage(*message(&discord.MessageCreate{}, "Connect bot to voice channel", MessageTypeDefault, 0))
}

func (h *MusicHandler) onDisconnect(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	client := e.Client()

	_, ok := client.Caches().VoiceState(*e.GuildID(), e.User().ID)
	if !ok {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "You can't disconnect the bot when it's not in a voice channel", MessageTypeDefault, discord.MessageFlagEphemeral))
	}

	p := h.Lavalink.ExistingPlayer(*e.GuildID())
	if p != nil {
		err := p.Destroy(e.Ctx)
		if err != nil {
			return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to stop the music player", MessageTypeError, 0))
		}
	}

	err := client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, false)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to disconnect bot from the voice channel", MessageTypeError, 0))
	}

	if p != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Stopped playing music", MessageTypeDefault, 0))
	}

	return nil
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

	p := h.Lavalink.Player(*e.GuildID())
	p.Node().LoadTracksHandler(e.Ctx, q, disgolink.NewResultHandler(
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
	p := h.Lavalink.Player(*e.GuildID())
	track, err := lavaqueue.QueueNextTrack(e.Ctx, p.Node(), *e.GuildID())

	if err != nil {
		// NOTE:
		// Currently, lavalink.Error does not implement an unwrap interface,
		// which in turn means that we can't use errors.As to unwrap
		// and check for the http.StatusNotFound error code in the original error.
		// Instead we just do a straight-up string comparison (stupid, yes)
		if err.Error() == "No next track found" {
			return e.CreateMessage(*message(&discord.MessageCreate{}, "No more songs in the queue", MessageTypeDefault, 0))
		}
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to skip the current song", MessageTypeError, 0))
	}

	return e.UpdateMessage(*player.Message(&discord.MessageUpdate{}, "Now playing", *track, true))
}

func (h *MusicHandler) onStopButton(e *handler.ComponentEvent) error {
	client := e.Client()
	p := h.Lavalink.Player(*e.GuildID())
	err := p.Destroy(e.Ctx)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to stop the music player", MessageTypeError, 0))
	}

	err = client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, false)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to disconnect bot from the voice channel", MessageTypeError, 0))
	}

	return e.UpdateMessage(*message(&discord.MessageUpdate{}, "Stopped playing music", MessageTypeDefault, 0))
}
