package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"github.com/kaanevranportfolio/Commercium/internal/user/models"
	"github.com/kaanevranportfolio/Commercium/internal/user/repository"
	"github.com/kaanevranportfolio/Commercium/pkg/auth"
	"github.com/kaanevranportfolio/Commercium/pkg/config"
	"github.com/kaanevranportfolio/Commercium/pkg/database"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
)

// UserService defines the interface for user business logic
type UserService interface {
	Register(ctx context.Context, req *models.CreateUserRequest) (*models.UserResponse, error)
	Login(ctx context.Context, req *models.LoginRequest) (*models.AuthTokens, error)
	RefreshToken(ctx context.Context, refreshToken string) (*models.AuthTokens, error)
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateUserRequest) (*models.UserResponse, error)
	ChangePassword(ctx context.Context, userID uuid.UUID, req *models.ChangePasswordRequest) error
	ForgotPassword(ctx context.Context, req *models.ForgotPasswordRequest) error
	ResetPassword(ctx context.Context, req *models.ResetPasswordRequest) error
	VerifyEmail(ctx context.Context, token string) error
	ResendEmailVerification(ctx context.Context, userID uuid.UUID) error
	
	// Address management
	CreateAddress(ctx context.Context, userID uuid.UUID, address *models.UserAddress) (*models.UserAddress, error)
	GetAddresses(ctx context.Context, userID uuid.UUID) ([]*models.UserAddress, error)
	UpdateAddress(ctx context.Context, userID uuid.UUID, addressID uuid.UUID, address *models.UserAddress) (*models.UserAddress, error)
	DeleteAddress(ctx context.Context, userID uuid.UUID, addressID uuid.UUID) error
}

// userService implements the UserService interface
type userService struct {
	repo       repository.UserRepository
	jwtService *auth.JWTService
	redis      *database.Redis
	config     *config.Config
	logger     *logger.Logger
}

// NewUserService creates a new user service
func NewUserService(
	repo repository.UserRepository,
	jwtService *auth.JWTService,
	redis *database.Redis,
	config *config.Config,
	logger *logger.Logger,
) UserService {
	return &userService{
		repo:       repo,
		jwtService: jwtService,
		redis:      redis,
		config:     config,
		logger:     logger,
	}
}

// Register creates a new user account
func (s *userService) Register(ctx context.Context, req *models.CreateUserRequest) (*models.UserResponse, error) {
	// Check if user already exists
	existingUser, err := s.repo.GetByEmail(ctx, req.Email)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with email %s already exists", req.Email)
	}

	existingUser, err = s.repo.GetByUsername(ctx, req.Username)
	if err == nil && existingUser != nil {
		return nil, fmt.Errorf("user with username %s already exists", req.Username)
	}

	// Hash password
	hashedPassword, err := s.hashPassword(req.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Create user
	user := &models.User{
		ID:           uuid.New(),
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: hashedPassword,
		FirstName:    req.FirstName,
		LastName:     req.LastName,
		Phone:        req.Phone,
		IsActive:     true,
		IsVerified:   false,
		Role:         "customer",
	}

	err = s.repo.Create(ctx, user)
	if err != nil {
		s.logger.Error("Failed to create user", "error", err, "email", req.Email)
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create default profile
	profile := &models.UserProfile{
		UserID:      user.ID,
		Preferences: make(map[string]interface{}),
	}
	
	err = s.repo.CreateProfile(ctx, profile)
	if err != nil {
		s.logger.Warn("Failed to create user profile", "error", err, "user_id", user.ID)
	}

	// Generate email verification token
	err = s.generateEmailVerificationToken(ctx, user.ID)
	if err != nil {
		s.logger.Warn("Failed to generate email verification token", "error", err, "user_id", user.ID)
	}

	s.logger.Info("User registered successfully", "user_id", user.ID, "email", user.Email)
	return user.ToResponse(), nil
}

// Login authenticates a user and returns tokens
func (s *userService) Login(ctx context.Context, req *models.LoginRequest) (*models.AuthTokens, error) {
	// Get user by username or email
	var user *models.User
	var err error

	// Try to find by email first, then by username
	user, err = s.repo.GetByEmail(ctx, req.Username)
	if err != nil {
		user, err = s.repo.GetByUsername(ctx, req.Username)
		if err != nil {
			return nil, fmt.Errorf("invalid credentials")
		}
	}

	// Check if user is active
	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	// Verify password
	if !s.verifyPassword(req.Password, user.PasswordHash) {
		return nil, fmt.Errorf("invalid credentials")
	}

	// Generate tokens
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Email, user.Username, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate tokens", "error", err, "user_id", user.ID)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update last login
	err = s.repo.UpdateLastLogin(ctx, user.ID)
	if err != nil {
		s.logger.Warn("Failed to update last login", "error", err, "user_id", user.ID)
	}

	// Cache refresh token in Redis
	refreshKey := fmt.Sprintf("refresh_token:%s", user.ID.String())
	err = s.redis.SetWithExpiration(ctx, refreshKey, tokenPair.RefreshToken, s.config.Auth.JWT.RefreshExpiration)
	if err != nil {
		s.logger.Warn("Failed to cache refresh token", "error", err, "user_id", user.ID)
	}

	s.logger.Info("User logged in successfully", "user_id", user.ID, "email", user.Email)
	
	return &models.AuthTokens{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// RefreshToken generates new tokens using a refresh token
func (s *userService) RefreshToken(ctx context.Context, refreshToken string) (*models.AuthTokens, error) {
	// Validate refresh token
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	userID, err := uuid.Parse(claims.Subject)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID in token: %w", err)
	}

	// Check if refresh token exists in Redis
	refreshKey := fmt.Sprintf("refresh_token:%s", userID.String())
	cachedToken, err := s.redis.GetString(ctx, refreshKey)
	if err != nil || cachedToken != refreshToken {
		return nil, fmt.Errorf("refresh token not found or expired")
	}

	// Get user details
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("account is deactivated")
	}

	// Generate new token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(user.ID, user.Email, user.Username, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate tokens", "error", err, "user_id", user.ID)
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	// Update cached refresh token
	err = s.redis.SetWithExpiration(ctx, refreshKey, tokenPair.RefreshToken, s.config.Auth.JWT.RefreshExpiration)
	if err != nil {
		s.logger.Warn("Failed to update cached refresh token", "error", err, "user_id", user.ID)
	}

	return &models.AuthTokens{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		TokenType:    tokenPair.TokenType,
		ExpiresIn:    tokenPair.ExpiresIn,
	}, nil
}

// GetProfile retrieves a user's profile
func (s *userService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return user.ToResponse(), nil
}

// UpdateProfile updates a user's profile
func (s *userService) UpdateProfile(ctx context.Context, userID uuid.UUID, req *models.UpdateUserRequest) (*models.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	// Update fields
	if req.FirstName != nil {
		user.FirstName = req.FirstName
	}
	if req.LastName != nil {
		user.LastName = req.LastName
	}
	if req.Phone != nil {
		user.Phone = req.Phone
	}

	err = s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to update user", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to update user: %w", err)
	}

	s.logger.Info("User profile updated", "user_id", userID)
	return user.ToResponse(), nil
}

