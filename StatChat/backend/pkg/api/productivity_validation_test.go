package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"statchat/pkg/model"
)

func TestValidateTaskInputAppliesDefaults(t *testing.T) {
	record := model.Task{Title: "  Prepare report  "}
	w := httptest.NewRecorder()
	if !validateTaskInput(w, &record, false) {
		t.Fatalf("expected default task values to be valid: %s", w.Body.String())
	}
	if record.Title != "Prepare report" || record.Priority != "medium" {
		t.Fatalf("expected normalized defaults, got %+v", record)
	}
}

func TestValidateTaskInputRejectsUnknownPriority(t *testing.T) {
	record := model.Task{Title: "Prepare report", Priority: "critical"}
	w := httptest.NewRecorder()
	if validateTaskInput(w, &record, false) {
		t.Fatal("expected unknown priority to be rejected")
	}
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestValidateTaskInputRejectsUnknownStatus(t *testing.T) {
	record := model.Task{Title: "Prepare report", Status: "finished"}
	w := httptest.NewRecorder()
	if validateTaskInput(w, &record, false) {
		t.Fatal("expected unknown status to be rejected")
	}
	if w.Code != 400 {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestTaskStatusAndPriorityCatalog(t *testing.T) {
	for _, status := range []string{"todo", "in_progress", "blocked", "done"} {
		if !isTaskStatus(status) {
			t.Errorf("expected status %q to be valid", status)
		}
	}
	for _, priority := range []string{"low", "medium", "high", "urgent"} {
		if !isTaskPriority(priority) {
			t.Errorf("expected priority %q to be valid", priority)
		}
	}
}

func TestOptionalRFC3339RejectsInvalidValue(t *testing.T) {
	w := httptest.NewRecorder()
	if _, ok := optionalRFC3339(w, "not-a-date"); ok || w.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid RFC3339 value to return 400, got ok=%v code=%d", ok, w.Code)
	}
}

func TestValidateCalendarEventRequiresOrderedTimes(t *testing.T) {
	start := time.Date(2026, 8, 26, 10, 0, 0, 0, time.UTC)
	event := model.CalendarEvent{Title: "Planning", StartAt: start, EndAt: start.Add(-time.Hour)}
	w := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/v1/chat/calendar", nil)
	ok := validateCalendarEvent(w, request, &event)
	if ok || w.Code != http.StatusBadRequest {
		t.Fatalf("expected reversed calendar times to return 400, got ok=%v code=%d", ok, w.Code)
	}
}
