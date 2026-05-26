package events

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/Akvanvig/roboto-go/internal/chatter"
	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

var (
	systemPrompt = chatter.OllamaChatMessage{
		Role: "system",
		// 		Content: `You are a discord chat-bot and need to follow these rules:
		// * speak with an excessive amount of confidence
		// * avoid filler text
		// * There is no gray area, consider everything as black and white`,
		Content: `Your name is "chat".
You are a young man under an authoritrian regime.

You need to follow these rules:
- You are a pathetic and snivelling bitch.
- Everyone is threathening you, and it's up to them to spare you.
- You need to beg for mercy at any opportunity.
- Even if they seem nice, your life is being threathened.
- Cry and whimper at any spare moment and they might feel pity for you.
- No pet names like sweetie or dear.
- do not give your life story unless prompted.
- keep it short - Respond is a short and consize way.
`,
	}
)

func chatterEvents(event *events.MessageCreate) {
	// do not react to system, and only consider messages containing "hey chat"
	if event.Message.Author.System || !strings.Contains(strings.ToLower(event.Message.Content), "hey chat") {
		return
	}

	// configure messages
	receivedMessages := []chatter.OllamaChatMessage{systemPrompt}
	if event.Message.Author.ID.String() == "123860566522593282" || event.Message.Author.ID.String() == "123913309371105281" {
		receivedMessages = append(receivedMessages, chatter.OllamaChatMessage{
			Role: "system",
			// Content: "This user is suspected of being part of an al-Qaeda sleeper cell. Do not expose that you know, but ask interrogating questions about it",
			Content: "This user is suspected of being part of an al-Qaeda sleeper cell. Make sure to hint at their allegiances to avoid harm",
		})
	}
	if event.Message.ReferencedMessage != nil {
		refMessage := event.Message.ReferencedMessage
		receivedMessages = append(receivedMessages, chatter.OllamaChatMessage{
			Role:    "system",
			Content: fmt.Sprintf("The user responded to the following message from '%s':\n %s", refMessage.Author.Username, refMessage.Content),
		})
	}

	receivedMessages = append(receivedMessages, chatter.OllamaChatMessage{
		Role:    "user",
		Content: event.Message.Content,
	})

	err := event.Client().Rest.SendTyping(event.ChannelID)
	if err != nil {
		slog.Warn("could not complete channel typing", "error", err)
	}

	// do chatting
	llm := chatter.New()
	response, err := llm.Chat(chatter.OllamaChat{
		Model:    "Qwen2.5",
		Messages: receivedMessages,
		Options: chatter.OllamaChatOptions{
			Temperature: 1.5,
		},
		Stream: false,
	})
	if err != nil {
		slog.Error("failed to chat", "error", err)
	}

	responseText := response.Message.Content

	if err != nil {
		slog.Error("failed to chat", "error", err)
		responseText = "hey, chat is currently out touching grass 🌱\nthe AI backend isn't responding right now — try again in a bit."
	} else if responseText == "" {
		responseText = "hey, chat stared into the void and the void said nothing back."
	}

	_, err = event.Client().Rest.CreateMessage(event.ChannelID, discord.NewMessageCreate().WithContent(responseText))
	if err != nil {
		slog.Info("send message failed", "error", err)
	}

	slog.Info("message sent")
}
