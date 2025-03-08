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
)

// -- INITIALIZER --

// SEE https://github.com/KittyBot-Org/KittyBotGo/blob/master/service/bot/commands/player.go
func music(bot *bot.RobotoBot, r *handler.Mux) discord.ApplicationCommandCreate {
	if bot.Lavalink == nil {
		return nil
	}

	cmds := discord.SlashCommandCreate{
		Name:        "music",
		Description: "Shows the music ??.",
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
		Lavalink: bot.Lavalink,
	}
	r.Route("/music", func(r handler.Router) {
		r.SlashCommand("/play", h.onMusicPlay)
		r.Group(func(r handler.Router) {
			// Middleware to check for existence of player
			r.Use(func(next handler.Handler) handler.Handler {
				return func(e *handler.InteractionEvent) error {
					player := h.Lavalink.ExistingPlayer(*e.GuildID())
					if player == nil {

						return fmt.Errorf("LOOOL")
						//return e.Respond(discord.InteractionResponseTypeCreateMessage, res.CreateError("No player found"))
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

			r.Component("/skip", h.onMusicSkipButton)
			r.Component("/stop", h.onMusicStopButton)
		})
	})

	return cmds
}

// -- COMPONENTS --

func fmtDuration(duration lavalink.Duration) string {
	if duration == 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", duration.Minutes(), duration.SecondsPart())
}

func fmtTrack(track lavalink.Track, pos lavalink.Duration) string {
	var txt string
	if pos > 0 {
		txt = fmt.Sprintf("`%s/%s`", fmtDuration(pos), fmtDuration(track.Info.Length))
	} else {
		txt = fmt.Sprintf("`%s`", fmtDuration(track.Info.Length))
	}

	return txt
}

type TrackUserData struct {
	User        string    `json:"username"`
	UserIconURL string    `json:"icon_url"`
	Timestamp   time.Time `json:"timestamp"`
}

func playerMessage[T *discord.MessageCreate | *discord.MessageUpdate](dst T, txt string, track lavalink.Track, buttons bool) T {

	var embeds []discord.Embed
	{
		var data TrackUserData
		err := json.Unmarshal(track.UserData, &data)
		if err != nil {

		}

		var url string
		if track.Info.URI != nil {
			url = *track.Info.URI
		}

		var thumbnail *discord.EmbedResource
		if track.Info.ArtworkURL != nil {
			thumbnail = &discord.EmbedResource{
				URL: *track.Info.ArtworkURL,
			}
		}

		embeds = []discord.Embed{
			{
				Author: &discord.EmbedAuthor{
					Name:    txt,
					IconURL: "https://media.tenor.com/V0PyK4xovxAAAAAC/peepo-dance-pepe.gif",
				},
				Title:     track.Info.Title,
				URL:       url,
				Thumbnail: thumbnail,
				Fields: []discord.EmbedField{
					{
						Name:  "Uploader",
						Value: track.Info.Author,
					},
					{
						Name:  "Length",
						Value: fmtTrack(track, 0),
					},
				},
				Footer: &discord.EmbedFooter{
					Text:    data.User,
					IconURL: data.UserIconURL,
				},
				Timestamp: &data.Timestamp,
				Color:     0,
			},
		}
	}

	var components []discord.ContainerComponent
	if buttons {
		components = []discord.ContainerComponent{discord.ActionRowComponent{
			discord.NewPrimaryButton("", "/music/pause_play").WithEmoji(discord.ComponentEmoji{Name: "⏯"}),
			discord.NewPrimaryButton("", "/music/skip").WithEmoji(discord.ComponentEmoji{Name: "⏭"}),
			discord.NewPrimaryButton("", "/music/stop").WithEmoji(discord.ComponentEmoji{Name: "⏹"}),
		}}
	}

	switch t := any(dst).(type) {
	case *discord.MessageCreate:
		t.Embeds = embeds
		t.Components = components

	case *discord.MessageUpdate:
		t.Embeds = &embeds
		t.Components = &components
	}

	return dst
}

// -- HANDLERS --

type MusicHandler struct {
	Lavalink disgolink.Client
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

	user := e.User()
	userData, err := json.Marshal(TrackUserData{
		User:        user.Username,
		UserIconURL: *user.AvatarURL(),
		Timestamp:   time.Now(),
	})
	if err != nil {
		_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		return err
	}

	player := h.Lavalink.Player(*e.GuildID())
	queueTracks := make([]lavaqueue.QueueTrack, len(tracks))
	for i := range tracks {
		track := tracks[i]
		queueTracks[i] = lavaqueue.QueueTrack{
			Encoded:  track.Encoded,
			UserData: userData,
		}
	}

	track, err := lavaqueue.AddQueueTracks(e.Ctx, player.Node(), *e.GuildID(), queueTracks)
	if err != nil {
		_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, err.Error(), MessageTypeError, 0))
		return err
	}

	_, err = e.UpdateInteractionResponse(*message(&discord.MessageUpdate{}, fmt.Sprintf("Added %d songs to the queue", len(tracks)), MessageTypeDefault, 0))
	if err != nil {
		return err
	}

	// NOTE:
	// The AddQueueTracks method is very dumb. It will *always* return a valid pointer,
	// but the unmarshalled data will be missing when a song is added to a non-empty queue.
	// I.e. we have to check a random field afterwards to determine if the track has actually
	// been returned, instead of comparing aginst nil.
	if track != nil && track.Encoded != "" {
		_, err = e.CreateFollowupMessage(*playerMessage(&discord.MessageCreate{}, "Now playing", *track, true))
	}

	return err
}

// NOTE:
// This isn't as easy as you would first expect.
func (h *MusicHandler) onMusicPlay(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
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

	player := h.Lavalink.Player(*e.GuildID())
	player.Node().LoadTracksHandler(e.Ctx, q, disgolink.NewResultHandler(
		func(track lavalink.Track) {
			err = h.musicPlay(*vs.ChannelID, []lavalink.Track{track}, e)
		},
		func(playlist lavalink.Playlist) {
			err = h.musicPlay(*vs.ChannelID, playlist.Tracks, e)
		},
		// Only play first track in search
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

func (h *MusicHandler) onMusicSkipButton(e *handler.ComponentEvent) error {
	player := h.Lavalink.Player(*e.GuildID())
	track, err := lavaqueue.QueueNextTrack(e.Ctx, player.Node(), *e.GuildID())

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

	return e.UpdateMessage(*playerMessage(&discord.MessageUpdate{}, "Now playing", *track, true))
}

func (h *MusicHandler) onMusicStopButton(e *handler.ComponentEvent) error {
	client := e.Client()
	player := h.Lavalink.Player(*e.GuildID())
	err := player.Destroy(e.Ctx)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to stop the music player", MessageTypeError, 0))
	}

	err = client.UpdateVoiceState(e.Ctx, *e.GuildID(), nil, false, false)
	if err != nil {
		return e.CreateMessage(*message(&discord.MessageCreate{}, "Failed to disconnect bot from the voice channel", MessageTypeError, 0))
	}

	return e.UpdateMessage(*message(&discord.MessageUpdate{}, "Stopped playing music", MessageTypeDefault, 0))
}
