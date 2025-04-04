package helpers

import (
	"net/http"

	"github.com/gorilla/websocket"
)

// WS Route upgrader ( Transforms http request into ws)
var WSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all connections
	},
}
