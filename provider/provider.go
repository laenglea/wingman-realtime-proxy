package provider

import (
	"net/http"

	"github.com/gorilla/websocket"
)

type Handler interface {
	Dial(r *http.Request) (*websocket.Conn, *http.Response, error)
}
