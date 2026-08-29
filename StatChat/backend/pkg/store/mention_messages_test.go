package store

import "testing"

func TestNormalizeMentionTargets(t *testing.T) {
	members := []string{"sender", "member-1", "member-2"}
	targets, err := NormalizeMentionTargets([]string{"member-2", "member-1", "member-2", "sender"}, members, "sender", false)
	if err != nil {
		t.Fatalf("NormalizeMentionTargets() error = %v", err)
	}
	if len(targets) != 2 || targets[0] != "member-2" || targets[1] != "member-1" {
		t.Fatalf("unexpected normalized targets: %#v", targets)
	}
	if _, err := NormalizeMentionTargets([]string{"outsider"}, members, "sender", false); err == nil {
		t.Fatal("expected outsider mention to be rejected")
	}
	all, err := NormalizeMentionTargets(nil, members, "sender", true)
	if err != nil || len(all) != 2 {
		t.Fatalf("mention all = %#v, %v; want two recipients", all, err)
	}
}

func TestShouldRouteMessageNotification(t *testing.T) {
	prefs := messageNotificationPreferences{messages: true, groups: false, mentions: true}
	if !shouldRouteMessageNotification(prefs, "direct", false, false) {
		t.Fatal("direct messages should follow the enabled message preference")
	}
	if shouldRouteMessageNotification(prefs, "group", false, false) {
		t.Fatal("group messages should follow the disabled group preference")
	}
	if shouldRouteMessageNotification(prefs, "direct", true, false) {
		t.Fatal("muted ordinary messages should not route")
	}
	if !shouldRouteMessageNotification(prefs, "group", true, true) {
		t.Fatal("enabled explicit mentions should override conversation mute")
	}
	prefs.mentions = false
	if shouldRouteMessageNotification(prefs, "direct", false, true) {
		t.Fatal("mentions should follow the mention preference")
	}
}
