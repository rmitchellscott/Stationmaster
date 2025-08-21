package utils

import (
	"fmt"
	"net"
	"net/url"
	"strings"

	"github.com/rmitchellscott/stationmaster/internal/config"
)

var (
	privateIPRanges = []*net.IPNet{
		// RFC 1918 - Private IPv4 address ranges
		mustParseCIDR("10.0.0.0/8"),     // 10.0.0.0 - 10.255.255.255
		mustParseCIDR("172.16.0.0/12"),  // 172.16.0.0 - 172.31.255.255
		mustParseCIDR("192.168.0.0/16"), // 192.168.0.0 - 192.168.255.255
		// RFC 3927 - Link-local addresses
		mustParseCIDR("169.254.0.0/16"), // 169.254.0.0 - 169.254.255.255
		// Localhost
		mustParseCIDR("127.0.0.0/8"), // 127.0.0.0 - 127.255.255.255
		// IPv6 localhost
		mustParseCIDR("::1/128"),
		// IPv6 link-local
		mustParseCIDR("fe80::/10"),
		// IPv6 unique local addresses
		mustParseCIDR("fc00::/7"),
	}
)

func mustParseCIDR(cidr string) *net.IPNet {
	_, ipNet, err := net.ParseCIDR(cidr)
	if err != nil {
		panic(fmt.Sprintf("failed to parse CIDR %s: %v", cidr, err))
	}
	return ipNet
}

// URLValidationConfig holds configuration for URL validation
type URLValidationConfig struct {
	BlockPrivateIPs bool
	BlockedDomains  []string
}

// GetURLValidationConfig returns the current URL validation configuration from environment variables
func GetURLValidationConfig() URLValidationConfig {
	blockedDomainsStr := config.Get("BLOCKED_DOMAINS", "")
	var blockedDomains []string
	if blockedDomainsStr != "" {
		for _, domain := range strings.Split(blockedDomainsStr, ",") {
			domain = strings.TrimSpace(domain)
			if domain != "" {
				blockedDomains = append(blockedDomains, strings.ToLower(domain))
			}
		}
	}

	return URLValidationConfig{
		BlockPrivateIPs: config.GetBool("BLOCK_PRIVATE_IPS", false),
		BlockedDomains:  blockedDomains,
	}
}

// ValidateURL validates a URL according to the configured security policies
func ValidateURL(urlStr string) error {
	return ValidateURLWithConfig(urlStr, GetURLValidationConfig())
}

// ValidateURLWithConfig validates a URL with the provided configuration
func ValidateURLWithConfig(urlStr string, cfg URLValidationConfig) error {
	// Parse the URL
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Only validate HTTP/HTTPS URLs
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("unsupported URL scheme: %s (only http and https are allowed)", parsedURL.Scheme)
	}

	// Extract hostname
	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("URL missing hostname")
	}

	// Check domain blocklist
	if cfg.BlockedDomains != nil && len(cfg.BlockedDomains) > 0 {
		hostnameLower := strings.ToLower(hostname)
		for _, blockedDomain := range cfg.BlockedDomains {
			if hostnameLower == blockedDomain || strings.HasSuffix(hostnameLower, "."+blockedDomain) {
				return fmt.Errorf("domain %s is blocked", hostname)
			}
		}
	}

	// Check private IP ranges if enabled
	if cfg.BlockPrivateIPs {
		// Try to resolve hostname to IP
		ips, err := net.LookupIP(hostname)
		if err != nil {
			// If we can't resolve, let it proceed - the actual request will fail naturally
			return nil
		}

		// Check all resolved IPs
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return fmt.Errorf("private IP address %s is blocked for hostname %s", ip.String(), hostname)
			}
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is in a private range
func isPrivateIP(ip net.IP) bool {
	for _, privateRange := range privateIPRanges {
		if privateRange.Contains(ip) {
			return true
		}
	}
	return false
}