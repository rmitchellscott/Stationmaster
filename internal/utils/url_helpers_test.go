package utils

import (
	"crypto/tls"
	"net/http"
	"testing"
)

func TestURLFromRequest(t *testing.T) {
	tests := []struct {
		name               string
		tls                *tls.ConnectionState
		xForwardedProto    string
		xForwardedHost     string
		host               string
		expectedScheme     string
		expectedHost       string
	}{
		{
			name:           "HTTP request without proxy",
			tls:            nil,
			xForwardedProto: "",
			xForwardedHost:  "",
			host:           "example.com",
			expectedScheme: "http",
			expectedHost:   "example.com",
		},
		{
			name:           "HTTPS request with direct TLS",
			tls:            &tls.ConnectionState{},
			xForwardedProto: "",
			xForwardedHost:  "",
			host:           "example.com",
			expectedScheme: "https",
			expectedHost:   "example.com",
		},
		{
			name:           "HTTP request with X-Forwarded-Proto https",
			tls:            nil,
			xForwardedProto: "https",
			xForwardedHost:  "",
			host:           "example.com",
			expectedScheme: "https",
			expectedHost:   "example.com",
		},
		{
			name:           "HTTP request with X-Forwarded-Host",
			tls:            nil,
			xForwardedProto: "",
			xForwardedHost:  "public.example.com",
			host:           "localhost:8080",
			expectedScheme: "http",
			expectedHost:   "public.example.com",
		},
		{
			name:           "HTTPS request with X-Forwarded-Host and X-Forwarded-Proto",
			tls:            nil,
			xForwardedProto: "https",
			xForwardedHost:  "api.example.com",
			host:           "internal-server:3000",
			expectedScheme: "https",
			expectedHost:   "api.example.com",
		},
		{
			name:           "X-Forwarded-Host with port",
			tls:            nil,
			xForwardedProto: "https",
			xForwardedHost:  "staging.example.com:8443",
			host:           "localhost:8080",
			expectedScheme: "https",
			expectedHost:   "staging.example.com:8443",
		},
		{
			name:           "HTTPS request with X-Forwarded-Proto http (TLS takes precedence)",
			tls:            &tls.ConnectionState{},
			xForwardedProto: "http",
			xForwardedHost:  "",
			host:           "example.com",
			expectedScheme: "https",
			expectedHost:   "example.com",
		},
		{
			name:           "Direct TLS with forwarded host (common CDN scenario)",
			tls:            &tls.ConnectionState{},
			xForwardedProto: "",
			xForwardedHost:  "mysite.com",
			host:           "server.internal.com",
			expectedScheme: "https",
			expectedHost:   "mysite.com",
		},
		{
			name:           "Empty X-Forwarded-Host should use original host",
			tls:            nil,
			xForwardedProto: "https",
			xForwardedHost:  "",
			host:           "example.com:8080",
			expectedScheme: "https",
			expectedHost:   "example.com:8080",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &http.Request{
				Host: tt.host,
				TLS:  tt.tls,
				Header: make(http.Header),
			}
			
			if tt.xForwardedProto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.xForwardedProto)
			}
			
			if tt.xForwardedHost != "" {
				req.Header.Set("X-Forwarded-Host", tt.xForwardedHost)
			}

			url := URLFromRequest(req)
			
			if url.Scheme != tt.expectedScheme {
				t.Errorf("URLFromRequest() scheme = %v, expected %v", url.Scheme, tt.expectedScheme)
			}
			
			if url.Host != tt.expectedHost {
				t.Errorf("URLFromRequest() host = %v, expected %v", url.Host, tt.expectedHost)
			}
		})
	}
}

func TestBaseURLFromRequest(t *testing.T) {
	req := &http.Request{
		Host:   "internal-server:8080",
		TLS:    nil,
		Header: make(http.Header),
	}
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "api.example.com")

	baseURL := BaseURLFromRequest(req)
	expected := "https://api.example.com"
	
	if baseURL != expected {
		t.Errorf("BaseURLFromRequest() = %v, expected %v", baseURL, expected)
	}
}