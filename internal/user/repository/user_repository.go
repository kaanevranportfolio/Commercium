package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"

	"github.com/kaanevranportfolio/Commercium/internal/user/models"
	"github.com/kaanevranportfolio/Commercium/pkg/database"
	"github.com/kaanevranportfolio/Commercium/pkg/logger"
)

// UserRepository defines the interface for user data operations
type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	GetByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	GetByEmail(ctx context.Context, email string) (*models.User, error)
	GetByUsername(ctx context.Context, username string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, limit, offset int) ([]*models.User, error)
	UpdateLastLogin(ctx context.Context, userID uuid.UUID) error
	
	// Profile operations
	CreateProfile(ctx context.Context, profile *models.UserProfile) error
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error)
	UpdateProfile(ctx context.Context, profile *models.UserProfile) error
	
	// Address operations
	CreateAddress(ctx context.Context, address *models.UserAddress) error
	GetAddresses(ctx context.Context, userID uuid.UUID) ([]*models.UserAddress, error)
	GetAddressByID(ctx context.Context, id uuid.UUID) (*models.UserAddress, error)
	UpdateAddress(ctx context.Context, address *models.UserAddress) error
	DeleteAddress(ctx context.Context, id uuid.UUID) error
	
	// Token operations
	CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error
	GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error)
	MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error
	
	CreateEmailVerificationToken(ctx context.Context, token *models.EmailVerificationToken) error
	GetEmailVerificationToken(ctx context.Context, token string) (*models.EmailVerificationToken, error)
	MarkEmailVerificationTokenUsed(ctx context.Context, tokenID uuid.UUID) error
}

// userRepository implements the UserRepository interface
type userRepository struct {
	db     *database.DB
	logger *logger.Logger
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *database.DB, logger *logger.Logger) UserRepository {
	return &userRepository{
		db:     db,
		logger: logger,
	}
}

// Create creates a new user
func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, first_name, last_name, phone, role)
		VALUES (:id, :username, :email, :password_hash, :first_name, :last_name, :phone, :role)
		RETURNING created_at, updated_at`
	
	rows, err := r.db.NamedQueryContext(ctx, query, user)
	if err != nil {
		r.logger.Error("Failed to create user", "error", err)
		return fmt.Errorf("failed to create user: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan timestamps: %w", err)
		}
	}
	
	return nil
}

// GetByID retrieves a user by ID
func (r *userRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, phone, 
		       is_active, is_verified, role, created_at, updated_at, last_login_at
		FROM users 
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, user, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.Error("Failed to get user by ID", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// GetByEmail retrieves a user by email
func (r *userRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, phone, 
		       is_active, is_verified, role, created_at, updated_at, last_login_at
		FROM users 
		WHERE email = $1`
	
	err := r.db.GetContext(ctx, user, query, email)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.Error("Failed to get user by email", "error", err, "email", email)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// GetByUsername retrieves a user by username
func (r *userRepository) GetByUsername(ctx context.Context, username string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, phone, 
		       is_active, is_verified, role, created_at, updated_at, last_login_at
		FROM users 
		WHERE username = $1`
	
	err := r.db.GetContext(ctx, user, query, username)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		r.logger.Error("Failed to get user by username", "error", err, "username", username)
		return nil, fmt.Errorf("failed to get user: %w", err)
	}
	
	return user, nil
}

// Update updates a user
func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	query := `
		UPDATE users 
		SET first_name = :first_name, last_name = :last_name, phone = :phone, 
		    is_active = :is_active, is_verified = :is_verified, updated_at = NOW()
		WHERE id = :id`
	
	result, err := r.db.NamedExecContext(ctx, query, user)
	if err != nil {
		r.logger.Error("Failed to update user", "error", err, "id", user.ID)
		return fmt.Errorf("failed to update user: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// Delete soft deletes a user
func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE users SET is_active = false, updated_at = NOW() WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete user", "error", err, "id", id)
		return fmt.Errorf("failed to delete user: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}
	
	return nil
}

// List retrieves a paginated list of users
func (r *userRepository) List(ctx context.Context, limit, offset int) ([]*models.User, error) {
	users := []*models.User{}
	query := `
		SELECT id, username, email, password_hash, first_name, last_name, phone, 
		       is_active, is_verified, role, created_at, updated_at, last_login_at
		FROM users 
		WHERE is_active = true
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`
	
	err := r.db.SelectContext(ctx, &users, query, limit, offset)
	if err != nil {
		r.logger.Error("Failed to list users", "error", err)
		return nil, fmt.Errorf("failed to list users: %w", err)
	}
	
	return users, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *userRepository) UpdateLastLogin(ctx context.Context, userID uuid.UUID) error {
	query := `UPDATE users SET last_login_at = NOW() WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, userID)
	if err != nil {
		r.logger.Error("Failed to update last login", "error", err, "user_id", userID)
		return fmt.Errorf("failed to update last login: %w", err)
	}
	
	return nil
}

