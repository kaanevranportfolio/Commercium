package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"

	"github.com/kaanevranportfolio/Commercium/internal/api-gateway/config"
	"github.com/kaanevranportfolio/Commercium/internal/api-gateway/server"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
	"github.com/kaanevranportfolio/Commercium/pkg/metrics"
	"github.com/kaanevranportfolio/Commercium/pkg/tracing"
)

const serviceName = "api-gateway"

func main() {
	// Initialize configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize logger
	logger, err := logger.New(cfg.Logger, serviceName)
	if err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Starting API Gateway", 
		"version", cfg.Version,
		"environment", cfg.Environment,
		"port", cfg.Server.Port,
	)

	// Initialize tracing
	tracerProvider, err := tracing.NewTracerProvider(cfg.Tracing, serviceName)
	if err != nil {
		logger.Fatal("Failed to initialize tracing", "error", err)
	}
	defer func() {
		if err := tracerProvider.Shutdown(context.Background()); err != nil {
			logger.Error("Failed to shutdown tracer provider", "error", err)
		}
	}()

	// Initialize metrics
	metricsRegistry, err := metrics.NewRegistry(cfg.Metrics, serviceName)
	if err != nil {
		logger.Fatal("Failed to initialize metrics", "error", err)
	}

	// Set Gin mode
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Create server
	srv, err := server.New(cfg, logger, metricsRegistry)
	if err != nil {
		logger.Fatal("Failed to create server", "error", err)
	}

	// Create HTTP server
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      srv.Handler(),
		ReadTimeout:  time.Duration(cfg.Server.ReadTimeout) * time.Second,
		WriteTimeout: time.Duration(cfg.Server.WriteTimeout) * time.Second,
		IdleTimeout:  time.Duration(cfg.Server.IdleTimeout) * time.Second,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Server starting", "address", httpServer.Addr)
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Create a deadline for shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Attempt graceful shutdown
	if err := httpServer.Shutdown(ctx); err != nil {
		logger.Error("Server forced to shutdown", "error", err)
	} else {
		logger.Info("Server shutdown complete")
	}
}

func init() {
	// Set up Viper to read configuration
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("./configs")
	viper.AddConfigPath(".")
	
	// Enable reading from environment variables
	viper.AutomaticEnv()
	
	// Set default values
	viper.SetDefault("server.port", 8080)
	viper.SetDefault("environment", "development")
	viper.SetDefault("logger.level", "info")
}
