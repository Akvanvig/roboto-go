package player

import (
	"context"
	"encoding/json"
	"errors"
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

func FormatDuration(duration lavalink.Duration) string {
	if duration == 0 {
		return "00:00"
	}
	return fmt.Sprintf("%02d:%02d", duration.Minutes(), duration.SecondsPart())
}

func FormatTrack(track lavalink.Track, pos lavalink.Duration) string {
	var txt string
	if pos > 0 {
		txt = fmt.Sprintf("`%s/%s`", FormatDuration(pos), FormatDuration(track.Info.Length))
	} else {
		txt = fmt.Sprintf("`%s`", FormatDuration(track.Info.Length))
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
						Value: FormatTrack(track, 0),
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
			discord.NewPrimaryButton("", "/music/queue").WithEmoji(discord.ComponentEmoji{Name: "📜"}),
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

type Player struct {
	discord         bot.Client
	lavalink        disgolink.Client
	playingChannels map[snowflake.ID]snowflake.ID
	playingMessages map[string]snowflake.ID
	// NOTE:
	// This mutex is currently global, but it should be per guild
	m sync.Mutex
}

func (p *Player) ChannelID(guildID snowflake.ID) *snowflake.ID {
	p.m.Lock()
	defer p.m.Unlock()

	channelID, ok := p.playingChannels[guildID]
	if !ok {
		return nil
	}
	return &channelID
}

func (p *Player) Volume(ctx context.Context, guildID snowflake.ID, volume int) error {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return fmt.Errorf("no active nodes")
	}

	return lp.Update(ctx, lavalink.WithVolume(volume))
}

// TODO:
// This can probably be waaaay improved
type SearchResultHandler func(tracks ...lavalink.Track)
type SearchResultErrorHandler func(err error)

func (p *Player) Search(ctx context.Context, guildID snowflake.ID, query string, onResult SearchResultHandler, onError SearchResultErrorHandler) error {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return fmt.Errorf("no active nodes")
	}

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

	return nil
}

func (p *Player) Add(ctx context.Context, guildID snowflake.ID, channelID snowflake.ID, user discord.User, tracks ...lavalink.Track) error {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return fmt.Errorf("no active nodes")
	}

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
		p.playingChannels[guildID] = channelID
	} else {
		// TODO:
		// We should disable and enable the buttons depending on the queue state
		/*id := p.playingMessages[guildID]
		p.discord.Rest().UpdateMessage(channelID, id, discord.MessageUpdate{
			Components: &[]discord.ContainerComponent{discord.ActionRowComponent{
				discord.NewPrimaryButton("", "/music/skip").WithEmoji(discord.ComponentEmoji{Name: "⏭"}),
				discord.NewPrimaryButton("", "/music/stop").WithEmoji(discord.ComponentEmoji{Name: "⏹"}),
				discord.NewPrimaryButton("", "/music/queue").WithEmoji(discord.ComponentEmoji{Name: "📜"}),
			}},
		})*/
	}

	return nil
}

func (p *Player) Queue(ctx context.Context, guildID snowflake.ID) ([]lavalink.Track, error) {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return nil, fmt.Errorf("no active nodes")
	}

	// TODO:
	// Look into this shit
	queue, err := lavaqueue.GetQueue(ctx, lp.Node(), guildID)
	if err != nil {
		return nil, err
	}

	return queue.Tracks, nil
}

func (p *Player) Next(ctx context.Context, guildID snowflake.ID) (*lavalink.Track, error) {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return nil, fmt.Errorf("no active nodes")
	}

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

func (p *Player) AddNodes(ctx context.Context, configs ...disgolink.NodeConfig) ([]disgolink.Node, error) {
	var (
		errs  error
		nodes []disgolink.Node
		wg    sync.WaitGroup
		m     sync.Mutex
	)

	for i := range configs {
		cfg := configs[i]
		wg.Add(1)
		go func() {
			defer wg.Done()

			node, err := p.lavalink.AddNode(ctx, disgolink.NodeConfig{
				Name:     cfg.Name,
				Address:  cfg.Address,
				Password: cfg.Password,
				Secure:   cfg.Secure,
			})

			m.Lock()
			if node != nil {
				nodes = append(nodes, node)
			}
			errs = errors.Join(errs, err)
			m.Unlock()
		}()
	}

	wg.Wait()
	return nodes, errs
}

func (p *Player) Close() {
	//p.m.Lock()
	//defer p.m.Unlock()

	// TODO:
	// Investigate if this causes our handlers to be fucked?
	p.lavalink.Close()
}

func New(discord bot.Client) *Player {
	lavalink := disgolink.New(discord.ApplicationID(),
		disgolink.WithPlugins(
			lavaqueue.New(),
		),
	)

	player := &Player{
		discord:         discord,
		lavalink:        lavalink,
		playingChannels: make(map[snowflake.ID]snowflake.ID),
		playingMessages: make(map[string]snowflake.ID),
	}

	discord.AddEventListeners(
		bot.NewListenerFunc(player.onVoiceServerUpdate),
		bot.NewListenerFunc(player.onGuildVoiceStateUpdate),
	)
	lavalink.AddListeners(
		disgolink.NewListenerFunc(player.onTrackStart),
		disgolink.NewListenerFunc(player.onTrackEnd),
		disgolink.NewListenerFunc(player.onTrackException),
		disgolink.NewListenerFunc(player.onQueueEnd),
		disgolink.NewListenerFunc(player.onWebSocketClosed),
	)

	return player
}
