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

	// Run auto-migration
	if err := runMigrations("STARTUP"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	// Initialize default system settings
	if err := initializeSystemSettings(); err != nil {
		return fmt.Errorf("failed to initialize system settings: %w", err)
	}

	// Initialize default plugins
	if err := initializeDefaultPlugins(); err != nil {
		return fmt.Errorf("failed to initialize default plugins: %w", err)
	}

	logging.Logf("[STARTUP] Database initialized successfully (type: %s)", config.Type)
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

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
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

	return db, nil
}

// runMigrations runs GORM auto-migration for all models
func runMigrations(logPrefix string) error {
	logging.Logf("[%s] Running GORM auto-migrations...", logPrefix)
	
	models := GetAllModels()

	// Force migration of all models
	for _, model := range models {
		if err := DB.AutoMigrate(model); err != nil {
			return fmt.Errorf("failed to migrate %T: %w", model, err)
		}
	}

	logging.Logf("[%s] GORM auto-migration completed successfully", logPrefix)
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
			Value:       "true",
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

// initializeDefaultPlugins creates default system plugins if they don't exist
func initializeDefaultPlugins() error {
	defaultPlugins := []Plugin{
		{
			Name:        "Text Display",
			Type:        "display",
			Description: "Display custom text messages on your TRMNL device",
			ConfigSchema: `{
				"type": "object",
				"properties": {
					"text": {
						"type": "string",
						"title": "Text to Display",
						"description": "The text message to show on the device",
						"default": "Hello World!"
					},
					"font_size": {
						"type": "integer",
						"title": "Font Size",
						"description": "Size of the text font",
						"minimum": 12,
						"maximum": 72,
						"default": 24
					}
				},
				"required": ["text"]
			}`,
			Version: "1.0.0",
			Author:  "Stationmaster",
			IsActive: true,
		},
		{
			Name:        "Clock",
			Type:        "time",
			Description: "Display current time and date",
			ConfigSchema: `{
				"type": "object",
				"properties": {
					"format": {
						"type": "string",
						"title": "Time Format",
						"description": "Format for displaying time",
						"enum": ["12-hour", "24-hour"],
						"default": "12-hour"
					},
					"timezone": {
						"type": "string",
						"title": "Timezone",
						"description": "Timezone for displaying time",
						"default": "UTC"
					},
					"show_date": {
						"type": "boolean",
						"title": "Show Date",
						"description": "Whether to display the date along with time",
						"default": true
					}
				}
			}`,
			Version: "1.0.0",
			Author:  "Stationmaster",
			IsActive: true,
		},
		{
			Name:        "Weather",
			Type:        "weather",
			Description: "Display weather information",
			ConfigSchema: `{
				"type": "object",
				"properties": {
					"location": {
						"type": "string",
						"title": "Location",
						"description": "City or location for weather data",
						"default": "New York, NY"
					},
					"units": {
						"type": "string",
						"title": "Temperature Units",
						"description": "Units for temperature display",
						"enum": ["celsius", "fahrenheit"],
						"default": "fahrenheit"
					},
					"api_key": {
						"type": "string",
						"title": "API Key",
						"description": "Weather API key (OpenWeatherMap, etc.)",
						"format": "password"
					}
				},
				"required": ["location", "api_key"]
			}`,
			Version: "1.0.0",
			Author:  "Stationmaster",
			IsActive: true,
		},
		{
			Name:        "RSS Feed",
			Type:        "content",
			Description: "Display RSS feed headlines",
			ConfigSchema: `{
				"type": "object",
				"properties": {
					"feed_url": {
						"type": "string",
						"title": "RSS Feed URL",
						"description": "URL of the RSS feed to display",
						"default": "https://feeds.bbci.co.uk/news/rss.xml"
					},
					"max_items": {
						"type": "integer",
						"title": "Maximum Items",
						"description": "Maximum number of feed items to display",
						"minimum": 1,
						"maximum": 10,
						"default": 5
					}
				},
				"required": ["feed_url"]
			}`,
			Version: "1.0.0",
			Author:  "Stationmaster",
			IsActive: true,
		},
		{
			Name:        "Image Display",
			Type:        "media",
			Description: "Display custom images",
			ConfigSchema: `{
				"type": "object",
				"properties": {
					"image_url": {
						"type": "string",
						"title": "Image URL",
						"description": "URL of the image to display",
						"default": "https://via.placeholder.com/800x480"
					},
					"fit_mode": {
						"type": "string",
						"title": "Fit Mode",
						"description": "How to fit the image on the display",
						"enum": ["contain", "cover", "fill"],
						"default": "contain"
					}
				},
				"required": ["image_url"]
			}`,
			Version: "1.0.0",
			Author:  "Stationmaster",
			IsActive: true,
		},
		{
			Name:        "Redirect",
			Type:        "redirect",
			Description: "Proxy JSON response from external endpoint (TRMNL BYOS Redirect plugin)",
			ConfigSchema: `{
				"type": "object",
				"properties": {
					"endpoint_url": {
						"type": "string",
						"title": "JSON Endpoint URL",
						"description": "URL to fetch JSON response from (must return filename, url, refresh_rate fields)",
						"placeholder": "https://your-server.com/api/plugin-endpoint"
					},
					"timeout_seconds": {
						"type": "integer",
						"title": "Request Timeout",
						"description": "Timeout for HTTP requests in seconds (max 10)",
						"minimum": 1,
						"maximum": 10,
						"default": 2
					}
				},
				"required": ["endpoint_url"]
			}`,
			Version: "1.0.0",
			Author:  "TRMNL",
			IsActive: true,
		},
	}

	for _, plugin := range defaultPlugins {
		var existing Plugin
		if err := DB.First(&existing, "name = ?", plugin.Name).Error; err == gorm.ErrRecordNotFound {
			if err := DB.Create(&plugin).Error; err != nil {
				return fmt.Errorf("failed to create default plugin %s: %w", plugin.Name, err)
			}
			logging.Logf("[STARTUP] Created default plugin: %s", plugin.Name)
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