package database

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/glebarez/sqlite"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/config"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	Type     string // "sqlite" or "postgres"
	Host     string
	Port     int
	User     string
	Password string
	DBName   string
	SSLMode  string
	DataDir  string // For SQLite
}

// GetDatabaseConfig reads database configuration from environment variables
func GetDatabaseConfig() *DatabaseConfig {
	cfg := &DatabaseConfig{
		Type:     config.Get("DB_TYPE", "sqlite"),
		Host:     config.Get("DB_HOST", "localhost"),
		Port:     config.GetInt("DB_PORT", 5432),
		User:     config.Get("DB_USER", "stationmaster"),
		Password: config.Get("DB_PASSWORD", ""),
		DBName:   config.Get("DB_NAME", "stationmaster"),
		SSLMode:  config.Get("DB_SSLMODE", "disable"),
		DataDir:  config.Get("DATA_DIR", "/data"),
	}

	return cfg
}

// Initialize sets up the database connection and runs migrations
func Initialize() error {
	config := GetDatabaseConfig()

	var err error
	switch config.Type {
	case "postgres":
		DB, err = initPostgres(config)
	case "sqlite":
		DB, err = initSQLite(config)
	default:
		return fmt.Errorf("unsupported database type: %s", config.Type)
	}

	if err != nil {
		return fmt.Errorf("failed to initialize database: %w", err)
	}

	// Run gormigrate migrations first
	if err := RunMigrations("STARTUP"); err != nil {
		return fmt.Errorf("failed to run gormigrate migrations: %w", err)
	}

	// Run auto-migration for remaining models
	if err := runMigrations("STARTUP"); err != nil {
		return fmt.Errorf("failed to run auto-migrations: %w", err)
	}

	// Initialize default system settings
	if err := initializeSystemSettings(); err != nil {
		return fmt.Errorf("failed to initialize system settings: %w", err)
	}


	logging.Info("[STARTUP] Database initialized successfully", "type", config.Type)
	return nil
}

// initPostgres initializes PostgreSQL connection
func initPostgres(config *DatabaseConfig) (*gorm.DB, error) {
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%d sslmode=%s",
		config.Host, config.User, config.Password, config.DBName, config.Port, config.SSLMode)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: getGormLogger(),
	})
	if err != nil {
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(25)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(5 * time.Minute)

	return db, nil
}

// initSQLite initializes SQLite connection
func initSQLite(config *DatabaseConfig) (*gorm.DB, error) {
	// Ensure data directory exists
	if err := os.MkdirAll(config.DataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %w", err)
	}

	dbPath := filepath.Join(config.DataDir, "stationmaster.db")

	db, err := gorm.Open(sqlite.Open(dbPath+"?_pragma=foreign_keys(1)"), &gorm.Config{
		Logger: getGormLogger(),
	})
	if err != nil {
		return nil, err
	}

	// Configure SQLite settings
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	sqlDB.SetMaxOpenConns(1) // SQLite doesn't support concurrent writes
	sqlDB.SetMaxIdleConns(1)

	// Enable foreign keys for SQLite
	if err := db.Exec("PRAGMA foreign_keys = ON").Error; err != nil {
		return nil, err
	}

	// Verify foreign keys are enabled
	var fkEnabled int
	if err := db.Raw("PRAGMA foreign_keys").Scan(&fkEnabled).Error; err != nil {
		return nil, fmt.Errorf("failed to check foreign keys status: %w", err)
	}
	if fkEnabled != 1 {
		return nil, fmt.Errorf("foreign keys are not enabled (got %d, expected 1)", fkEnabled)
	}

	return db, nil
}

// runMigrations runs GORM auto-migration for all models
func runMigrations(logPrefix string) error {
	logging.Info("Running GORM auto-migrations", "component", logPrefix)

	models := GetAllModels()

	// Force migration of all models
	for _, model := range models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	logging.Info("GORM auto-migration completed successfully", "component", logPrefix)
	return nil
}

// RunAutoMigrations runs GORM auto-migration for all models (public wrapper)
func RunAutoMigrations(logPrefix string) error {
	return runMigrations(logPrefix)
}

