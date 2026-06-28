package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestIsCurl(t *testing.T) {
	tests := []struct {
		userAgent string
		accept    string
		expected  bool
	}{
		{"curl/7.68.0", "*/*", true},
		{"Wget/1.20.3 (linux-gnu)", "*/*", true},
		{"Mozilla/5.0", "text/html", false},
		{"Mozilla/5.0", "text/plain", true},
		{"curl/7.68.0", "text/html", false},
	}

	for _, tt := range tests {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Header.Set("User-Agent", tt.userAgent)
		req.Header.Set("Accept", tt.accept)

		if got := isCurl(req); got != tt.expected {
			t.Errorf("isCurl() = %v for UA=%q Accept=%q; want %v", got, tt.userAgent, tt.accept, tt.expected)
		}
	}
}
