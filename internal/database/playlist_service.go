package database

import (
	"time"

	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/logging"
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

// AddItemToPlaylist adds a plugin instance to a playlist
func (pls *PlaylistService) AddItemToPlaylist(playlistID, pluginInstanceID uuid.UUID, importance bool, durationOverride *int) (*PlaylistItem, error) {
	// Get the next order index
	var maxOrder int
	pls.db.Model(&PlaylistItem{}).Where("playlist_id = ?", playlistID).Select("COALESCE(MAX(order_index), 0)").Scan(&maxOrder)

	playlistItem := &PlaylistItem{
		PlaylistID:       playlistID,
		PluginInstanceID: pluginInstanceID,
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
	err := pls.db.Preload("PluginInstance").Preload("PluginInstance.PluginDefinition").Preload("Schedules").
		Where("playlist_id = ?", playlistID).
		Order("order_index ASC").
		Find(&items).Error
		
	return items, err
}

// GetPlaylistItemByID returns a playlist item by its ID
func (pls *PlaylistService) GetPlaylistItemByID(itemID uuid.UUID) (*PlaylistItem, error) {
	var item PlaylistItem
	err := pls.db.Preload("PluginInstance").Preload("PluginInstance.PluginDefinition").Preload("Schedules").
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
			logging.Debug("[DeletePlaylistItem] Playlist item not found", "item_id", itemID.String(), "error", err)
			return err
		}

		playlistID := existingItem.PlaylistID
		logging.Debug("[DeletePlaylistItem] Deleting item from playlist", "item_id", itemID.String(), "playlist_id", playlistID.String())

		// Delete playlist item (schedules will cascade due to foreign key constraints)
		result := tx.Delete(&PlaylistItem{}, "id = ?", itemID)
		if result.Error != nil {
			logging.Error("[DeletePlaylistItem] Failed to delete playlist item", "item_id", itemID.String(), "error", result.Error)
			return result.Error
		}

		// Verify the item was actually deleted
		if result.RowsAffected == 0 {
			logging.Debug("[DeletePlaylistItem] No rows affected when deleting item", "item_id", itemID.String())
			return gorm.ErrRecordNotFound
		}

		logging.Debug("[DeletePlaylistItem] Successfully deleted item", "item_id", itemID.String(), "rows_affected", result.RowsAffected)

		// Compact the order for the playlist
		if err := pls.compactPlaylistOrderInTx(tx, playlistID); err != nil {
			logging.Error("[DeletePlaylistItem] Failed to compact order for playlist", "playlist_id", playlistID.String(), "error", err)
			return err
		}

		logging.Debug("[DeletePlaylistItem] Successfully compacted order for playlist", "playlist_id", playlistID.String())
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

// ConsolidateDevicePlaylists ensures each device has exactly one default playlist
// Merges multiple playlists into a single default playlist per device
func (pls *PlaylistService) ConsolidateDevicePlaylists() error {
	logging.Info("[CONSOLIDATE] Starting playlist consolidation for all devices")
	
	// Get all devices
	var devices []Device
	if err := pls.db.Find(&devices).Error; err != nil {
		return err
	}
	
	consolidatedCount := 0
	for _, device := range devices {
		if err := pls.consolidatePlaylistsForDevice(device.ID); err != nil {
			logging.Error("[CONSOLIDATE] Error consolidating playlists for device", "device_id", device.ID.String(), "error", err)
			return err
		}
		consolidatedCount++
	}
	
	logging.Info("[CONSOLIDATE] Successfully consolidated playlists", "device_count", consolidatedCount)
	return nil
}

// consolidatePlaylistsForDevice merges all playlists for a single device into one default playlist
func (pls *PlaylistService) consolidatePlaylistsForDevice(deviceID uuid.UUID) error {
	// Get all playlists for this device
	var playlists []Playlist
	if err := pls.db.Where("device_id = ?", deviceID).Find(&playlists).Error; err != nil {
		return err
	}
	
	if len(playlists) <= 1 {
		// Device has 0 or 1 playlist, ensure it's marked as default
		if len(playlists) == 1 && !playlists[0].IsDefault {
			playlists[0].IsDefault = true
			if err := pls.db.Save(&playlists[0]).Error; err != nil {
				return err
			}
			logging.Debug("[CONSOLIDATE] Marked single playlist as default", "device_id", deviceID.String())
		}
		return nil
	}
	
	logging.Debug("[CONSOLIDATE] Device has multiple playlists, consolidating", "device_id", deviceID.String(), "playlist_count", len(playlists))
	
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// Find or create the target default playlist
		var targetPlaylist *Playlist
		for i := range playlists {
			if playlists[i].IsDefault {
				targetPlaylist = &playlists[i]
				break
			}
		}
		
		// If no default playlist exists, use the first one
		if targetPlaylist == nil {
			targetPlaylist = &playlists[0]
			targetPlaylist.IsDefault = true
			if err := tx.Save(targetPlaylist).Error; err != nil {
				return err
			}
		}
		
		logging.Debug("[CONSOLIDATE] Using playlist as target default", "playlist_name", targetPlaylist.Name, "device_id", deviceID.String())
		
		// Collect all items from all playlists and move them to the target playlist
		orderIndex := 1
		playlistsToDelete := []uuid.UUID{}
		
		for _, playlist := range playlists {
			if playlist.ID == targetPlaylist.ID {
				continue // Skip the target playlist
			}
			
			// Get all items from this playlist
			var items []PlaylistItem
			if err := tx.Where("playlist_id = ?", playlist.ID).Find(&items).Error; err != nil {
				return err
			}
			
			logging.Debug("[CONSOLIDATE] Moving items from playlist to default", "item_count", len(items), "playlist_name", playlist.Name)
			
			// Move each item to the target playlist
			for _, item := range items {
				// Update the item to belong to the target playlist with new order
				updates := map[string]interface{}{
					"playlist_id": targetPlaylist.ID,
					"order_index": orderIndex,
					"updated_at":  time.Now(),
				}
				
				if err := tx.Model(&item).Updates(updates).Error; err != nil {
					return err
				}
				orderIndex++
			}
			
			// Mark this playlist for deletion
			playlistsToDelete = append(playlistsToDelete, playlist.ID)
		}
		
		// Delete the empty playlists
		for _, playlistID := range playlistsToDelete {
			if err := tx.Delete(&Playlist{}, "id = ?", playlistID).Error; err != nil {
				return err
			}
		}
		
		logging.Debug("[CONSOLIDATE] Deleted extra playlists for device", "deleted_count", len(playlistsToDelete), "device_id", deviceID.String())
		
		return nil
	})
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

	for _, item := range items {
		if !item.IsVisible {
			continue
		}
		
		// Skip items that need config updates - they shouldn't be served to devices
		if item.PluginInstance.NeedsConfigUpdate {
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

			// Load the schedule's timezone with validation
			scheduleTimezone := utils.NormalizeTimezone(schedule.Timezone)

			loc, err := time.LoadLocation(scheduleTimezone)
			if err != nil {
				loc = time.UTC
				scheduleTimezone = "UTC"
			}

			// Convert current time to schedule's timezone
			currentTimeInScheduleTZ := currentTime.In(loc)
			weekday := int(currentTimeInScheduleTZ.Weekday())
			dayBit := 1 << weekday
			currentTimeStr := currentTimeInScheduleTZ.Format("15:04:05")

			// Check day mask
			dayMatch := (schedule.DayMask & dayBit) != 0
			if !dayMatch {
				continue
			}

			// Check time range (schedule times are stored as local times in the schedule's timezone)
			// Handle overnight schedules where end_time < start_time (crosses midnight)
			var timeMatch bool
			if schedule.EndTime < schedule.StartTime {
				// Overnight schedule: active if current time is >= start OR <= end
				timeMatch = currentTimeStr >= schedule.StartTime || currentTimeStr <= schedule.EndTime
			} else {
				// Normal schedule: active if current time is between start and end
				timeMatch = currentTimeStr >= schedule.StartTime && currentTimeStr <= schedule.EndTime
			}
			if timeMatch {
				activeItems = append(activeItems, item)
				break
			}
		}
	}


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
		return importantItems, nil
	}

	// If no important items, return all active items (normal behavior)
	return activeItems, nil
}

