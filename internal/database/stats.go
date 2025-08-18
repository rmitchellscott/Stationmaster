package database

import (
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DatabaseStats holds statistics about the database
type DatabaseStats struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveUsers   int64 `json:"active_users"`
	TotalAPIKeys  int64 `json:"total_api_keys"`
	ActiveAPIKeys int64 `json:"active_api_keys"`
	TotalSessions int64 `json:"total_sessions"`
}

// GetDatabaseStats returns database statistics
func GetDatabaseStats(db *gorm.DB) (*DatabaseStats, error) {
	stats := &DatabaseStats{}

	// Count users
	if err := db.Model(&User{}).Count(&stats.TotalUsers).Error; err != nil {
		return nil, err
	}

	// Count active users
	if err := db.Model(&User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers).Error; err != nil {
		return nil, err
	}

	// Count API keys
	if err := db.Model(&APIKey{}).Count(&stats.TotalAPIKeys).Error; err != nil {
		return nil, err
	}

	// Count active API keys
	if err := db.Model(&APIKey{}).Where("is_active = ?", true).Count(&stats.ActiveAPIKeys).Error; err != nil {
		return nil, err
	}

	// Count sessions
	if err := db.Model(&UserSession{}).Count(&stats.TotalSessions).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

// CleanupOldData removes old sessions and login attempts
func CleanupOldData(db *gorm.DB, sessionDays, loginAttemptDays int) error {
	// Clean up expired sessions
	if err := db.Where("expires_at < NOW()").Delete(&UserSession{}).Error; err != nil {
		return err
	}

	// Clean up old login attempts
	if loginAttemptDays > 0 {
		if err := db.Where("attempted_at < NOW() - INTERVAL ? DAY", loginAttemptDays).Delete(&LoginAttempt{}).Error; err != nil {
			// SQLite syntax
			if err := db.Where("attempted_at < datetime('now', '-' || ? || ' days')", loginAttemptDays).Delete(&LoginAttempt{}).Error; err != nil {
				return err
			}
		}
	}

	return nil
}

// MigrateSingleUserData is a placeholder for single-user to multi-user migration
// In Stationmaster, this is simplified as we don't have document-specific data
func MigrateSingleUserData(db *gorm.DB, userID uuid.UUID) error {
	// In the original Aviary, this migrated documents and folders
	// For Stationmaster, we just return nil as there's nothing to migrate
	return nil
}
