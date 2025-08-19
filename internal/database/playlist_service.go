package database

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/utils"
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
func (pls *PlaylistService) AddItemToPlaylist(playlistID, userPluginID uuid.UUID, importance bool, durationOverride *int) (*PlaylistItem, error) {
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

// ReorderPlaylistItemsByArray updates playlist items to match the provided order array
func (pls *PlaylistService) ReorderPlaylistItemsByArray(playlistID uuid.UUID, orderedItemIDs []uuid.UUID) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// Update each item's order_index based on its position in the array
		for i, itemID := range orderedItemIDs {
			newOrderIndex := i + 1 // Start from 1
			if err := tx.Model(&PlaylistItem{}).Where("id = ? AND playlist_id = ?", itemID, playlistID).Update("order_index", newOrderIndex).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// DeletePlaylistItem deletes a playlist item and its schedules, then compacts the order
func (pls *PlaylistService) DeletePlaylistItem(itemID uuid.UUID) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// First, verify the playlist item exists
		var existingItem PlaylistItem
		if err := tx.Where("id = ?", itemID).First(&existingItem).Error; err != nil {
			log.Printf("[DeletePlaylistItem] Playlist item not found: %s, error: %v", itemID.String(), err)
			return err
		}

		playlistID := existingItem.PlaylistID
		log.Printf("[DeletePlaylistItem] Deleting item %s from playlist %s", itemID.String(), playlistID.String())

		// Delete playlist item (schedules will cascade due to foreign key constraints)
		result := tx.Delete(&PlaylistItem{}, "id = ?", itemID)
		if result.Error != nil {
			log.Printf("[DeletePlaylistItem] Failed to delete playlist item %s: %v", itemID.String(), result.Error)
			return result.Error
		}

		// Verify the item was actually deleted
		if result.RowsAffected == 0 {
			log.Printf("[DeletePlaylistItem] No rows affected when deleting item %s", itemID.String())
			return gorm.ErrRecordNotFound
		}

		log.Printf("[DeletePlaylistItem] Successfully deleted item %s, rows affected: %d", itemID.String(), result.RowsAffected)

		// Compact the order for the playlist
		if err := pls.compactPlaylistOrderInTx(tx, playlistID); err != nil {
			log.Printf("[DeletePlaylistItem] Failed to compact order for playlist %s: %v", playlistID.String(), err)
			return err
		}

		log.Printf("[DeletePlaylistItem] Successfully compacted order for playlist %s", playlistID.String())
		return nil
	})
}

// CompactPlaylistOrder renumbers all playlist items to have sequential order_index values (1, 2, 3...)
func (pls *PlaylistService) CompactPlaylistOrder(playlistID uuid.UUID) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		return pls.compactPlaylistOrderInTx(tx, playlistID)
	})
}

// compactPlaylistOrderInTx compacts order within an existing transaction
func (pls *PlaylistService) compactPlaylistOrderInTx(tx *gorm.DB, playlistID uuid.UUID) error {
	// Get all items ordered by current order_index
	var items []PlaylistItem
	if err := tx.Where("playlist_id = ?", playlistID).Order("order_index ASC").Find(&items).Error; err != nil {
		return err
	}

	// Update each item with a new sequential order_index
	for i, item := range items {
		newOrderIndex := i + 1 // Start from 1
		if err := tx.Model(&item).Update("order_index", newOrderIndex).Error; err != nil {
			return err
		}
	}

	return nil
}

// AddScheduleToPlaylistItem adds a schedule to a playlist item
func (pls *PlaylistService) AddScheduleToPlaylistItem(playlistItemID uuid.UUID, name string, dayMask int, startTime, endTime, timezone string, isActive bool) (*Schedule, error) {
	schedule := &Schedule{
		PlaylistItemID: playlistItemID,
		Name:           name,
		DayMask:        dayMask,
		StartTime:      startTime,
		EndTime:        endTime,
		Timezone:       timezone,
		IsActive:       isActive,
	}

	if err := pls.db.Create(schedule).Error; err != nil {
		return nil, err
	}

	return schedule, nil
}

