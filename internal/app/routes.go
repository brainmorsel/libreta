package app

import (
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/brainmorsel/libreta/internal/api"
	"github.com/brainmorsel/libreta/ui"
)

func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	config *Config,
	apiNodeContent *api.NodeContent,
	apiRPC *api.RPC,
) {
	mux.Handle("POST /api/rpc/{method_name}", logErrorHandler{logger, apiRPC.HandleRequest})
	mux.Handle("POST /api/content", logErrorHandler{logger, apiNodeContent.Upload})
	mux.Handle("GET /api/content/{node_id}", logErrorHandler{logger, apiNodeContent.Download})
	if config.DevServerURL.String() != "" {
		logger.Info("proxy to dev server used", slog.String("url", config.DevServerURL.String()))
		mux.Handle("/", httputil.NewSingleHostReverseProxy(&config.DevServerURL))
	} else {
		mux.Handle("/", ui.FileServer())
	}
}

type logErrorHandler struct {
	logger *slog.Logger
	f      func(http.ResponseWriter, *http.Request) error
}

func (h logErrorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := h.f(w, r); err != nil {
		h.logger.ErrorContext(r.Context(), "http handler error", slog.Any("error", err))
	}
}
