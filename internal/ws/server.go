package ws

import (
	"context"
)

type EventHandler interface {
	Handle(ctx context.Context, request Message) (Message, error)
}

type Server struct {
	hub     *hub
	handlers map[string]EventHandler
}

func NewServer(hub *hub, handlers map[string]EventHandler) *Server {
	return &Server{
		hub:     hub,
		handlers: handlers,
	}
}
