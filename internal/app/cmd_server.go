package app

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"github.com/brainmorsel/libreta/internal/api"
	"github.com/brainmorsel/libreta/internal/core"
	"github.com/brainmorsel/libreta/internal/storage"
)

func cmdServer(ctx context.Context, logger *slog.Logger, config *Config) error {
	storage, err := storage.NewStorage(logger, config.DataDir)
	if err != nil {
		return fmt.Errorf("new storage: %w", err)
	}
	if err := storage.Open(ctx, false); err != nil {
		return fmt.Errorf("open storage: %w", err)
	}
	core := core.NewCore(logger, storage)
	_ = core

	apiNodeContent, err := api.NewNodeContent(logger, storage)
	if err != nil {
		return fmt.Errorf("new api.NodeContent: %w", err)
	}
	apiRPC, err := api.NewRPC(logger, storage)
	if err != nil {
		return fmt.Errorf("new api.RPC: %w", err)
	}

	srv := http.NewServeMux()
	addRoutes(
		srv,
		logger,
		config,
		apiNodeContent,
		apiRPC,
	)
	httpServer := &http.Server{
		Addr:    config.BindAddr,
		Handler: srv,
	}
	go func() {
		logger.InfoContext(ctx, "run server", slog.String("bind", httpServer.Addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.ErrorContext(ctx, "error listening and serving", slog.Any("error", err))
		}
	}()
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-ctx.Done()
		if err := httpServer.Shutdown(ctx); err != nil {
			logger.ErrorContext(ctx, "error shutting down http server", slog.Any("error", err))
		}
	}()
	wg.Wait()
	return nil
}
