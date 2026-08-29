package main

import (
	"strings"
	"testing"
)

func TestValidateAdvancedSQL(t *testing.T) {
	tests := []struct {
		name    string
		sql     string
		wantErr string
	}{
		{
			name:    "empty",
			sql:     "   ",
			wantErr: "SQL query cannot be empty",
		},
		{
			name:    "simple select ok",
			sql:     "SELECT a FROM schema.t WHERE id = 1",
			wantErr: "",
		},
		{
			name:    "select with trailing semicolon ok",
			sql:     "SELECT 1;",
			wantErr: "",
		},
		{
			name:    "insert blocked",
			sql:     "INSERT INTO t VALUES (1)",
			wantErr: "SQL contains blocked keyword: INSERT",
		},
		{
			name:    "update blocked",
			sql:     "UPDATE t SET a = 1",
			wantErr: "SQL contains blocked keyword: UPDATE",
		},
		{
			name:    "delete blocked",
			sql:     "DELETE FROM t",
			wantErr: "SQL contains blocked keyword: DELETE",
		},
		{
			name:    "pg_tables blocked",
			sql:     "SELECT * FROM pg_tables",
			wantErr: "SQL contains blocked keyword: PG_TABLES",
		},
		{
			name:    "information_schema blocked",
			sql:     "SELECT * FROM information_schema.tables",
			wantErr: "SQL contains blocked keyword: INFORMATION_SCHEMA",
		},
		{
			name:    "pg_sleep blocked",
			sql:     "SELECT pg_sleep(1)",
			wantErr: "SQL contains blocked keyword: PG_SLEEP",
		},
		{
			name:    "pg_read blocked as standalone token",
			sql:     "SELECT PG_READ FROM t",
			wantErr: "SQL contains blocked keyword: PG_READ",
		},
		{
			name:    "multiple statements blocked",
			sql:     "SELECT 1; SELECT 2",
			wantErr: "only single SQL statements are allowed",
		},
		{
			name:    "string literal do not false positive on do",
			sql:     "SELECT col FROM t WHERE note = '%do not panic%'",
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateAdvancedSQL(tt.sql)
			if tt.wantErr == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tt.wantErr)
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("error %q should contain %q", err.Error(), tt.wantErr)
			}
		})
	}
}
