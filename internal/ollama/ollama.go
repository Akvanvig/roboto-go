package ollama

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"slices"

	"github.com/Akvanvig/roboto-go/internal/config"
	"github.com/disgoorg/disgo/bot"
	"github.com/disgoorg/snowflake/v2"
)

// structs
// message model for the chat endpoint of ollama
// https://docs.ollama.com/api/chat
type OllamaChat struct {
	Model       string              `json:"model"`    // required
	Messages    []OllamaChatMessage `json:"messages"` // required
	Tools       []OllamaChatTools   `json:"tools,omitzero"`
	Format      string              `json:"format,omitempty"`
	Options     OllamaChatOptions   `json:"options,omitzero"`
	Stream      bool                `json:"stream"`
	Think       string              `json:"think,omitempty"` // high, medium, low
	KeepAlive   string              `json:"keep_alive,omitempty"`
	Logprobs    bool                `json:"logprobs,omitempty"`
	TopLogprobs int                 `json:"top_logprobs,omitempty"`
}

type OllamaChatMessage struct {
	Role      OllamaChatMessageRole `json:"role"`             // required "system","user","assistant" or "tool"
	Content   string                `json:"content"`          // required
	Images    []string              `json:"images,omitempty"` // base64-encoded image content
	ToolCalls []OllamaChatToolCalls `json:"tool_calls,omitempty"`
}

type OllamaChatMessageRole = string

const (
	OllamaChatMessageRoleSystem    OllamaChatMessageRole = "system"
	OllamaChatMessageRoleUser      OllamaChatMessageRole = "user"
	OllamaChatMessageRoleAssistant OllamaChatMessageRole = "assistant"
	OllamaChatMessageRoleTool      OllamaChatMessageRole = "tool"
)

// TODO
type OllamaChatTools struct {
}

// TODO
type OllamaChatToolCalls struct {
}

type OllamaChatOptions struct {
	Seed        int     `json:"seed,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
	TopK        int     `json:"top_k,omitempty"`
	TopP        float64 `json:"top_p,omitempty"`
	MinP        float64 `json:"min_p,omitempty"`
	Stop        string  `json:"stop,omitempty"`
	NumCtx      int     `json:"num_ctx,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

type OllamaChatResponse struct {
	Model              string               `json:"model"`
	CreatedAt          string               `json:"created_at"`
	Message            OllamaChatMessage    `json:"message"`
	Done               bool                 `json:"done"`
	DoneReason         string               `json:"done_reason"`
	TotalDuration      int                  `json:"total_duration"`
	LoadDuration       int                  `json:"load_duration"`
	PromptEvalCount    int                  `json:"prompt_eval_count"`
	PromptEvalDuration int                  `json:"prompt_eval_duration"`
	EvalCount          int                  `json:"eval_count"`
	EvalDuration       int                  `json:"eval_duration"`
	Logprobs           []OllamaChatLogProbs `json:"logprobs"`
}

// TODO
type OllamaChatLogProbs struct {
}

// data for connecting to ollama server
type Ollama struct {
	logger *slog.Logger
	cfg    *config.OllamaConfig
}

// Get system model
func (o *Ollama) model(guildID snowflake.ID, channelID snowflake.ID) string {
	if cfg := o.cfg.ChannelPrompts[channelID]; cfg.Model != "" {
		return cfg.Model
	}
	if cfg := o.cfg.ServerPrompts[guildID]; cfg.Model != "" {
		return cfg.Model
	}
	return o.cfg.DefaultPrompt.Model
}

// Get system prompts
func (o *Ollama) prompts(guildID snowflake.ID, channelID snowflake.ID) []OllamaChatMessage {
	prompts := make([]OllamaChatMessage, 0, 3)

	if cfg := o.cfg.ChannelPrompts[channelID]; cfg.SystemPrompt != "" {
		prompts = append(prompts, OllamaChatMessage{
			Role:    OllamaChatMessageRoleSystem,
			Content: cfg.SystemPrompt,
		})
		if cfg.Exclusive {
			return prompts
		}
	}
	if cfg := o.cfg.ServerPrompts[guildID]; cfg.SystemPrompt != "" {
		prompts = slices.Insert(prompts, 0, OllamaChatMessage{
			Role:    OllamaChatMessageRoleSystem,
			Content: cfg.SystemPrompt,
		})
		if cfg.Exclusive {
			return prompts
		}
	}

	// Default
	prompts = slices.Insert(prompts, 0, OllamaChatMessage{
		Role:    OllamaChatMessageRoleSystem,
		Content: o.cfg.DefaultPrompt.SystemPrompt,
	})
	return prompts
}

// do stuff
func (o *Ollama) Chat(chat OllamaChat) (OllamaChatResponse, error) {
	// bad validation probably
	o.logger.Info("doing request", slog.Any("chat", chat))

	// invoke stuff
	endpoint, _ := url.JoinPath(o.cfg.Server, o.cfg.ChatPath)
	jsonData, err := json.Marshal(chat)
	if err != nil {
		return OllamaChatResponse{}, err
	}

	resp, err := http.Post(endpoint, "application/json", bytes.NewBufferString(string(jsonData)))
	if err != nil {
		return OllamaChatResponse{}, err
	}
	if resp.StatusCode != 200 {
		return OllamaChatResponse{}, fmt.Errorf("received non-successful error code '%d' status '%s", resp.StatusCode, resp.Status)
	}

	var chatResp OllamaChatResponse
	jsonDecoder := json.NewDecoder(resp.Body)
	err = jsonDecoder.Decode(&chatResp)

	return chatResp, nil
}

func New(discord *bot.Client, cfg *config.OllamaConfig) *Ollama {
	ollama := &Ollama{
		logger: discord.Logger,
		cfg:    cfg,
	}
	discord.AddEventListeners(
		bot.NewListenerFunc(ollama.onMessageCreate),
	)

	return ollama
}