// GetSchedulesByPlaylistItemID returns all schedules for a playlist item
func (pls *PlaylistService) GetSchedulesByPlaylistItemID(playlistItemID uuid.UUID) ([]Schedule, error) {
	var schedules []Schedule
	err := pls.db.Where("playlist_item_id = ?", playlistItemID).Order("created_at ASC").Find(&schedules).Error
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

	log.Printf("[SCHEDULE DEBUG] Device: %s, Current time UTC: %s",
		deviceID.String(), currentTime.UTC().Format("2006-01-02 15:04:05"))
	log.Printf("[SCHEDULE DEBUG] Found %d playlist items", len(items))

	for i, item := range items {
		log.Printf("[SCHEDULE DEBUG] Item %d: ID=%s, Visible=%t, Schedules=%d",
			i, item.ID.String(), item.IsVisible, len(item.Schedules))

		if !item.IsVisible {
			log.Printf("[SCHEDULE DEBUG] Item %d: Skipping - not visible", i)
			continue
		}

		// If no schedules, item is always active
		if len(item.Schedules) == 0 {
			log.Printf("[SCHEDULE DEBUG] Item %d: Active - no schedules (always active)", i)
			activeItems = append(activeItems, item)
			continue
		}

		// Check if any schedule matches current time
		itemMatched := false
		for j, schedule := range item.Schedules {
			log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Name=%s, Active=%t, DayMask=%d, Start=%s, End=%s, Timezone=%s",
				i, j, schedule.Name, schedule.IsActive, schedule.DayMask, schedule.StartTime, schedule.EndTime, schedule.Timezone)

			if !schedule.IsActive {
				log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Skipping - not active", i, j)
				continue
			}

			// Load the schedule's timezone with validation
			scheduleTimezone := utils.NormalizeTimezone(schedule.Timezone)

			loc, err := time.LoadLocation(scheduleTimezone)
			if err != nil {
				log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Invalid timezone %s, falling back to UTC", i, j, scheduleTimezone)
				loc = time.UTC
				scheduleTimezone = "UTC"
			}

			// Convert current time to schedule's timezone
			currentTimeInScheduleTZ := currentTime.In(loc)
			weekday := int(currentTimeInScheduleTZ.Weekday())
			dayBit := 1 << weekday
			currentTimeStr := currentTimeInScheduleTZ.Format("15:04:05")

			log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Current time in %s: %s, Weekday: %d, DayBit: %d",
				i, j, scheduleTimezone, currentTimeInScheduleTZ.Format("2006-01-02 15:04:05"), weekday, dayBit)

			// Check day mask
			dayMatch := (schedule.DayMask & dayBit) != 0
			log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Day match = %t (schedule mask %d & current bit %d)",
				i, j, dayMatch, schedule.DayMask, dayBit)
			if !dayMatch {
				continue
			}

			// Check time range (schedule times are stored as local times in the schedule's timezone)
			// Handle overnight schedules where end_time < start_time (crosses midnight)
			var timeMatch bool
			if schedule.EndTime < schedule.StartTime {
				// Overnight schedule: active if current time is >= start OR <= end
				timeMatch = currentTimeStr >= schedule.StartTime || currentTimeStr <= schedule.EndTime
				log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Overnight schedule - Time match = %t (%s >= %s || %s <= %s)",
					i, j, timeMatch, currentTimeStr, schedule.StartTime, currentTimeStr, schedule.EndTime)
			} else {
				// Normal schedule: active if current time is between start and end
				timeMatch = currentTimeStr >= schedule.StartTime && currentTimeStr <= schedule.EndTime
				log.Printf("[SCHEDULE DEBUG] Item %d, Schedule %d: Normal schedule - Time match = %t (%s >= %s && %s <= %s)",
					i, j, timeMatch, currentTimeStr, schedule.StartTime, currentTimeStr, schedule.EndTime)
			}
			if timeMatch {
				log.Printf("[SCHEDULE DEBUG] Item %d: ACTIVE - matched schedule %d", i, j)
				activeItems = append(activeItems, item)
				itemMatched = true
				break
			}
		}

		if !itemMatched {
			log.Printf("[SCHEDULE DEBUG] Item %d: No matching schedules", i)
		}
	}

	log.Printf("[SCHEDULE DEBUG] Found %d active items before importance filtering", len(activeItems))

	// Check if any important items are active
	importantItems := make([]PlaylistItem, 0)
	normalItems := make([]PlaylistItem, 0)

	for _, item := range activeItems {
		if item.Importance {
			importantItems = append(importantItems, item)
		} else {
			normalItems = append(normalItems, item)
		}
	}

	// If important items are active, only return important items
	if len(importantItems) > 0 {
		log.Printf("[SCHEDULE DEBUG] %d important items found - filtering out %d normal items", len(importantItems), len(normalItems))
		return importantItems, nil
	}

	// If no important items, return all active items (normal behavior)
	log.Printf("[SCHEDULE DEBUG] No important items found - returning all %d active items", len(activeItems))
	return activeItems, nil
}
