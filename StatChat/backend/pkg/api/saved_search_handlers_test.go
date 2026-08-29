package api

import (
	"net/http/httptest"
	"testing"
	"time"
)

func TestParseMessageSearchFilters(t *testing.T) {
	request := httptest.NewRequest("GET", "/v1/search?q=report&conversationId=group-1&sender=Amina&from=2026-08-01&to=2026-08-28&hasAttachment=true&savedOnly=true", nil)
	filters, active, err := parseMessageSearchFilters(request)
	if err != nil {
		t.Fatalf("parseMessageSearchFilters() error = %v", err)
	}
	if !active || filters.Query != "report" || filters.ConversationID != "group-1" || filters.Sender != "Amina" || filters.HasAttachment == nil || !*filters.HasAttachment || !filters.SavedOnly {
		t.Fatalf("unexpected filters: %#v", filters)
	}
	if filters.DateFrom == nil || filters.DateTo == nil || filters.DateTo.Sub(*filters.DateFrom) != 28*24*time.Hour {
		t.Fatalf("unexpected date range: %v to %v", filters.DateFrom, filters.DateTo)
	}
}

func TestParseMessageSearchFiltersRejectsInvalidValues(t *testing.T) {
	for _, path := range []string{
		"/v1/search?from=not-a-date",
		"/v1/search?from=2026-08-20&to=2026-08-01",
		"/v1/search?hasAttachment=perhaps",
		"/v1/search?savedOnly=perhaps",
	} {
		request := httptest.NewRequest("GET", path, nil)
		if _, _, err := parseMessageSearchFilters(request); err == nil {
			t.Fatalf("expected %s to be rejected", path)
		}
	}
}
