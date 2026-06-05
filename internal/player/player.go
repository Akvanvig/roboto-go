package player

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/Akvanvig/roboto-go/internal/config"
	"github.com/disgoorg/json"
	"golang.org/x/sync/errgroup"

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
	logger          *slog.Logger
	cfg             *config.LavalinkConfig
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

// See https://github.com/CyberFlameGO/Lavalink-Client/tree/3ea412523817694cae8cc93ba2cc5f5c941f767c/src/main/java/lavalink/client/io/filters

type FilterType string

const (
	FilterTypeKaraoke FilterType = "karaoke"
	FilterTypeVibrato FilterType = "vibrato"
)

var FilterDefaultKaraoke = lavalink.Karaoke{
	Level:       1.0,
	MonoLevel:   1.0,
	FilterBand:  220.0,
	FilterWidth: 100.0,
}

var FilterDefaultVibrato = lavalink.Vibrato{
	Frequency: 2.0,
	Depth:     0.5,
}

func (p *Player) Filter(ctx context.Context, guildID snowflake.ID, filter FilterType) (bool, error) {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return false, fmt.Errorf("no active nodes")
	}

	filters := lp.Filters()
	enable := false
	switch filter {
	case FilterTypeKaraoke:
		enable = !(filters.Karaoke != nil && *filters.Karaoke != FilterDefaultKaraoke)
		if enable {
			filters.Karaoke = &lavalink.Karaoke{
				Level:       5.0,
				MonoLevel:   1.0,
				FilterBand:  220.0,
				FilterWidth: 100.0,
			}
		} else {
			filters.Karaoke = &FilterDefaultKaraoke
		}
	case FilterTypeVibrato:
		enable = !(filters.Vibrato != nil && *filters.Vibrato != FilterDefaultVibrato)
		if enable {
			filters.Vibrato = &lavalink.Vibrato{
				Frequency: 10.0,
				Depth:     1,
			}
		} else {
			filters.Vibrato = &FilterDefaultVibrato
		}
	default:
		return enable, fmt.Errorf("currently unsupported filter type: %s", filter)
	}

	return enable, lp.Update(ctx, lavalink.WithFilters(filters))
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
		// NOTE:
		// SoundCloud uses the wrong handler for empty search results
		func(tracks []lavalink.Track) {
			if len(tracks) > 0 {
				onResult(tracks[0])
			} else {
				onResult()
			}
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
		_, err = p.discord.Rest.UpdateMessage(channelID, messageID, discord.MessageUpdate{
			Components: new(Components(false)),
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

	p.m.Lock()
	defer p.m.Unlock()

	err := lavaqueue.ClearQueue(ctx, lp.Node(), guildID)
	if err != nil {
		return err
	}

	channelID := p.playingMessages[guildID]
	messageID := p.playingMessages[channelID]
	_, err = p.discord.Rest.UpdateMessage(channelID, messageID, discord.MessageUpdate{
		Components: new(Components(true)),
	})

	return err
}

func (p *Player) Skip(ctx context.Context, guildID snowflake.ID, count int) (*lavalink.Track, error) {
	lp := p.lavalink.Player(guildID)
	if lp == nil {
		return nil, fmt.Errorf("no active nodes")
	}

	track, err := lavaqueue.QueueNextTrack(ctx, lp.Node(), guildID, count)
	if err != nil {
		if lavaErr, ok := errors.AsType[lavalink.Error](err); ok {
			if lavaErr.Status == http.StatusNotFound {
				return nil, nil
			}
		}
		return nil, err
	}

	return track, nil
}

func (p *Player) Connect(ctx context.Context) error {
	node := p.lavalink.BestNode()
	if node != nil {
		status := node.Status()
		if status == disgolink.StatusConnecting || status == disgolink.StatusConnected || status == disgolink.StatusReconnecting {
			return fmt.Errorf("nodes are already active")
		}
	}

	var g errgroup.Group
	for _, cfg := range p.cfg.Nodes {
		g.Go(func() error {
			_, err := p.lavalink.AddNode(ctx, disgolink.NodeConfig{
				Name:     cfg.Name,
				Address:  cfg.Address,
				Password: cfg.Password,
				Secure:   cfg.Secure,
			})
			return err
		})
	}

	err := g.Wait()
	return err
}

func (p *Player) Disconnect() {
	p.lavalink.Close()

	p.m.Lock()
	defer p.m.Unlock()

	// NOTE:
	// We gracefully clean up sent messages to avoid user confusion.
	for channelID, messageID := range p.playingMessages {
		p.discord.Rest.DeleteMessage(channelID, messageID)
		delete(p.playingMessages, channelID)
	}

	for guildID := range p.playingChannels {
		delete(p.playingChannels, guildID)
	}
}

func New(discord bot.Client, cfg *config.LavalinkConfig) *Player {
	lavalink := disgolink.New(discord.ApplicationID,
		disgolink.WithPlugins(
			lavaqueue.New(),
		),
	)

	player := &Player{
		logger:          discord.Logger,
		cfg:             cfg,
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
