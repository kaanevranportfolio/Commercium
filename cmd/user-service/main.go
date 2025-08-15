package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/kaanevranportfolio/Commercium/internal/user/handlers"
	"github.com/kaanevranportfolio/Commercium/internal/user/repository"
	"github.com/kaanevranportfolio/Commercium/internal/user/service"
	"github.com/kaanevranportfolio/Commercium/pkg/auth"
	"github.com/kaanevranportfolio/Commercium/pkg/config"
	"github.com/kaanevranportfolio/Commercium/pkg/database"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
	"github.com/kaanevranportfolio/Commercium/pkg/metrics"
	"github.com/kaanevranportfolio/Commercium/pkg/tracing"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		panic(fmt.Sprintf("Failed to load configuration: %v", err))
	}

	// Initialize logger
	log, err := logger.New(cfg.Logger, "user-service")
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer log.Sync()

	log.Info("Starting User Service", 
		"version", cfg.Version,
		"environment", cfg.Environment,
		"port", cfg.Server.Port,
	)

	// Initialize tracing
	tracerProvider, err := tracing.NewTracerProvider(cfg.Tracing, "user-service")
	if err != nil {
		log.Error("Failed to initialize tracing", "error", err)
	} else {
		defer func() {
			if err := tracerProvider.Shutdown(context.Background()); err != nil {
				log.Error("Failed to shutdown tracer", "error", err)
			}
		}()
	}

	// Initialize metrics
	metricsRegistry, err := metrics.NewRegistry(cfg.Metrics, "user-service")
	if err != nil {
		log.Error("Failed to initialize metrics", "error", err)
	}
	
	// Initialize database
	db, err := database.New(cfg.Database, log)
	if err != nil {
		log.Fatal("Failed to connect to database", "error", err)
	}
	defer db.Close()

	// Run database migrations
	migrator, err := database.NewMigrator(db.DB, "./migrations", log)
	if err != nil {
		log.Fatal("Failed to create migrator", "error", err)
	}
	defer migrator.Close()

	if err := migrator.Up(); err != nil {
		log.Fatal("Failed to run database migrations", "error", err)
	}

	// Initialize Redis
	redis, err := database.NewRedis(cfg.Redis, log)
	if err != nil {
		log.Fatal("Failed to connect to Redis", "error", err)
	}
	defer redis.Close()

	// Initialize JWT service
	jwtService := auth.NewJWTService(&cfg.Auth.JWT)

	// Initialize repositories
	userRepo := repository.NewUserRepository(db, log)

	// Initialize services  
	userService := service.NewUserService(userRepo, jwtService, redis, cfg, log)

	// Initialize handlers
	userHandler := handlers.NewUserHandler(userService, jwtService, log)

	// Setup Gin router
	if cfg.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	
	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	// Health checks
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"service": "user-service",
			"timestamp": time.Now().Unix(),
		})
	})
	
	router.GET("/readiness", func(c *gin.Context) {
		// Check database connectivity
		if err := db.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error": "database connection failed",
			})
			return
		}
		
		// Check Redis connectivity
		if err := redis.HealthCheck(); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready", 
				"error": "redis connection failed",
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
			"service": "user-service",
		})
	})

	// Setup user routes
	userHandler.SetupRoutes(router)

	// Setup metrics endpoint
	router.GET("/metrics", func(c *gin.Context) {
		if metricsRegistry != nil {
			metricsRegistry.Handler().ServeHTTP(c.Writer, c.Request)
		} else {
			c.JSON(http.StatusServiceUnavailable, gin.H{"error": "metrics not available"})
		}
	})

	// Start HTTP server
	srv := &http.Server{
		Addr:         fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		log.Info("User service starting", "address", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal("Failed to start server", "error", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("Shutting down User Service...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Error("Server forced to shutdown", "error", err)
	}

	log.Info("User Service stopped")
}
