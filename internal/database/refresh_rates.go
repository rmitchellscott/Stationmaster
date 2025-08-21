package database

// Predefined refresh rate constants (in seconds)
const (
	RefreshRate15Min  = 900   // 15 minutes
	RefreshRate30Min  = 1800  // 30 minutes
	RefreshRateHourly = 3600  // 1 hour
	RefreshRate2Hours = 7200  // 2 hours
	RefreshRate4Hours = 14400 // 4 hours
	RefreshRate4xDay  = 21600 // 6 hours (4 times daily)
	RefreshRate3xDay  = 28800 // 8 hours (3 times daily)
	RefreshRate2xDay  = 43200 // 12 hours (twice daily)
	RefreshRateDaily  = 86400 // 24 hours (daily)
)

// RefreshRateOption represents a user-friendly refresh rate option
type RefreshRateOption struct {
	Label   string `json:"label"`
	Value   int    `json:"value"`
	Default bool   `json:"default,omitempty"`
}

// GetRefreshRateOptions returns all available refresh rate options
func GetRefreshRateOptions() []RefreshRateOption {
	return []RefreshRateOption{
		{Label: "15 minutes", Value: RefreshRate15Min},
		{Label: "30 minutes", Value: RefreshRate30Min},
		{Label: "Hourly", Value: RefreshRateHourly},
		{Label: "Every 2 hours", Value: RefreshRate2Hours},
		{Label: "Every 4 hours", Value: RefreshRate4Hours},
		{Label: "4 times daily", Value: RefreshRate4xDay},
		{Label: "3 times daily", Value: RefreshRate3xDay},
		{Label: "Twice daily", Value: RefreshRate2xDay},
		{Label: "Daily", Value: RefreshRateDaily, Default: true},
	}
}

// IsValidRefreshRate checks if a refresh rate value is valid
func IsValidRefreshRate(rate int) bool {
	validRates := []int{
		RefreshRate15Min,
		RefreshRate30Min,
		RefreshRateHourly,
		RefreshRate2Hours,
		RefreshRate4Hours,
		RefreshRate4xDay,
		RefreshRate3xDay,
		RefreshRate2xDay,
		RefreshRateDaily,
	}
	
	for _, validRate := range validRates {
		if rate == validRate {
			return true
		}
	}
	return false
}

// GetDefaultRefreshRate returns the default refresh rate
func GetDefaultRefreshRate() int {
	return RefreshRateDaily
}