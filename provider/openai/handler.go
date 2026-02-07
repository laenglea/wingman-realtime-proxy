package openai

import (
	"errors"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

type Handler struct {
	baseURL string

	apiKey string

	defaultModel string
}

func NewFromEnvironment() (*Handler, error) {
	handler := &Handler{
		baseURL: os.Getenv("OPENAI_BASE_URL"),

		apiKey: os.Getenv("OPENAI_API_KEY"),

		defaultModel: "gpt-realtime",
	}

	if handler.baseURL == "" {
		handler.baseURL = "https://api.openai.com/v1"
	}

	if handler.baseURL == "" {
		return nil, errors.New("OPENAI_BASE_URL environment variable is not set")
	}

	if handler.apiKey == "" {
		return nil, errors.New("OPENAI_API_KEY environment variable is not set")
	}

	return handler, nil
}

func (h *Handler) Dial(r *http.Request) (*websocket.Conn, *http.Response, error) {
	u, _ := url.Parse(h.baseURL)

	u.Scheme = "wss"
	u.Path = "/v1/realtime"

	query := u.Query()

	if h.defaultModel != "" {
		query.Set("model", h.defaultModel)
	}

	if model := r.URL.Query().Get("model"); model != "" {
		query.Set("model", model)
	}

	u.RawQuery = query.Encode()

	headers := http.Header{
		"Authorization": []string{"Bearer " + h.apiKey},
	}

	dialer := websocket.Dialer{}

	return dialer.Dial(u.String(), headers)
}
