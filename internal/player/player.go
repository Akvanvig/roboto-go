package player

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavaqueue-plugin"
	"github.com/disgoorg/snowflake/v2"
)

type TrackUserData struct {
	User        string    `json:"username"`
	UserIconURL string    `json:"icon_url"`
	Timestamp   time.Time `json:"timestamp"`
}

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

func Message[T *discord.MessageCreate | *discord.MessageUpdate](dst T, txt string, track lavalink.Track, buttons bool) T {

	var embeds []discord.Embed
	{
		var data TrackUserData
		err := json.Unmarshal(track.UserData, &data)
		if err != nil {
			// TODO
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
			//discord.NewPrimaryButton("", "/music/pause_play").WithEmoji(discord.ComponentEmoji{Name: "⏯"}),
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

func AddedToQueue() {

}

type Player struct {
	Discord         bot.Client
	Lavalink        disgolink.Client
	PlayingChannels map[snowflake.ID]snowflake.ID
	PlayingMessages map[string]snowflake.ID
	// NOTE:
	// This mutex is currently global, but it should be per guild
	m sync.Mutex
}

func (p *Player) ChannelID(guildID snowflake.ID) *snowflake.ID {
	p.m.Lock()
	defer p.m.Unlock()

	channelID, ok := p.PlayingChannels[guildID]
	if !ok {
		return nil
	}
	return &channelID
}

// TODO:
// This can probably be waaaay improved
type SearchResultHandler func(tracks ...lavalink.Track)
type SearchResultErrorHandler func(err error)

func (p *Player) Search(ctx context.Context, guildID snowflake.ID, query string, onResult SearchResultHandler, onError SearchResultErrorHandler) {
	lp := p.Lavalink.Player(guildID)
	lp.Node().LoadTracksHandler(ctx, query, disgolink.NewResultHandler(
		func(track lavalink.Track) {
			onResult(track)
		},
		func(playlist lavalink.Playlist) {
			onResult(playlist.Tracks...)
		},
		func(tracks []lavalink.Track) {
			onResult(tracks[0])
		},
		func() {
			onResult()
		},
		func(err error) {
			onError(err)
		},
	))
}

func (p *Player) Add(ctx context.Context, guildID snowflake.ID, channelID snowflake.ID, user discord.User, tracks ...lavalink.Track) error {
	lp := p.Lavalink.Player(guildID)

	data, err := json.Marshal(TrackUserData{
		User:        user.Username,
		UserIconURL: *user.AvatarURL(),
		Timestamp:   time.Now(),
	})
	if err != nil {
		return err
	}

	queued := make([]lavaqueue.QueueTrack, len(tracks))
	for i := range tracks {
		track := tracks[i]
		queued[i] = lavaqueue.QueueTrack{
			Encoded:  track.Encoded,
			UserData: data,
		}
	}

	p.m.Lock()
	defer p.m.Unlock()

	track, err := lavaqueue.AddQueueTracks(ctx, lp.Node(), guildID, queued)
	if err != nil {
		return err
	}

	if track != nil {
		p.PlayingChannels[guildID] = channelID
	}

	return nil
}

func (p *Player) Next(ctx context.Context, guildID snowflake.ID) (*lavalink.Track, error) {
	lp := p.Lavalink.Player(guildID)

	track, err := lavaqueue.QueueNextTrack(ctx, lp.Node(), guildID)
	if err != nil {
		// NOTE:
		// Currently, lavalink.Error does not implement an unwrap interface,
		// which in turn means that we can't use errors.As to unwrap
		// and check for the http.StatusNotFound error code in the original error.
		// Instead we just do a straight-up string comparison (stupid, yes)
		if err.Error() == "No next track found" {
			return nil, nil
		}
		return nil, err
	}

	return track, nil
}

func (p *Player) Stop(ctx context.Context, guildID snowflake.ID) error {
	lp := p.Lavalink.Player(guildID)
	return lp.Destroy(ctx)
}

func New(discord bot.Client, lavalink disgolink.Client) *Player {
	player := &Player{
		Discord:         discord,
		Lavalink:        lavalink,
		PlayingChannels: make(map[snowflake.ID]snowflake.ID),
		PlayingMessages: make(map[string]snowflake.ID),
	}

	discord.AddEventListeners(bot.NewListenerFunc(player.OnDiscordEvent))
	lavalink.AddListeners(disgolink.NewListenerFunc(player.OnLavalinkEvent))

	return player
}