// CreateProfile creates a user profile
func (r *userRepository) CreateProfile(ctx context.Context, profile *models.UserProfile) error {
	query := `
		INSERT INTO user_profiles (user_id, avatar_url, date_of_birth, gender, bio, preferences)
		VALUES (:user_id, :avatar_url, :date_of_birth, :gender, :bio, :preferences)
		RETURNING created_at, updated_at`
	
	rows, err := r.db.NamedQueryContext(ctx, query, profile)
	if err != nil {
		r.logger.Error("Failed to create user profile", "error", err)
		return fmt.Errorf("failed to create user profile: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&profile.CreatedAt, &profile.UpdatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan timestamps: %w", err)
		}
	}
	
	return nil
}

// GetProfile retrieves a user profile
func (r *userRepository) GetProfile(ctx context.Context, userID uuid.UUID) (*models.UserProfile, error) {
	profile := &models.UserProfile{}
	query := `
		SELECT user_id, avatar_url, date_of_birth, gender, bio, preferences, created_at, updated_at
		FROM user_profiles 
		WHERE user_id = $1`
	
	err := r.db.GetContext(ctx, profile, query, userID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user profile not found")
		}
		r.logger.Error("Failed to get user profile", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}
	
	return profile, nil
}

// UpdateProfile updates a user profile
func (r *userRepository) UpdateProfile(ctx context.Context, profile *models.UserProfile) error {
	query := `
		UPDATE user_profiles 
		SET avatar_url = :avatar_url, date_of_birth = :date_of_birth, gender = :gender, 
		    bio = :bio, preferences = :preferences, updated_at = NOW()
		WHERE user_id = :user_id`
	
	result, err := r.db.NamedExecContext(ctx, query, profile)
	if err != nil {
		r.logger.Error("Failed to update user profile", "error", err, "user_id", profile.UserID)
		return fmt.Errorf("failed to update user profile: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("user profile not found")
	}
	
	return nil
}

// CreateAddress creates a user address
func (r *userRepository) CreateAddress(ctx context.Context, address *models.UserAddress) error {
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		// If this is being set as default, unset other default addresses
		if address.IsDefault {
			_, err := tx.ExecContext(ctx, 
				`UPDATE user_addresses SET is_default = false WHERE user_id = $1 AND type = $2`, 
				address.UserID, address.Type)
			if err != nil {
				return fmt.Errorf("failed to unset default addresses: %w", err)
			}
		}
		
		query := `
			INSERT INTO user_addresses 
			(id, user_id, type, first_name, last_name, company, address_line1, address_line2, 
			 city, state, postal_code, country, phone, is_default)
			VALUES (:id, :user_id, :type, :first_name, :last_name, :company, :address_line1, 
			        :address_line2, :city, :state, :postal_code, :country, :phone, :is_default)
			RETURNING created_at, updated_at`
		
		stmt, err := tx.PrepareNamedContext(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to prepare statement: %w", err)
		}
		defer stmt.Close()
		
		rows, err := stmt.QueryContext(ctx, address)
		if err != nil {
			return fmt.Errorf("failed to create address: %w", err)
		}
		defer rows.Close()
		
		if rows.Next() {
			err = rows.Scan(&address.CreatedAt, &address.UpdatedAt)
			if err != nil {
				return fmt.Errorf("failed to scan timestamps: %w", err)
			}
		}
		
		return nil
	})
}

// GetAddresses retrieves all addresses for a user
func (r *userRepository) GetAddresses(ctx context.Context, userID uuid.UUID) ([]*models.UserAddress, error) {
	addresses := []*models.UserAddress{}
	query := `
		SELECT id, user_id, type, first_name, last_name, company, address_line1, address_line2,
		       city, state, postal_code, country, phone, is_default, created_at, updated_at
		FROM user_addresses 
		WHERE user_id = $1
		ORDER BY is_default DESC, created_at DESC`
	
	err := r.db.SelectContext(ctx, &addresses, query, userID)
	if err != nil {
		r.logger.Error("Failed to get user addresses", "error", err, "user_id", userID)
		return nil, fmt.Errorf("failed to get user addresses: %w", err)
	}
	
	return addresses, nil
}

// GetAddressByID retrieves an address by ID
func (r *userRepository) GetAddressByID(ctx context.Context, id uuid.UUID) (*models.UserAddress, error) {
	address := &models.UserAddress{}
	query := `
		SELECT id, user_id, type, first_name, last_name, company, address_line1, address_line2,
		       city, state, postal_code, country, phone, is_default, created_at, updated_at
		FROM user_addresses 
		WHERE id = $1`
	
	err := r.db.GetContext(ctx, address, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("address not found")
		}
		r.logger.Error("Failed to get address by ID", "error", err, "id", id)
		return nil, fmt.Errorf("failed to get address: %w", err)
	}
	
	return address, nil
}

