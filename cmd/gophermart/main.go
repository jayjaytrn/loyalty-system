package main

import (
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/loyalty-system/config"
	"github.com/jayjaytrn/loyalty-system/internal/db"
	"github.com/jayjaytrn/loyalty-system/internal/handlers"
	"github.com/jayjaytrn/loyalty-system/internal/middleware"
	"github.com/jayjaytrn/loyalty-system/logging"
	"go.uber.org/zap"
	"net/http"
)

func main() {
	logger := logging.GetSugaredLogger()
	defer logger.Sync() // TODO прочитать зачем

	cfg := config.GetConfig()

	database, err := db.NewManager(cfg)
	if err != nil {
		logger.Fatal(err)
	}
	defer database.Close()

	h := handlers.Handler{
		Config:   cfg,
		Database: *database,
		Logger:   logger,
	}

	r := initRouter(h, logger)

	err = http.ListenAndServe(cfg.RunAddress, r)
	logger.Fatalw("failed to start server", "error", err)
}

func initRouter(h handlers.Handler, logger *zap.SugaredLogger) *chi.Mux {
	r := chi.NewRouter()
	r.Post(`/api/user/register`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Register),
				logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateCredentialsAndHashLogin,
			).ServeHTTP(w, r)
		},
	)
	r.Post(`/api/user/login`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Login),
				logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateCredentialsAndHashLogin,
			).ServeHTTP(w, r)
		},
	)
	r.Post(`/api/user/orders`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Login),
				logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateAuth,
			).ServeHTTP(w, r)
		},
	)
	return r
}
