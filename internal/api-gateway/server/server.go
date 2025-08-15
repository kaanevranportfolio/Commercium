package server

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/kaanevranportfolio/Commercium/pkg/config"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
	"github.com/kaanevranportfolio/Commercium/pkg/metrics"
)

// Server represents the API Gateway server
type Server struct {
	config   *config.Config
	logger   *logger.Logger
	metrics  *metrics.Registry
	router   *gin.Engine
}

// New creates a new API Gateway server
func New(cfg *config.Config, log *logger.Logger, metricsRegistry *metrics.Registry) (*Server, error) {
	server := &Server{
		config:  cfg,
		logger:  log,
		metrics: metricsRegistry,
		router:  gin.New(),
	}

	if err := server.setupRoutes(); err != nil {
		return nil, err
	}

	return server, nil
}

// Handler returns the HTTP handler
func (s *Server) Handler() http.Handler {
	return s.router
}

// setupRoutes configures the server routes
func (s *Server) setupRoutes() error {
	// Middleware
	s.router.Use(gin.Logger())
	s.router.Use(gin.Recovery())
	s.router.Use(s.metrics.HTTPMiddleware("api-gateway"))

	// Health check endpoint
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/readiness", s.readinessCheck)

	// Metrics endpoint
	s.router.GET("/metrics", gin.WrapH(s.metrics.Handler()))

	// API routes
	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/status", s.getStatus)
	}

	// GraphQL endpoint (placeholder for now)
	s.router.POST("/graphql", s.graphqlHandler)
	s.router.GET("/playground", s.playgroundHandler)

	return nil
}

// healthCheck handles health check requests
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "healthy",
		"service": "api-gateway",
		"version": s.config.Version,
	})
}

// readinessCheck handles readiness check requests
func (s *Server) readinessCheck(c *gin.Context) {
	// TODO: Add actual readiness checks (database connectivity, etc.)
	c.JSON(http.StatusOK, gin.H{
		"status":  "ready",
		"service": "api-gateway",
		"version": s.config.Version,
	})
}

// getStatus handles status requests
func (s *Server) getStatus(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"service":     "api-gateway",
		"version":     s.config.Version,
		"environment": s.config.Environment,
		"uptime":      "calculated_uptime", // TODO: Calculate actual uptime
	})
}

// graphqlHandler handles GraphQL requests (placeholder)
func (s *Server) graphqlHandler(c *gin.Context) {
	// TODO: Implement actual GraphQL handler
	c.JSON(http.StatusOK, gin.H{
		"message": "GraphQL endpoint - implementation pending",
		"data":    nil,
	})
}

// playgroundHandler serves the GraphQL playground (placeholder)
func (s *Server) playgroundHandler(c *gin.Context) {
	// TODO: Implement GraphQL playground
	c.HTML(http.StatusOK, "playground.html", gin.H{
		"title": "GraphQL Playground",
	})
}
