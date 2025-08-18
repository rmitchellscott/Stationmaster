package database

import (
	"fmt"

	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/logging"
)

// MigrateToMultiUser ensures initial admin user exists and runs migrations
func MigrateToMultiUser() error {

	userService := NewUserService(DB)

	// Check if any users exist
	var userCount int64
	if err := DB.Model(&User{}).Count(&userCount).Error; err != nil {
		return fmt.Errorf("failed to count users: %w", err)
	}

	if userCount > 0 {
		logging.Logf("[STARTUP] Users already exist, skipping user creation migration")
		// Still run schema migrations even if users exist
		return RunMigrations("STARTUP")
	}

	// Create admin user from environment variables
	username := config.Get("ADMIN_USERNAME", "")
	password := config.Get("ADMIN_PASSWORD", "")
	email := config.Get("ADMIN_EMAIL", "")

	if username == "" || password == "" {
		logging.Logf("[STARTUP] No admin user configured - navigate to /register to create the first admin account")
		return RunMigrations("STARTUP")
	}

	if email == "" {
		email = username + "@localhost" // Default email if not provided
	}

	logging.Logf("[STARTUP] Creating initial admin user: %s", username)

	// Create the admin user using the service method
	_, err := userService.CreateUser(username, email, password, true)
	if err != nil {
		// Check if user already exists
		if existingUser, authErr := userService.AuthenticateUser(username, password); authErr == nil && existingUser != nil {
			logging.Logf("[STARTUP] Admin user already exists")
			return RunMigrations("STARTUP")
		}
		return fmt.Errorf("failed to create admin user: %w", err)
	}

	logging.Logf("[STARTUP] Successfully created admin user")

	// Run database migrations
	return RunMigrations("STARTUP")
}
