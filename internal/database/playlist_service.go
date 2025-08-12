package database

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PlaylistService handles playlist-related database operations
type PlaylistService struct {
	db *gorm.DB
}

// NewPlaylistService creates a new playlist service
func NewPlaylistService(db *gorm.DB) *PlaylistService {
	return &PlaylistService{db: db}
}

// CreatePlaylist creates a new playlist for a device
func (pls *PlaylistService) CreatePlaylist(userID, deviceID uuid.UUID, name string, isDefault bool) (*Playlist, error) {
	// If this is set as default, unset any other default playlists for this device
	if isDefault {
		if err := pls.db.Model(&Playlist{}).Where("device_id = ?", deviceID).Update("is_default", false).Error; err != nil {
			return nil, err
		}
	}

	playlist := &Playlist{
		UserID:    userID,
		DeviceID:  deviceID,
		Name:      name,
		IsDefault: isDefault,
	}

	if err := pls.db.Create(playlist).Error; err != nil {
		return nil, err
	}

	return playlist, nil
}

// GetPlaylistsByDeviceID returns all playlists for a device
func (pls *PlaylistService) GetPlaylistsByDeviceID(deviceID uuid.UUID) ([]Playlist, error) {
	var playlists []Playlist
	err := pls.db.Where("device_id = ?", deviceID).Order("is_default DESC, created_at DESC").Find(&playlists).Error
	return playlists, err
}

// GetPlaylistsByUserID returns all playlists for a user
func (pls *PlaylistService) GetPlaylistsByUserID(userID uuid.UUID) ([]Playlist, error) {
	var playlists []Playlist
	err := pls.db.Preload("Device").Where("user_id = ?", userID).Order("created_at DESC").Find(&playlists).Error
	return playlists, err
}

// GetPlaylistByID returns a playlist by its ID
func (pls *PlaylistService) GetPlaylistByID(playlistID uuid.UUID) (*Playlist, error) {
	var playlist Playlist
	err := pls.db.Preload("Device").First(&playlist, "id = ?", playlistID).Error
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// GetDefaultPlaylistForDevice returns the default playlist for a device
func (pls *PlaylistService) GetDefaultPlaylistForDevice(deviceID uuid.UUID) (*Playlist, error) {
	var playlist Playlist
	err := pls.db.First(&playlist, "device_id = ? AND is_default = ?", deviceID, true).Error
	if err != nil {
		return nil, err
	}
	return &playlist, nil
}

// UpdatePlaylist updates a playlist
func (pls *PlaylistService) UpdatePlaylist(playlist *Playlist) error {
	// If this is being set as default, unset any other default playlists for this device
	if playlist.IsDefault {
		if err := pls.db.Model(&Playlist{}).Where("device_id = ? AND id != ?", playlist.DeviceID, playlist.ID).Update("is_default", false).Error; err != nil {
			return err
		}
	}

	return pls.db.Save(playlist).Error
}

// DeletePlaylist deletes a playlist and all its items
func (pls *PlaylistService) DeletePlaylist(playlistID uuid.UUID) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// Delete playlist will cascade to playlist items and schedules
		return tx.Delete(&Playlist{}, "id = ?", playlistID).Error
	})
}

// AddItemToPlaylist adds a user plugin to a playlist
func (pls *PlaylistService) AddItemToPlaylist(playlistID, userPluginID uuid.UUID, importance int, durationOverride *int) (*PlaylistItem, error) {
	// Get the next order index
	var maxOrder int
	pls.db.Model(&PlaylistItem{}).Where("playlist_id = ?", playlistID).Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)

	playlistItem := &PlaylistItem{
		PlaylistID:       playlistID,
		UserPluginID:     userPluginID,
		OrderIndex:       maxOrder + 1,
		IsVisible:        true,
		Importance:       importance,
		DurationOverride: durationOverride,
	}

	if err := pls.db.Create(playlistItem).Error; err != nil {
		return nil, err
	}

	return playlistItem, nil
}

// GetPlaylistItems returns all items in a playlist with their associated data
func (pls *PlaylistService) GetPlaylistItems(playlistID uuid.UUID) ([]PlaylistItem, error) {
	var items []PlaylistItem
	err := pls.db.Preload("UserPlugin").Preload("UserPlugin.Plugin").Preload("Schedules").
		Where("playlist_id = ?", playlistID).
		Order("order_index ASC").
		Find(&items).Error
	return items, err
}

