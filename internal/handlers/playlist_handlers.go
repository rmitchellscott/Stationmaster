package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/rmitchellscott/stationmaster/internal/auth"
	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/sse"
	"github.com/rmitchellscott/stationmaster/internal/utils"
	"gorm.io/gorm"
)

// GetPlaylistsHandler returns all playlists for the current user, optionally filtered by device
func GetPlaylistsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Check if device_id parameter is provided
	deviceIDStr := c.Query("device_id")
	var playlists []database.Playlist
	var err error

	if deviceIDStr != "" {
		// Filter by device ID
		deviceID, parseErr := uuid.Parse(deviceIDStr)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid device ID"})
			return
		}

		// Verify device ownership first
		deviceService := database.NewDeviceService(db)
		device, deviceErr := deviceService.GetDeviceByID(deviceID)
		if deviceErr != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
			return
		}

		if device.UserID == nil || *device.UserID != userUUID {
			c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
			return
		}

		playlists, err = playlistService.GetPlaylistsByDeviceID(deviceID)
	} else {
		// Return all playlists for user
		playlists, err = playlistService.GetPlaylistsByUserID(userUUID)
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlists"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playlists": playlists})
}

// CreatePlaylistHandler creates a new playlist
func CreatePlaylistHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID

	var req struct {
		DeviceID  uuid.UUID `json:"device_id" binding:"required"`
		Name      string    `json:"name" binding:"required"`
		IsDefault bool      `json:"is_default"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	deviceService := database.NewDeviceService(db)
	playlistService := database.NewPlaylistService(db)

	// Verify device ownership
	device, err := deviceService.GetDeviceByID(req.DeviceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Device not found"})
		return
	}

	if device.UserID == nil || *device.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	playlist, err := playlistService.CreatePlaylist(userUUID, req.DeviceID, req.Name, req.IsDefault)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create playlist"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"playlist": playlist})
}

// GetPlaylistHandler returns a specific playlist with its items
func GetPlaylistHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	playlistIDStr := c.Param("id")

	playlistID, err := uuid.Parse(playlistIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	playlist, err := playlistService.GetPlaylistByID(playlistID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	// Verify ownership
	if playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Get playlist items
	items, err := playlistService.GetPlaylistItems(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch playlist items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playlist": playlist, "items": items})
}

// UpdatePlaylistHandler updates a playlist
func UpdatePlaylistHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	playlistIDStr := c.Param("id")

	playlistID, err := uuid.Parse(playlistIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		IsDefault *bool  `json:"is_default"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	playlist, err := playlistService.GetPlaylistByID(playlistID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	// Verify ownership
	if playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	if req.Name != "" {
		playlist.Name = req.Name
	}
	if req.IsDefault != nil {
		playlist.IsDefault = *req.IsDefault
	}

	err = playlistService.UpdatePlaylist(playlist)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"playlist": playlist})
}

// DeletePlaylistHandler deletes a playlist
func DeletePlaylistHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	playlistIDStr := c.Param("id")

	playlistID, err := uuid.Parse(playlistIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	playlist, err := playlistService.GetPlaylistByID(playlistID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	// Verify ownership
	if playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Don't allow deletion of default playlist
	if playlist.IsDefault {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot delete default playlist"})
		return
	}

	err = playlistService.DeletePlaylist(playlistID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete playlist"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist deleted successfully"})
}

