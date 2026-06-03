package ollama

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func (o *Ollama) onMessageCreate(e *events.MessageCreate) {
	// do not react to system, and only consider messages containing "hey chat" and responses to bot
	if e.Message.Author.System || !strings.Contains(strings.ToLower(e.Message.Content), "hey chat") {
		// does message reference bot?
		if !(e.Message.ReferencedMessage != nil && e.Message.ReferencedMessage.Author.ID == e.Client().ID()) {
			return
		}
	}

	// configure messages
	receivedMessages := o.systemPromts(uint64(*e.GuildID), uint64(e.ChannelID))
	if e.Message.Author.ID.String() == "123860566522593282" || e.Message.Author.ID.String() == "123913309371105281" {
		receivedMessages = append(receivedMessages, OllamaChatMessage{
			Role:    "system",
			Content: "This user is suspected of being part of an al-Qaeda sleeper cell. Make sure to hint at their allegiances to avoid harm",
		})
	}

	// go through referenced messages to get more context
	// limited to 100 messages
	nextMessage := e.Message.ReferencedMessage
	for range 100 {
		// if no message is referenced, drop out of loop
		if nextMessage == nil {
			break
		}

		// tag roboto messages as assistant
		if nextMessage.Author.Bot {
			receivedMessages = append([]OllamaChatMessage{{
				Role:    "assistant",
				Content: nextMessage.Content,
			}}, receivedMessages...)
		} else { // otherwise tag with user and include displayname
			receivedMessages = append([]OllamaChatMessage{{
				Role:    "user",
				Content: fmt.Sprintf("'%s' says:\n%s", nextMessage.Author.EffectiveName(), nextMessage.Content),
			}}, receivedMessages...)
		}

		// set referenced message to next reference
		nextMessage = nextMessage.ReferencedMessage
	}

	channelPrompt := o.systemPromts(uint64(*e.Message.GuildID), uint64(e.ChannelID))

	receivedMessages = append(receivedMessages, channelPrompt...)
	receivedMessages = append(receivedMessages, OllamaChatMessage{
		Role:    "user",
		Content: e.Message.Content,
	})

	err := e.Client().Rest.SendTyping(e.ChannelID)
	if err != nil {
		slog.Warn("could not complete channel typing", "error", err)
	}

	// do chatting
	response, err := o.Chat(OllamaChat{
		Model:    o.model(uint64(*e.Message.GuildID), uint64(e.ChannelID)),
		Messages: receivedMessages,
		Options: OllamaChatOptions{
			Temperature: 1.5,
		},
		Stream: false,
	})
	responseText := response.Message.Content

	if err != nil {
		slog.Error("failed to chat", "error", err)
		responseText = "hey, chat is currently out touching grass 🌱\nthe AI backend isn't responding right now — try again in a bit."
	} else if responseText == "" {
		responseText = "hey, chat stared into the void and the void said nothing back."
	}

	_, err = e.Client().Rest.CreateMessage(e.ChannelID, discord.NewMessageCreate().WithContent(responseText).WithMessageReferenceByID(*&e.Message.ID))
	if err != nil {
		slog.Info("send message failed", "error", err)
	}

	slog.Info("message sent")
}
