package trmnl

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/rmitchellscott/stationmaster/internal/database"
	"github.com/rmitchellscott/stationmaster/internal/logging"
	"github.com/rmitchellscott/stationmaster/internal/plugins"
	"github.com/rmitchellscott/stationmaster/internal/rendering"
	"github.com/rmitchellscott/stationmaster/internal/sse"
	"github.com/rmitchellscott/stationmaster/internal/storage"
)

// PluginProcessor handles processing plugins with the unified architecture
type PluginProcessor struct {
	imageStorage        *storage.ImageStorage
	db                  *gorm.DB
	queueManager        *rendering.QueueManager
	pluginService       *database.UnifiedPluginService
	pluginFactory       *plugins.UnifiedPluginFactory
	browserlessRenderer *rendering.BrowserlessRenderer
}

// generateRandomString creates a cryptographically secure random string
func generateRandomString(length int) string {
	bytes := make([]byte, length/2)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based randomness if crypto/rand fails
		return fmt.Sprintf("%x", time.Now().UnixNano())[:length]
	}
	return hex.EncodeToString(bytes)[:length]
}

// findItemByID finds a playlist item by UUID in the active items array
func findItemByID(activeItems []database.PlaylistItem, itemID *uuid.UUID) *database.PlaylistItem {
	if itemID == nil {
		return nil
	}
	for i := range activeItems {
		if activeItems[i].ID == *itemID {
			return &activeItems[i]
		}
	}
	return nil
}

// findNextActiveItem finds the next active item after the given item by order_index
func findNextActiveItem(activeItems []database.PlaylistItem, currentItem *database.PlaylistItem) *database.PlaylistItem {
	if len(activeItems) == 0 {
		return nil
	}
	
	if len(activeItems) == 1 {
		return &activeItems[0] // Only one item, return it
	}
	
	if currentItem == nil {
		// No current item, return first by order_index
		return &activeItems[0]
	}
	
	// Sort items by order_index to ensure consistent ordering
	sortedItems := make([]database.PlaylistItem, len(activeItems))
	copy(sortedItems, activeItems)
	
	// Items are already sorted by order_index from GetActivePlaylistItemsForTime
	// Find current item and return next one
	for i, item := range sortedItems {
		if item.ID == currentItem.ID {
			// Return next item (wrap around if at end)
			nextIndex := (i + 1) % len(sortedItems)
			return &sortedItems[nextIndex]
		}
	}
	
	// Current item not found in active items, return first one
	return &sortedItems[0]
}

// NewPluginProcessor creates a new plugin processor with unified architecture
func NewPluginProcessor(db *gorm.DB) (*PluginProcessor, error) {
	imageStorage := storage.GetDefaultImageStorage()
	queueManager := rendering.NewQueueManager(db)
	pluginService := database.NewUnifiedPluginService(db)
	pluginFactory := plugins.NewUnifiedPluginFactory(db)

	// Initialize browserless renderer (optional dependency)
	browserlessRenderer, err := rendering.NewBrowserlessRenderer()
	if err != nil {
		logging.Warn("[PLUGIN_PROCESSOR] Browserless renderer not available", "error", err)
		browserlessRenderer = nil
	}

	return &PluginProcessor{
		imageStorage:        imageStorage,
		db:                  db,
		queueManager:        queueManager,
		pluginService:       pluginService,
		pluginFactory:       pluginFactory,
		browserlessRenderer: browserlessRenderer,
	}, nil
}

// Close cleans up resources
func (pp *PluginProcessor) Close() error {
	return nil
}

