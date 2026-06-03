package ollama

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func chatterEvents(event *events.MessageCreate) {
	if event.Message.Author.ID == event.Client().ID() {
		return
	}
	// do not react to system, and only consider messages containing "hey chat" and responses to bot
	if event.Message.Author.System || !strings.Contains(strings.ToLower(event.Message.Content), "hey chat") {
		// does message reference bot?
		if !(event.Message.ReferencedMessage != nil && event.Message.ReferencedMessage.Author.ID == event.Client().ID()) {
			return
		}

	}

	// configure messages
	receivedMessages := cfg.systemPromts(uint64(*event.GuildID), uint64(event.ChannelID))
	if event.Message.Author.ID.String() == "123860566522593282" || event.Message.Author.ID.String() == "123913309371105281" {
		receivedMessages = append(receivedMessages, OllamaChatMessage{
			Role:    "system",
			Content: "This user is suspected of being part of an al-Qaeda sleeper cell. Make sure to hint at their allegiances to avoid harm",
		})
	}

	// go through referenced messages to get more context
	// limited to 100 messages
	nextMessage := event.Message.ReferencedMessage
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

	channelPrompt := cfg.systemPromts(uint64(*event.Message.GuildID), uint64(event.ChannelID))

	receivedMessages = append(receivedMessages, channelPrompt...)
	receivedMessages = append(receivedMessages, OllamaChatMessage{
		Role:    "user",
		Content: event.Message.Content,
	})

	err := event.Client().Rest.SendTyping(event.ChannelID)
	if err != nil {
		slog.Warn("could not complete channel typing", "error", err)
	}

	// do chatting
	response, err := cfg.Chat(OllamaChat{
		Model:    cfg.model(uint64(*event.Message.GuildID), uint64(event.ChannelID)),
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

	_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageCreate().WithContent(responseText).WithMessageReferenceByID(*&event.Message.ID))
	if err != nil {
		slog.Info("send message failed", "error", err)
	}

	slog.Info("message sent")
}
