package events

import (
	"github.com/disgoorg/disgo/bot"
)

var EventListeners = []bot.EventListener{
	bot.NewListenerFunc(chatterEvents),
}
