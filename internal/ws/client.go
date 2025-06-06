package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/gorilla/websocket"
)

type client struct {
	hub  *hub
	conn *websocket.Conn
	send chan Message

	handlers map[string]EventHandler
}

func NewClient(conn *websocket.Conn, hub *hub, handlers map[string]EventHandler) *client {
	return &client{
		conn:     conn,
		hub:      hub,
		handlers: handlers,
		send:     make(chan Message),
	}
}

func (c *client) readPump(ctx context.Context) {
	defer func() {
		c.hub.unregister <- c
		c.conn.Close()
	}()

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			slog.Error(err.Error())
			return
		}

		var request Message
		if err := json.Unmarshal(data, &request); err != nil {
			slog.Error(err.Error())
			continue
		}

		s := strings.Split(string(request.Event), ":")
		key := s[0]

		handler, ok := c.handlers[key]
		if !ok {
			slog.Warn("handler not found for: " + key)
			continue
		}

		response, err := handler.Handle(ctx, request)
		if err != nil {
			slog.Error(err.Error())
			continue
		}

		c.hub.Broadcast(response)
	}
}

func (c *client) writePump() {
	defer c.conn.Close()

	for {
		select {
		case message, ok := <-c.send:
			if !ok {
				slog.Error("hub closed channel")
				return
			}

			if err := c.conn.WriteJSON(message); err != nil {
				slog.Error(fmt.Errorf("websocket write json: %w", err).Error())
			}
		}
	}
}