// ChangePassword changes a user's password
func (s *userService) ChangePassword(ctx context.Context, userID uuid.UUID, req *models.ChangePasswordRequest) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Verify current password
	if !s.verifyPassword(req.CurrentPassword, user.PasswordHash) {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	user.PasswordHash = hashedPassword
	err = s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to update user password", "error", err, "user_id", userID)
		return fmt.Errorf("failed to update password: %w", err)
	}

	s.logger.Info("User password changed", "user_id", userID)
	return nil
}

// ForgotPassword generates a password reset token
func (s *userService) ForgotPassword(ctx context.Context, req *models.ForgotPasswordRequest) error {
	user, err := s.repo.GetByEmail(ctx, req.Email)
	if err != nil {
		// Don't reveal if user exists for security
		s.logger.Info("Password reset requested for non-existent email", "email", req.Email)
		return nil
	}

	// Generate secure token
	token, err := s.generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	// Create password reset token
	resetToken := &models.PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour), // 1 hour expiration
	}

	err = s.repo.CreatePasswordResetToken(ctx, resetToken)
	if err != nil {
		s.logger.Error("Failed to create password reset token", "error", err, "user_id", user.ID)
		return fmt.Errorf("failed to create reset token: %w", err)
	}

	// TODO: Send email with reset link
	s.logger.Info("Password reset token generated", "user_id", user.ID, "email", user.Email)
	return nil
}

// ResetPassword resets a user's password using a token
func (s *userService) ResetPassword(ctx context.Context, req *models.ResetPasswordRequest) error {
	// Get and validate token
	resetToken, err := s.repo.GetPasswordResetToken(ctx, req.Token)
	if err != nil {
		return fmt.Errorf("invalid or expired reset token")
	}

	// Get user
	user, err := s.repo.GetByID(ctx, resetToken.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Hash new password
	hashedPassword, err := s.hashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	// Update password
	user.PasswordHash = hashedPassword
	err = s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to reset user password", "error", err, "user_id", user.ID)
		return fmt.Errorf("failed to reset password: %w", err)
	}

	// Mark token as used
	err = s.repo.MarkPasswordResetTokenUsed(ctx, resetToken.ID)
	if err != nil {
		s.logger.Warn("Failed to mark reset token as used", "error", err, "token_id", resetToken.ID)
	}

	s.logger.Info("Password reset successfully", "user_id", user.ID)
	return nil
}

