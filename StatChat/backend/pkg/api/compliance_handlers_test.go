package api

import (
	"encoding/csv"
	"strings"
	"testing"

	"statchat/pkg/model"
)

func TestComplianceAdministratorRoles(t *testing.T) {
	for _, role := range []string{"admin", "superadmin", "tenant_admin", "platform_admin", "TENANT_ADMIN"} {
		if !isComplianceAdministrator(model.User{Roles: []string{role}}) {
			t.Fatalf("expected %q to administer compliance", role)
		}
	}
	for _, role := range []string{"member", "analyst", "channel_admin", ""} {
		if isComplianceAdministrator(model.User{Roles: []string{role}}) {
			t.Fatalf("did not expect %q to administer compliance", role)
		}
	}
}

func TestCSVSafePreventsSpreadsheetFormulas(t *testing.T) {
	for _, value := range []string{"=SUM(A1:A2)", "+cmd", "-1+2", "@IMPORT", "  =hidden"} {
		if escaped := csvSafe(value); !strings.HasPrefix(escaped, "'") {
			t.Fatalf("expected dangerous value %q to be escaped, got %q", value, escaped)
		}
	}
	if escaped := csvSafe("ordinary text"); escaped != "ordinary text" {
		t.Fatalf("ordinary text changed to %q", escaped)
	}
}

func TestConversationCSVHasStableColumns(t *testing.T) {
	body, err := encodeConversationCSV([]model.Message{{ID: "message-1", SenderID: "user-1", Sender: "Analyst", Text: "=unsafe", Status: "active"}})
	if err != nil {
		t.Fatal(err)
	}
	records, err := csv.NewReader(strings.NewReader(string(body))).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 || len(records[0]) != 9 || records[1][4] != "'=unsafe" {
		t.Fatalf("unexpected export records: %#v", records)
	}
}

func TestSafeExportName(t *testing.T) {
	if got := safeExportName(`tenant/conversation:42`); got != "tenant-conversation-42" {
		t.Fatalf("unexpected safe name %q", got)
	}
}
