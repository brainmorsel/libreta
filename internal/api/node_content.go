package api

import (
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/brainmorsel/libreta/internal/storage"
	"github.com/brainmorsel/libreta/pkg/jmsgp"
)

func NewNodeContent(logger *slog.Logger, storage *storage.Storage) (*NodeContent, error) {
	if logger == nil {
		return nil, fmt.Errorf("logger is nil")
	}
	if storage == nil {
		return nil, fmt.Errorf("storage is nil")
	}
	return &NodeContent{
		logger:  logger,
		storage: storage,
	}, nil
}

type NodeContent struct {
	logger  *slog.Logger
	storage *storage.Storage
}

type NodeContentUploadResult struct {
	Name     string `json:"name"`
	Hash     string `json:"hash"`
	Mimetype string `json:"mimetype"`
	Length   int64  `json:"length"`
}

func (nc *NodeContent) Upload(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	requestID := r.Header.Get(jmsgp.DefaultMessageIdHTTPHeader)
	contentType := r.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "multipart/form-data;") {
		return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, ErrInvalidContentType(contentType))
	}
	r.ParseMultipartForm(1024 * 1024)
	file, handler, err := r.FormFile("file")
	if err != nil {
		return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, ErrInternal(fmt.Errorf("parse form file: %w", err)))
	}
	defer file.Close()

	hash, err := nc.storage.NodeContentSave(ctx, file)
	if err != nil {
		return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, ErrInternal(fmt.Errorf("save content: %w", err)))
	}

	return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, NodeContentUploadResult{
		Name:     handler.Filename,
		Hash:     hash,
		Mimetype: handler.Header.Get("Content-Type"),
		Length:   handler.Size,
	})
}

func (nc *NodeContent) Download(w http.ResponseWriter, r *http.Request) error {
	ctx := r.Context()
	requestID := r.Header.Get(jmsgp.DefaultMessageIdHTTPHeader)
	nodeID := r.PathValue("node_id")

	nodes, err := nc.storage.NodesLoad(ctx, []string{nodeID})
	if err != nil {
		return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, ErrInternal(fmt.Errorf("load node %q: %w", nodeID, err)))
	}
	node, ok := nodes[nodeID]
	if !ok {
		return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, ErrNotFound())
	}

	if r.Method == http.MethodHead {
		w.Header().Set("Content-Type", node.ContentMimetype)
		w.Header().Set("Content-Length", fmt.Sprintf("%d", node.ContentLength))
		w.WriteHeader(http.StatusOK)
		return nil
	}

	content, err := nc.storage.NodeContentLoad(ctx, node.ContentHash)
	if err != nil {
		return jmsgp.WriteHTTPResponse(ctx, w, "", requestID, ErrInternal(fmt.Errorf("load content %q: %w", node.ContentHash, err)))
	}
	w.Header().Set("Content-Type", node.ContentMimetype)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", node.ContentLength))
	_, err = io.Copy(w, content)
	if err != nil {
		return fmt.Errorf("write response: %w", err)
	}
	return nil
}
