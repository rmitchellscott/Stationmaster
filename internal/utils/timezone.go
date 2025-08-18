package utils

import (
	"fmt"
	"time"
)

// ValidateTimezone checks if a timezone string is valid according to IANA database
func ValidateTimezone(timezone string) error {
	if timezone == "" {
		return fmt.Errorf("timezone cannot be empty")
	}

	_, err := time.LoadLocation(timezone)
	if err != nil {
		return fmt.Errorf("invalid timezone: %s", timezone)
	}

	return nil
}

// GetValidTimezones returns a list of common/supported timezones
func GetValidTimezones() []string {
	return []string{
		"UTC",
		"America/New_York",
		"America/Chicago",
		"America/Denver",
		"America/Los_Angeles",
		"America/Anchorage",
		"Pacific/Honolulu",
		"Europe/London",
		"Europe/Paris",
		"Europe/Berlin",
		"Europe/Rome",
		"Europe/Madrid",
		"Europe/Amsterdam",
		"Europe/Stockholm",
		"Europe/Helsinki",
		"Europe/Warsaw",
		"Europe/Prague",
		"Europe/Vienna",
		"Europe/Zurich",
		"Europe/Brussels",
		"Europe/Dublin",
		"Europe/Lisbon",
		"Europe/Athens",
		"Europe/Istanbul",
		"Europe/Moscow",
		"Asia/Tokyo",
		"Asia/Shanghai",
		"Asia/Hong_Kong",
		"Asia/Singapore",
		"Asia/Seoul",
		"Asia/Bangkok",
		"Asia/Jakarta",
		"Asia/Manila",
		"Asia/Kuala_Lumpur",
		"Asia/Taipei",
		"Asia/Kolkata",
		"Asia/Dubai",
		"Asia/Tehran",
		"Asia/Jerusalem",
		"Asia/Riyadh",
		"Australia/Sydney",
		"Australia/Melbourne",
		"Australia/Brisbane",
		"Australia/Perth",
		"Australia/Adelaide",
		"Pacific/Auckland",
		"Africa/Cairo",
		"Africa/Johannesburg",
		"Africa/Lagos",
		"Africa/Nairobi",
		"America/Toronto",
		"America/Vancouver",
		"America/Montreal",
		"America/Sao_Paulo",
		"America/Argentina/Buenos_Aires",
		"America/Mexico_City",
		"America/Bogota",
		"America/Lima",
		"America/Santiago",
		"America/Caracas",
	}
}

// ConvertTimeToTimezone converts a time to a specific timezone
func ConvertTimeToTimezone(t time.Time, timezone string) (time.Time, error) {
	if timezone == "" {
		timezone = "UTC"
	}

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid timezone: %s", timezone)
	}

	return t.In(loc), nil
}

// IsValidTimezone checks if a timezone string is valid
func IsValidTimezone(timezone string) bool {
	return ValidateTimezone(timezone) == nil
}

// NormalizeTimezone returns a normalized timezone string or default
func NormalizeTimezone(timezone string) string {
	if timezone == "" {
		return "UTC"
	}

	if IsValidTimezone(timezone) {
		return timezone
	}

	return "UTC"
}
