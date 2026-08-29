package main

import "testing"

func TestCanAccessFileRecordRequiresMatchingTenant(t *testing.T) {
	record := FileRecord{ID: "file-1", TenantID: "tenant-alpha"}

	if !canAccessFileRecord(record, "tenant-alpha", "viewer") {
		t.Fatal("expected matching tenant to access file")
	}
	if canAccessFileRecord(record, "tenant-beta", "viewer") {
		t.Fatal("expected different tenant to be denied")
	}
	if canAccessFileRecord(record, "", "viewer") {
		t.Fatal("expected missing tenant context to be denied")
	}
}

func TestCanAccessFileRecordAllowsPlatformRoles(t *testing.T) {
	record := FileRecord{ID: "file-1", TenantID: "tenant-alpha"}

	for _, role := range []string{"admin", "superadmin", "platform_admin"} {
		if !canAccessFileRecord(record, "tenant-beta", role) {
			t.Fatalf("expected platform role %q to access cross-tenant file", role)
		}
	}
}

func TestCanAccessFileRecordDeniesTenantlessRecordsForTenantUsers(t *testing.T) {
	record := FileRecord{ID: "legacy-file"}

	if canAccessFileRecord(record, "tenant-alpha", "viewer") {
		t.Fatal("expected tenant user to be denied tenantless legacy record")
	}
	if !canAccessFileRecord(record, "tenant-alpha", "platform_admin") {
		t.Fatal("expected platform administrator to access tenantless legacy record")
	}
}

func TestResolveEventTenantID(t *testing.T) {
	if got := resolveEventTenantID(DomainEvent{TenantID: "tenant-alpha"}); got != "tenant-alpha" {
		t.Fatalf("expected explicit tenant, got %q", got)
	}

	got := resolveEventTenantID(DomainEvent{Payload: map[string]interface{}{"tenant_id": "tenant-beta"}})
	if got != "tenant-beta" {
		t.Fatalf("expected payload tenant, got %q", got)
	}

	if got := resolveEventTenantID(DomainEvent{}); got != "" {
		t.Fatalf("expected empty tenant fallback, got %q", got)
	}
}
