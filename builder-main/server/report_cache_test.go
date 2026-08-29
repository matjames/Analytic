package main

import (
	"testing"
	"time"
)

func TestBuildReportCacheKey(t *testing.T) {
	f1 := ReportFilters{Year: "2024", Month: "3"}
	f2 := ReportFilters{Year: "2024", Month: "4"}
	k1 := buildReportCacheKey("community/a", "hash1", f1)
	k2 := buildReportCacheKey("community/a", "hash1", f1)
	k3 := buildReportCacheKey("community/a", "hash1", f2)
	if k1 != k2 {
		t.Fatalf("same inputs should match")
	}
	if k1 == k3 {
		t.Fatalf("different filters should not match")
	}
}

func TestFullReportCacheGetSetTTL(t *testing.T) {
	c := &fullReportResponseCache{
		enabled:    true,
		ttl:        time.Hour,
		maxEntries: 10,
		maxBytes:   1 << 20,
	}
	key := buildReportCacheKey("r/x", "abc", ReportFilters{Year: "2024"})
	body := []byte(`{"id":"r/x"}`)
	c.Set(key, body, "abc")
	got := c.Get(key, "abc")
	if string(got) != string(body) {
		t.Fatalf("got %q want %q", got, body)
	}
	// wrong yaml hash
	c.Set(key, body, "abc")
	if c.Get(key, "wrong") != nil {
		t.Fatal("expected miss on yaml mismatch")
	}
}