// CopyPlaylistItems copies all playlist items from source device to target device
func (pls *PlaylistService) CopyPlaylistItems(sourceDeviceID, targetDeviceID uuid.UUID) error {
	// First, get the target device's user ID for the copied playlists
	var targetDevice Device
	if err := pls.db.First(&targetDevice, "id = ?", targetDeviceID).Error; err != nil {
		return err
	}

	if targetDevice.UserID == nil {
		return gorm.ErrRecordNotFound
	}

	// Get the target device's default playlist (or create one if it doesn't exist)
	var targetPlaylist Playlist
	err := pls.db.Where("device_id = ? AND is_default = ?", targetDeviceID, true).First(&targetPlaylist).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// Create a default playlist for the target device
			targetPlaylist = Playlist{
				UserID:    *targetDevice.UserID,
				DeviceID:  targetDeviceID,
				Name:      "Default Playlist",
				IsDefault: true,
			}
			if err := pls.db.Create(&targetPlaylist).Error; err != nil {
				return err
			}
			logging.Debug("[MIRROR] Created default playlist for target device", "device_id", targetDeviceID.String())
		} else {
			return err
		}
	}

	logging.Debug("[MIRROR] Using target device default playlist", "name", targetPlaylist.Name, "id", targetPlaylist.ID.String())

	// Get all playlists from source device to copy their items
	var sourcePlaylists []Playlist
	if err := pls.db.Where("device_id = ?", sourceDeviceID).Find(&sourcePlaylists).Error; err != nil {
		return err
	}

	logging.Debug("[MIRROR] Starting to copy items from source playlists", "source_count", len(sourcePlaylists), "target_device_id", targetDeviceID.String())

	// Begin transaction
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// Clear existing items in target playlist
		if err := tx.Where("playlist_id = ?", targetPlaylist.ID).Delete(&PlaylistItem{}).Error; err != nil {
			return err
		}
		logging.Debug("[MIRROR] Cleared existing items from target playlist")

		// Copy items from all source playlists into the single target default playlist
		orderIndex := 1
		for playlistIndex, sourcePlaylist := range sourcePlaylists {
			logging.Debug("[MIRROR] Processing source playlist", "current", playlistIndex+1, "total", len(sourcePlaylists), "name", sourcePlaylist.Name, "id", sourcePlaylist.ID.String())

			// Get all playlist items from source playlist
			var sourceItems []PlaylistItem
			if err := tx.Where("playlist_id = ?", sourcePlaylist.ID).Find(&sourceItems).Error; err != nil {
				return err
			}
			
			logging.Debug("[MIRROR] Found items in source playlist", "item_count", len(sourceItems), "playlist_name", sourcePlaylist.Name)

			// Copy each playlist item to the target default playlist
			for itemIndex, sourceItem := range sourceItems {
				logging.Debug("[MIRROR] Copying item", "current", itemIndex+1, "total", len(sourceItems), "plugin_instance_id", sourceItem.PluginInstanceID, "is_visible", sourceItem.IsVisible, "order_index", sourceItem.OrderIndex)

				// Create item with minimum required fields to avoid foreign key constraint errors
				targetItem := PlaylistItem{
					PlaylistID:       targetPlaylist.ID,
					PluginInstanceID: sourceItem.PluginInstanceID,
				}
				if err := tx.Create(&targetItem).Error; err != nil {
					logging.Error("[MIRROR] Error creating target item with required fields", "error", err)
					return err
				}

				logging.Debug("[MIRROR] Target item before update", "is_visible", sourceItem.IsVisible)

				// Use Updates to set remaining fields including false values
				// Use a sequential order index across all source playlists
				updates := map[string]interface{}{
					"order_index":       orderIndex,
					"is_visible":        sourceItem.IsVisible,
					"importance":        sourceItem.Importance,
					"duration_override": sourceItem.DurationOverride,
					"updated_at":        time.Now(),
				}
				orderIndex++ // Increment for next item

				if err := tx.Model(&targetItem).Updates(updates).Error; err != nil {
					logging.Error("[MIRROR] Error updating target item", "error", err)
					return err
				}

				// Verify the item was created correctly
				var verifyItem PlaylistItem
				if err := tx.First(&verifyItem, "id = ?", targetItem.ID).Error; err == nil {
					logging.Debug("[MIRROR] Created item verified", "id", verifyItem.ID, "is_visible", verifyItem.IsVisible)
				} else {
					logging.Error("[MIRROR] Error verifying created item", "error", err)
				}

				// Copy schedules associated with this playlist item
				var sourceSchedules []Schedule
				if err := tx.Where("playlist_item_id = ?", sourceItem.ID).Find(&sourceSchedules).Error; err != nil {
					return err
				}
				
				logging.Debug("[MIRROR] Found schedules for item", "schedule_count", len(sourceSchedules), "item_id", sourceItem.ID.String())

				for scheduleIndex, sourceSchedule := range sourceSchedules {
					logging.Debug("[MIRROR] Copying schedule", "current", scheduleIndex+1, "total", len(sourceSchedules), "name", sourceSchedule.Name, "is_active", sourceSchedule.IsActive, "day_mask", sourceSchedule.DayMask)
					targetSchedule := Schedule{
						PlaylistItemID: targetItem.ID,
						Name:           sourceSchedule.Name,
						DayMask:        sourceSchedule.DayMask,
						StartTime:      sourceSchedule.StartTime,
						EndTime:        sourceSchedule.EndTime,
						Timezone:       sourceSchedule.Timezone,
						IsActive:       sourceSchedule.IsActive,
						CreatedAt:      time.Now(),
						UpdatedAt:      time.Now(),
					}

					if err := tx.Create(&targetSchedule).Error; err != nil {
						logging.Error("[MIRROR] Error creating schedule", "error", err)
						return err
					}
					logging.Debug("[MIRROR] Successfully created schedule", "name", targetSchedule.Name, "id", targetSchedule.ID.String())
				}
			}
		}
		
		logging.Info("[MIRROR] Successfully completed mirroring transaction", "source_device_id", sourceDeviceID.String(), "target_device_id", targetDeviceID.String())

		return nil
	})
}

