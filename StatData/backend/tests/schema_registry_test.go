package tests

import (
	"context"
	"testing"

	"statdata-backend/internal/models"
	"statdata-backend/internal/store"
)

func TestSchemaRegistryAndEvolution(t *testing.T) {
	ctx := context.Background()
	memStore := store.NewMemStore()

	// 1. Register v1 schema
	schV1 := &models.SchemaDefinition{
		ID:            "sch-hospital-v1",
		Subject:       "hospital_admissions",
		Version:       1,
		SchemaType:    "JSON_SCHEMA",
		SchemaContent: `{"type":"object","properties":{"patient_id":{"type":"string"}},"required":["patient_id"]}`,
		Compatibility: "BACKWARD",
		Fields: []models.SchemaField{
			{Name: "patient_id", Type: "STRING", Nullable: false},
			{Name: "facility_code", Type: "STRING", Nullable: false},
		},
		TenantID:  "default",
		CreatedBy: "admin",
	}

	if err := memStore.RegisterSchema(ctx, schV1); err != nil {
		t.Fatalf("Failed to register schema v1: %v", err)
	}

	// 2. Register v2 schema
	schV2 := &models.SchemaDefinition{
		ID:            "sch-hospital-v2",
		Subject:       "hospital_admissions",
		Version:       2,
		SchemaType:    "JSON_SCHEMA",
		SchemaContent: `{"type":"object","properties":{"patient_id":{"type":"string"},"admission_date":{"type":"string"}},"required":["patient_id"]}`,
		Compatibility: "BACKWARD",
		Fields: []models.SchemaField{
			{Name: "patient_id", Type: "STRING", Nullable: false},
			{Name: "facility_code", Type: "STRING", Nullable: false},
			{Name: "admission_date", Type: "TIMESTAMP", Nullable: true},
		},
		TenantID:  "default",
		CreatedBy: "admin",
	}

	if err := memStore.RegisterSchema(ctx, schV2); err != nil {
		t.Fatalf("Failed to register schema v2: %v", err)
	}

	// 3. Fetch latest schema by subject
	latest, err := memStore.GetLatestSchemaBySubject(ctx, "hospital_admissions", "default")
	if err != nil {
		t.Fatalf("GetLatestSchemaBySubject failed: %v", err)
	}
	if latest.Version != 2 {
		t.Errorf("Expected latest schema version 2, got %d", latest.Version)
	}
	if len(latest.Fields) != 3 {
		t.Errorf("Expected 3 fields in v2, got %d", len(latest.Fields))
	}
}
