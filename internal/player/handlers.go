package player

import (
	"context"
	"log/slog"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
	"github.com/disgoorg/lavaqueue-plugin"
)

func (p *Player) onVoiceServerUpdate(e *events.VoiceServerUpdate) {
	if e.Endpoint != nil {
		p.lavalink.OnVoiceServerUpdate(context.Background(), e.GuildID, e.Token, *e.Endpoint)
	}
}

func (p *Player) onGuildVoiceStateUpdate(e *events.GuildVoiceStateUpdate) {
	if e.VoiceState.UserID == e.Client().ApplicationID {
		p.lavalink.OnVoiceStateUpdate(context.Background(), e.VoiceState.GuildID, e.VoiceState.ChannelID, e.VoiceState.SessionID)
	}
}

func (p *Player) onTrackStart(lp disgolink.Player, e lavalink.TrackStartEvent) {
	ctx := context.Background()
	guildID := lp.GuildID()

	queue, _ := p.Queue(ctx, guildID)

	p.m.Lock()
	defer p.m.Unlock()

	channelID := p.playingChannels[guildID]
	msg, err := p.discord.Rest.CreateMessage(channelID, discord.MessageCreate{
		Embeds:     Embeds("Now playing", false, e.Track),
		Components: Components(len(queue) < 1),
	})

	if err == nil {
		p.playingMessages[channelID] = msg.ID
	}
}

func (p *Player) onTrackEnd(lp disgolink.Player, e lavalink.TrackEndEvent) {
	guildID := lp.GuildID()

	p.m.Lock()
	defer p.m.Unlock()

	channelID := p.playingChannels[guildID]
	messageID, ok := p.playingMessages[channelID]
	if !ok {
		p.logger.Warn("Failed to find the playing message", slog.Any("channel_id", channelID))
		return
	}

	err := p.discord.Rest.DeleteMessage(channelID, messageID)
	if err != nil {
		p.logger.Warn("Failed to delete playing message", slog.Any("channel_id", channelID), slog.Any("message_id", messageID))
	}
}

func (p *Player) onTrackException(lp disgolink.Player, e lavalink.TrackExceptionEvent) {
	// A suspicious exception indicates that youtube tried blocking us
	if e.Exception.Severity == lavalink.SeveritySuspicious {
		p.logger.Warn("Failed to play track", slog.String("track_name", e.Track.Info.Title), slog.Any("error", e.Exception))
	}
}

func (p *Player) onQueueEnd(lp disgolink.Player, e lavaqueue.QueueEndEvent) {
	go func() {
		time.Sleep(time.Second * 10)
		track := lp.Track()
		if track == nil {
			err := p.discord.UpdateVoiceState(context.Background(), e.GuildID(), nil, false, false)
			if err != nil {
				p.logger.Warn("Failed to update voice state", slog.Any("error", err))
			}
		}

	}()
}

func (p *Player) onWebSocketClosed(lp disgolink.Player, e lavalink.WebSocketClosedEvent) {
	p.m.Lock()
	defer p.m.Unlock()

	guildID := lp.GuildID()
	channelID := p.playingChannels[guildID]
	messageID := p.playingMessages[channelID]

	p.discord.Rest.DeleteMessage(channelID, messageID)

	delete(p.playingChannels, guildID)
	delete(p.playingMessages, channelID)
}
