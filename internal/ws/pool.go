package ws

import (
	"fmt"
	"log/slog"
	"sync"
)

type pool struct {
	clients    map[*client]bool
	register   chan *client
	unregister chan *client
	mu         sync.RWMutex
}

func NewPool() *pool {
	return &pool{
		clients:    make(map[*client]bool),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

func (p *pool) Start() {
	slog.Info("Starting WebSocket pool...")

	for {
		select {
		case client := <-p.register:
			p.clients[client] = true

			slog.Info("User has connected.")
			slog.Info(fmt.Sprintf("Size of pool: %d", len(p.clients)))

		case client := <-p.unregister:
			p.mu.Lock()
			if _, ok := p.clients[client]; ok {
				delete(p.clients, client)

				slog.Info("User has disconnected.")
				slog.Info(fmt.Sprintf("Size of pool: %d", len(p.clients)))
			}
			p.mu.Unlock()
		}
	}
}