// VerifyEmail verifies a user's email using a token
func (s *userService) VerifyEmail(ctx context.Context, token string) error {
	// Get and validate token
	verificationToken, err := s.repo.GetEmailVerificationToken(ctx, token)
	if err != nil {
		return fmt.Errorf("invalid or expired verification token")
	}

	// Get user
	user, err := s.repo.GetByID(ctx, verificationToken.UserID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	// Mark user as verified
	user.IsVerified = true
	err = s.repo.Update(ctx, user)
	if err != nil {
		s.logger.Error("Failed to verify user email", "error", err, "user_id", user.ID)
		return fmt.Errorf("failed to verify email: %w", err)
	}

	// Mark token as used
	err = s.repo.MarkEmailVerificationTokenUsed(ctx, verificationToken.ID)
	if err != nil {
		s.logger.Warn("Failed to mark verification token as used", "error", err, "token_id", verificationToken.ID)
	}

	s.logger.Info("Email verified successfully", "user_id", user.ID, "email", user.Email)
	return nil
}

// ResendEmailVerification generates a new email verification token
func (s *userService) ResendEmailVerification(ctx context.Context, userID uuid.UUID) error {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.IsVerified {
		return fmt.Errorf("email is already verified")
	}

	err = s.generateEmailVerificationToken(ctx, userID)
	if err != nil {
		return fmt.Errorf("failed to generate verification token: %w", err)
	}

	return nil
}

// CreateAddress creates a new user address
func (s *userService) CreateAddress(ctx context.Context, userID uuid.UUID, address *models.UserAddress) (*models.UserAddress, error) {
	address.ID = uuid.New()
	address.UserID = userID

	err := s.repo.CreateAddress(ctx, address)
	if err != nil {
		s.logger.Error("Failed to create user address", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to create address: %w", err)
	}

	s.logger.Info("Address created", "user_id", userID, "address_id", address.ID)
	return address, nil
}

// GetAddresses retrieves all user addresses
func (s *userService) GetAddresses(ctx context.Context, userID uuid.UUID) ([]*models.UserAddress, error) {
	addresses, err := s.repo.GetAddresses(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get addresses: %w", err)
	}

	return addresses, nil
}

// UpdateAddress updates a user address
func (s *userService) UpdateAddress(ctx context.Context, userID uuid.UUID, addressID uuid.UUID, address *models.UserAddress) (*models.UserAddress, error) {
	// Verify address belongs to user
	existingAddress, err := s.repo.GetAddressByID(ctx, addressID)
	if err != nil {
		return nil, fmt.Errorf("address not found: %w", err)
	}

	if existingAddress.UserID != userID {
		return nil, fmt.Errorf("address does not belong to user")
	}

	// Update address
	address.ID = addressID
	address.UserID = userID
	
	err = s.repo.UpdateAddress(ctx, address)
	if err != nil {
		s.logger.Error("Failed to update user address", "error", err, "address_id", addressID)
		return nil, fmt.Errorf("failed to update address: %w", err)
	}

	s.logger.Info("Address updated", "user_id", userID, "address_id", addressID)
	return address, nil
}

// DeleteAddress deletes a user address
func (s *userService) DeleteAddress(ctx context.Context, userID uuid.UUID, addressID uuid.UUID) error {
	// Verify address belongs to user
	address, err := s.repo.GetAddressByID(ctx, addressID)
	if err != nil {
		return fmt.Errorf("address not found: %w", err)
	}

	if address.UserID != userID {
		return fmt.Errorf("address does not belong to user")
	}

	err = s.repo.DeleteAddress(ctx, addressID)
	if err != nil {
		s.logger.Error("Failed to delete user address", "error", err, "address_id", addressID)
		return fmt.Errorf("failed to delete address: %w", err)
	}

	s.logger.Info("Address deleted", "user_id", userID, "address_id", addressID)
	return nil
}

// hashPassword hashes a password using bcrypt
func (s *userService) hashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// verifyPassword verifies a password against its hash
func (s *userService) verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// generateSecureToken generates a cryptographically secure token
func (s *userService) generateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// generateEmailVerificationToken generates and stores an email verification token
func (s *userService) generateEmailVerificationToken(ctx context.Context, userID uuid.UUID) error {
	token, err := s.generateSecureToken(32)
	if err != nil {
		return fmt.Errorf("failed to generate token: %w", err)
	}

	verificationToken := &models.EmailVerificationToken{
		ID:        uuid.New(),
		UserID:    userID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour), // 24 hours expiration
	}

	err = s.repo.CreateEmailVerificationToken(ctx, verificationToken)
	if err != nil {
		return fmt.Errorf("failed to create verification token: %w", err)
	}

	// TODO: Send verification email
	s.logger.Info("Email verification token generated", "user_id", userID)
	return nil
}
