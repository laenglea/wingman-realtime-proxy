package server

import (
	"context"
	"io"
	"log"
	"net/http"

	"github.com/adrianliechti/wingman-realtime-proxy/provider"

	"github.com/gorilla/websocket"
)

type Server struct {
	handler provider.Handler
}

func New(h provider.Handler) *Server {
	s := &Server{
		handler: h,
	}

	return s
}

func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/realtime", s.handleRealtime)

	return http.ListenAndServe(addr, mux)
}

func (s *Server) handleRealtime(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	downstream, err := upgrader.Upgrade(w, r, nil)

	if err != nil {
		log.Printf("Failed to upgrade connection: %v", err)
		return
	}

	defer downstream.Close()

	upstream, resp, err := s.handler.Dial(r)

	if err != nil {
		log.Printf("Failed to connect to upstream: %v", err)

		if resp != nil {
			data, _ := io.ReadAll(resp.Body)
			log.Print(string(data))
		}

		return
	}

	defer upstream.Close()

	log.Printf("Connected to Upstream Realtime API")

	go func() {
		defer cancel()

		for {
			messageType, message, err := downstream.ReadMessage()

			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					log.Printf("Client connection error: %v", err)
				}

				return
			}

			if err := upstream.WriteMessage(messageType, message); err != nil {
				log.Printf("Failed to write to Provider: %v", err)
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
					log.Printf("Provider connection error: %v", err)
				}

				return
			}

			if err := downstream.WriteMessage(messageType, message); err != nil {
				log.Printf("Failed to write to client: %v", err)
				return
			}
		}
	}()

	<-ctx.Done()

	log.Printf("Connection closed")
}

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}