// processUnifiedPluginInstance processes a unified plugin instance
func (pp *PluginProcessor) processUnifiedPluginInstance(device *database.Device, pluginInstance *database.PluginInstance) (gin.H, error) {
	// Get the plugin definition
	definition, err := pp.pluginService.GetPluginDefinitionByID(pluginInstance.PluginDefinitionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin definition: %w", err)
	}
	
	// Create the plugin using the factory
	plugin, err := pp.pluginFactory.CreatePlugin(definition, pluginInstance)
	if err != nil {
		return nil, fmt.Errorf("failed to create plugin: %w", err)
	}
	
	// Check if plugin requires processing before looking for pre-rendered content
	var renderedContent *database.RenderedContent
	if plugin.RequiresProcessing() {
		// First, try to get pre-rendered content
		renderedContent, err = pp.getPreRenderedContentForInstance(pluginInstance.ID, device)
		if err != nil {
			logging.Error("[PLUGIN] Failed to check for pre-rendered content", "error", err)
		}
	}
	
	var response gin.H
	var pluginErr error
	
	if renderedContent != nil {
		// Use pre-rendered content
		var imageURL string
		if strings.HasPrefix(renderedContent.ImagePath, "/static/rendered/") {
			// Already a properly formatted URL
			imageURL = renderedContent.ImagePath
		} else if filepath.IsAbs(renderedContent.ImagePath) {
			// Local file path - convert to URL
			relPath, err := filepath.Rel(pp.imageStorage.GetBasePath(), renderedContent.ImagePath)
			if err != nil {
				logging.Error("[PLUGIN] Failed to compute relative path", "path", renderedContent.ImagePath, "error", err)
				imageURL = renderedContent.ImagePath // Fallback to original path
			} else {
				imageURL = "/static/rendered/" + relPath
			}
		} else {
			// URL reference
			imageURL = renderedContent.ImagePath
		}
		
		response = gin.H{
			"image_url": imageURL,
			"filename":  fmt.Sprintf("pre_rendered_%s", time.Now().Format("20060102150405")),
		}
		
		logging.Info("[PLUGIN] Using pre-rendered content", 
			"plugin_type", plugin.Type(), 
			"plugin_name", pluginInstance.Name)
	} else {
		// No pre-rendered content available - skip this playlist item instead of blocking
		if plugin.RequiresProcessing() {
			logging.Info("[PLUGIN] No pre-rendered content available, skipping playlist item", "plugin_type", plugin.Type(), "plugin_name", pluginInstance.Name)
			// Schedule an immediate render job so it's ready next time
			pp.scheduleImmediateRenderForInstance(pluginInstance.ID)
			
			// Return a special response indicating this item should be skipped
			return gin.H{
				"skip_item": true,
				"reason":    "no_pre_rendered_content",
				"plugin_type": plugin.Type(),
				"plugin_name": pluginInstance.Name,
			}, nil
		}
		
		// For plugins that don't require processing, we can still process them on-demand
		// Create unified plugin context
		ctx, err := pp.createUnifiedPluginContext(device, pluginInstance)
		if err != nil {
			return nil, fmt.Errorf("failed to create plugin context: %w", err)
		}
		
		// Process the plugin (only for non-processing plugins)
		response, pluginErr = plugin.Process(ctx)
		if pluginErr != nil {
			logging.Error("[PLUGIN] Plugin processing failed", "plugin_type", plugin.Type(), "error", pluginErr)
			// Return error response but don't fail the whole request
			response = gin.H{
				"image_url": getImageURLForDevice(device),
				"filename":  fmt.Sprintf("error_%s", time.Now().Format("20060102150405")),
			}
		} else {
			// Since plugin doesn't require processing, we can use the response directly
			logging.Debug("[PLUGIN] Plugin processed successfully (no processing required)", "plugin_type", plugin.Type())
		}
	}
	
	return response, pluginErr
}

// renderHTMLToImage converts HTML content to an image using browserless
func (pp *PluginProcessor) renderHTMLToImage(htmlContent string, device *database.Device) ([]byte, error) {
	if device.DeviceModel == nil {
		return nil, fmt.Errorf("device model not available")
	}
	
	// Check if browserless renderer is available
	if pp.browserlessRenderer == nil {
		// Return error image SVG when browserless is not available
		errorSVG := fmt.Sprintf(`
			<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
				<rect width="100%%" height="100%%" fill="#fee"/>
				<text x="50%%" y="45%%" text-anchor="middle" dominant-baseline="middle" 
					  font-family="Arial, sans-serif" font-size="14" fill="#c33">
					Browserless Service Unavailable
				</text>
				<text x="50%%" y="55%%" text-anchor="middle" dominant-baseline="middle" 
					  font-family="Arial, sans-serif" font-size="12" fill="#666">
					Configure BROWSERLESS_URL to render HTML
				</text>
			</svg>
		`, device.DeviceModel.ScreenWidth, device.DeviceModel.ScreenHeight)
		return []byte(errorSVG), nil
	}
	
	// Use browserless to render HTML to PNG
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	
	imageData, err := pp.browserlessRenderer.RenderHTML(ctx, htmlContent, device.DeviceModel.ScreenWidth, device.DeviceModel.ScreenHeight)
	if err != nil {
		logging.Error("[PLUGIN_PROCESSOR] Browserless rendering failed", "error", err)
		
		// Return error image SVG when browserless fails
		errorSVG := fmt.Sprintf(`
			<svg width="%d" height="%d" xmlns="http://www.w3.org/2000/svg">
				<rect width="100%%" height="100%%" fill="#fee"/>
				<text x="50%%" y="45%%" text-anchor="middle" dominant-baseline="middle" 
					  font-family="Arial, sans-serif" font-size="14" fill="#c33">
					HTML Rendering Failed
				</text>
				<text x="50%%" y="55%%" text-anchor="middle" dominant-baseline="middle" 
					  font-family="Arial, sans-serif" font-size="12" fill="#666">
					Check browserless service status
				</text>
			</svg>
		`, device.DeviceModel.ScreenWidth, device.DeviceModel.ScreenHeight)
		return []byte(errorSVG), nil
	}
	
	return imageData, nil
}

