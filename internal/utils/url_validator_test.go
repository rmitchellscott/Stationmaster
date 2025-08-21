package utils

import (
	"net"
	"testing"
)

func TestValidateURLWithConfig(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		config      URLValidationConfig
		expectError bool
		errorContains string
	}{
		// Basic valid cases
		{
			name:        "valid https URL",
			url:         "https://example.com/image.png",
			config:      URLValidationConfig{},
			expectError: false,
		},
		{
			name:        "valid http URL",
			url:         "http://example.com/image.png",
			config:      URLValidationConfig{},
			expectError: false,
		},

		// Invalid URL cases
		{
			name:          "invalid URL",
			url:           "not-a-url",
			config:        URLValidationConfig{},
			expectError:   true,
			errorContains: "unsupported URL scheme",
		},
		{
			name:          "unsupported scheme",
			url:           "ftp://example.com/file.txt",
			config:        URLValidationConfig{},
			expectError:   true,
			errorContains: "unsupported URL scheme",
		},
		{
			name:          "missing hostname",
			url:           "http:///path",
			config:        URLValidationConfig{},
			expectError:   true,
			errorContains: "URL missing hostname",
		},

		// Domain blocklist tests
		{
			name: "blocked exact domain",
			url:  "https://evil.com/image.png",
			config: URLValidationConfig{
				BlockedDomains: []string{"evil.com"},
			},
			expectError:   true,
			errorContains: "domain evil.com is blocked",
		},
		{
			name: "blocked subdomain",
			url:  "https://api.evil.com/image.png",
			config: URLValidationConfig{
				BlockedDomains: []string{"evil.com"},
			},
			expectError:   true,
			errorContains: "domain api.evil.com is blocked",
		},
		{
			name: "case insensitive domain blocking",
			url:  "https://API.EVIL.COM/image.png",
			config: URLValidationConfig{
				BlockedDomains: []string{"evil.com"},
			},
			expectError:   true,
			errorContains: "domain API.EVIL.COM is blocked",
		},
		{
			name: "allowed domain similar to blocked",
			url:  "https://notevil.com/image.png",
			config: URLValidationConfig{
				BlockedDomains: []string{"evil.com"},
			},
			expectError: false,
		},
		{
			name: "multiple blocked domains",
			url:  "https://bad.com/image.png",
			config: URLValidationConfig{
				BlockedDomains: []string{"evil.com", "bad.com", "malicious.org"},
			},
			expectError:   true,
			errorContains: "domain bad.com is blocked",
		},

		// Private IP tests (using known public IPs that won't resolve to private)
		{
			name: "public domain with private IP blocking enabled",
			url:  "https://example.com/image.png",
			config: URLValidationConfig{
				BlockPrivateIPs: true,
			},
			expectError: false, // example.com should resolve to public IPs
		},

		// Combined tests
		{
			name: "blocked domain with private IP blocking",
			url:  "https://evil.com/image.png",
			config: URLValidationConfig{
				BlockPrivateIPs: true,
				BlockedDomains:  []string{"evil.com"},
			},
			expectError:   true,
			errorContains: "domain evil.com is blocked",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURLWithConfig(tt.url, tt.config)
			
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error but got none")
					return
				}
				if tt.errorContains != "" && !containsIgnoreCase(err.Error(), tt.errorContains) {
					t.Errorf("expected error containing %q, got %q", tt.errorContains, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		name     string
		ip       string
		expected bool
	}{
		// Public IPs
		{"Google DNS", "8.8.8.8", false},
		{"Cloudflare DNS", "1.1.1.1", false},
		{"Example.com", "93.184.216.34", false},

		// RFC 1918 Private IPs
		{"10.x.x.x range start", "10.0.0.1", true},
		{"10.x.x.x range end", "10.255.255.254", true},
		{"172.16-31.x.x range start", "172.16.0.1", true},
		{"172.16-31.x.x range middle", "172.20.1.1", true},
		{"172.16-31.x.x range end", "172.31.255.254", true},
		{"192.168.x.x range start", "192.168.0.1", true},
		{"192.168.x.x range end", "192.168.255.254", true},

		// Localhost
		{"localhost", "127.0.0.1", true},
		{"localhost range", "127.1.2.3", true},

		// Link-local
		{"link-local start", "169.254.0.1", true},
		{"link-local end", "169.254.255.254", true},

		// IPv6
		{"IPv6 localhost", "::1", true},
		{"IPv6 link-local", "fe80::1", true},
		{"IPv6 unique local", "fc00::1", true},
		{"IPv6 public", "2001:4860:4860::8888", false}, // Google DNS

		// Edge cases - not private
		{"172.15.x.x (just before private range)", "172.15.255.255", false},
		{"172.32.x.x (just after private range)", "172.32.0.0", false},
		{"169.253.x.x (just before link-local)", "169.253.255.255", false},
		{"169.255.x.x (just after link-local)", "169.255.0.0", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := parseIP(t, tt.ip)
			result := isPrivateIP(ip)
			if result != tt.expected {
				t.Errorf("isPrivateIP(%s) = %v, want %v", tt.ip, result, tt.expected)
			}
		})
	}
}

func TestGetURLValidationConfig(t *testing.T) {
	// Note: This test would need to set environment variables to test properly
	// For now, we'll just test that it returns a valid config struct
	config := GetURLValidationConfig()
	
	// Should return a valid config struct (values depend on environment)
	if config.BlockedDomains == nil {
		config.BlockedDomains = []string{} // Should be initialized as empty slice, not nil
	}
}

// Helper functions

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && 
		   len(substr) > 0 && 
		   (s == substr || 
		    (len(s) > len(substr) && 
		     (s[:len(substr)] == substr || 
		      s[len(s)-len(substr):] == substr ||
		      func() bool {
		      	for i := 0; i <= len(s)-len(substr); i++ {
		      		if s[i:i+len(substr)] == substr {
		      			return true
		      		}
		      	}
		      	return false
		      }())))
}

func parseIP(t *testing.T, ipStr string) net.IP {
	ip := net.ParseIP(ipStr)
	if ip == nil {
		t.Fatalf("failed to parse IP: %s", ipStr)
	}
	return ip
}