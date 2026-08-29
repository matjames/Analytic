// Package main: Configs folder tree and role-based report permissions.
// Reads configs directory for the Role Permissions UI; stores allowed paths per role in role-permissions.json.
package main

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// ConfigsTreeNode represents a folder or report in the configs tree.
type ConfigsTreeNode struct {
	ID       string            `json:"id"`   // path without .yaml (e.g. "Deaths", "Deaths/sub", "community/report")
	Name     string            `json:"name"` // display name (folder or report name)
	Type     string            `json:"type"` // "folder" or "report"
	Children []ConfigsTreeNode `json:"children,omitempty"`
}

const rolePermissionsFile = "configs/role-permissions.json"

// BuildConfigsTree walks the configs directory and returns a nested tree (categories → subcategories → .yaml reports).
func BuildConfigsTree() ([]ConfigsTreeNode, error) {
	// path (with /) -> type: "dir" or "report"
	type entry struct {
		path string
		dir  bool
	}
	var entries []entry
	err := filepath.Walk("configs", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		rel, e := filepath.Rel("configs", path)
		if e != nil || rel == "." {
			return nil
		}
		rel = filepath.ToSlash(rel)
		if info.IsDir() {
			entries = append(entries, entry{rel, true})
			return nil
		}
		if !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		parts := strings.Split(rel, "/")
		if len(parts) > 3 {
			return nil
		}
		entries = append(entries, entry{strings.TrimSuffix(rel, ".yaml"), false})
		return nil
	})
	if err != nil {
		return nil, err
	}

	// Build nested tree: folder id -> list of child nodes (folder or report)
	type child struct {
		id   string
		name string
		typ  string
	}
	folderChildren := make(map[string][]child)
	seenFolders := make(map[string]bool)
	for _, e := range entries {
		if e.dir {
			seenFolders[e.path] = true
			parts := strings.Split(e.path, "/")
			parent := ""
			if len(parts) > 1 {
				parent = strings.Join(parts[:len(parts)-1], "/")
			}
			name := parts[len(parts)-1]
			folderChildren[parent] = append(folderChildren[parent], child{e.path, name, "folder"})
		} else {
			parts := strings.Split(e.path, "/")
			name := parts[len(parts)-1]
			parent := ""
			if len(parts) > 1 {
				parent = strings.Join(parts[:len(parts)-1], "/")
			}
			folderChildren[parent] = append(folderChildren[parent], child{e.path, name, "report"})
		}
	}
	for k := range folderChildren {
		sort.Slice(folderChildren[k], func(i, j int) bool { return folderChildren[k][i].name < folderChildren[k][j].name })
	}

	var build func(prefix string) []ConfigsTreeNode
	build = func(prefix string) []ConfigsTreeNode {
		ch := folderChildren[prefix]
		var out []ConfigsTreeNode
		for _, c := range ch {
			node := ConfigsTreeNode{ID: c.id, Name: c.name, Type: c.typ}
			if c.typ == "folder" {
				node.Children = build(c.id)
			}
			out = append(out, node)
		}
		return out
	}

	result := build("")
	return result, nil
}

// UserPermissions holds the effective permissions for a user (merged from all their roles).
type UserPermissions struct {
	Paths   []string // allowed report/folder paths (excludes __builder, __admin)
	Builder bool     // can access Builder
	Admin   bool     // can access User Management
}

// GetEffectivePermissionsForUser merges role-permissions for the given client roles.
// Returns nil if no role-permissions.json or no roles; caller may treat as "no access" or "full access" by policy.
func GetEffectivePermissionsForUser(clientRoles []string) (*UserPermissions, error) {
	data, err := os.ReadFile(rolePermissionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var all map[string][]string
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	out := &UserPermissions{Paths: []string{}}
	pathSet := make(map[string]bool)
	for _, role := range clientRoles {
		paths, ok := all[role]
		if !ok {
			// Case-insensitive match so Keycloak "Builder" matches role-permissions "builder"
			for k, v := range all {
				if strings.EqualFold(role, k) {
					paths = v
					ok = true
					break
				}
			}
			if !ok {
				continue
			}
		}
		for _, p := range paths {
			switch p {
			case "__builder":
				out.Builder = true
			case "__admin":
				out.Admin = true
			default:
				pathSet[p] = true
			}
		}
	}
	for p := range pathSet {
		out.Paths = append(out.Paths, p)
	}
	sort.Strings(out.Paths)
	return out, nil
}

// UserCanAccessReport returns true if the user is allowed to access the report (by path or parent folder).
// If perms is nil or Paths is empty, returns false (no access).
// If perms.Paths contains "*", returns true.
// Otherwise returns true if reportID equals an allowed path or has an allowed path as prefix (e.g. "Deaths" allows "Deaths/foo").
func UserCanAccessReport(reportID string, perms *UserPermissions) bool {
	if perms == nil {
		return false
	}
	for _, p := range perms.Paths {
		if p == "*" {
			return true
		}
		if p == reportID {
			return true
		}
		// folder: "Deaths" allows "Deaths" and "Deaths/..."
		if strings.HasPrefix(reportID+"/", p+"/") {
			return true
		}
	}
	return false
}

// GetRolePermissions returns the list of allowed paths (folder or report ids) for a role.
func GetRolePermissions(role string) ([]string, error) {
	data, err := os.ReadFile(rolePermissionsFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var all map[string][]string
	if err := json.Unmarshal(data, &all); err != nil {
		return nil, err
	}
	return all[role], nil
}

// SaveRolePermissions persists allowed paths for a role.
func SaveRolePermissions(role string, paths []string) error {
	var all map[string][]string
	data, err := os.ReadFile(rolePermissionsFile)
	if err == nil {
		_ = json.Unmarshal(data, &all)
	}
	if all == nil {
		all = make(map[string][]string)
	}
	if len(paths) == 0 {
		delete(all, role)
	} else {
		all[role] = paths
	}
	data, err = json.MarshalIndent(all, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(rolePermissionsFile, data, 0644)
}

func keycloakConfigsTree(w http.ResponseWriter, r *http.Request) {
	tree, err := BuildConfigsTree()
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"tree": tree})
}

func keycloakGetRolePermissions(w http.ResponseWriter, r *http.Request) {
	role := r.URL.Query().Get("role")
	if role == "" {
		writeJSONError(w, http.StatusBadRequest, "role query parameter required")
		return
	}
	paths, err := GetRolePermissions(role)
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if paths == nil {
		paths = []string{}
	}
	writeJSON(w, map[string]interface{}{"role": role, "paths": paths})
}

func keycloakSaveRolePermissions(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Role  string   `json:"role"`
		Paths []string `json:"paths"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON")
		return
	}
	if body.Role == "" {
		writeJSONError(w, http.StatusBadRequest, "role is required")
		return
	}
	if err := SaveRolePermissions(body.Role, body.Paths); err != nil {
		writeJSONError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, map[string]interface{}{"role": body.Role, "saved": true})
}
