package ws

import (
	"context"
)

type EventHandler interface {
	Handle(ctx context.Context, request Message) (Message, error)
}

type Server struct {
	pool     *pool
	handlers map[string]EventHandler
}

func NewServer(pool *pool, handlers map[string]EventHandler) *Server {
	return &Server{
		pool:     pool,
		handlers: handlers,
	}
}
