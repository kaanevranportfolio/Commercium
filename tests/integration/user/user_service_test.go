package user_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kaanevranportfolio/Commercium/internal/user/handlers"
	"github.com/kaanevranportfolio/Commercium/internal/user/models"
	"github.com/kaanevranportfolio/Commercium/internal/user/repository"
	"github.com/kaanevranportfolio/Commercium/internal/user/service"
	"github.com/kaanevranportfolio/Commercium/pkg/auth"
	"github.com/kaanevranportfolio/Commercium/pkg/config"
	"github.com/kaanevranportfolio/Commercium/pkg/database"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
)

type TestSuite struct {
	handler     *handlers.UserHandler
	userService service.UserService
	userRepo    repository.UserRepository
	db          *database.DB
	redis       *database.Redis
	jwtService  *auth.JWTService
	router      *gin.Engine
	logger      *logger.Logger
}

func setupTestSuite(t *testing.T) *TestSuite {
	// Load test configuration
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:         "localhost",
			Port:         5432,
			User:         "commercium_user", 
			Password:     "commercium_password",
			Database:     "commercium_test_db",
			SSLMode:      "disable",
			MaxOpenConns: 10,
			MaxIdleConns: 5,
			MaxLifetime:  30 * time.Minute,
			MaxIdleTime:  15 * time.Minute,
		},
		Redis: config.RedisConfig{
			Host:         "localhost",
			Port:         6379,
			Password:     "",
			Database:     1, // Use different DB for tests
			PoolSize:     5,
			PoolTimeout:  30 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		},
		Auth: config.AuthConfig{
			JWT: config.JWTConfig{
				SecretKey:         "test-secret-key-for-testing-only",
				Issuer:            "commercium-test",
				Expiration:        15 * time.Minute,
				RefreshExpiration: 24 * time.Hour,
			},
		},
	}

	// Initialize logger
	log, err := logger.New(config.LoggerConfig{
		Level:  "info",
		Format: "json",
		Output: "stdout",
	}, "user-service-test")
	require.NoError(t, err)

	// Initialize database (skip if not available)
	db, err := database.New(cfg.Database, log)
	if err != nil {
		t.Skipf("Database not available for integration tests: %v", err)
	}

	// Initialize Redis (skip if not available)
	redis, err := database.NewRedis(cfg.Redis, log)
	if err != nil {
		t.Skipf("Redis not available for integration tests: %v", err)
	}

	// Initialize JWT service
	jwtService := auth.NewJWTService(&cfg.Auth.JWT)

	// Initialize repository and service
	userRepo := repository.NewUserRepository(db, log)
	userService := service.NewUserService(userRepo, jwtService, redis, cfg, log)

	// Initialize handler
	userHandler := handlers.NewUserHandler(userService, jwtService, log)

	// Setup Gin router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	userHandler.SetupRoutes(router)

	return &TestSuite{
		handler:     userHandler,
		userService: userService,
		userRepo:    userRepo,
		db:          db,
		redis:       redis,
		jwtService:  jwtService,
		router:      router,
		logger:      log,
	}
}

func (ts *TestSuite) cleanup() {
	// Clean up test data
	ctx := context.Background()
	
	// Clean up Redis test data
	ts.redis.Client.FlushDB(ctx)
	
	// Close connections
	ts.db.Close()
	ts.redis.Close()
}