// GetPlaylistItemByID returns a playlist item by its ID
func (pls *PlaylistService) GetPlaylistItemByID(itemID uuid.UUID) (*PlaylistItem, error) {
	var item PlaylistItem
	err := pls.db.Preload("UserPlugin").Preload("UserPlugin.Plugin").Preload("Schedules").
		First(&item, "id = ?", itemID).Error
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdatePlaylistItem updates a playlist item
func (pls *PlaylistService) UpdatePlaylistItem(item *PlaylistItem) error {
	return pls.db.Save(item).Error
}

// ReorderPlaylistItems updates the order of multiple playlist items
func (pls *PlaylistService) ReorderPlaylistItems(playlistID uuid.UUID, itemOrders map[uuid.UUID]int) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		for itemID, orderIndex := range itemOrders {
			if err := tx.Model(&PlaylistItem{}).Where("id = ? AND playlist_id = ?", itemID, playlistID).Update("order_index", orderIndex).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeletePlaylistItem deletes a playlist item and its schedules
func (pls *PlaylistService) DeletePlaylistItem(itemID uuid.UUID) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// Delete playlist item will cascade to schedules
		return tx.Delete(&PlaylistItem{}, "id = ?", itemID).Error
	})
}

// AddScheduleToPlaylistItem adds a schedule to a playlist item
func (pls *PlaylistService) AddScheduleToPlaylistItem(playlistItemID uuid.UUID, name string, dayMask int, startTime, endTime, timezone string) (*Schedule, error) {
	schedule := &Schedule{
		PlaylistItemID: playlistItemID,
		Name:           name,
		DayMask:        dayMask,
		StartTime:      startTime,
		EndTime:        endTime,
		Timezone:       timezone,
		IsActive:       true,
	}

	if err := pls.db.Create(schedule).Error; err != nil {
		return nil, err
	}

	return schedule, nil
}

// GetSchedulesByPlaylistItemID returns all schedules for a playlist item
func (pls *PlaylistService) GetSchedulesByPlaylistItemID(playlistItemID uuid.UUID) ([]Schedule, error) {
	var schedules []Schedule
	err := pls.db.Where("playlist_item_id = ? AND is_active = ?", playlistItemID, true).Order("created_at ASC").Find(&schedules).Error
	return schedules, err
}

// UpdateSchedule updates a schedule
func (pls *PlaylistService) UpdateSchedule(schedule *Schedule) error {
	return pls.db.Save(schedule).Error
}

// DeleteSchedule deletes a schedule
func (pls *PlaylistService) DeleteSchedule(scheduleID uuid.UUID) error {
	return pls.db.Delete(&Schedule{}, "id = ?", scheduleID).Error
}

// GetActivePlaylistItemsForTime returns playlist items that should be active at a given time
func (pls *PlaylistService) GetActivePlaylistItemsForTime(deviceID uuid.UUID, currentTime time.Time) ([]PlaylistItem, error) {
	// Get the default playlist for the device
	playlist, err := pls.GetDefaultPlaylistForDevice(deviceID)
	if err != nil {
		return nil, err
	}

	// Get all playlist items with their schedules
	items, err := pls.GetPlaylistItems(playlist.ID)
	if err != nil {
		return nil, err
	}

	// Filter items that match the current time
	var activeItems []PlaylistItem
	
	// Get current day of week (0=Sunday, 1=Monday, etc.)
	weekday := int(currentTime.Weekday())
	dayBit := 1 << weekday
	
	currentTimeStr := currentTime.Format("15:04:05")
	
	for _, item := range items {
		if !item.IsVisible {
			continue
		}
		
		// If no schedules, item is always active
		if len(item.Schedules) == 0 {
			activeItems = append(activeItems, item)
			continue
		}
		
		// Check if any schedule matches current time
		for _, schedule := range item.Schedules {
			if !schedule.IsActive {
				continue
			}
			
			// Check day mask
			if (schedule.DayMask & dayBit) == 0 {
				continue
			}
			
			// Check time range
			if currentTimeStr >= schedule.StartTime && currentTimeStr <= schedule.EndTime {
				activeItems = append(activeItems, item)
				break
			}
		}
	}
	
	return activeItems, nil
}