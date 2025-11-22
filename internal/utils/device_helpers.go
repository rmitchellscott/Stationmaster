package utils

import (
	"math"
	"time"
)

// CalculateBatteryPercentage converts battery voltage to percentage
// using piecewise linear interpolation based on official API data
func CalculateBatteryPercentage(voltage float64) int {
	// Clamp voltage to valid range
	if voltage <= 3.1 {
		return 1
	}
	if voltage >= 4.06 {
		return 100
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
			return int(math.Round(p1 + ratio*(p2-p1)))
		}
	}

	return 1 // Fallback
}

// SignalQuality represents WiFi signal quality information
type SignalQuality struct {
	Quality  string
	Strength int
	Color    string
}

// GetSignalQuality converts RSSI to signal quality information
func GetSignalQuality(rssi int) SignalQuality {
	if rssi > -50 {
		return SignalQuality{Quality: "Excellent", Strength: 5, Color: ""}
	}
	if rssi > -60 {
		return SignalQuality{Quality: "Good", Strength: 4, Color: ""}
	}
	if rssi > -70 {
		return SignalQuality{Quality: "Fair", Strength: 3, Color: ""}
	}
	if rssi > -80 {
		return SignalQuality{Quality: "Poor", Strength: 2, Color: "text-destructive"}
	}
	return SignalQuality{Quality: "Very Poor", Strength: 1, Color: "text-destructive"}
}

// CalculateWiFiPercentage converts RSSI to WiFi strength percentage
func CalculateWiFiPercentage(rssi int) int {
	quality := GetSignalQuality(rssi)
	return quality.Strength * 20 // Convert 1-5 strength to 20-100 percentage
}

// GetTimezoneFriendlyName returns a timezone abbreviation using Go's built-in formatting
func GetTimezoneFriendlyName(timezoneIANA string) string {
	location, err := time.LoadLocation(timezoneIANA)
	if err != nil {
		// Fallback to the IANA name if we can't load the location
		return timezoneIANA
	}
	
	// Get the current time in that timezone to determine the abbreviation
	now := time.Now().UTC().In(location)
	zoneName, _ := now.Zone()
	
	// Return just the timezone abbreviation (e.g., "MDT", "EST", "PST")
	return zoneName
}