func TestUserServiceIntegration(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanup()

	t.Run("User Registration and Authentication Flow", func(t *testing.T) {
		// Test user registration
		registerReq := models.CreateUserRequest{
			Username: "testuser123",
			Email:    "test@example.com",
			Password: "SecurePassword123!",
		}

		// Register user
		registerBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var registerResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &registerResp)
		require.NoError(t, err)
		assert.Equal(t, "User registered successfully", registerResp["message"])

		// Test user login
		loginReq := models.LoginRequest{
			Username: "testuser123",
			Password: "SecurePassword123!",
		}

		loginBody, _ := json.Marshal(loginReq)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var tokens models.AuthTokens
		err = json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)
		assert.NotEmpty(t, tokens.AccessToken)
		assert.NotEmpty(t, tokens.RefreshToken)
		assert.Equal(t, "Bearer", tokens.TokenType)

		// Test accessing protected endpoint
		req = httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var userResp models.UserResponse
		err = json.Unmarshal(w.Body.Bytes(), &userResp)
		require.NoError(t, err)
		assert.Equal(t, "testuser123", userResp.Username)
		assert.Equal(t, "test@example.com", userResp.Email)
		assert.True(t, userResp.IsActive)
		assert.False(t, userResp.IsVerified) // Should be false initially

		// Test token refresh
		refreshReq := map[string]string{
			"refresh_token": tokens.RefreshToken,
		}
		
		refreshBody, _ := json.Marshal(refreshReq)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", bytes.NewReader(refreshBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var newTokens models.AuthTokens
		err = json.Unmarshal(w.Body.Bytes(), &newTokens)
		require.NoError(t, err)
		assert.NotEmpty(t, newTokens.AccessToken)
		assert.NotEqual(t, tokens.AccessToken, newTokens.AccessToken) // Should be different
	})

	t.Run("User Profile Management", func(t *testing.T) {
		// Register a test user first
		registerReq := models.CreateUserRequest{
			Username: "profileuser",
			Email:    "profile@example.com",
			Password: "Password123!",
		}

		registerBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		// Login to get token
		loginReq := models.LoginRequest{
			Username: "profileuser",
			Password: "Password123!",
		}

		loginBody, _ := json.Marshal(loginReq)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var tokens models.AuthTokens
		err := json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)

		// Test profile update
		firstName := "John"
		lastName := "Doe"
		phone := "+1234567890"
		
		updateReq := models.UpdateUserRequest{
			FirstName: &firstName,
			LastName:  &lastName,
			Phone:     &phone,
		}

		updateBody, _ := json.Marshal(updateReq)
		req = httptest.NewRequest(http.MethodPut, "/api/v1/users/profile", bytes.NewReader(updateBody))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var updateResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &updateResp)
		require.NoError(t, err)
		assert.Equal(t, "Profile updated successfully", updateResp["message"])

		user := updateResp["user"].(map[string]interface{})
		assert.Equal(t, "John", user["first_name"])
		assert.Equal(t, "Doe", user["last_name"])
		assert.Equal(t, "+1234567890", user["phone"])
	})

	t.Run("Address Management", func(t *testing.T) {
		// Register and login user
		registerReq := models.CreateUserRequest{
			Username: "addressuser",
			Email:    "address@example.com",
			Password: "Password123!",
		}

		registerBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		loginReq := models.LoginRequest{
			Username: "addressuser",
			Password: "Password123!",
		}

		loginBody, _ := json.Marshal(loginReq)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)
		require.Equal(t, http.StatusOK, w.Code)

		var tokens models.AuthTokens
		err := json.Unmarshal(w.Body.Bytes(), &tokens)
		require.NoError(t, err)

		// Create address
		address := models.UserAddress{
			Type:         "shipping",
			FirstName:    "John",
			LastName:     "Doe",
			AddressLine1: "123 Main St",
			City:         "New York",
			PostalCode:   "10001",
			Country:      "US",
			IsDefault:    true,
		}

		addressBody, _ := json.Marshal(address)
		req = httptest.NewRequest(http.MethodPost, "/api/v1/users/addresses", bytes.NewReader(addressBody))
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
		req.Header.Set("Content-Type", "application/json")
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var createResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &createResp)
		require.NoError(t, err)
		assert.Equal(t, "Address created successfully", createResp["message"])

		// Get addresses
		req = httptest.NewRequest(http.MethodGet, "/api/v1/users/addresses", nil)
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tokens.AccessToken))
		w = httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var getResp map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &getResp)
		require.NoError(t, err)

		addresses := getResp["addresses"].([]interface{})
		assert.Len(t, addresses, 1)

		firstAddress := addresses[0].(map[string]interface{})
		assert.Equal(t, "shipping", firstAddress["type"])
		assert.Equal(t, "123 Main St", firstAddress["address_line1"])
		assert.Equal(t, true, firstAddress["is_default"])
	})
}

func TestUserServiceErrors(t *testing.T) {
	ts := setupTestSuite(t)
	defer ts.cleanup()

	t.Run("Registration Validation Errors", func(t *testing.T) {
		// Test missing required fields
		registerReq := models.CreateUserRequest{
			Username: "", // Missing username
			Email:    "invalid-email", // Invalid email
			Password: "123", // Too short password
		}

		registerBody, _ := json.Marshal(registerReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/register", bytes.NewReader(registerBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("Login with Invalid Credentials", func(t *testing.T) {
		loginReq := models.LoginRequest{
			Username: "nonexistentuser",
			Password: "wrongpassword",
		}

		loginBody, _ := json.Marshal(loginReq)
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", bytes.NewReader(loginBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "Invalid credentials", errorResp["error"])
	})

	t.Run("Unauthorized Access", func(t *testing.T) {
		// Try to access protected endpoint without token
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "Authorization header required", errorResp["error"])
	})

	t.Run("Invalid Token", func(t *testing.T) {
		// Try to access protected endpoint with invalid token
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users/profile", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		ts.router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)

		var errorResp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &errorResp)
		require.NoError(t, err)
		assert.Equal(t, "Invalid token", errorResp["error"])
	})
}
