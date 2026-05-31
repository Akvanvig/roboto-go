package ollama

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/url"
)

// data for connecting to ollama server
type Ollama struct {
	Server         string                        `yaml:"server,omitempty"`
	ChatPath       string                        `yaml:"chatPath,omitempty"`
	GeneratePath   string                        `yaml:"generatePath,omitempty"`
	DefaultPrompt  SystemPromptConfig            `yaml:"defaultPrompt,omitempty"`
	ServerPrompts  map[uint64]SystemPromptConfig `yaml:"serverPrompts,omitempty"`  // server/channel id as key
	ChannelPrompts map[uint64]SystemPromptConfig `yaml:"channelPrompts,omitempty"` // server/channel id as key
}

// config used for ollama queries
type SystemPromptConfig struct {
	Name         string `yaml:"name"`         // ¯\_(ツ)_/¯
	Model        string `yaml:"model"`        // override model to use in ollama request. requires model present in ollama
	Exclusive    bool   `yaml:"exclusive"`    // removes system-prompts earlier in the chain Default < Server < Channel
	SystemPrompt string `yaml:"systemPrompt"` // system-prompt to provide when used
}

// returns a list of messages containing system prompt
func (o *Ollama) model(server, channel uint64) string {
	channelConfig := o.ChannelPrompts[channel]
	serverConfig := o.ServerPrompts[server]
	if channelConfig.Model != "" {
		return serverConfig.Model
	}
	if serverConfig.Model != "" {
		return serverConfig.Model
	}
	return o.DefaultPrompt.Model
}

// returns a list of messages containing system prompt
func (o *Ollama) systemPromts(server, channel uint64) (response []OllamaChatMessage) {
	serverConfig := o.ServerPrompts[server]
	channelConfig := o.ChannelPrompts[channel]

	// channel specific config
	if channelConfig.SystemPrompt != "" {
		response = append(response, OllamaChatMessage{
			Role:    "system",
			Content: channelConfig.SystemPrompt,
		})
		if channelConfig.Exclusive {
			return
		}
	}

	// server specific config
	if serverConfig.SystemPrompt != "" {
		response = append([]OllamaChatMessage{{
			Role:    "system",
			Content: serverConfig.SystemPrompt,
		}}, response...)
		if serverConfig.Exclusive {
			return
		}
	}

	// default config
	response = append([]OllamaChatMessage{{
		Role:    "system",
		Content: o.DefaultPrompt.SystemPrompt,
	}}, response...)

	return response
}

// do stuff
func (o *Ollama) Chat(chat OllamaChat) (OllamaChatResponse, error) {

	// bad validation probably
	slog.Info("doing request", "chat", chat)

	// invoke stuff
	endpoint, _ := url.JoinPath(o.Server, o.ChatPath)
	jsonData, err := json.Marshal(chat)
	if err != nil {
		return OllamaChatResponse{}, err
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBufferString(string(jsonData)))
	if err != nil {
		return OllamaChatResponse{}, err
	}
	if resp.StatusCode != 200 {
		slog.Warn("responsecode is not 200", "code", resp.StatusCode, "status", resp.Status)
	}

	var chatResp OllamaChatResponse
	jsonDecoder := json.NewDecoder(resp.Body)
	err = jsonDecoder.Decode(&chatResp)

	return chatResp, nil
}
