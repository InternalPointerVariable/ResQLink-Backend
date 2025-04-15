package ws

import (
	"fmt"
	"log/slog"
	"sync"
)

type hub struct {
	clients    map[*client]bool
	register   chan *client
	unregister chan *client
	mu         sync.RWMutex
}

func NewHub() *hub {
	return &hub{
		clients:    make(map[*client]bool),
		register:   make(chan *client),
		unregister: make(chan *client),
	}
}

func (p *hub) Start() {
	slog.Info("Starting WebSocket hub...")

	for {
		select {
		case client := <-p.register:
			p.mu.Lock()
			p.clients[client] = true
			p.mu.Unlock()

			slog.Info("User has connected.")
			slog.Info(fmt.Sprintf("Size of hub: %d", len(p.clients)))

		case client := <-p.unregister:
			p.mu.Lock()
			if _, ok := p.clients[client]; ok {
				delete(p.clients, client)

				slog.Info("User has disconnected.")
				slog.Info(fmt.Sprintf("Size of hub: %d", len(p.clients)))
			}
			p.mu.Unlock()
		}
	}
}

func (h *hub) Broadcast(msg Message) {
	for client := range h.clients {
		client.send <- msg
	}
}
