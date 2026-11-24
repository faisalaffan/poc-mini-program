package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/faisalaffan/poc-mini-program/backend/internal/config"
	"github.com/faisalaffan/poc-mini-program/backend/internal/repository/postgres"
	"github.com/faisalaffan/poc-mini-program/backend/internal/service"
	httptransport "github.com/faisalaffan/poc-mini-program/backend/internal/transport/http"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed_to_load_config", "error", err)
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := postgres.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed_to_connect_postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := postgres.MigrateAndSeed(ctx, pool); err != nil {
		logger.Error("failed_to_migrate_and_seed", "error", err)
		os.Exit(1)
	}

	repo := postgres.NewGachaRepository(pool)
	rng := service.NewCryptoWeightedRNG()
	authSvc := service.NewAuthService(cfg.JWTSecret)
	gachaSvc := service.NewGachaService(repo, rng, cfg.AppEnv)
	sduiSvc := service.NewSDUIService(repo)

	router := httptransport.NewRouter(httptransport.RouterDeps{
		Config:       cfg,
		Logger:       logger,
		AuthService:  authSvc,
		GachaService: gachaSvc,
		SDUIService:  sduiSvc,
	})

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		logger.Info("server_started", "port", cfg.Port, "env", cfg.AppEnv, "version", cfg.BuildVersion)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server_listen_failed", "error", err)
			stop()
		}
	}()

	<-ctx.Done()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server_shutdown_error", "error", err)
	}
	logger.Info("server_stopped")
}