// AddPlaylistItemHandler adds an item to a playlist
func AddPlaylistItemHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	playlistIDStr := c.Param("id")

	playlistID, err := uuid.Parse(playlistIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	var req struct {
		PluginInstanceID uuid.UUID `json:"plugin_instance_id" binding:"required"`
		Importance       bool      `json:"importance"`
		DurationOverride *int      `json:"duration_override"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)
	unifiedPluginService := database.NewUnifiedPluginService(db)

	// Verify playlist ownership
	playlist, err := playlistService.GetPlaylistByID(playlistID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Verify plugin instance ownership
	pluginInstance, err := unifiedPluginService.GetPluginInstanceByID(req.PluginInstanceID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Plugin instance not found"})
		return
	}

	if pluginInstance.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	item, err := playlistService.AddItemToPlaylist(playlistID, req.PluginInstanceID, req.Importance, req.DurationOverride)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add item to playlist"})
		return
	}

	// Schedule immediate independent render for the plugin instance
	renderJob := database.RenderQueue{
		ID:               uuid.New(),
		PluginInstanceID: req.PluginInstanceID,
		Priority:         999, // High priority for immediate render on playlist addition
		ScheduledFor:     time.Now(),
		Status:           "pending",
		IndependentRender: true, // Playlist additions are independent renders
	}
	
	err = db.Create(&renderJob).Error
	if err != nil {
		logging.Error("[PLAYLIST] Failed to schedule immediate render job for playlist addition", "plugin_instance_id", req.PluginInstanceID, "error", err)
		// Don't fail the playlist addition if render scheduling fails
	} else {
		logging.Info("[PLAYLIST] Scheduled immediate render for playlist addition", "plugin_instance_id", req.PluginInstanceID, "job_id", renderJob.ID)
	}

	// Broadcast playlist item added event
	sseService := sse.GetSSEService()
	sseService.BroadcastToDevice(playlist.DeviceID, sse.Event{
		Type: "playlist_item_added",
		Data: map[string]interface{}{
			"device_id":     playlist.DeviceID.String(),
			"playlist_id":   playlist.ID.String(),
			"playlist_item": item,
			"timestamp":     time.Now().UTC(),
		},
	})

	c.JSON(http.StatusCreated, gin.H{"playlist_item": item})
}

// UpdatePlaylistItemHandler updates a playlist item
func UpdatePlaylistItemHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	itemIDStr := c.Param("itemId")

	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var req struct {
		IsVisible        *bool `json:"is_visible"`
		Importance       *bool `json:"importance"`
		DurationOverride *int  `json:"duration_override"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	item, err := playlistService.GetPlaylistItemByID(itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist item not found"})
		return
	}

	// Verify ownership through playlist
	playlist, err := playlistService.GetPlaylistByID(item.PlaylistID)
	if err != nil || playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	if req.IsVisible != nil {
		item.IsVisible = *req.IsVisible
	}
	if req.Importance != nil {
		item.Importance = *req.Importance
	}
	// Always update duration_override field when provided (including null values)
	item.DurationOverride = req.DurationOverride

	err = playlistService.UpdatePlaylistItem(item)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update playlist item"})
		return
	}

	// Determine the event type based on what was updated
	eventType := "playlist_item_updated"
	if req.IsVisible != nil {
		eventType = "playlist_item_visibility_changed"
	}

	// Broadcast playlist item update event
	sseService := sse.GetSSEService()
	sseService.BroadcastToDevice(playlist.DeviceID, sse.Event{
		Type: eventType,
		Data: map[string]interface{}{
			"device_id":     playlist.DeviceID.String(),
			"playlist_id":   playlist.ID.String(),
			"playlist_item": item,
			"timestamp":     time.Now().UTC(),
		},
	})

	c.JSON(http.StatusOK, gin.H{"playlist_item": item})
}

// DeletePlaylistItemHandler removes an item from a playlist
func DeletePlaylistItemHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	itemIDStr := c.Param("itemId")

	logging.InfoWithComponent(logging.ComponentPlaylist, "User attempting to delete playlist item", "user", userUUID.String(), "item_id", itemIDStr)

	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		logging.WarnWithComponent(logging.ComponentPlaylist, "Invalid item ID format", "item_id", itemIDStr)
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID format"})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Get the playlist item with detailed logging
	item, err := playlistService.GetPlaylistItemByID(itemID)
	if err != nil {
		logging.WarnWithComponent(logging.ComponentPlaylist, "Playlist item not found", "item_id", itemID.String(), "error", err)
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist item not found"})
		return
	}

	logging.InfoWithComponent(logging.ComponentPlaylist, "Found playlist item", "item_id", itemID.String(), "playlist_id", item.PlaylistID.String())

	// Verify ownership through playlist
	playlist, err := playlistService.GetPlaylistByID(item.PlaylistID)
	if err != nil {
		logging.ErrorWithComponent(logging.ComponentPlaylist, "Failed to get playlist", "playlist_id", item.PlaylistID.String(), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify playlist ownership"})
		return
	}

	if playlist.UserID != userUUID {
		logging.WarnWithComponent(logging.ComponentPlaylist, "Access denied - user does not own playlist", "user", userUUID.String(), "playlist_id", playlist.ID.String(), "owner", playlist.UserID.String())
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	logging.InfoWithComponent(logging.ComponentPlaylist, "Ownership verified, proceeding with deletion")

	// Attempt to delete the playlist item
	err = playlistService.DeletePlaylistItem(itemID)
	if err != nil {
		logging.ErrorWithComponent(logging.ComponentPlaylist, "Failed to delete playlist item", "item_id", itemID.String(), "error", err)

		// Provide more specific error messages
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "Playlist item not found"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete playlist item: " + err.Error()})
		}
		return
	}

	logging.InfoWithComponent(logging.ComponentPlaylist, "Successfully deleted playlist item", "item_id", itemID.String())

	// Broadcast playlist item removed event
	sseService := sse.GetSSEService()
	sseService.BroadcastToDevice(playlist.DeviceID, sse.Event{
		Type: "playlist_item_removed",
		Data: map[string]interface{}{
			"device_id":   playlist.DeviceID.String(),
			"playlist_id": playlist.ID.String(),
			"item_id":     itemID.String(),
			"timestamp":   time.Now().UTC(),
		},
	})


	c.JSON(http.StatusOK, gin.H{"message": "Playlist item deleted successfully"})
}

