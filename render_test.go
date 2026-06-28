package main

import (
	"testing"
	"time"
)

func TestGetHourColor(t *testing.T) {
	tests := []struct {
		hour     int
		expected string
	}{
		{10, "\x1b[33m"}, // 10 AM, Work hours
		{18, "\x1b[36m"}, // 6 PM, Evening
		{7, "\x1b[36m"},  // 7 AM, Morning
		{2, "\x1b[34m"},  // 2 AM, Night
	}

	for _, tt := range tests {
		dt := time.Date(2023, 1, 1, tt.hour, 0, 0, 0, time.UTC)
		if got := getHourColor(dt); got != tt.expected {
			t.Errorf("getHourColor for hour %d = %q; want %q", tt.hour, got, tt.expected)
		}
	}
}
