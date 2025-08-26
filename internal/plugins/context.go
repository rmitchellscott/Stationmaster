package plugins

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/rmitchellscott/stationmaster/internal/database"
)

// NewPluginContext creates a new plugin context with parsed settings
func NewPluginContext(device *database.Device, pluginInstance *database.PluginInstance, user *database.User) (PluginContext, error) {
	settings := make(map[string]interface{})
	
	if len(pluginInstance.Settings) > 0 {
		if err := json.Unmarshal(pluginInstance.Settings, &settings); err != nil {
			return PluginContext{}, fmt.Errorf("failed to parse plugin settings: %w", err)
		}
	}
	
	return PluginContext{
		Device:         device,
		PluginInstance: pluginInstance,
		User:           user,
		Settings:       settings,
	}, nil
}

// GetStringSetting returns a string setting value with fallback
func (ctx PluginContext) GetStringSetting(key string, fallback string) string {
	if val, ok := ctx.Settings[key].(string); ok {
		return val
	}
	return fallback
}

// GetIntSetting returns an integer setting value with fallback
func (ctx PluginContext) GetIntSetting(key string, fallback int) int {
	if val, ok := ctx.Settings[key].(float64); ok {
		return int(val)
	}
	if val, ok := ctx.Settings[key].(int); ok {
		return val
	}
	return fallback
}

// GetBoolSetting returns a boolean setting value with fallback
func (ctx PluginContext) GetBoolSetting(key string, fallback bool) bool {
	if val, ok := ctx.Settings[key].(bool); ok {
		return val
	}
	return fallback
}

// GetFloatSetting returns a float setting value with fallback
func (ctx PluginContext) GetFloatSetting(key string, fallback float64) float64 {
	if val, ok := ctx.Settings[key].(float64); ok {
		return val
	}
	return fallback
}

// HasSetting checks if a setting exists
func (ctx PluginContext) HasSetting(key string) bool {
	_, exists := ctx.Settings[key]
	return exists
}

// BatteryVoltageToPercentage converts battery voltage to percentage
// using piecewise linear interpolation based on official TRMNL API data
func BatteryVoltageToPercentage(voltage float64) float64 {
	if voltage <= 0 {
		return 0 // Invalid voltage reading
	}

	// Clamp voltage to valid range
	if voltage <= 3.1 {
		return 1.0
	}
	if voltage >= 4.06 {
		return 100.0
	}

	// Piecewise linear interpolation based on official API data
	points := [][2]float64{
		{3.1, 1},
		{3.65, 54},
		{3.70, 58},
		{3.75, 62},
		{3.80, 66},
		{3.85, 70},
		{3.88, 73},
		{3.90, 75},
		{3.92, 76},
		{3.98, 81},
		{4.00, 90},
		{4.02, 95},
		{4.05, 95},
		{4.06, 100},
	}

	// Find the two points to interpolate between
	for i := 0; i < len(points)-1; i++ {
		v1, p1 := points[i][0], points[i][1]
		v2, p2 := points[i+1][0], points[i+1][1]

		if voltage >= v1 && voltage <= v2 {
			// Linear interpolation between the two points
			ratio := (voltage - v1) / (v2 - v1)
			result := p1 + ratio*(p2-p1)
			// Round to 2 decimal places to match TRMNL API format (e.g., 75.83)
			return math.Round(result*100) / 100
		}
	}

	return 1.0 // Fallback
}

// RSSIToWifiStrengthPercentage converts RSSI (dBm) to wifi strength percentage
// RSSI values typically range from -100 dBm (very poor) to -30 dBm (excellent)
func RSSIToWifiStrengthPercentage(rssi int) int {
	if rssi == 0 {
		return 0 // No signal or invalid reading
	}

	const (
		minRSSI = -100 // 0% signal strength
		maxRSSI = -30  // 100% signal strength
	)

	// Clamp RSSI to expected range
	if rssi < minRSSI {
		return 0
	}
	if rssi > maxRSSI {
		return 100
	}

	// Linear conversion from RSSI range to percentage
	percentage := ((rssi - minRSSI) * 100) / (maxRSSI - minRSSI)
	
	// Ensure result is within 0-100 range
	if percentage < 0 {
		return 0
	}
	if percentage > 100 {
		return 100
	}
	
	return percentage
}

// GetDeviceBatteryPercentage returns the device battery percentage
func (ctx PluginContext) GetDeviceBatteryPercentage() float64 {
	if ctx.Device == nil {
		return 0
	}
	return BatteryVoltageToPercentage(ctx.Device.BatteryVoltage)
}

// GetDeviceWifiStrengthPercentage returns the device wifi strength percentage  
func (ctx PluginContext) GetDeviceWifiStrengthPercentage() int {
	if ctx.Device == nil {
		return 0
	}
	return RSSIToWifiStrengthPercentage(ctx.Device.RSSI)
}