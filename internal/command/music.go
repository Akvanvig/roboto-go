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
			//r.SlashCommand("/skip", h.onMusicSkip)
			r.Component("/skip", h.onMusicSkipButton)
		})
	})

	return cmds
}

// -- COMPONENTS --

type TrackUserData struct {
	User        string    `json:"username"`
	UserIconURL string    `json:"icon_url"`
	Timestamp   time.Time `json:"timestamp"`
}

/*
func (txt string, track *lavalink.Track) {
	var data TrackUserData
	err := json.Unmarshal(track.UserData, &userInfo)
	if err != nil {

	}

	return []discord.Embed{
		{
			Author: &discord.EmbedAuthor{
				Name: txt,
				IconURL: "https://media.tenor.com/V0PyK4xovxAAAAAC/peepo-dance-pepe.gif",
			},
			Title: track.Info.Title,
			URL: *track.Info.URI,
			Thumbnail: &discord.EmbedResource{
				URL: *track.Info.ArtworkURL,
			},
			Fields: []discord.EmbedField{
				{
					Name: "Uploader",
					Value: track.Info.Author,
				},
				{
					Name: "Length",
					Value: track.Info.Author,
				},
			},
			Footer: &discord.EmbedFooter{
				Text: data.User,
				IconURL: data.UserIconURL,

			},
			Timestamp: &data.Timestamp,
			Color:       0,
		},
	},
}
*/

func player(txt string) discord.MessageCreate {
	return discord.MessageCreate{
		Embeds: []discord.Embed{
			{
				Description: txt,
				Color:       0,
			},
		},
		Components: []discord.ContainerComponent{discord.ActionRowComponent{
			discord.NewPrimaryButton("", "/player/pause_play").WithEmoji(discord.ComponentEmoji{Name: "⏯"}),
			discord.NewPrimaryButton("", "/player/skip").WithEmoji(discord.ComponentEmoji{Name: "⏭"}),
			discord.NewPrimaryButton("", "/player/stop").WithEmoji(discord.ComponentEmoji{Name: "⏹"}),
		}},
	}
}

func playerUpdate(txt string) discord.MessageUpdate {
	return discord.MessageUpdate{
		Embeds: &[]discord.Embed{
			{
				Description: txt,
				Color:       0,
			},
		},
		Components: &[]discord.ContainerComponent{discord.ActionRowComponent{
			discord.NewPrimaryButton("", "/player/pause_play").WithEmoji(discord.ComponentEmoji{Name: "⏯"}),
			discord.NewPrimaryButton("", "/player/skip").WithEmoji(discord.ComponentEmoji{Name: "⏭"}),
			discord.NewPrimaryButton("", "/player/stop").WithEmoji(discord.ComponentEmoji{Name: "⏹"}),
		}},
	}
}

// -- HANDLERS --

type MusicHandler struct {
	Lavalink disgolink.Client
}

// TEMPORARY
func FormatTrack(track lavalink.Track, position lavalink.Duration) string {
	var positionStr string
	if position > 0 {
		positionStr = fmt.Sprintf("`%s/%s`", FormatDuration(position), FormatDuration(track.Info.Length))
	} else {
		positionStr = fmt.Sprintf("`%s`", FormatDuration(track.Info.Length))
	}

	if track.Info.URI != nil {
		return fmt.Sprintf("[`%s`](<%s>) - `%s` %s", track.Info.Title, *track.Info.URI, track.Info.Author, positionStr)
	}
	return fmt.Sprintf("`%s` - `%s` %s`", track.Info.Title, track.Info.Author, positionStr)
}

func FormatDuration(duration lavalink.Duration) string {
	if duration == 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", duration.Minutes(), duration.SecondsPart())
}

func (h *MusicHandler) play(id snowflake.ID, tracks []lavalink.Track, e *handler.CommandEvent) error {
	discord := e.Client()
	_, ok := discord.Caches().VoiceState(*e.GuildID(), e.ApplicationID())

	if !ok {
		err := discord.UpdateVoiceState(context.Background(), *e.GuildID(), &id, false, false)
		if err != nil {
			_, err = e.UpdateInteractionResponse(errorMessageUpdate(err))
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
		_, err = e.UpdateInteractionResponse(errorMessageUpdate(err))
		return err
	}

	queueTracks := make([]lavaqueue.QueueTrack, len(tracks))
	for i := range tracks {
		track := tracks[i]
		queueTracks[i] = lavaqueue.QueueTrack{
			Encoded:  track.Encoded,
			UserData: userData,
		}
	}

	player := h.Lavalink.Player(*e.GuildID())
	track, err := lavaqueue.AddQueueTracks(e.Ctx, player.Node(), *e.GuildID(), queueTracks)
	if err != nil {
		_, err = e.UpdateInteractionResponse(errorMessageUpdate(err))
		return err
	}

	var content string
	numTracks := len(tracks)

	if track != nil {
		content = fmt.Sprintf("▶ Playing: %s", FormatTrack(*track, 0))
		numTracks -= 1
	}
	if numTracks > 0 {
		content += fmt.Sprintf("\nAdded %d songs to the queue", numTracks)
	}

	_, err = e.UpdateInteractionResponse(playerUpdate(content))
	return err
}

// NOTE:
// This isn't as easy as you would first expect.
func (h *MusicHandler) onMusicPlay(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	discord := e.Client()
	vs, ok := discord.Caches().VoiceState(*e.GuildID(), e.User().ID)
	if !ok {
		return e.CreateMessage(message("You must be in a voice channel to queue music"))
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

	err := e.DeferCreateMessage(true)
	if err != nil {
		return err
	}

	player := h.Lavalink.Player(*e.GuildID())
	player.Node().LoadTracksHandler(e.Ctx, q, disgolink.NewResultHandler(
		func(track lavalink.Track) {
			err = h.play(*vs.ChannelID, []lavalink.Track{track}, e)
		},
		func(playlist lavalink.Playlist) {
			err = h.play(*vs.ChannelID, playlist.Tracks, e)
		},
		func(tracks []lavalink.Track) {
			err = h.play(*vs.ChannelID, []lavalink.Track{tracks[0]}, e)
		},
		func() {
			_, err = e.UpdateInteractionResponse(messageUpdate(fmt.Sprintf("No results found for %s", q)))
		},
		func(err error) {
			_, err = e.UpdateInteractionResponse(errorMessageUpdate(err))
		},
	))

	return err
}

func (h *MusicHandler) onMusicSkip(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	return nil
}

func (h *MusicHandler) onMusicSkipButton(e *handler.ComponentEvent) error {
	/*
		player := h.Lavalink.Player(*e.GuildID())
		track, err := lavaqueue.QueueNextTrack(e.Ctx, player.Node(), *e.GuildID())
		if err != nil {
			var eErr *lavalink.Error
			if errors.As(err, &eErr) && eErr.Status == http.StatusNotFound {
				return e.CreateMessage(res.CreateError("No more songs in queue"))
			}
			return e.CreateMessage(res.CreateErr("Failed to skip to the next song", err))
		}

		return e.UpdateMessage(res.UpdatePlayerf("▶ Playing: %s", true, res.FormatTrack(*track, 0)))
	*/
	return nil
}

func (h *MusicHandler) onMusicStopButton(data discord.SlashCommandInteractionData, e *handler.CommandEvent) error {
	return nil
}
