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
	Server       string
	ChatPath     string
	GeneratePath string
}

// temporary hardcoded stuff
func New() Ollama {
	return Ollama{
		Server:       "http://192.168.10.23:32300",
		ChatPath:     "/api/chat",
		GeneratePath: "/api/generate",
	}
}

// do stuff
func (o *Ollama) Chat(chat OllamaChat) (OllamaChatResponse, error) {

	// bad validation probably
	slog.Info("doing request", "chat", chat, "ollama", o)

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