// getPreRenderedContentForInstance attempts to get pre-rendered content for a plugin instance
func (pp *PluginProcessor) getPreRenderedContentForInstance(pluginInstanceID uuid.UUID, device *database.Device) (*database.RenderedContent, error) {
	var renderedContent database.RenderedContent
	
	// Get device specifications from device model
	if device.DeviceModel == nil {
		return nil, nil // No device model, can't match resolution
	}
	
	// Look for pre-rendered content matching this device model's specifications
	err := pp.db.Where("plugin_instance_id = ? AND width = ? AND height = ? AND bit_depth = ?",
		pluginInstanceID, device.DeviceModel.ScreenWidth, device.DeviceModel.ScreenHeight, device.DeviceModel.BitDepth).
		Order("rendered_at DESC").
		First(&renderedContent).Error
	
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No pre-rendered content available
		}
		return nil, fmt.Errorf("failed to query rendered content: %w", err)
	}
	
	return &renderedContent, nil
}

// scheduleRenderIfNeededForInstance schedules a render job for a plugin instance if no recent content exists
func (pp *PluginProcessor) scheduleRenderIfNeededForInstance(pluginInstanceID uuid.UUID) {
	// TODO: Update QueueManager to work with plugin instances
	// For now, skip scheduling for unified instances
	logging.Debug("[PLUGIN_PROCESSOR] Skipping render scheduling for unified plugin instance", "instance_id", pluginInstanceID)
}

// scheduleImmediateRenderForInstance schedules an immediate high-priority render job for a plugin instance
func (pp *PluginProcessor) scheduleImmediateRenderForInstance(pluginInstanceID uuid.UUID) {
	if pp.queueManager != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		
		err := pp.queueManager.ScheduleImmediateRender(ctx, pluginInstanceID)
		if err != nil {
			logging.Error("[PLUGIN_PROCESSOR] Failed to schedule immediate render", "plugin_id", pluginInstanceID, "error", err)
		} else {
			logging.Info("[PLUGIN_PROCESSOR] Scheduled immediate render", "plugin_id", pluginInstanceID)
		}
	} else {
		logging.Warn("[PLUGIN_PROCESSOR] Queue manager not available for immediate render", "plugin_id", pluginInstanceID)
	}
}

// createUnifiedPluginContext creates a plugin context for unified plugin instances
func (pp *PluginProcessor) createUnifiedPluginContext(device *database.Device, pluginInstance *database.PluginInstance) (plugins.PluginContext, error) {
	// Parse instance settings
	settings, err := pp.pluginService.GetPluginInstanceSettings(pluginInstance.ID)
	if err != nil {
		return plugins.PluginContext{}, fmt.Errorf("failed to get plugin instance settings: %w", err)
	}
	
	// Use PluginInstance directly with PluginContext
	// TODO: Update PluginContext to work directly with PluginInstance
	return plugins.PluginContext{
		Device:         device,
		PluginInstance: pluginInstance,
		Settings:       settings,
	}, nil
}

// getPreRenderedContent attempts to get pre-rendered content for a plugin
func (pp *PluginProcessor) getPreRenderedContent(userPluginID uuid.UUID, device *database.Device) (*database.RenderedContent, error) {
	var renderedContent database.RenderedContent

	// Get device specifications from device model
	if device.DeviceModel == nil {
		return nil, nil // No device model, can't match resolution
	}

	// Look for pre-rendered content matching this device model's specifications
	err := pp.db.Where("plugin_instance_id = ? AND width = ? AND height = ? AND bit_depth = ?",
		userPluginID, device.DeviceModel.ScreenWidth, device.DeviceModel.ScreenHeight, device.DeviceModel.BitDepth).
		Order("rendered_at DESC").
		First(&renderedContent).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil // No pre-rendered content available
		}
		return nil, fmt.Errorf("failed to query rendered content: %w", err)
	}

	return &renderedContent, nil
}

