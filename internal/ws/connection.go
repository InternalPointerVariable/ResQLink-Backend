package ws

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin:     func(r *http.Request) bool { return true },
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

func upgrade(w http.ResponseWriter, r *http.Request) (*websocket.Conn, error) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}

	return conn, err
}

func (s *Server) HandleConnection(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	conn, err := upgrade(w, r)
	defer conn.Close()

	if err != nil {
		slog.Error(err.Error())
		return
	}

	// TODO: Authenticate user

	client := &client{
		conn:     conn,
		pool:     s.pool,
		handlers: s.handlers,
	}

	s.pool.register <- client

	go client.writePump()
	go client.readPump(ctx)
}
