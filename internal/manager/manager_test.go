package manager

import (
	"testing"
	"time"

	"github.com/sol-strategies/solana-validator-version-sync/internal/config"
)

func TestCalculateNextBoundary(t *testing.T) {
	// Create a minimal manager for testing
	cfg := &config.Config{}
	m := &Manager{cfg: cfg}

	tests := []struct {
		name             string
		now              time.Time
		intervalDuration time.Duration
		expectedBoundary time.Time
		description      string
	}{
		{
			name:             "10 seconds - aligns to :00, :10, :20, etc",
			now:              time.Date(2024, 1, 15, 9, 53, 37, 0, time.UTC),
			intervalDuration: 10 * time.Second,
			expectedBoundary: time.Date(2024, 1, 15, 9, 53, 40, 0, time.UTC),
			description:      "9:53:37 should align to 9:53:40",
		},
		{
			name:             "10 seconds - at boundary",
			now:              time.Date(2024, 1, 15, 9, 53, 40, 0, time.UTC),
			intervalDuration: 10 * time.Second,
			expectedBoundary: time.Date(2024, 1, 15, 9, 53, 50, 0, time.UTC),
			description:      "9:53:40 should align to 9:53:50",
		},
		{
			name:             "10 minutes - aligns to :00, :10, :20, etc",
			now:              time.Date(2024, 1, 15, 9, 53, 0, 0, time.UTC),
			intervalDuration: 10 * time.Minute,
			expectedBoundary: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			description:      "9:53 should align to 10:00",
		},
		{
			name:             "10 minutes - at boundary",
			now:              time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			intervalDuration: 10 * time.Minute,
			expectedBoundary: time.Date(2024, 1, 15, 10, 10, 0, 0, time.UTC),
			description:      "10:00 should align to 10:10",
		},
		{
			name:             "1 hour - aligns to :00 of each hour",
			now:              time.Date(2024, 1, 15, 9, 53, 0, 0, time.UTC),
			intervalDuration: 1 * time.Hour,
			expectedBoundary: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			description:      "9:53 should align to 10:00",
		},
		{
			name:             "1 hour - at boundary",
			now:              time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			intervalDuration: 1 * time.Hour,
			expectedBoundary: time.Date(2024, 1, 15, 11, 0, 0, 0, time.UTC),
			description:      "10:00 should align to 11:00",
		},
		{
			name:             "1 day - aligns to midnight",
			now:              time.Date(2024, 1, 15, 9, 53, 0, 0, time.UTC),
			intervalDuration: 24 * time.Hour,
			expectedBoundary: time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			description:      "Jan 15 9:53 should align to Jan 16 00:00",
		},
		{
			name:             "1 day - at boundary",
			now:              time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC),
			intervalDuration: 24 * time.Hour,
			expectedBoundary: time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC),
			description:      "Jan 15 00:00 should align to Jan 16 00:00",
		},
		{
			name:             "30 seconds",
			now:              time.Date(2024, 1, 15, 9, 53, 45, 0, time.UTC),
			intervalDuration: 30 * time.Second,
			expectedBoundary: time.Date(2024, 1, 15, 9, 54, 0, 0, time.UTC),
			description:      "9:53:45 should align to 9:54:00",
		},
		{
			name:             "5 minutes",
			now:              time.Date(2024, 1, 15, 9, 57, 0, 0, time.UTC),
			intervalDuration: 5 * time.Minute,
			expectedBoundary: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
			description:      "9:57 should align to 10:00",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := m.calculateNextBoundary(tt.now, tt.intervalDuration)
			if !result.Equal(tt.expectedBoundary) {
				t.Errorf("%s: got %v, want %v", tt.description, result, tt.expectedBoundary)
			}
		})
	}
}