// scheduleRenderIfNeeded schedules a render job if no recent content exists
func (pp *PluginProcessor) scheduleRenderIfNeeded(userPluginID uuid.UUID) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := pp.queueManager.ScheduleImmediateRender(ctx, userPluginID)
	if err != nil {
		logging.Error("[PLUGIN_PROCESSOR] Failed to schedule render", "plugin_id", userPluginID, "error", err)
	}
}

// processActivePlugins processes plugins using the unified plugin architecture only
func (pp *PluginProcessor) processActivePlugins(device *database.Device, activeItems []database.PlaylistItem) (gin.H, *database.PlaylistItem, error) {
	if len(activeItems) == 0 {
		return nil, nil, fmt.Errorf("no active playlist items")
	}

	// Find the currently showing item by UUID
	currentItem := findItemByID(activeItems, device.LastPlaylistItemID)
	
	// Get the next item in rotation using UUID-based logic
	nextItem := findNextActiveItem(activeItems, currentItem)
	if nextItem == nil {
		return nil, nil, fmt.Errorf("no valid next item found")
	}
	
	item := *nextItem

	// Check if plugin instance ID is valid
	if item.PluginInstanceID == uuid.Nil {
		errorMsg := "Playlist item has no plugin instance configured"
		logging.Warn("[PLUGIN] Skipping playlist item", "error", errorMsg, "item_id", item.ID)
		
		return gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  fmt.Sprintf("no_plugin_%s", time.Now().Format("20060102150405")),
		}, &item, fmt.Errorf(errorMsg)
	}

	// Get the plugin instance
	pluginInstance, err := pp.pluginService.GetPluginInstanceByID(item.PluginInstanceID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get plugin instance: %w", err)
	}

	// Process using unified system
	response, err := pp.processUnifiedPluginInstance(device, pluginInstance)
	if err != nil {
		logging.Error("[PLUGIN] Unified plugin processing failed", "plugin_instance_id", pluginInstance.ID, "error", err)
		// Return error response but don't fail the whole request
		response = gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  fmt.Sprintf("error_%s", time.Now().Format("20060102150405")),
		}
		return response, &item, err
	}

	// Check if the plugin requested to skip this item
	if skipItem, ok := response["skip_item"].(bool); ok && skipItem {
		logging.Info("[PLUGIN] Skipping playlist item due to missing pre-rendered content", "plugin_type", response["plugin_type"], "plugin_name", response["plugin_name"])
		
		// Update device's last item to current (skipped) item to track progression
		deviceService := database.NewDeviceService(pp.db)
		if err := deviceService.UpdateLastPlaylistItemID(device.ID, item.ID); err != nil {
			logging.Warn("[PLUGIN] Failed to update last playlist item ID for skipped item", "error", err)
		}
		
		// Find next item after the one we're skipping
		nextItemAfterSkip := findNextActiveItem(activeItems, &item)
		
		// Prevent infinite loop if all items are skipped (only one item available and it's the skipped one)
		if nextItemAfterSkip != nil && nextItemAfterSkip.ID == item.ID && len(activeItems) == 1 {
			logging.Warn("[PLUGIN] All playlist items require skipping, returning error image")
			return gin.H{
				"image_url": getImageURLForDevice(device),
				"filename":  fmt.Sprintf("all_skipped_%s", time.Now().Format("20060102150405")),
			}, &item, fmt.Errorf("all playlist items require skipping")
		}
		
		// Update device to track next item and recursively try it
		if nextItemAfterSkip != nil {
			device.LastPlaylistItemID = &nextItemAfterSkip.ID
		}
		return pp.processActivePlugins(device, activeItems)
	}

	// Apply duration override (takes priority over plugin refresh_rate)
	if item.DurationOverride != nil {
		response["refresh_rate"] = fmt.Sprintf("%d", *item.DurationOverride)
	}
	
	// Return the successful result
	return response, &item, nil
}

// renderDataPlugin renders a data plugin response to an image
func (pp *PluginProcessor) renderDataPlugin(response plugins.PluginResponse, device *database.Device, pluginType string) (gin.H, error) {
	return nil, fmt.Errorf("HTML rendering not available - data plugins are not supported without Chromium")
}

