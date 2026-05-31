package ollama

import (
	"log/slog"

	"github.com/disgoorg/disgo/bot"
)

var (
	EventListeners = []bot.EventListener{
		bot.NewListenerFunc(chatterEvents),
	}
	cfg = Ollama{}
)

func New(config Ollama) []bot.EventListener {
	cfg = config
	if config.Server == "" {
		slog.Info("ollama integrations disabled")
		return []bot.EventListener{}
	}

	slog.Info("ollama integrations enabled")
	return EventListeners
}
