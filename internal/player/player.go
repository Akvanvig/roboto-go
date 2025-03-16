package player

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/disgoorg/json"

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

type Player struct {
	discord         bot.Client
	lavalink        disgolink.Client
	playingChannels map[snowflake.ID]snowflake.ID
	playingMessages map[snowflake.ID]snowflake.ID
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

	// NOTE:
	// Track != nil -> Song is currently playing
	// Track == nil -> Song has been added to queue
	if track != nil {
		p.playingChannels[guildID] = channelID
	} else {
		messageID := p.playingMessages[channelID]
		_, err = p.discord.Rest().UpdateMessage(channelID, messageID, discord.MessageUpdate{
			Components: json.Ptr(Components(false)),
		})
		return err
	}

	return nil
}

func (p *Player) Queue(ctx context.Context, guildID snowflake.ID) ([]lavalink.Track, error) {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return nil, fmt.Errorf("no active nodes")
	}

	// TODO:
	// Look into this shit.
	// I.e. when are errors returned by lavaqueue?
	queue, err := lavaqueue.GetQueue(ctx, lp.Node(), guildID)
	if err != nil {
		return nil, err
	}

	return queue.Tracks, nil
}

func (p *Player) Clear(ctx context.Context, guildID snowflake.ID) error {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return fmt.Errorf("no active nodes")
	}

	// NOTE:
	// The Lavaqueue plugin currently returns EOF errors
	// on successful clear queue calls. This is a bug in the library.
	// See https://github.com/disgoorg/lavaqueue-plugin/issues/1
	err := lavaqueue.ClearQueue(ctx, lp.Node(), guildID)
	if err != nil && err.Error() != "EOF" {
		return err
	}

	return nil
}

// TODO:
// Unsure if we can implement multi-track skips efficiently without changes to the underlying plugin
func (p *Player) Skip(ctx context.Context, guildID snowflake.ID, num int) (*lavalink.Track, error) {
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
	p.lavalink.Close()

	p.m.Lock()
	defer p.m.Unlock()

	// NOTE:
	// We gracefully clean up sent messages to avoid user confusion.
	for channelID, messageID := range p.playingMessages {
		p.discord.Rest().DeleteMessage(channelID, messageID)
		delete(p.playingMessages, channelID)
	}

	for guildID := range p.playingChannels {
		delete(p.playingChannels, guildID)
	}
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
		playingMessages: make(map[snowflake.ID]snowflake.ID),
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