// processCurrentPlugin processes the current plugin without advancing the index (unified system only)
func (pp *PluginProcessor) processCurrentPlugin(device *database.Device, activeItems []database.PlaylistItem) (gin.H, error) {
	if len(activeItems) == 0 {
		return nil, fmt.Errorf("no active playlist items")
	}

	// Get the current item based on UUID
	currentItem := findItemByID(activeItems, device.LastPlaylistItemID)
	if currentItem == nil {
		// No current item set or item not found, use first available
		if len(activeItems) > 0 {
			currentItem = &activeItems[0]
		} else {
			return nil, fmt.Errorf("no active playlist items available")
		}
	}

	item := *currentItem

	// Check if plugin instance ID is valid
	if item.PluginInstanceID == uuid.Nil {
		errorMsg := "Current playlist item has no plugin instance configured"
		logging.Warn("[PLUGIN] Skipping current playlist item", "error", errorMsg, "item_id", item.ID)
		
		return gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  fmt.Sprintf("no_plugin_%s", time.Now().Format("20060102150405")),
		}, fmt.Errorf(errorMsg)
	}

	// Get the plugin instance
	pluginInstance, err := pp.pluginService.GetPluginInstanceByID(item.PluginInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to get plugin instance: %w", err)
	}

	// Process using unified system
	response, err := pp.processUnifiedPluginInstance(device, pluginInstance)
	if err != nil {
		logging.Error("[PLUGIN] Unified plugin processing failed (current)", "plugin_instance_id", pluginInstance.ID, "error", err)
		// Return error response
		response = gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  fmt.Sprintf("error_%s", time.Now().Format("20060102150405")),
		}
		return response, err
	}

	// Check if the plugin requested to skip this item
	if skipItem, ok := response["skip_item"].(bool); ok && skipItem {
		logging.Info("[PLUGIN] Current playlist item needs to be skipped, returning error image", "plugin_type", response["plugin_type"], "plugin_name", response["plugin_name"])
		
		// For the current item request, we can't easily skip to the next item since this function
		// doesn't manage playlist state. Return an error image instead.
		return gin.H{
			"image_url": getImageURLForDevice(device),
			"filename":  fmt.Sprintf("skipped_current_%s", time.Now().Format("20060102150405")),
		}, fmt.Errorf("current playlist item requires skipping due to missing pre-rendered content")
	}

	// Apply duration override (takes priority over plugin refresh_rate)
	if item.DurationOverride != nil {
		response["refresh_rate"] = fmt.Sprintf("%d", *item.DurationOverride)
	}

	return response, nil
}

// broadcastPlaylistChange broadcasts playlist changes via SSE
func (pp *PluginProcessor) broadcastPlaylistChange(device *database.Device, currentItem database.PlaylistItem, activeItems []database.PlaylistItem) {
	// Get user timezone for sleep calculations
	userTimezone := "UTC" // Default fallback
	if device.UserID != nil {
		db := database.GetDB()
		userService := database.NewUserService(db)
		user, err := userService.GetUserByID(*device.UserID)
		if err == nil && user.Timezone != "" {
			userTimezone = user.Timezone
		}
	}

	// Check if device is currently in sleep period for SSE event
	currentlySleeping := isInSleepPeriod(device, userTimezone)

	// Calculate current index for compatibility (frontend still expects it)
	currentIndex := -1
	for i, activeItem := range activeItems {
		if activeItem.ID == currentItem.ID {
			currentIndex = i
			break
		}
	}

	// Broadcast playlist change to connected SSE clients
	sseService := sse.GetSSEService()
	sseService.BroadcastToDevice(device.ID, sse.Event{
		Type: "playlist_index_changed",
		Data: map[string]interface{}{
			"device_id":     device.ID.String(),
			"current_index": currentIndex,
			"current_item":  currentItem,
			"active_items":  activeItems,
			"timestamp":     time.Now().UTC(),
			"sleep_config": map[string]interface{}{
				"enabled":            device.SleepEnabled,
				"start_time":         device.SleepStartTime,
				"end_time":           device.SleepEndTime,
				"show_screen":        device.SleepShowScreen,
				"currently_sleeping": currentlySleeping,
			},
		},
	})
}

// Global plugin processor instance
var globalProcessor *PluginProcessor

// GetPluginProcessor returns the global plugin processor instance
func GetPluginProcessor() *PluginProcessor {
	return globalProcessor
}

// InitPluginProcessor initializes the global plugin processor
func InitPluginProcessor(db *gorm.DB) error {
	var err error
	globalProcessor, err = NewPluginProcessor(db)
	return err
}

// CleanupPluginProcessor cleans up the global plugin processor
func CleanupPluginProcessor() error {
	if globalProcessor != nil {
		return globalProcessor.Close()
	}
	return nil
}
