package server

import (
	"fmt"
	"net/http"
	"strings"
)

// OfficialStatisticsScope is the tenant/workspace boundary for Phase 6
// resources. Admin keys authorize the operation; these values authorize the
// data partition being operated on.
type OfficialStatisticsScope struct {
	TenantID    string
	WorkspaceID string
}

func defaultOfficialStatisticsScope() OfficialStatisticsScope {
	scope := OfficialStatisticsScope{TenantID: "default", WorkspaceID: "default"}
	if cfg != nil {
		if strings.TrimSpace(cfg.TenantID) != "" {
			scope.TenantID = strings.TrimSpace(cfg.TenantID)
		}
		if strings.TrimSpace(cfg.WorkspaceID) != "" {
			scope.WorkspaceID = strings.TrimSpace(cfg.WorkspaceID)
		}
	}
	return scope
}

func officialStatisticsScope(r *http.Request) (OfficialStatisticsScope, error) {
	scope := defaultOfficialStatisticsScope()
	if value := strings.TrimSpace(r.Header.Get("X-Tenant-ID")); value != "" {
		scope.TenantID = value
	}
	if value := strings.TrimSpace(r.Header.Get("X-Workspace-ID")); value != "" {
		scope.WorkspaceID = value
	}
	if !validOfficialScopeValue(scope.TenantID) || !validOfficialScopeValue(scope.WorkspaceID) {
		return OfficialStatisticsScope{}, fmt.Errorf("tenant and workspace headers must be 1-128 characters using letters, numbers, '.', '_' or '-'")
	}
	return scope, nil
}

func validOfficialScopeValue(value string) bool {
	if value == "" || len(value) > 128 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func officialScopeOrDefault(scopes []OfficialStatisticsScope) OfficialStatisticsScope {
	if len(scopes) > 0 && scopes[0].TenantID != "" && scopes[0].WorkspaceID != "" {
		return scopes[0]
	}
	return defaultOfficialStatisticsScope()
}

func requireOfficialStatisticsScope(w http.ResponseWriter, r *http.Request) (OfficialStatisticsScope, bool) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return OfficialStatisticsScope{}, false
	}
	scope, err := officialStatisticsScope(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return OfficialStatisticsScope{}, false
	}
	return scope, true
}

// requireOfficialStatisticsReadScope allows authenticated field and analyst
// clients to read workspace-owned templates while keeping mutations admin-only.
func requireOfficialStatisticsReadScope(w http.ResponseWriter, r *http.Request) (OfficialStatisticsScope, bool) {
	if !checkAPIKey(r) && !checkInternalKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return OfficialStatisticsScope{}, false
	}
	scope, err := officialStatisticsScope(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return OfficialStatisticsScope{}, false
	}
	return scope, true
}
