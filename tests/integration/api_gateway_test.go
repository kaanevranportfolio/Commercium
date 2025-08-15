package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaanevranportfolio/Commercium/internal/api-gateway/config"
	"github.com/kaanevranportfolio/Commercium/internal/api-gateway/server"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
	"github.com/kaanevranportfolio/Commercium/pkg/metrics"
)

func TestAPIGatewayEndpoints(t *testing.T) {
	// Create test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Create logger
	log, err := logger.New(cfg.Logger, "api-gateway-test")
	require.NoError(t, err)

	// Create metrics registry
	metricsRegistry, err := metrics.NewRegistry(cfg.Metrics, "api-gateway-test")
	require.NoError(t, err)

	// Create server
	srv, err := server.New(cfg, log, metricsRegistry)
	require.NoError(t, err)

	// Test cases
	testCases := []struct {
		name           string
		method         string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "Health check",
			method:         "GET",
			path:           "/health",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Readiness check",
			method:         "GET",
			path:           "/readiness",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Status endpoint",
			method:         "GET",
			path:           "/api/v1/status",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GraphQL endpoint",
			method:         "POST",
			path:           "/graphql",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, nil)
			w := httptest.NewRecorder()

			srv.Handler().ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code)
			if tc.expectedBody != "" {
				assert.JSONEq(t, tc.expectedBody, w.Body.String())
			} else {
				// For non-exact body checks, just verify we got a response
				assert.NotEmpty(t, w.Body.String())
			}
		})
	}
}

func TestMetricsEndpoint(t *testing.T) {
	// Create test configuration
	cfg, err := config.Load()
	require.NoError(t, err)

	// Enable metrics for this test
	cfg.Metrics.Enabled = true

	// Create logger
	log, err := logger.New(cfg.Logger, "api-gateway-test")
	require.NoError(t, err)

	// Create metrics registry
	metricsRegistry, err := metrics.NewRegistry(cfg.Metrics, "api-gateway-test")
	require.NoError(t, err)

	// Create server
	srv, err := server.New(cfg, log, metricsRegistry)
	require.NoError(t, err)

	// Test metrics endpoint
	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()

	srv.Handler().ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	// Just check that some metrics are present
	body := w.Body.String()
	assert.Contains(t, body, "# HELP")
	assert.Contains(t, body, "# TYPE")
}
