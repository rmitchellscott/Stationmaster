package database

// Predefined refresh rate constants (in seconds)
const (
	// Frequent refresh rates (requires enable_frequent_refreshes setting)
	RefreshRate1Min   = 60    // 1 minute
	RefreshRate5Min   = 300   // 5 minutes
	RefreshRate10Min  = 600   // 10 minutes
	
	// Standard refresh rates
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

// GetRefreshRateOptionsWithFrequent returns all available refresh rate options including frequent rates if enabled
func GetRefreshRateOptionsWithFrequent(includeFrequent bool) []RefreshRateOption {
	options := []RefreshRateOption{}
	
	// Add frequent refresh options if enabled
	if includeFrequent {
		options = append(options, []RefreshRateOption{
			{Label: "1 minute", Value: RefreshRate1Min},
			{Label: "5 minutes", Value: RefreshRate5Min},
			{Label: "10 minutes", Value: RefreshRate10Min},
		}...)
	}
	
	// Add standard refresh options
	options = append(options, []RefreshRateOption{
		{Label: "15 minutes", Value: RefreshRate15Min},
		{Label: "30 minutes", Value: RefreshRate30Min},
		{Label: "Hourly", Value: RefreshRateHourly},
		{Label: "Every 2 hours", Value: RefreshRate2Hours},
		{Label: "Every 4 hours", Value: RefreshRate4Hours},
		{Label: "4 times daily", Value: RefreshRate4xDay},
		{Label: "3 times daily", Value: RefreshRate3xDay},
		{Label: "Twice daily", Value: RefreshRate2xDay},
		{Label: "Daily", Value: RefreshRateDaily, Default: true},
	}...)
	
	return options
}

// IsValidRefreshRate checks if a refresh rate value is valid
func IsValidRefreshRate(rate int) bool {
	return IsValidRefreshRateWithFrequent(rate, true)
}

// IsValidRefreshRateWithFrequent checks if a refresh rate value is valid, optionally including frequent rates
func IsValidRefreshRateWithFrequent(rate int, includeFrequent bool) bool {
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
	
	// Add frequent rates if enabled
	if includeFrequent {
		validRates = append([]int{
			RefreshRate1Min,
			RefreshRate5Min,
			RefreshRate10Min,
		}, validRates...)
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