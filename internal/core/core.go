package core

import (
	"log/slog"

	"github.com/brainmorsel/libreta/internal/storage"
)

type Core struct {
	logger  *slog.Logger
	storage *storage.Storage
}

func NewCore(logger *slog.Logger, storage *storage.Storage) *Core {
	return &Core{
		logger:  logger,
		storage: storage,
	}
}