// ClearMirroredPlaylists removes all playlist items from a device that was mirroring
func (pls *PlaylistService) ClearMirroredPlaylists(deviceID uuid.UUID) error {
	return pls.db.Transaction(func(tx *gorm.DB) error {
		// Get all playlists for this device
		var playlists []Playlist
		if err := tx.Where("device_id = ?", deviceID).Find(&playlists).Error; err != nil {
			return err
		}

		// For each playlist, delete all playlist items and their schedules
		for _, playlist := range playlists {
			// Delete schedules for all playlist items in this playlist
			if err := tx.Where("playlist_item_id IN (SELECT id FROM playlist_items WHERE playlist_id = ?)", playlist.ID).Delete(&Schedule{}).Error; err != nil {
				return err
			}
			
			// Delete all playlist items
			if err := tx.Where("playlist_id = ?", playlist.ID).Delete(&PlaylistItem{}).Error; err != nil {
				return err
			}
		}

		return nil
	})
}

// GetDevicesUsingPluginInstance returns all devices that have the given plugin instance in their playlists
func (pls *PlaylistService) GetDevicesUsingPluginInstance(pluginInstanceID uuid.UUID) ([]Device, error) {
	var devices []Device
	
	// Find all playlists that have items with this plugin instance
	err := pls.db.Distinct().
		Preload("DeviceModel").
		Joins("JOIN playlists ON devices.id = playlists.device_id").
		Joins("JOIN playlist_items ON playlists.id = playlist_items.playlist_id").
		Where("playlist_items.plugin_instance_id = ? AND devices.is_active = ?", pluginInstanceID, true).
		Find(&devices).Error
	
	return devices, err
}
