package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/kaanevranportfolio/Commercium/internal/user/models"
	"github.com/kaanevranportfolio/Commercium/internal/user/service"
	"github.com/kaanevranportfolio/Commercium/pkg/auth"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService service.UserService
	jwtService  *auth.JWTService
	logger      *logger.Logger
}

// NewUserHandler creates a new user handler
func NewUserHandler(userService service.UserService, jwtService *auth.JWTService, logger *logger.Logger) *UserHandler {
	return &UserHandler{
		userService: userService,
		jwtService:  jwtService,
		logger:      logger,
	}
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	var req models.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.Register(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Registration failed", "error", err)
		
		// Check for specific errors
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Registration failed"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": user,
	})
}

// Login handles user authentication
func (h *UserHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	tokens, err := h.userService.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Login failed", "error", err)
		
		if strings.Contains(err.Error(), "credentials") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}
		
		if strings.Contains(err.Error(), "deactivated") {
			c.JSON(http.StatusForbidden, gin.H{"error": "Account is deactivated"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Login failed"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// RefreshToken handles token refresh
func (h *UserHandler) RefreshToken(c *gin.Context) {
	var req struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	tokens, err := h.userService.RefreshToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		h.logger.Error("Token refresh failed", "error", err)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
		return
	}

	c.JSON(http.StatusOK, tokens)
}

// GetProfile retrieves the user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	user, err := h.userService.GetProfile(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get user profile", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get profile"})
		return
	}

	c.JSON(http.StatusOK, user)
}

// UpdateProfile updates the user's profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	user, err := h.userService.UpdateProfile(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Failed to update user profile", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": user,
	})
}

// ChangePassword handles password change requests
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err := h.userService.ChangePassword(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Password change failed", "error", err, "user_id", userID)
		
		if strings.Contains(err.Error(), "incorrect") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Current password is incorrect"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to change password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password changed successfully"})
}

// ForgotPassword handles forgot password requests
func (h *UserHandler) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err := h.userService.ForgotPassword(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Forgot password failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process request"})
		return
	}

	// Always return success for security (don't reveal if email exists)
	c.JSON(http.StatusOK, gin.H{
		"message": "If the email exists, a password reset link has been sent",
	})
}

// ResetPassword handles password reset requests
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	err := h.userService.ResetPassword(c.Request.Context(), &req)
	if err != nil {
		h.logger.Error("Password reset failed", "error", err)
		
		if strings.Contains(err.Error(), "invalid or expired") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired reset token"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Password reset successfully"})
}

// VerifyEmail handles email verification
func (h *UserHandler) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Verification token is required"})
		return
	}

	err := h.userService.VerifyEmail(c.Request.Context(), token)
	if err != nil {
		h.logger.Error("Email verification failed", "error", err)
		
		if strings.Contains(err.Error(), "invalid or expired") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid or expired verification token"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email verified successfully"})
}

// ResendEmailVerification resends email verification
func (h *UserHandler) ResendEmailVerification(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	err := h.userService.ResendEmailVerification(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to resend email verification", "error", err, "user_id", userID)
		
		if strings.Contains(err.Error(), "already verified") {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Email is already verified"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to resend verification email"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Verification email sent"})
}

// CreateAddress creates a new user address
func (h *UserHandler) CreateAddress(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var address models.UserAddress
	if err := c.ShouldBindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	createdAddress, err := h.userService.CreateAddress(c.Request.Context(), userID, &address)
	if err != nil {
		h.logger.Error("Failed to create address", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create address"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Address created successfully",
		"address": createdAddress,
	})
}

// GetAddresses retrieves user addresses
func (h *UserHandler) GetAddresses(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addresses, err := h.userService.GetAddresses(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get addresses", "error", err, "user_id", userID)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get addresses"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"addresses": addresses})
}

// UpdateAddress updates a user address
func (h *UserHandler) UpdateAddress(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addressIDStr := c.Param("id")
	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	var address models.UserAddress
	if err := c.ShouldBindJSON(&address); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request data",
			"details": err.Error(),
		})
		return
	}

	updatedAddress, err := h.userService.UpdateAddress(c.Request.Context(), userID, addressID, &address)
	if err != nil {
		h.logger.Error("Failed to update address", "error", err, "user_id", userID, "address_id", addressID)
		
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not belong") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Address updated successfully",
		"address": updatedAddress,
	})
}

// DeleteAddress deletes a user address
func (h *UserHandler) DeleteAddress(c *gin.Context) {
	userID := h.getUserIDFromContext(c)
	if userID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	addressIDStr := c.Param("id")
	addressID, err := uuid.Parse(addressIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid address ID"})
		return
	}

	err = h.userService.DeleteAddress(c.Request.Context(), userID, addressID)
	if err != nil {
		h.logger.Error("Failed to delete address", "error", err, "user_id", userID, "address_id", addressID)
		
		if strings.Contains(err.Error(), "not found") || strings.Contains(err.Error(), "does not belong") {
			c.JSON(http.StatusNotFound, gin.H{"error": "Address not found"})
			return
		}
		
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete address"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Address deleted successfully"})
}

// AuthMiddleware validates JWT tokens
func (h *UserHandler) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Extract token from "Bearer <token>"
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		token := parts[1]
		claims, err := h.jwtService.ValidateAccessToken(token)
		if err != nil {
			h.logger.Error("Token validation failed", "error", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Store user information in context
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_username", claims.Username)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// getUserIDFromContext extracts user ID from gin context
func (h *UserHandler) getUserIDFromContext(c *gin.Context) uuid.UUID {
	userID, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil
	}
	
	id, ok := userID.(uuid.UUID)
	if !ok {
		return uuid.Nil
	}
	
	return id
}

// SetupRoutes sets up the user routes
func (h *UserHandler) SetupRoutes(r *gin.Engine) {
	// Public routes
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/register", h.Register)
		auth.POST("/login", h.Login)
		auth.POST("/refresh", h.RefreshToken)
		auth.POST("/forgot-password", h.ForgotPassword)
		auth.POST("/reset-password", h.ResetPassword)
		auth.GET("/verify-email", h.VerifyEmail)
	}

	// Protected routes
	users := r.Group("/api/v1/users")
	users.Use(h.AuthMiddleware())
	{
		users.GET("/profile", h.GetProfile)
		users.PUT("/profile", h.UpdateProfile)
		users.POST("/change-password", h.ChangePassword)
		users.POST("/resend-verification", h.ResendEmailVerification)
		
		// Address management
		users.POST("/addresses", h.CreateAddress)
		users.GET("/addresses", h.GetAddresses)
		users.PUT("/addresses/:id", h.UpdateAddress)
		users.DELETE("/addresses/:id", h.DeleteAddress)
	}
}

// HealthCheck provides a health check endpoint
func (h *UserHandler) HealthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "healthy",
		"service": "user-service",
	})
}
