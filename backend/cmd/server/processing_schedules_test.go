package main

import (
	"testing"
	"time"
)

func TestNextProcessingScheduleTime(t *testing.T) {
	now := time.Date(2026, 10, 5, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name        string
		frequency   string
		scheduledAt time.Time
		want        time.Time
	}{
		{
			name:        "hourly schedule rolls past missed executions",
			frequency:   "hourly",
			scheduledAt: now.Add(-3 * time.Hour),
			want:        now.Add(time.Hour),
		},
		{
			name:        "monthly schedule advances by calendar month",
			frequency:   "monthly",
			scheduledAt: now.AddDate(0, -1, 0),
			want:        now.AddDate(0, 1, 0),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := nextProcessingScheduleTime(tt.frequency, tt.scheduledAt, now)
			if got == nil || !got.Equal(tt.want) {
				t.Fatalf("nextProcessingScheduleTime() = %v, want %v", got, tt.want)
			}
		})
	}
	if got := nextProcessingScheduleTime("on_demand", now, now); got != nil {
		t.Fatalf("on_demand schedule should not recur, got %v", got)
	}
}
