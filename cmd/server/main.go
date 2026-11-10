package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/nats-io/nats.go"
	goredis "github.com/redis/go-redis/v9"

	"github.com/mugishajc/orderforge/internal/api/handler"
	"github.com/mugishajc/orderforge/internal/api/middleware"
	"github.com/mugishajc/orderforge/internal/application"
	"github.com/mugishajc/orderforge/internal/config"
	infraNats "github.com/mugishajc/orderforge/internal/infrastructure/nats"
	infraPostgres "github.com/mugishajc/orderforge/internal/infrastructure/postgres"
	infraRedis "github.com/mugishajc/orderforge/internal/infrastructure/redis"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(logger)

	cfg, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", slog.String("error", err.Error()))
		os.Exit(1)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// PostgreSQL
	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		logger.Error("failed to connect to database", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer pool.Close()

	if err := pool.Ping(ctx); err != nil {
		logger.Error("database ping failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	// Redis
	redisOpt, err := goredis.ParseURL(cfg.RedisURL)
	if err != nil {
		logger.Error("invalid redis URL", slog.String("error", err.Error()))
		os.Exit(1)
	}
	redisClient := goredis.NewClient(redisOpt)
	defer redisClient.Close()

	// NATS
	nc, err := nats.Connect(cfg.NatsURL)
	if err != nil {
		logger.Error("failed to connect to NATS", slog.String("error", err.Error()))
		os.Exit(1)
	}
	defer nc.Close()

	// Repositories
	orderRepo := infraPostgres.NewOrderRepository(pool)
	productRepo := infraPostgres.NewProductRepository(pool)
	couponRepo := infraPostgres.NewCouponRepository(pool)
	productCache := infraRedis.NewProductCache(redisClient)
	eventPublisher := infraNats.NewPublisher(nc)

	// Use cases
	createOrder := application.NewCreateOrderUseCase(orderRepo, productRepo, productCache, eventPublisher)
	cancelOrder := application.NewCancelOrderUseCase(orderRepo, productRepo, productCache, eventPublisher)
	getHistory := application.NewGetOrderHistoryUseCase(orderRepo)
	applyDiscount := application.NewApplyDiscountUseCase(orderRepo, couponRepo, eventPublisher)
	transitionOrder := application.NewTransitionOrderUseCase(orderRepo, eventPublisher)

	// HTTP
	orderHandler := handler.NewOrderHandler(createOrder, cancelOrder, getHistory, applyDiscount, transitionOrder)
	healthHandler := handler.NewHealthHandler(pool)

	r := chi.NewRouter()
	r.Use(chimiddleware.RequestID)
	r.Use(middleware.Recovery(logger))
	r.Use(middleware.Logging(logger))

	// Public health endpoints
	r.Get("/healthz", healthHandler.Liveness)
	r.Get("/readyz", healthHandler.Readiness)

	// Protected API routes
	r.Group(func(r chi.Router) {
		r.Use(middleware.Auth)
		orderHandler.Routes(r)
	})

	addr := fmt.Sprintf(":%d", cfg.ServerPort)
	srv := &http.Server{
		Addr:         addr,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info("server starting", slog.String("addr", addr))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server error", slog.String("error", err.Error()))
			os.Exit(1)
		}
	}()

	sig := <-shutdown
	logger.Info("shutdown signal received", slog.String("signal", sig.String()))

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("server shutdown error", slog.String("error", err.Error()))
	}

	logger.Info("server stopped")
}
