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

// PluginProcessor handles processing plugins with the new architecture
type PluginProcessor struct {
	imageStorage *storage.ImageStorage
	db           *gorm.DB
	queueManager *rendering.QueueManager
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

// NewPluginProcessor creates a new plugin processor
func NewPluginProcessor(db *gorm.DB) (*PluginProcessor, error) {
	imageStorage := storage.GetDefaultImageStorage()
	queueManager := rendering.NewQueueManager(db)

	return &PluginProcessor{
		imageStorage: imageStorage,
		db:           db,
		queueManager: queueManager,
	}, nil
}

// Close cleans up resources
func (pp *PluginProcessor) Close() error {
	return nil
}

// getPreRenderedContent attempts to get pre-rendered content for a plugin
func (pp *PluginProcessor) getPreRenderedContent(userPluginID uuid.UUID, device *database.Device) (*database.RenderedContent, error) {
	var renderedContent database.RenderedContent

	// Get device specifications from device model
	if device.DeviceModel == nil {
		return nil, nil // No device model, can't match resolution
	}

	// Look for pre-rendered content matching this device model's specifications
	err := pp.db.Where("user_plugin_id = ? AND width = ? AND height = ? AND bit_depth = ?",
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

// processActivePlugins processes plugins using the new plugin architecture
func (pp *PluginProcessor) processActivePlugins(device *database.Device, activeItems []database.PlaylistItem) (gin.H, *database.PlaylistItem, error) {
	if len(activeItems) == 0 {
		return nil, nil, fmt.Errorf("no active playlist items")
	}

	// Calculate next item index for rotation
	nextIndex := 0
	if len(activeItems) > 1 {
		nextIndex = (device.LastPlaylistIndex + 1) % len(activeItems)
	}

	// Get the next item in rotation
	item := activeItems[nextIndex]

	// Get the user plugin details
	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(item.UserPluginID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user plugin: %w", err)
	}

	// Check if plugin requires processing before looking for pre-rendered content
	var renderedContent *database.RenderedContent
	plugin, exists := plugins.Get(userPlugin.Plugin.Type)
	if exists && plugin.RequiresProcessing() {
		// First, try to get pre-rendered content
		var err error
		renderedContent, err = pp.getPreRenderedContent(userPlugin.ID, device)
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
			// Already a properly formatted URL (from image plugins)
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

		logging.Info("[PLUGIN] Using pre-rendered content", "plugin_type", userPlugin.Plugin.Type)
	} else {
		// No pre-rendered content available, fall back to on-demand processing
		if exists && plugin.RequiresProcessing() {
			logging.Debug("[PLUGIN] No pre-rendered content, falling back to on-demand", "plugin_type", userPlugin.Plugin.Type)
			// Schedule a render job for next time (only if plugin requires processing)
			pp.scheduleRenderIfNeeded(userPlugin.ID)
		}

		if !exists {
			// Fallback for unknown plugin types
			response = gin.H{
				"image_url": getImageURLForDevice(device),
				"filename":  "display.png",
			}
		} else {
			// Create plugin context
			ctx, err := plugins.NewPluginContext(device, userPlugin)
			if err != nil {
				return nil, nil, fmt.Errorf("failed to create plugin context: %w", err)
			}

			// Process the plugin
			response, pluginErr = plugin.Process(ctx)
			if pluginErr != nil {
				logging.Error("[PLUGIN] Plugin processing failed", "plugin_type", plugin.Type(), "error", pluginErr)
				// Return error response but don't fail the whole request
				response = gin.H{
					"image_url": getImageURLForDevice(device),
					"filename":  fmt.Sprintf("error_%s", time.Now().Format("20060102150405")),
				}
			} else {
				// Handle plugins that require processing
				if plugin.RequiresProcessing() {
					if plugins.IsDataResponse(response) {
						// Data plugin - needs HTML template rendering
						response, err = pp.renderDataPlugin(response, device, plugin.Type())
						if err != nil {
							logging.Error("[PLUGIN] Failed to render data plugin", "plugin_type", plugin.Type(), "error", err)
							// Fallback to default response
							response = gin.H{
								"image_url": getImageURLForDevice(device),
								"filename":  fmt.Sprintf("render_error_%s", time.Now().Format("20060102150405")),
							}
						}
					} else if plugins.IsImageResponse(response) {
						// Image plugin - check if it has image data that needs to be stored
						if imageData, ok := plugins.GetImageData(response); ok {
							// Store the image data and replace with URL
							randomString := generateRandomString(10)
							filename := fmt.Sprintf("%s_%s_%s.png", plugin.Type(), time.Now().Format("20060102_150405"), randomString)
							imageURL, err := pp.imageStorage.StoreImage(imageData, device.ID, plugin.Type())
							if err != nil {
								logging.Error("[PLUGIN] Failed to store image data", "plugin_type", plugin.Type(), "error", err)
								response = gin.H{
									"image_url": getImageURLForDevice(device),
									"filename":  fmt.Sprintf("store_error_%s", time.Now().Format("20060102150405")),
								}
							} else {
								// Replace image_data with image_url
								newResponse := gin.H{
									"image_url": imageURL,
									"filename":  filename,
								}
								// Only include refresh_rate if plugin provided one
								if refreshRate := response["refresh_rate"]; refreshRate != nil {
									newResponse["refresh_rate"] = refreshRate
								}
								response = newResponse
								logging.Debug("[PLUGIN] Stored image data", "plugin_type", plugin.Type(), "url", imageURL)
							}
						} else {
							// Already has URL, ready to serve
							logging.Debug("[PLUGIN] Image plugin processed successfully", "plugin_type", plugin.Type())
						}
					} else {
						// Unknown plugin response type
						logging.Warn("[PLUGIN] Unknown plugin response type", "plugin_type", plugin.Type())
						response = gin.H{
							"image_url": getImageURLForDevice(device),
							"filename":  fmt.Sprintf("unknown_type_%s", time.Now().Format("20060102150405")),
						}
					}
				}
			}
		}
	}

	// Only update the playlist index if plugin processing was successful
	if pluginErr == nil {
		deviceService := database.NewDeviceService(db)
		if err := deviceService.UpdateLastPlaylistIndex(device.ID, nextIndex); err != nil {
			logging.Error("[PLAYLIST] Failed to update last playlist index", "device_mac", device.MacAddress, "error", err)
		} else {
			pp.broadcastPlaylistChange(device, nextIndex, item, activeItems)
		}
	}

	return response, &item, pluginErr
}

// renderDataPlugin renders a data plugin response to an image
func (pp *PluginProcessor) renderDataPlugin(response plugins.PluginResponse, device *database.Device, pluginType string) (gin.H, error) {
	return nil, fmt.Errorf("HTML rendering not available - data plugins are not supported without Chromium")
}

// processCurrentPlugin processes the current plugin without advancing the index
func (pp *PluginProcessor) processCurrentPlugin(device *database.Device, activeItems []database.PlaylistItem) (gin.H, error) {
	if len(activeItems) == 0 {
		return nil, fmt.Errorf("no active playlist items")
	}

	// Get the current item based on existing LastPlaylistIndex
	currentIndex := device.LastPlaylistIndex
	if currentIndex < 0 || currentIndex >= len(activeItems) {
		currentIndex = 0
	}

	item := activeItems[currentIndex]

	// Get the user plugin details
	db := database.GetDB()
	pluginService := database.NewPluginService(db)

	userPlugin, err := pluginService.GetUserPluginByID(item.UserPluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user plugin: %w", err)
	}

	// Check if plugin requires processing before looking for pre-rendered content
	var renderedContent *database.RenderedContent
	plugin, exists := plugins.Get(userPlugin.Plugin.Type)
	if exists && plugin.RequiresProcessing() {
		// First, try to get pre-rendered content
		var err error
		renderedContent, err = pp.getPreRenderedContent(userPlugin.ID, device)
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
			// Already a properly formatted URL (from image plugins)
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

		logging.Info("[PLUGIN] Using pre-rendered content (current)", "plugin_type", userPlugin.Plugin.Type)
	} else {
		// No pre-rendered content available, fall back to on-demand processing
		if exists && plugin.RequiresProcessing() {
			logging.Debug("[PLUGIN] No pre-rendered content, falling back to on-demand (current)", "plugin_type", userPlugin.Plugin.Type)
			// Schedule a render job for next time
			pp.scheduleRenderIfNeeded(userPlugin.ID)
		}
		if !exists {
			response = gin.H{
				"image_url": getImageURLForDevice(device),
				"filename":  "display.png",
			}
		} else {
			// Create plugin context
			ctx, err := plugins.NewPluginContext(device, userPlugin)
			if err != nil {
				return nil, fmt.Errorf("failed to create plugin context: %w", err)
			}

			// Process the plugin
			response, pluginErr = plugin.Process(ctx)
			if pluginErr != nil {
				response = gin.H{
					"image_url": getImageURLForDevice(device),
					"filename":  fmt.Sprintf("error_%s", time.Now().Format("20060102150405")),
				}
			} else if plugins.IsDataResponse(response) {
				response, err = pp.renderDataPlugin(response, device, plugin.Type())
				if err != nil {
					response = gin.H{
						"image_url": getImageURLForDevice(device),
						"filename":  fmt.Sprintf("render_error_%s", time.Now().Format("20060102150405")),
					}
				}
			} else if plugins.IsImageResponse(response) {
				// Image plugin - check if it has image data that needs to be stored
				if imageData, ok := plugins.GetImageData(response); ok {
					// Store the image data and replace with URL
					randomString := generateRandomString(10)
					filename := fmt.Sprintf("%s_%s_%s.png", plugin.Type(), time.Now().Format("20060102_150405"), randomString)
					imageURL, err := pp.imageStorage.StoreImage(imageData, device.ID, plugin.Type())
					if err != nil {
						logging.Error("[PLUGIN] Failed to store image data", "plugin_type", plugin.Type(), "error", err)
						response = gin.H{
							"image_url": getImageURLForDevice(device),
							"filename":  fmt.Sprintf("store_error_%s", time.Now().Format("20060102150405")),
						}
					} else {
						// Replace image_data with image_url
						newResponse := gin.H{
							"image_url": imageURL,
							"filename":  filename,
						}
						// Only include refresh_rate if plugin provided one
						if refreshRate := response["refresh_rate"]; refreshRate != nil {
							newResponse["refresh_rate"] = refreshRate
						}
						response = newResponse
						logging.Debug("[PLUGIN] Stored image data (current)", "plugin_type", plugin.Type(), "url", imageURL)
					}
				} else {
					// Already has URL, ready to serve
					logging.Debug("[PLUGIN] Image plugin processed successfully (current)", "plugin_type", plugin.Type())
				}
			}
		}
	}

	// Apply duration override (takes priority over plugin refresh_rate)
	if item.DurationOverride != nil {
		response["refresh_rate"] = fmt.Sprintf("%d", *item.DurationOverride)
	}

	return response, pluginErr
}

// broadcastPlaylistChange broadcasts playlist index changes via SSE
func (pp *PluginProcessor) broadcastPlaylistChange(device *database.Device, nextIndex int, item database.PlaylistItem, activeItems []database.PlaylistItem) {
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

	// Broadcast playlist index change to connected SSE clients
	sseService := sse.GetSSEService()
	sseService.BroadcastToDevice(device.ID, sse.Event{
		Type: "playlist_index_changed",
		Data: map[string]interface{}{
			"device_id":     device.ID.String(),
			"current_index": nextIndex,
			"current_item":  item,
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
