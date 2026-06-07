package ollama

import (
	"fmt"
	"log/slog"
	"slices"
	"strings"

	"github.com/disgoorg/disgo/discord"
	"github.com/disgoorg/disgo/events"
)

func (o *Ollama) onMessageCreate(e *events.MessageCreate) {
	if e.Message.Author.System || e.Message.Author.ID == e.Client().ID() {
		return
	}

	// Only consider messages containing "hey chat" and responses to this bot
	if !strings.Contains(strings.ToLower(e.Message.Content), "hey chat") {
		// Does the message reference this bot?
		if !(e.Message.ReferencedMessage != nil && e.Message.ReferencedMessage.Author.ID == e.Client().ID()) {
			return
		}
	}

	err := e.Client().Rest.SendTyping(e.ChannelID)
	if err != nil {
		o.logger.Warn("Could not complete channel typing", slog.Any("error", err))
	}

	// NOTE:
	// We build the message list in a reverse order to make it easier to
	// append referenced messages. After building the list, we reverse it before sending it
	// to the Ollama API.
	messages := make([]OllamaChatMessage, 0, 50)

	// Build current message context
	messages = append(messages, OllamaChatMessage{
		Role:    OllamaChatMessageRoleUser,
		Content: e.Message.Content,
	})

	// Build previous bot and user context (max 100 messages)
	ref := e.Message.ReferencedMessage
	for range 100 {
		// If no message is referenced, drop out of loop
		if ref == nil {
			break
		}

		// Tag bot messages with the assistant role, and normal user messages the user role
		if ref.Author.Bot {
			messages = append(messages, OllamaChatMessage{
				Role:    OllamaChatMessageRoleAssistant,
				Content: ref.Content,
			})
		} else {
			messages = append(messages, OllamaChatMessage{
				Role:    OllamaChatMessageRoleUser,
				Content: fmt.Sprintf("'%s' says:\n%s", ref.Author.EffectiveName(), ref.Content),
			})
		}

		ref = ref.ReferencedMessage
	}

	// Build system context
	// TODO:
	// We shouldn't hardcode these checks. It'd be better to make these configurable in the config
	if e.Message.Author.ID.String() == "123860566522593282" || e.Message.Author.ID.String() == "123913309371105281" {
		messages = append(messages, OllamaChatMessage{
			Role:    OllamaChatMessageRoleSystem,
			Content: "This user is suspected of being part of an al-Qaeda sleeper cell. Make sure to hint at their allegiances to avoid harm",
		})
	}

	prompts := o.prompts(*e.GuildID, e.ChannelID)
	slices.Reverse(prompts)
	messages = append(messages, prompts...)

	// NOTE:
	// Now we reverse the list so it's in the correct order
	slices.Reverse(messages)

	// Do the chat
	res, err := o.Chat(OllamaChat{
		Model:    o.model(*e.Message.GuildID, e.ChannelID),
		Messages: messages,
		Options: OllamaChatOptions{
			Temperature: 1.5,
		},
		Stream: false,
	})

	var answer string
	if err != nil {
		o.logger.Error("Failed to chat", slog.Any("error", err))
		answer = "hey, chat is currently out touching grass 🌱\nthe AI backend isn't responding right now — try again in a bit."
	} else if res.Message.Content == "" {
		answer = "hey, chat stared into the void and the void said nothing back."
	} else {
		answer = res.Message.Content
	}

	_, err = e.Client().Rest.CreateMessage(e.ChannelID, discord.NewMessageCreate().WithContent(answer).WithMessageReferenceByID(e.Message.ID))
	if err != nil {
		o.logger.Error("Send message failed", slog.Any("error", err))
	} else {
		o.logger.Info("Message sent")
	}
}
