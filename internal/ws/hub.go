package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/redis/go-redis/v9"
)

type hub struct {
	clients     map[*client]bool
	register    chan *client
	unregister  chan *client
	mu          sync.RWMutex
	redisClient *redis.Client
}

func NewHub(rds *redis.Client) *hub {
	return &hub{
		clients:     make(map[*client]bool),
		register:    make(chan *client),
		unregister:  make(chan *client),
		redisClient: rds,
	}
}

func (h *hub) Start() {
	slog.Info("Starting WebSocket hub...")

	ctx := context.Background()
	go h.listenToPubSub(ctx)

	for {
		select {
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()

			slog.Info("User has connected.")
			slog.Info(fmt.Sprintf("Size of hub: %d", len(h.clients)))

		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)

				slog.Info("User has disconnected.")
				slog.Info(fmt.Sprintf("Size of hub: %d", len(h.clients)))
			}
			h.mu.Unlock()
		}
	}
}

func (h *hub) Broadcast(msg Message) {
	for client := range h.clients {
		client.send <- msg
	}
}

func (h *hub) listenToPubSub(ctx context.Context) {
	// TODO: The event type for the WebSocket and channels for PubSub should be different types
	sub := h.redisClient.Subscribe(ctx, "disaster:create_report")
	defer sub.Close()

	if _, err := sub.Receive(ctx); err != nil {
		slog.Error(err.Error())
		return
	}

	ch := sub.Channel()

	for msg := range ch {
		fmt.Println(msg.String())
		foo := Message{
			Event: msg.Channel,
			Data:  json.RawMessage(msg.Payload),
		}

		h.Broadcast(foo)
	}
}
