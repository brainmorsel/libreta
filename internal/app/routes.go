package app

import (
	"log/slog"
	"net/http"
	"net/http/httputil"

	"github.com/brainmorsel/libreta/ui"
)

func addRoutes(
	mux *http.ServeMux,
	logger *slog.Logger,
	config *Config,
) {
	if config.DevServerURL.String() != "" {
		logger.Info("proxy to dev server used", slog.String("url", config.DevServerURL.String()))
		mux.Handle("/", httputil.NewSingleHostReverseProxy(&config.DevServerURL))
	} else {
		mux.Handle("/", ui.FileServer())
	}
}
