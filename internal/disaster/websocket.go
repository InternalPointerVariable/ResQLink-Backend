package disaster

import (
	"context"
	"encoding/json"

	"github.com/InternalPointerVariable/ResQLink-Backend/internal/ws"
)

type SocketServer struct {
	repository Repository
}

func NewSocketServer(repository Repository) *SocketServer {
	return &SocketServer{
		repository: repository,
	}
}

const (
	createReport = "disaster:create_report" // Used as a PubSub channel
	saveLocation = "disaster:save_location"
)

func (s *SocketServer) Handle(ctx context.Context, msg ws.Message) (ws.Message, error) {
	switch msg.Event {
	case saveLocation:
		var req saveLocationRequest
		if err := json.Unmarshal(msg.Data, &req); err != nil {
			return ws.Message{}, err
		}

		if err := s.repository.SaveLocation(ctx, req); err != nil {
			return ws.Message{}, err
		}

		return msg.Response(req)
	}

	return ws.Message{}, nil
}
