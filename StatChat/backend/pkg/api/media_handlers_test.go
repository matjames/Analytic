package api

import "testing"

func TestMediaCatalogSearchesLabelsAndTags(t *testing.T) {
	item, ok := findMediaItem("celebrate")
	if !ok || item.Kind != "gif" || item.PreviewURL == "" {
		t.Fatalf("expected built-in celebrate animation, got %+v", item)
	}
	if item.PreviewURL[:26] != "data:image/svg+xml;base64," {
		t.Fatalf("expected self-contained SVG data URL")
	}
}

func TestMediaCatalogRejectsUnknownID(t *testing.T) {
	if _, ok := findMediaItem("arbitrary-remote-url"); ok {
		t.Fatal("expected unknown media ID to be rejected")
	}
}
