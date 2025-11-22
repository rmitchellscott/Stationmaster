package database

import (
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserService provides user-related database operations
type UserService struct {
	db *gorm.DB
}

// UserCacheInvalidateFunc is a function type for invalidating user cache
type UserCacheInvalidateFunc func(userID uuid.UUID) error

// Global hook for cache invalidation to avoid import cycles
var userCacheInvalidateHook UserCacheInvalidateFunc

// NewUserService creates a new user service
func NewUserService(db *gorm.DB) *UserService {
	return &UserService{db: db}
}

// SetUserCacheInvalidateHook sets the function to call when user cache needs invalidation
func SetUserCacheInvalidateHook(hook UserCacheInvalidateFunc) {
	userCacheInvalidateHook = hook
}

// InvalidateUserCache calls the hook to invalidate user cache if set
func InvalidateUserCache(userID uuid.UUID) error {
	if userCacheInvalidateHook != nil {
		return userCacheInvalidateHook(userID)
	}
	return nil
}

// CreateUser creates a new user with hashed password
func (s *UserService) CreateUser(username, email, password string, isAdmin bool, timezone ...string) (*User, error) {
	var existingUser User
	if err := s.db.Where("LOWER(username) = LOWER(?) OR LOWER(email) = LOWER(?)", username, email).First(&existingUser).Error; err == nil {
		return nil, errors.New("user with this username or email already exists")
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Handle timezone parameter
	userTimezone := "UTC"
	if len(timezone) > 0 && timezone[0] != "" {
		userTimezone = utils.NormalizeTimezone(timezone[0])
	}

	user := &User{
		ID:        uuid.New(),
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		Timezone:  userTimezone,
		IsAdmin:   isAdmin,
		IsActive:  true,
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	if err := s.db.Create(user).Error; err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	return user, nil
}

// AuthenticateUser validates user credentials and returns user if valid
func (s *UserService) AuthenticateUser(username, password string) (*User, error) {
	var user User
	if err := s.db.Where("LOWER(username) = LOWER(?)", username).First(&user).Error; err != nil {
		return nil, errors.New("invalid credentials")
	}

	if !user.IsActive {
		return nil, errors.New("account disabled")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return nil, errors.New("invalid credentials")
	}

	// Update last login
	now := time.Now().UTC()
	user.LastLogin = &now
	s.db.Save(&user)

	return &user, nil
}

// UpdateUserPassword updates a user's password
func (s *UserService) UpdateUserPassword(userID uuid.UUID, newPassword string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"password":            string(hashedPassword),
		"updated_at":          time.Now().UTC(),
		"reset_token":         nil,
		"reset_token_expires": nil,
	}).Error
}

// GeneratePasswordResetToken generates a reset token for password recovery
func (s *UserService) GeneratePasswordResetToken(email string) (string, error) {
	var user User
	if err := s.db.Where("LOWER(email) = LOWER(?) AND is_active = ?", email, true).First(&user).Error; err != nil {
		return "", errors.New("user not found")
	}

	// Generate random token
	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}
	token := fmt.Sprintf("%x", tokenBytes)

	// Set token expiration (24 hours from now)
	expires := time.Now().UTC().Add(24 * time.Hour)

	if err := s.db.Model(&user).Updates(map[string]interface{}{
		"reset_token":         token,
		"reset_token_expires": expires,
		"updated_at":          time.Now().UTC(),
	}).Error; err != nil {
		return "", fmt.Errorf("failed to save reset token: %w", err)
	}

	return token, nil
}

// ValidatePasswordResetToken validates a reset token and returns the user
func (s *UserService) ValidatePasswordResetToken(token string) (*User, error) {
	var user User
	if err := s.db.Where("reset_token = ? AND reset_token_expires > ? AND is_active = ?",
		token, time.Now().UTC(), true).First(&user).Error; err != nil {
		return nil, errors.New("invalid or expired reset token")
	}

	return &user, nil
}

