package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/platform/order-service/internal/awsclient"
	"github.com/platform/order-service/internal/handler"
	"github.com/platform/order-service/internal/idempotency"
	"github.com/platform/order-service/internal/logger"
	"github.com/platform/order-service/internal/middleware"
	"github.com/platform/order-service/internal/publisher"
	"github.com/platform/order-service/internal/repository"
	"github.com/platform/order-service/internal/security"
)

func main() {
	if os.Getenv("JWT_SECRET") == "" {
		logger.Error("JWT_SECRET environment variable is required", logger.Fields{})
		os.Exit(1)
	}

	ctx := context.Background()

	dynamo := awsclient.NewDynamo(ctx)
	snsClient := awsclient.NewSNS(ctx)

	orderRepo := repository.NewOrderRepository(dynamo)
	snsPub := publisher.NewSNSPublisher(snsClient)
	idempot := idempotency.New(dynamo)

	orderHandler := handler.NewOrderHandler(orderRepo, snsPub, idempot)
	authHandler := handler.NewAuthHandler()

	rateLimiter := security.NewRateLimiter(10, 30)

	r := chi.NewRouter()
	r.Use(chimiddleware.Recoverer)
	r.Use(security.SecureHeaders)
	r.Use(rateLimiter.Middleware)
	r.Use(middleware.CORS)
	r.Use(middleware.CorrelationID)
	r.Use(middleware.RequestLogger)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"ok","service":"order-service"}`))
	})

	r.Post("/auth/token", authHandler.IssueToken)

	r.Route("/orders", func(r chi.Router) {
		r.Use(middleware.Auth)
		r.Post("/", orderHandler.Create)
		r.Get("/", orderHandler.List)
		r.Get("/{orderId}", orderHandler.GetByID)
	})

	port := os.Getenv("ORDER_SERVICE_PORT")
	if port == "" {
		port = "3001"
	}

	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      r,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		logger.Info("order-service started", logger.Fields{"port": port})
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("Server error", logger.Fields{"error": err.Error()})
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down order-service")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
}
