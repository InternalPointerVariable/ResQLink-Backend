package disaster

import (
	"context"

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
	createReport = "disaster:create-report"
)

func (s *SocketServer) Handle(ctx context.Context, request ws.Message) (ws.Message, error) {
	return ws.Message{}, nil
}
