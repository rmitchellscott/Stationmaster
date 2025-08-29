package utils

import (
	"net/http"
	"net/url"
)

// URLFromRequest creates a URL with the correct scheme and host based on the request.
// It detects HTTPS from either direct TLS connection or X-Forwarded-Proto header,
// and uses X-Forwarded-Host for the hostname when available
// (commonly used by reverse proxies like nginx, Cloudflare, load balancers).
func URLFromRequest(r *http.Request) *url.URL {
	u := &url.URL{
		Scheme: "http",
		Host:   r.Host,
	}
	if v := r.Header.Get("X-Forwarded-Host"); v != "" {
		u.Host = v
	}
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		u.Scheme = "https"
	}
	return u
}

// BaseURLFromRequest returns the base URL as a string from the request.
// This is a convenience function that calls URLFromRequest and returns the string representation.
func BaseURLFromRequest(r *http.Request) string {
	return URLFromRequest(r).String()
}