// initializeSystemSettings creates default system settings if they don't exist
func initializeSystemSettings() error {
	defaultSettings := map[string]SystemSetting{
		"smtp_enabled": {
			Key:         "smtp_enabled",
			Value:       "false",
			Description: "Whether SMTP is configured for password resets",
		},
		"smtp_host": {
			Key:         "smtp_host",
			Value:       "",
			Description: "SMTP server hostname",
		},
		"smtp_port": {
			Key:         "smtp_port",
			Value:       "587",
			Description: "SMTP server port",
		},
		"smtp_username": {
			Key:         "smtp_username",
			Value:       "",
			Description: "SMTP username",
		},
		"smtp_password": {
			Key:         "smtp_password",
			Value:       "",
			Description: "SMTP password",
		},
		"smtp_from": {
			Key:         "smtp_from",
			Value:       "",
			Description: "From email address for system emails",
		},
		"smtp_tls": {
			Key:         "smtp_tls",
			Value:       "true",
			Description: "Whether to use TLS for SMTP",
		},
		"registration_enabled": {
			Key:         "registration_enabled",
			Value:       config.Get("PUBLIC_REGISTRATION_ENABLED", "false"),
			Description: "Whether new user registration is enabled",
		},
		"max_api_keys_per_user": {
			Key:         "max_api_keys_per_user",
			Value:       "10",
			Description: "Maximum API keys per user",
		},
		"password_reset_timeout_hours": {
			Key:         "password_reset_timeout_hours",
			Value:       "24",
			Description: "Password reset token timeout in hours",
		},
		"enable_frequent_refreshes": {
			Key:         "enable_frequent_refreshes",
			Value:       "false",
			Description: "Enable 1, 5, and 10 minute screen render refresh rates",
		},
	}

	for _, setting := range defaultSettings {
		var existing SystemSetting
		if err := DB.First(&existing, "key = ?", setting.Key).Error; err == gorm.ErrRecordNotFound {
			if err := DB.Create(&setting).Error; err != nil {
				return fmt.Errorf("failed to create system setting %s: %w", setting.Key, err)
			}
		}
	}

	return nil
}


// getGormLogger returns appropriate GORM logger based on environment
func getGormLogger() logger.Interface {
	logLevel := logger.Warn
	if config.Get("GIN_MODE", "") == "debug" {
		logLevel = logger.Info
	}

	return logger.Default.LogMode(logLevel)
}

// Helper functions
// IsMultiUserMode always returns true for Stationmaster
// (kept for compatibility with existing code)
func IsMultiUserMode() bool {
	return true
}

// GetCurrentUser gets the current user from the database by ID
func GetCurrentUser(userID uuid.UUID) (*User, error) {
	var user User
	if err := DB.First(&user, "id = ? AND is_active = ?", userID, true).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsername gets a user by username
func GetUserByUsername(username string) (*User, error) {
	var user User
	if err := DB.Where("LOWER(username) = LOWER(?) AND is_active = ?", username, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByOIDCSubject gets a user by OIDC subject
func GetUserByOIDCSubject(oidcSubject string) (*User, error) {
	var user User
	if err := DB.First(&user, "oidc_subject = ? AND is_active = ?", oidcSubject, true).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByUsernameWithoutOIDC gets a user by username without an OIDC subject
func GetUserByUsernameWithoutOIDC(username string) (*User, error) {
	var user User
	if err := DB.Where("LOWER(username) = LOWER(?) AND is_active = ? AND (oidc_subject IS NULL OR oidc_subject = '')", username, true).First(&user).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmailWithoutOIDC gets a user by email without an OIDC subject
func GetUserByEmailWithoutOIDC(email string) (*User, error) {
	var user User
	if err := DB.First(&user, "LOWER(email) = LOWER(?) AND is_active = ? AND (oidc_subject IS NULL OR oidc_subject = '')", email, true).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetUserByEmail gets a user by email
func GetUserByEmail(email string) (*User, error) {
	var user User
	if err := DB.First(&user, "LOWER(email) = LOWER(?) AND is_active = ?", email, true).Error; err != nil {
		return nil, err
	}
	return &user, nil
}

// GetSystemSetting gets a system setting by key
func GetSystemSetting(key string) (string, error) {
	var setting SystemSetting
	if err := DB.First(&setting, "key = ?", key).Error; err != nil {
		return "", err
	}
	return setting.Value, nil
}

// GetRegistrationSetting gets the registration enabled setting, checking environment variable first
func GetRegistrationSetting() (string, bool) {
	envValue := config.Get("PUBLIC_REGISTRATION_ENABLED", "")
	if envValue != "" {
		return envValue, true // true means locked by environment
	}
	
	dbValue, err := GetSystemSetting("registration_enabled")
	if err != nil {
		return "false", false
	}
	return dbValue, false // false means not locked
}

// IsRegistrationSettingLocked returns true if registration setting is controlled by environment variable
func IsRegistrationSettingLocked() bool {
	return config.Get("PUBLIC_REGISTRATION_ENABLED", "") != ""
}

// SetSystemSetting sets a system setting
func SetSystemSetting(key, value string, updatedBy *uuid.UUID) error {
	setting := SystemSetting{
		Key:       key,
		Value:     value,
		UpdatedBy: updatedBy,
		UpdatedAt: time.Now(),
	}

	return DB.Save(&setting).Error
}

// GetDB returns the database instance
func GetDB() *gorm.DB {
	return DB
}

// Close closes the database connection
func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
