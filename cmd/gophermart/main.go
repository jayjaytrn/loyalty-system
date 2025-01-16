package main

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/jayjaytrn/loyalty-system/config"
	"github.com/jayjaytrn/loyalty-system/internal/accrual"
	"github.com/jayjaytrn/loyalty-system/internal/db"
	"github.com/jayjaytrn/loyalty-system/internal/handlers"
	"github.com/jayjaytrn/loyalty-system/internal/middleware"
	"github.com/jayjaytrn/loyalty-system/logging"
	"github.com/jayjaytrn/loyalty-system/models"
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

	ordersToAccrualSystem := make(chan models.OrderToAccrual)
	am := accrual.NewManager(ordersToAccrualSystem, database, cfg, logger)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go am.StartOrderProcessing(ctx)

	h := handlers.Handler{
		Config:         cfg,
		Database:       database,
		Logger:         logger,
		AccrualManager: am,
	}

	r := initRouter(h)

	err = http.ListenAndServe(cfg.RunAddress, r)
	logger.Fatalw("failed to start server", "error", err)
}

func initRouter(h handlers.Handler) *chi.Mux {
	r := chi.NewRouter()
	r.Post(`/api/user/register`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Register),
				h.Logger,
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
				h.Logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateCredentialsAndHashLogin,
			).ServeHTTP(w, r)
		},
	)
	r.Post(`/api/user/orders`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Orders),
				h.Logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateAuth,
			).ServeHTTP(w, r)
		},
	)
	r.Get(`/api/user/orders`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.OrdersGet),
				h.Logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateAuth,
			).ServeHTTP(w, r)
		},
	)
	r.Get(`/api/user/balance`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Balance),
				h.Logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateAuth,
			).ServeHTTP(w, r)
		},
	)
	r.Post(`/api/user/balance/withdraw`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Withdraw),
				h.Logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateAuth,
			).ServeHTTP(w, r)
		},
	)
	r.Get(`/api/user/withdrawals`,
		func(w http.ResponseWriter, r *http.Request) {
			middleware.Conveyor(
				http.HandlerFunc(h.Withdrawals),
				h.Logger,
				middleware.WriteWithCompression,
				middleware.ReadWithCompression,
				middleware.ValidateAuth,
			).ServeHTTP(w, r)
		},
	)
	return r
}