// GetUserByID retrieves a user by ID
func (s *UserService) GetUserByID(userID uuid.UUID) (*User, error) {
	var user User
	if err := s.db.Where("id = ? AND is_active = ?", userID, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetAllUsers retrieves all users (for admin)
func (s *UserService) GetAllUsers() ([]User, error) {
	var users []User
	if err := s.db.Find(&users).Error; err != nil {
		return nil, err
	}
	return users, nil
}

// UpdateUserSettings updates user-specific settings
func (s *UserService) UpdateUserSettings(userID uuid.UUID, settings map[string]interface{}) error {
	settings["updated_at"] = time.Now().UTC()
	return s.db.Model(&User{}).Where("id = ?", userID).Updates(settings).Error
}

// DeactivateUser deactivates a user instead of deleting
func (s *UserService) DeactivateUser(userID uuid.UUID) error {
	return s.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_active":  false,
		"updated_at": time.Now().UTC(),
	}).Error
}

// ActivateUser reactivates a user
func (s *UserService) ActivateUser(userID uuid.UUID) error {
	return s.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_active":  true,
		"updated_at": time.Now().UTC(),
	}).Error
}

// SetUserAdmin sets or removes admin privileges
func (s *UserService) SetUserAdmin(userID uuid.UUID, isAdmin bool) error {
	return s.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"is_admin":   isAdmin,
		"updated_at": time.Now().UTC(),
	}).Error
}

// GetUserStats returns user statistics
func (s *UserService) GetUserStats(userID uuid.UUID) (map[string]interface{}, error) {
	var stats map[string]interface{} = make(map[string]interface{})

	// Count API keys
	var apiKeyCount int64
	if err := s.db.Model(&APIKey{}).Where("user_id = ? AND is_active = ?", userID, true).Count(&apiKeyCount).Error; err != nil {
		return nil, err
	}
	stats["api_keys"] = apiKeyCount

	// Count active sessions
	var sessionCount int64
	if err := s.db.Model(&UserSession{}).Where("user_id = ? AND expires_at > ?", userID, time.Now().UTC()).Count(&sessionCount).Error; err != nil {
		return nil, err
	}
	stats["active_sessions"] = sessionCount

	return stats, nil
}

// CompleteOnboarding marks the user's onboarding as complete
func (s *UserService) CompleteOnboarding(userID uuid.UUID) error {
	return s.db.Model(&User{}).Where("id = ?", userID).Updates(map[string]interface{}{
		"onboarding_completed": true,
		"updated_at":           time.Now().UTC(),
	}).Error
}

// CleanupExpiredSessions removes expired sessions
func (s *UserService) CleanupExpiredSessions() error {
	return s.db.Where("expires_at < ?", time.Now().UTC()).Delete(&UserSession{}).Error
}

// CleanupExpiredResetTokens removes expired reset tokens
func (s *UserService) CleanupExpiredResetTokens() error {
	return s.db.Model(&User{}).Where("reset_token_expires < ?", time.Now().UTC()).Updates(map[string]interface{}{
		"reset_token":         nil,
		"reset_token_expires": nil,
	}).Error
}

// DeleteUser permanently deletes a user and all associated data
func (s *UserService) DeleteUser(userID uuid.UUID) error {
	// Start a transaction to ensure atomicity
	return s.db.Transaction(func(tx *gorm.DB) error {
		// Delete all user sessions
		if err := tx.Where("user_id = ?", userID).Delete(&UserSession{}).Error; err != nil {
			return fmt.Errorf("failed to delete user sessions: %w", err)
		}

		// Delete all API keys
		if err := tx.Where("user_id = ?", userID).Delete(&APIKey{}).Error; err != nil {
			return fmt.Errorf("failed to delete API keys: %w", err)
		}

		// Delete login attempts
		if err := tx.Where("username = (SELECT username FROM users WHERE id = ?)", userID).Delete(&LoginAttempt{}).Error; err != nil {
			return fmt.Errorf("failed to delete login attempts: %w", err)
		}

		// Finally, delete the user
		if err := tx.Where("id = ?", userID).Delete(&User{}).Error; err != nil {
			return fmt.Errorf("failed to delete user: %w", err)
		}

		return nil
	})
}
