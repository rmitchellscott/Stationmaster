package locales

import (
	"embed"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed *.yml
var localeFiles embed.FS

// LocaleManager manages TRMNL locale files and provides translation lookups
type LocaleManager struct {
	translations map[string]map[string]interface{}
	mutex        sync.RWMutex
}

// NewLocaleManager creates a new locale manager and loads all embedded locale files
func NewLocaleManager() (*LocaleManager, error) {
	lm := &LocaleManager{
		translations: make(map[string]map[string]interface{}),
	}
	
	if err := lm.loadEmbeddedLocales(); err != nil {
		return nil, fmt.Errorf("failed to load embedded locales: %w", err)
	}
	
	return lm, nil
}

// loadEmbeddedLocales loads all embedded YAML locale files
func (lm *LocaleManager) loadEmbeddedLocales() error {
	entries, err := localeFiles.ReadDir(".")
	if err != nil {
		return fmt.Errorf("failed to read embedded locale directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yml") {
			continue
		}

		// Extract locale code from filename (e.g., "en.yml" -> "en")
		locale := strings.TrimSuffix(entry.Name(), filepath.Ext(entry.Name()))
		
		// Read the YAML file
		data, err := localeFiles.ReadFile(entry.Name())
		if err != nil {
			return fmt.Errorf("failed to read locale file %s: %w", entry.Name(), err)
		}

		// Parse YAML content
		var translations map[string]interface{}
		if err := yaml.Unmarshal(data, &translations); err != nil {
			return fmt.Errorf("failed to parse YAML for locale %s: %w", locale, err)
		}

		lm.mutex.Lock()
		lm.translations[locale] = translations
		lm.mutex.Unlock()
	}

	return nil
}

// GetTranslation retrieves a translation for a given locale and key
func (lm *LocaleManager) GetTranslation(locale, key string) (string, bool) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	// Normalize locale (handle cases like "en-US" -> "en")
	normalizedLocale := normalizeLocale(locale)
	
	// Try exact match first
	if translations, exists := lm.translations[locale]; exists {
		if value, found := getNestedValue(translations, key); found {
			return value, true
		}
	}
	
	// Try normalized locale
	if locale != normalizedLocale {
		if translations, exists := lm.translations[normalizedLocale]; exists {
			if value, found := getNestedValue(translations, key); found {
				return value, true
			}
		}
	}
	
	// Fallback to English
	if locale != "en" && normalizedLocale != "en" {
		if translations, exists := lm.translations["en"]; exists {
			if value, found := getNestedValue(translations, key); found {
				return value, true
			}
		}
	}
	
	return "", false
}

// GetLocaleJSON returns the complete translation data for a locale as JSON
func (lm *LocaleManager) GetLocaleJSON(locale string) ([]byte, error) {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	normalizedLocale := normalizeLocale(locale)
	
	// Try exact match first
	if translations, exists := lm.translations[locale]; exists {
		return json.Marshal(translations)
	}
	
	// Try normalized locale
	if locale != normalizedLocale {
		if translations, exists := lm.translations[normalizedLocale]; exists {
			return json.Marshal(translations)
		}
	}
	
	// Fallback to English
	if locale != "en" && normalizedLocale != "en" {
		if translations, exists := lm.translations["en"]; exists {
			return json.Marshal(translations)
		}
	}
	
	return nil, fmt.Errorf("locale %s not found", locale)
}

// GetAvailableLocales returns a list of all available locale codes
func (lm *LocaleManager) GetAvailableLocales() []string {
	lm.mutex.RLock()
	defer lm.mutex.RUnlock()

	locales := make([]string, 0, len(lm.translations))
	for locale := range lm.translations {
		locales = append(locales, locale)
	}
	
	return locales
}

// normalizeLocale converts locale codes like "en-US" to "en"
func normalizeLocale(locale string) string {
	if idx := strings.Index(locale, "-"); idx != -1 {
		return locale[:idx]
	}
	return locale
}

// getNestedValue retrieves a value from nested map using dot notation
func getNestedValue(data map[string]interface{}, key string) (string, bool) {
	parts := strings.Split(key, ".")
	current := data
	
	for i, part := range parts {
		if i == len(parts)-1 {
			// Last part - return the string value
			if value, exists := current[part]; exists {
				if strValue, ok := value.(string); ok {
					return strValue, true
				}
			}
			return "", false
		} else {
			// Intermediate part - continue navigating
			if next, exists := current[part]; exists {
				if nextMap, ok := next.(map[string]interface{}); ok {
					current = nextMap
				} else {
					return "", false
				}
			} else {
				return "", false
			}
		}
	}
	
	return "", false
}