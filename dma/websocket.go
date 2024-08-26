package dma

import (
	"github.com/gorilla/websocket"
)

// ReadWebSocket should run as a goroutine, reading the connection and sending
// all text messages to the given channel until an error.
//
// The ReadMessage blocks until there is a message, which makes the 'select'
// pattern unworkable. And a timeout destroys the web socket. Hence reading in
// another goroutine.
func ReadWebSocket(conn *websocket.Conn, messages chan []byte) {
	for {
		t, b, err := conn.ReadMessage()
		if err != nil {
			return
		}
		if t != websocket.TextMessage {
			continue
		}
		messages <- b
	}
}
