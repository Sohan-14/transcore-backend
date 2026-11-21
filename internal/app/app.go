package app

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"github.com/sohan-14/transcore-backend/internal/platform/config"
	"github.com/sohan-14/transcore-backend/internal/platform/database"
	"github.com/sohan-14/transcore-backend/internal/platform/health"
	"github.com/sohan-14/transcore-backend/internal/platform/httpserver"
	"github.com/sohan-14/transcore-backend/internal/platform/logger"
	"github.com/sohan-14/transcore-backend/internal/platform/middleware"
)

// Run builds the application and serves
// ===============================================================
func Run(ctx context.Context) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log := logger.Setup(os.Stdout, &cfg.Logger)
	log.InfoContext(ctx, "starting application",
		"name", cfg.App.Name,
		"env", cfg.App.Environment,
	)

	db, err := database.New(ctx, &cfg.Database, log)
	if err != nil {
		return fmt.Errorf("init database: %w", err)
	}
	defer db.Close()

	mux := http.NewServeMux()

	// Health endpoints (operational, not API features)
	mux.HandleFunc("GET /healthz", health.Liveness())
	mux.HandleFunc("GET /readyz", health.Readiness(db))

	// Catch-all 404 handler (consistent error envelope)
	// mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
	// 	httpx.WriteError(w, r, apperr.New(apperr.KindNotFound, "route_not_found", "the requested resource was not found"))
	// })

	// Module routes here

	handler := middleware.Chain(mux,
		middleware.RequestID(),
		middleware.RequestLogger(log),
		middleware.PanicRecover(log),
		middleware.MaxBodySize(cfg.HTTP.MaxBodyBytes),
	)

	srv := httpserver.New(&cfg.HTTP, log, handler)
	srv.OnDrain(func() {
		log.InfoContext(ctx, "draining connections")
	})

	return srv.Run(ctx)
}
