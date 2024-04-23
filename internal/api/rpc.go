package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/brainmorsel/libreta/internal/storage"
	"github.com/brainmorsel/libreta/pkg/jmsgp"
)

func NewRPC(logger *slog.Logger, storage *storage.Storage) (*RPC, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage is nil")
	}
	rpc := &RPC{
		logger:  logger,
		storage: storage,
	}

	hub := jmsgp.NewHub()
	hub.AddHandler("GenerateNodeID", jmsgp.RPCHandler(rpc.GenerateNodeID))
	hub.AddHandler("NodeSave", jmsgp.RPCHandler(rpc.NodeSave))

	rpc.transport = jmsgp.NewHTTPServerTransport(hub)
	rpc.transport.ExtractTargetFunc = jmsgp.TargetFromHTTPRequestURLPathValue("method_name")
	return rpc, nil
}

type RPC struct {
	logger    *slog.Logger
	storage   *storage.Storage
	transport *jmsgp.HTTPServerTransport
}

func (rpc *RPC) HandleRequest(w http.ResponseWriter, r *http.Request) error {
	return rpc.transport.HandleRequest(w, r)
}

func (rpc *RPC) GenerateNodeID(ctx context.Context, _ struct{}) (string, error) {
	return rpc.storage.GenerateNodeID(ctx)
}

type Node struct {
	ID              string `json:"id"`
	Name            string `json:"name"`
	ContentHash     string `json:"content_hash"`
	ContentMimetype string `json:"content_mimetype"`
}

func (rpc *RPC) NodeSave(ctx context.Context, n Node) (string, error) {
	node := storage.Node{
		ID:              n.ID,
		Name:            n.Name,
		ContentHash:     n.ContentHash,
		ContentMimetype: n.ContentMimetype,
	}
	err := rpc.storage.NodeSave(ctx, node)
	if err != nil {
		return "", ErrInternal(err)
	}
	return "ok", nil
}
