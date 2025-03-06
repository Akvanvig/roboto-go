package bot

import (
	"context"

	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/disgo/events"
	"github.com/disgoorg/disgolink/v3/disgolink"
	"github.com/disgoorg/disgolink/v3/lavalink"
)

func (b *RobotoBot) OnDiscordEvent(event bot.Event) {
	switch e := event.(type) {
	case *events.VoiceServerUpdate:
		if e.Endpoint == nil {
			return
		}
		b.Lavalink.OnVoiceServerUpdate(context.Background(), e.GuildID, e.Token, *e.Endpoint)
	case *events.GuildVoiceStateUpdate:
		if e.VoiceState.UserID != e.Client().ApplicationID() {
			return
		}
		b.Lavalink.OnVoiceStateUpdate(context.Background(), e.VoiceState.GuildID, e.VoiceState.ChannelID, e.VoiceState.SessionID)
	}
}

func (b *RobotoBot) OnLavalinkEvent(p disgolink.Player, event lavalink.Event) {
	// player := b.Lavalink.Player(p.GuildID())
	/*
		switch e := event.(type) {
		case lavaqueue.QueueEndEvent:
			//slog.Info("queue end", slog.String("guild", p.GuildID().String()))

		case lavalink.TrackStartEvent:

		case lavalink.TrackExceptionEvent:
			//slog.Error("track exception", tint.Err(e.Exception))

		case lavalink.TrackStuckEvent:

		}*/
}
