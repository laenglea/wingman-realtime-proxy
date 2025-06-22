package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"

	"github.com/gorilla/websocket"
)

var (
	aoiBaseURL string

	aoiAPIKey     string
	aoiAPIVersion = "2024-10-01-preview"

	aoiModel      = "gpt-4o-mini-realtime-preview"
	aoiDeployment = "gpt-4o-mini-realtime-preview"
)

func main() {
	port := os.Getenv("PORT")

	if port == "" {
		port = "8080"
	}

	aoiAPIKey = os.Getenv("AZURE_OPENAI_API_KEY")
	aoiBaseURL = os.Getenv("AZURE_OPENAI_BASE_URL")

	if val := os.Getenv("AZURE_OPENAI_API_VERSION"); val != "" {
		aoiAPIVersion = val
	}

	if val := os.Getenv("AZURE_OPENAI_MODEL_NAME"); val != "" {
		aoiModel = val
	}

	if val := os.Getenv("AZURE_OPENAI_DEPLOYMENT_NAME"); val != "" {
		aoiDeployment = val
	}

	if aoiBaseURL == "" {
		log.Fatal("AZURE_OPENAI_BASE_URL environment variable is not set")
	}

	if aoiAPIKey == "" {
		log.Fatal("AZURE_OPENAI_API_KEY environment variable is not set")
	}

	http.HandleFunc("/v1/realtime", handleRealtime)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal("Server failed to start:", err)
	}
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func handleRealtime(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	conn, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	defer conn.Close()

	u, _ := url.Parse(aoiBaseURL)
	u.Scheme = "wss"
	u.Path = "/openai/realtime"

	query := u.Query()

	if aoiAPIKey != "" {
		query.Set("api-key", aoiAPIKey)
	}

	if aoiAPIVersion != "" {
		query.Set("api-version", aoiAPIVersion)
	}

	if aoiModel != "" {
		query.Set("model", aoiModel)
	}

	if aoiDeployment != "" {
		query.Set("deployment", aoiDeployment)
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

	upstream, resp, err := dialer.Dial(u.String(), headers)

	if resp != nil {
		log.Printf("Upstream connection response: %s", resp.Status)

		data, _ := io.ReadAll(resp.Body)
		println(string(data))
	}

	if err != nil {
		log.Printf("Failed to connect to OpenAI: %v", err)
		return
	}

	defer upstream.Close()

	log.Printf("Connected to Upstream Realtime API")

	go func() {
		defer cancel()

		for {
			messageType, message, err := conn.ReadMessage()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Client connection error: %v", err)
				}

				return
			}

			if err := upstream.WriteMessage(messageType, message); err != nil {
				log.Printf("Failed to write to OpenAI: %v", err)
				return
			}
		}
	}()

	go func() {
		defer cancel()

		for {
			messageType, message, err := upstream.ReadMessage()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("OpenAI connection error: %v", err)
				}

				return
			}

			if err := conn.WriteMessage(messageType, message); err != nil {
				log.Printf("Failed to write to client: %v", err)
				return
			}
		}
	}()

	<-ctx.Done()

	log.Printf("Connection closed")
}
