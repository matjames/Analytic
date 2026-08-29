package api

import (
	"testing"
	"time"
)

func TestValidateScheduledForNormalizesTimezoneToUTC(t *testing.T) {
	now := time.Date(2026, time.August, 27, 10, 0, 0, 0, time.UTC)
	got, err := validateScheduledFor("2026-08-27T15:30:00+03:00", now)
	if err != nil {
		t.Fatalf("expected valid schedule: %v", err)
	}
	want := time.Date(2026, time.August, 27, 12, 30, 0, 0, time.UTC)
	if !got.Equal(want) || got.Location() != time.UTC {
		t.Fatalf("expected %s UTC, got %s (%s)", want, got, got.Location())
	}
}

func TestValidateScheduledForRejectsPastAndExcessiveTimes(t *testing.T) {
	now := time.Date(2026, time.August, 27, 10, 0, 0, 0, time.UTC)
	for _, value := range []string{"2026-08-27T09:59:00Z", "2028-08-27T10:00:00Z", "not-a-time"} {
		if _, err := validateScheduledFor(value, now); err == nil {
			t.Fatalf("expected %q to be rejected", value)
		}
	}
}
