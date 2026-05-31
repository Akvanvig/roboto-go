package ollama

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func chatterEvents(event *events.MessageCreate) {
	// do not react to system, and only consider messages containing "hey chat"
	if event.Message.Author.System || !strings.Contains(strings.ToLower(event.Message.Content), "hey chat") {
		return
	}

	// configure messages
	receivedMessages := cfg.systemPromts(uint64(*event.GuildID), uint64(event.ChannelID))
	if event.Message.Author.ID.String() == "123860566522593282" || event.Message.Author.ID.String() == "123913309371105281" {
		receivedMessages = append(receivedMessages, OllamaChatMessage{
			Role:    "system",
			Content: "This user is suspected of being part of an al-Qaeda sleeper cell. Make sure to hint at their allegiances to avoid harm",
		})
	}
	if event.Message.ReferencedMessage != nil {
		refMessage := event.Message.ReferencedMessage
		if refMessage.Author.Bot {
			receivedMessages = append(receivedMessages, OllamaChatMessage{
				Role:    "assistant",
				Content: refMessage.Content,
			})
		} else {
			receivedMessages = append(receivedMessages, OllamaChatMessage{
				Role:    "system",
				Content: fmt.Sprintf("The user responded to the following message from '%s':\n %s", refMessage.Author.Username, refMessage.Content),
			})
		}
	}

	channelPrompt := cfg.systemPromts(uint64(*event.Message.GuildID), uint64(event.ChannelID))

	receivedMessages = append(receivedMessages, channelPrompt...)
	receivedMessages = append(receivedMessages, OllamaChatMessage{
		Role:    "user",
		Content: event.Message.Content,
	})

	context.WithTimeout(context.Background(), time.Second*20)
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

	_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageCreate().WithContent(responseText).WithMessageReference(event.Message.MessageReference))
	if err != nil {
		slog.Info("send message failed", "error", err)
	}

	slog.Info("message sent")
}
