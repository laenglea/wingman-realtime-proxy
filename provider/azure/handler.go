package azure

import (
	"errors"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

type Handler struct {
	baseURL string

	apiKey     string
	apiVersion string

	defaultModel      string
	defaultDeployment string
}

func NewFromEnvironment() (*Handler, error) {
	handler := &Handler{
		baseURL: os.Getenv("AZURE_OPENAI_BASE_URL"),

		apiKey:     os.Getenv("AZURE_OPENAI_API_KEY"),
		apiVersion: "2024-10-01-preview",

		defaultModel:      "gpt-4o-mini-realtime-preview",
		defaultDeployment: "gpt-4o-mini-realtime-preview",
	}

	if val := os.Getenv("AZURE_OPENAI_API_VERSION"); val != "" {
		handler.apiVersion = val
	}

	if val := os.Getenv("AZURE_OPENAI_MODEL_NAME"); val != "" {
		handler.defaultModel = val
	}

	if val := os.Getenv("AZURE_OPENAI_DEPLOYMENT_NAME"); val != "" {
		handler.defaultDeployment = val
	}

	if handler.baseURL == "" {
		return nil, errors.New("AZURE_OPENAI_BASE_URL environment variable is not set")
	}

	if handler.apiKey == "" {
		return nil, errors.New("AZURE_OPENAI_API_KEY environment variable is not set")
	}

	return handler, nil
}

func (h *Handler) Dial(r *http.Request) (*websocket.Conn, *http.Response, error) {
	u, _ := url.Parse(h.baseURL)

	u.Scheme = "wss"
	u.Path = "/openai/realtime"

	query := u.Query()

	if h.apiKey != "" {
		query.Set("api-key", h.apiKey)
	}

	if h.apiVersion != "" {
		query.Set("api-version", h.apiVersion)
	}

	if h.defaultModel != "" {
		query.Set("model", h.defaultModel)
	}

	if h.defaultDeployment != "" {
		query.Set("deployment", h.defaultDeployment)
	}

	if model := r.URL.Query().Get("model"); model != "" {
		query.Set("model", model)
	}

	u.RawQuery = query.Encode()

	headers := http.Header{}

	subprotocols := []string{
		"realtime",
		"openai-beta.realtime-v1",
	}

	dialer := websocket.Dialer{
		Subprotocols: subprotocols,
	}

	return dialer.Dial(u.String(), headers)
}
