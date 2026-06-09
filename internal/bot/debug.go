package bot

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/disgoorg/disgo/discord"
)

// A LevelHandler wraps a Handler with an Enabled method
// that returns false for levels below a minimum.
type DiscordDebugHandler struct {
	roboto    *RobotoBot
	debugging bool
	handler   slog.Handler
}

func NewDiscordDebugHandler(roboto *RobotoBot, h slog.Handler) *DiscordDebugHandler {
	// Optimization: avoid chains of NewDebugHandler.
	if dh, ok := h.(*DiscordDebugHandler); ok {
		h = dh.Handler()
	}
	return &DiscordDebugHandler{
		roboto:  roboto,
		handler: h,
	}
}

func (h *DiscordDebugHandler) Toggle() bool {
	h.debugging = !h.debugging
	return h.debugging
}

func (h *DiscordDebugHandler) Enabled(ctx context.Context, level slog.Level) bool {
	return h.handler.Enabled(ctx, level)
}

func (h *DiscordDebugHandler) Handle(ctx context.Context, r slog.Record) error {
	err := h.handler.Handle(ctx, r)
	if h.debugging && err != nil {
		channelID := h.roboto.Config.Discord.DebugChannelID
		if channelID > 0 {
			msg := fmt.Sprintf("%s %s %s", r.Time.UTC(), r.Level.String(), r.Message)
			_, err = h.roboto.Discord.Rest.CreateMessage(h.roboto.Config.Discord.DebugChannelID, discord.NewMessageCreate().WithContent(msg))
		}
	}
	return err
}

func (h *DiscordDebugHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return NewDiscordDebugHandler(h.roboto, h.handler.WithAttrs(attrs))
}

func (h *DiscordDebugHandler) WithGroup(name string) slog.Handler {
	return NewDiscordDebugHandler(h.roboto, h.handler.WithGroup(name))
}

func (h *DiscordDebugHandler) Handler() slog.Handler {
	return h
}
