package chatter

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
	Role      string                `json:"role"`             // required "system","user","assistant" or "tool"
	Content   string                `json:"content"`          // required
	Images    []string              `json:"images,omitempty"` //base64-encoded image content
	ToolCalls []OllamaChatToolCalls `json:"tool_calls,omitempty"`
}

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