// ReorderPlaylistItemsHandler updates the order of playlist items
func ReorderPlaylistItemsHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	playlistIDStr := c.Param("id")

	playlistID, err := uuid.Parse(playlistIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	var req struct {
		ItemOrders map[string]int `json:"item_orders" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Verify playlist ownership
	playlist, err := playlistService.GetPlaylistByID(playlistID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Convert string keys to UUIDs
	itemOrders := make(map[uuid.UUID]int)
	for itemIDStr, order := range req.ItemOrders {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID in order map"})
			return
		}
		itemOrders[itemID] = order
	}

	err = playlistService.ReorderPlaylistItems(playlistID, itemOrders)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder playlist items"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Playlist items reordered successfully"})
}

// ReorderPlaylistItemsArrayHandler updates the order of playlist items based on an ordered array
func ReorderPlaylistItemsArrayHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	playlistIDStr := c.Param("id")

	playlistID, err := uuid.Parse(playlistIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid playlist ID"})
		return
	}

	var req struct {
		ItemIDs []string `json:"item_ids" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Verify playlist ownership
	playlist, err := playlistService.GetPlaylistByID(playlistID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found"})
		return
	}

	if playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Convert string IDs to UUIDs
	orderedItemIDs := make([]uuid.UUID, len(req.ItemIDs))
	for i, itemIDStr := range req.ItemIDs {
		itemID, err := uuid.Parse(itemIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID in order array"})
			return
		}
		orderedItemIDs[i] = itemID
	}

	err = playlistService.ReorderPlaylistItemsByArray(playlistID, orderedItemIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reorder playlist items"})
		return
	}

	// Broadcast playlist item reordered event
	sseService := sse.GetSSEService()
	sseService.BroadcastToDevice(playlist.DeviceID, sse.Event{
		Type: "playlist_item_reordered",
		Data: map[string]interface{}{
			"device_id":        playlist.DeviceID.String(),
			"playlist_id":      playlist.ID.String(),
			"ordered_item_ids": req.ItemIDs,
			"timestamp":        time.Now().UTC(),
		},
	})

	c.JSON(http.StatusOK, gin.H{"message": "Playlist items reordered successfully"})
}

// Schedule handlers

// AddScheduleHandler adds a schedule to a playlist item
func AddScheduleHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	itemIDStr := c.Param("itemId")

	itemID, err := uuid.Parse(itemIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid item ID"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		DayMask   int    `json:"day_mask" binding:"required"`
		StartTime string `json:"start_time" binding:"required"`
		EndTime   string `json:"end_time" binding:"required"`
		Timezone  string `json:"timezone"`
		IsActive  *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Timezone == "" {
		req.Timezone = "UTC"
	}

	// Validate timezone
	if err := utils.ValidateTimezone(req.Timezone); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timezone: " + err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Verify ownership through playlist
	item, err := playlistService.GetPlaylistItemByID(itemID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist item not found"})
		return
	}

	playlist, err := playlistService.GetPlaylistByID(item.PlaylistID)
	if err != nil || playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	isActive := true // Default to active for backward compatibility
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	schedule, err := playlistService.AddScheduleToPlaylistItem(itemID, req.Name, req.DayMask, req.StartTime, req.EndTime, req.Timezone, isActive)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to add schedule"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"schedule": schedule})
}

// UpdateScheduleHandler updates a schedule
func UpdateScheduleHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	scheduleIDStr := c.Param("scheduleId")

	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	var req struct {
		Name      string `json:"name"`
		DayMask   *int   `json:"day_mask"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
		Timezone  string `json:"timezone"`
		IsActive  *bool  `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Get schedule and verify ownership
	var schedule database.Schedule
	err = db.Preload("PlaylistItem").Preload("PlaylistItem.Playlist").First(&schedule, "id = ?", scheduleID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	if schedule.PlaylistItem.Playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Update fields
	if req.Name != "" {
		schedule.Name = req.Name
	}
	if req.DayMask != nil {
		schedule.DayMask = *req.DayMask
	}
	if req.StartTime != "" {
		schedule.StartTime = req.StartTime
	}
	if req.EndTime != "" {
		schedule.EndTime = req.EndTime
	}
	if req.Timezone != "" {
		// Validate timezone
		if err := utils.ValidateTimezone(req.Timezone); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid timezone: " + err.Error()})
			return
		}
		schedule.Timezone = req.Timezone
	}
	if req.IsActive != nil {
		schedule.IsActive = *req.IsActive
	}

	err = playlistService.UpdateSchedule(&schedule)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"schedule": schedule})
}

// DeleteScheduleHandler deletes a schedule
func DeleteScheduleHandler(c *gin.Context) {
	user, ok := auth.RequireUser(c)
	if !ok {
		return
	}
	userUUID := user.ID
	scheduleIDStr := c.Param("scheduleId")

	scheduleID, err := uuid.Parse(scheduleIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid schedule ID"})
		return
	}

	db := database.GetDB()
	playlistService := database.NewPlaylistService(db)

	// Get schedule and verify ownership
	var schedule database.Schedule
	err = db.Preload("PlaylistItem").Preload("PlaylistItem.Playlist").First(&schedule, "id = ?", scheduleID).Error
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Schedule not found"})
		return
	}

	if schedule.PlaylistItem.Playlist.UserID != userUUID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	err = playlistService.DeleteSchedule(scheduleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete schedule"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Schedule deleted successfully"})
}