// UpdateAddress updates a user address
func (r *userRepository) UpdateAddress(ctx context.Context, address *models.UserAddress) error {
	return r.db.Transaction(func(tx *sqlx.Tx) error {
		// If this is being set as default, unset other default addresses
		if address.IsDefault {
			_, err := tx.ExecContext(ctx, 
				`UPDATE user_addresses SET is_default = false WHERE user_id = $1 AND type = $2 AND id != $3`, 
				address.UserID, address.Type, address.ID)
			if err != nil {
				return fmt.Errorf("failed to unset default addresses: %w", err)
			}
		}
		
		query := `
			UPDATE user_addresses 
			SET first_name = :first_name, last_name = :last_name, company = :company,
			    address_line1 = :address_line1, address_line2 = :address_line2, city = :city,
			    state = :state, postal_code = :postal_code, country = :country, phone = :phone,
			    is_default = :is_default, updated_at = NOW()
			WHERE id = :id`
		
		result, err := tx.NamedExecContext(ctx, query, address)
		if err != nil {
			return fmt.Errorf("failed to update address: %w", err)
		}
		
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("failed to get rows affected: %w", err)
		}
		
		if rowsAffected == 0 {
			return fmt.Errorf("address not found")
		}
		
		return nil
	})
}

// DeleteAddress deletes a user address
func (r *userRepository) DeleteAddress(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_addresses WHERE id = $1`
	
	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		r.logger.Error("Failed to delete address", "error", err, "id", id)
		return fmt.Errorf("failed to delete address: %w", err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}
	
	if rowsAffected == 0 {
		return fmt.Errorf("address not found")
	}
	
	return nil
}

// CreatePasswordResetToken creates a password reset token
func (r *userRepository) CreatePasswordResetToken(ctx context.Context, token *models.PasswordResetToken) error {
	query := `
		INSERT INTO password_reset_tokens (id, user_id, token, expires_at)
		VALUES (:id, :user_id, :token, :expires_at)
		RETURNING created_at`
	
	rows, err := r.db.NamedQueryContext(ctx, query, token)
	if err != nil {
		r.logger.Error("Failed to create password reset token", "error", err)
		return fmt.Errorf("failed to create password reset token: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&token.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan timestamp: %w", err)
		}
	}
	
	return nil
}

// GetPasswordResetToken retrieves a password reset token
func (r *userRepository) GetPasswordResetToken(ctx context.Context, token string) (*models.PasswordResetToken, error) {
	resetToken := &models.PasswordResetToken{}
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM password_reset_tokens 
		WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()`
	
	err := r.db.GetContext(ctx, resetToken, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired token")
		}
		r.logger.Error("Failed to get password reset token", "error", err)
		return nil, fmt.Errorf("failed to get password reset token: %w", err)
	}
	
	return resetToken, nil
}

// MarkPasswordResetTokenUsed marks a password reset token as used
func (r *userRepository) MarkPasswordResetTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `UPDATE password_reset_tokens SET used_at = NOW() WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		r.logger.Error("Failed to mark password reset token as used", "error", err, "token_id", tokenID)
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	
	return nil
}

// CreateEmailVerificationToken creates an email verification token
func (r *userRepository) CreateEmailVerificationToken(ctx context.Context, token *models.EmailVerificationToken) error {
	query := `
		INSERT INTO email_verification_tokens (id, user_id, token, expires_at)
		VALUES (:id, :user_id, :token, :expires_at)
		RETURNING created_at`
	
	rows, err := r.db.NamedQueryContext(ctx, query, token)
	if err != nil {
		r.logger.Error("Failed to create email verification token", "error", err)
		return fmt.Errorf("failed to create email verification token: %w", err)
	}
	defer rows.Close()
	
	if rows.Next() {
		err = rows.Scan(&token.CreatedAt)
		if err != nil {
			return fmt.Errorf("failed to scan timestamp: %w", err)
		}
	}
	
	return nil
}

// GetEmailVerificationToken retrieves an email verification token
func (r *userRepository) GetEmailVerificationToken(ctx context.Context, token string) (*models.EmailVerificationToken, error) {
	verificationToken := &models.EmailVerificationToken{}
	query := `
		SELECT id, user_id, token, expires_at, used_at, created_at
		FROM email_verification_tokens 
		WHERE token = $1 AND used_at IS NULL AND expires_at > NOW()`
	
	err := r.db.GetContext(ctx, verificationToken, query, token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("invalid or expired token")
		}
		r.logger.Error("Failed to get email verification token", "error", err)
		return nil, fmt.Errorf("failed to get email verification token: %w", err)
	}
	
	return verificationToken, nil
}

// MarkEmailVerificationTokenUsed marks an email verification token as used
func (r *userRepository) MarkEmailVerificationTokenUsed(ctx context.Context, tokenID uuid.UUID) error {
	query := `UPDATE email_verification_tokens SET used_at = NOW() WHERE id = $1`
	
	_, err := r.db.ExecContext(ctx, query, tokenID)
	if err != nil {
		r.logger.Error("Failed to mark email verification token as used", "error", err, "token_id", tokenID)
		return fmt.Errorf("failed to mark token as used: %w", err)
	}
	
	return nil
}
