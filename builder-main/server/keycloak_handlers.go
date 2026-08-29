// Package main: HTTP handlers for Keycloak Admin API proxy (users, roles, permissions).
// All handlers require the admin role (KEYCLOAK_ADMIN_ROLE) and use server-side admin token.
package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"strings"
)

const keycloakAPIPrefix = "/api/keycloak/"

// KeycloakAdminHandler routes /api/keycloak/* to the appropriate Keycloak Admin API.
// Wrap with RequireRole(KeycloakAdminRole, KeycloakAdminHandler).
func KeycloakAdminHandler(w http.ResponseWriter, r *http.Request) {
	if authMode == "off" {
		writeJSONError(w, http.StatusForbidden, "user management requires AUTH_MODE=on")
		return
	}
	path := strings.TrimPrefix(r.URL.Path, basePath+keycloakAPIPrefix)
	path = strings.Trim(path, "/")
	parts := strings.SplitN(path, "/", 10)

	switch {
	// --- Users ---
	case path == "users" && r.Method == http.MethodGet:
		keycloakListUsers(w, r)
		return
	case path == "users" && r.Method == http.MethodPost:
		keycloakCreateUser(w, r)
		return
	case len(parts) >= 3 && parts[0] == "users" && parts[2] == "role-mappings":
		userID := parts[1]
		if len(parts) == 3 && r.Method == http.MethodGet {
			keycloakGetUserRoleMappings(w, r, userID)
			return
		}
		if len(parts) == 4 && parts[3] == "realm" {
			if r.Method == http.MethodGet {
				keycloakGetUserRealmRoles(w, r, userID)
				return
			}
			if r.Method == http.MethodPost {
				keycloakAddUserRealmRoles(w, r, userID)
				return
			}
			if r.Method == http.MethodDelete {
				keycloakRemoveUserRealmRoles(w, r, userID)
				return
			}
		}
		if len(parts) == 5 && parts[3] == "clients" {
			clientID := parts[4]
			if r.Method == http.MethodGet {
				keycloakGetUserClientRoles(w, r, userID, clientID)
				return
			}
			if r.Method == http.MethodPost {
				keycloakAddUserClientRoles(w, r, userID, clientID)
				return
			}
			if r.Method == http.MethodDelete {
				keycloakRemoveUserClientRoles(w, r, userID, clientID)
				return
			}
		}
	case len(parts) >= 1 && parts[0] == "users" && r.Method == http.MethodGet:
		if len(parts) == 1 {
			http.Error(w, "user id required", http.StatusBadRequest)
			return
		}
		if len(parts) >= 3 && parts[2] == "groups" {
			keycloakGetUserGroups(w, r, parts[1])
			return
		}
		keycloakGetUser(w, r, parts[1])
		return
	case len(parts) == 4 && parts[0] == "users" && parts[2] == "groups" && r.Method == http.MethodPut:
		keycloakAddUserToGroup(w, r, parts[1], parts[3])
		return
	case len(parts) == 4 && parts[0] == "users" && parts[2] == "groups" && r.Method == http.MethodDelete:
		keycloakRemoveUserFromGroup(w, r, parts[1], parts[3])
		return
	case len(parts) >= 1 && parts[0] == "users" && r.Method == http.MethodPut:
		if len(parts) < 2 {
			http.Error(w, "user id required", http.StatusBadRequest)
			return
		}
		if len(parts) == 3 && parts[2] == "reset-password" {
			keycloakResetUserPassword(w, r, parts[1])
			return
		}
		keycloakUpdateUser(w, r, parts[1])
		return
	case len(parts) >= 1 && parts[0] == "users" && r.Method == http.MethodDelete:
		if len(parts) < 2 {
			http.Error(w, "user id required", http.StatusBadRequest)
			return
		}
		keycloakDeleteUser(w, r, parts[1])
		return
	// --- Client roles (clients/{clientId}/roles) ---
	case len(parts) >= 3 && parts[0] == "clients" && parts[2] == "roles" && r.Method == http.MethodGet:
		if len(parts) == 3 {
			keycloakListClientRoles(w, r, parts[1])
			return
		}
		if len(parts) == 4 {
			keycloakGetClientRoleByName(w, r, parts[1], parts[3])
			return
		}
	case len(parts) == 3 && parts[0] == "clients" && parts[2] == "roles" && r.Method == http.MethodPost:
		keycloakCreateClientRole(w, r, parts[1])
		return
	case len(parts) == 4 && parts[0] == "clients" && parts[2] == "roles" && r.Method == http.MethodPut:
		keycloakUpdateClientRole(w, r, parts[1], parts[3])
		return
	case len(parts) == 4 && parts[0] == "clients" && parts[2] == "roles" && r.Method == http.MethodDelete:
		keycloakDeleteClientRoleByName(w, r, parts[1], parts[3])
		return
	// --- Realm roles (kept for backwards compat / Role Permissions) ---
	case path == "roles" && r.Method == http.MethodGet:
		keycloakListRealmRoles(w, r)
		return
	case path == "roles" && r.Method == http.MethodPost:
		keycloakCreateRealmRole(w, r)
		return
	case len(parts) >= 1 && parts[0] == "roles" && r.Method == http.MethodGet:
		if len(parts) < 2 {
			http.Error(w, "role name required", http.StatusBadRequest)
			return
		}
		keycloakGetRealmRoleByName(w, r, parts[1])
		return
	case len(parts) >= 1 && parts[0] == "roles" && r.Method == http.MethodDelete:
		if len(parts) < 2 {
			http.Error(w, "role name required", http.StatusBadRequest)
			return
		}
		keycloakDeleteRealmRoleByName(w, r, parts[1])
		return
	// --- Role Permissions (realm + client roles view) ---
	case path == "permissions" && r.Method == http.MethodGet:
		keycloakListPermissions(w, r)
		return
	// --- Configs tree (for role permissions UI) ---
	case path == "configs-tree" && r.Method == http.MethodGet:
		keycloakConfigsTree(w, r)
		return
	// --- Role permissions (report/folder access per role) ---
	case path == "role-permissions" && r.Method == http.MethodGet:
		keycloakGetRolePermissions(w, r)
		return
	case (path == "role-permissions" && r.Method == http.MethodPost) || (path == "role-permissions" && r.Method == http.MethodPut):
		keycloakSaveRolePermissions(w, r)
		return
	// --- Groups ---
	case path == "groups" && r.Method == http.MethodGet:
		keycloakListGroups(w, r)
		return
	case path == "groups" && r.Method == http.MethodPost:
		keycloakCreateGroup(w, r)
		return
	case len(parts) == 5 && parts[0] == "groups" && parts[2] == "role-mappings" && parts[3] == "clients" && r.Method == http.MethodPost:
		keycloakAddGroupClientRoles(w, r, parts[1], parts[4])
		return
	case len(parts) == 5 && parts[0] == "groups" && parts[2] == "role-mappings" && parts[3] == "clients" && r.Method == http.MethodDelete:
		keycloakRemoveGroupClientRoles(w, r, parts[1], parts[4])
		return
	case len(parts) >= 1 && parts[0] == "groups" && r.Method == http.MethodGet:
		if len(parts) < 2 {
			http.Error(w, "group id required", http.StatusBadRequest)
			return
		}
		if len(parts) == 2 {
			keycloakGetGroup(w, r, parts[1])
			return
		}
		if len(parts) == 3 && parts[2] == "role-mappings" {
			keycloakGetGroupRoleMappings(w, r, parts[1])
			return
		}
		if len(parts) == 4 && parts[2] == "role-mappings" && parts[3] == "realm" {
			if r.Method == http.MethodGet {
				keycloakGetGroupRealmRoles(w, r, parts[1])
				return
			}
			if r.Method == http.MethodPost {
				keycloakAddGroupRealmRoles(w, r, parts[1])
				return
			}
			if r.Method == http.MethodDelete {
				keycloakRemoveGroupRealmRoles(w, r, parts[1])
				return
			}
		}
		if len(parts) == 5 && parts[2] == "role-mappings" && parts[3] == "clients" {
			if r.Method == http.MethodGet {
				keycloakGetGroupClientRoles(w, r, parts[1], parts[4])
				return
			}
			if r.Method == http.MethodPost {
				keycloakAddGroupClientRoles(w, r, parts[1], parts[4])
				return
			}
			if r.Method == http.MethodDelete {
				keycloakRemoveGroupClientRoles(w, r, parts[1], parts[4])
				return
			}
		}
	case len(parts) >= 1 && parts[0] == "groups" && r.Method == http.MethodPut:
		if len(parts) < 2 {
			http.Error(w, "group id required", http.StatusBadRequest)
			return
		}
		keycloakUpdateGroup(w, r, parts[1])
		return
	case len(parts) >= 1 && parts[0] == "groups" && r.Method == http.MethodDelete:
		if len(parts) < 2 {
			http.Error(w, "group id required", http.StatusBadRequest)
			return
		}
		keycloakDeleteGroup(w, r, parts[1])
		return
	}

	http.Error(w, "not found", http.StatusNotFound)
}

func keycloakListUsers(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path := "users?"
	if v := q.Get("first"); v != "" {
		path += "first=" + v + "&"
	}
	if v := q.Get("max"); v != "" {
		path += "max=" + v + "&"
	}
	if v := q.Get("search"); v != "" {
		path += "search=" + v + "&"
	}
	path = strings.TrimSuffix(path, "&")
	if path == "users?" {
		path = "users"
	}

	resp, err := keycloakAdminGetFunc(path)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakCreateUser(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "users", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusCreated {
		loc := resp.Header.Get("Location")
		if loc != "" {
			w.Header().Set("Location", loc)
		}
	}
	io.Copy(w, resp.Body)
}

func keycloakGetUser(w http.ResponseWriter, r *http.Request, id string) {
	resp, err := keycloakAdminGetFunc("users/" + id)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakGetUserGroups(w http.ResponseWriter, r *http.Request, userID string) {
	path := "users/" + userID + "/groups"
	if max := r.URL.Query().Get("max"); max != "" {
		path += "?max=" + max
	}
	resp, err := keycloakAdminGetFunc(path)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakAddUserToGroup(w http.ResponseWriter, r *http.Request, userID, groupID string) {
	resp, err := keycloakAdminRequest(http.MethodPut, "users/"+userID+"/groups/"+groupID, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	w.WriteHeader(resp.StatusCode)
	if len(body) > 0 {
		w.Write(body)
	}
}

func keycloakRemoveUserFromGroup(w http.ResponseWriter, r *http.Request, userID, groupID string) {
	resp, err := keycloakAdminRequest(http.MethodDelete, "users/"+userID+"/groups/"+groupID, nil)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
}

func keycloakUpdateUser(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPut, "users/"+id, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakDeleteUser(w http.ResponseWriter, r *http.Request, id string) {
	resp, err := keycloakAdminDelete("users/" + id)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakResetUserPassword(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPut, "users/"+id+"/reset-password", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode != http.StatusNoContent && resp.Body != nil {
		io.Copy(w, resp.Body)
	}
}

func keycloakGetUserRoleMappings(w http.ResponseWriter, r *http.Request, userID string) {
	resp, err := keycloakAdminGetFunc("users/" + userID + "/role-mappings")
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakGetUserRealmRoles(w http.ResponseWriter, r *http.Request, userID string) {
	resp, err := keycloakAdminGetFunc("users/" + userID + "/role-mappings/realm")
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakAddUserRealmRoles(w http.ResponseWriter, r *http.Request, userID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "users/"+userID+"/role-mappings/realm", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakRemoveUserRealmRoles(w http.ResponseWriter, r *http.Request, userID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodDelete, "users/"+userID+"/role-mappings/realm", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakGetUserClientRoles(w http.ResponseWriter, r *http.Request, userID, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := keycloakAdminGetFunc("users/" + userID + "/role-mappings/clients/" + clientUUID)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakAddUserClientRoles(w http.ResponseWriter, r *http.Request, userID, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "users/"+userID+"/role-mappings/clients/"+clientUUID, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakRemoveUserClientRoles(w http.ResponseWriter, r *http.Request, userID, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodDelete, "users/"+userID+"/role-mappings/clients/"+clientUUID, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakListClientRoles(w http.ResponseWriter, r *http.Request, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		log.Printf("[Keycloak] keycloakListClientRoles: failed to resolve clientID=%q for path=%q error=%v", clientID, r.URL.Path, err)
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	log.Printf("[Keycloak] keycloakListClientRoles: resolved clientID=%q to clientUUID=%q", clientID, clientUUID)
	resp, err := keycloakAdminGetFunc("clients/" + clientUUID + "/roles")
	if err != nil {
		log.Printf("[Keycloak] keycloakListClientRoles: failed to fetch roles for clientID=%q clientUUID=%q error=%v", clientID, clientUUID, err)
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakCreateClientRole(w http.ResponseWriter, r *http.Request, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "clients/"+clientUUID+"/roles", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusCreated {
		loc := resp.Header.Get("Location")
		if loc != "" {
			w.Header().Set("Location", loc)
		}
	}
	io.Copy(w, resp.Body)
}

func keycloakGetClientRoleByName(w http.ResponseWriter, r *http.Request, clientID, roleName string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := keycloakAdminGetFunc("clients/" + clientUUID + "/roles/" + roleName)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakUpdateClientRole(w http.ResponseWriter, r *http.Request, clientID, roleName string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPut, "clients/"+clientUUID+"/roles/"+roleName, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if resp.Body != nil {
		io.Copy(w, resp.Body)
	}
}

func keycloakDeleteClientRoleByName(w http.ResponseWriter, r *http.Request, clientID, roleName string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := keycloakAdminDelete("clients/" + clientUUID + "/roles/" + roleName)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakListGroups(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	path := "groups?"
	if v := q.Get("first"); v != "" {
		path += "first=" + v + "&"
	}
	if v := q.Get("max"); v != "" {
		path += "max=" + v + "&"
	}
	if v := q.Get("search"); v != "" {
		path += "search=" + v + "&"
	}
	path = strings.TrimSuffix(path, "&")
	if path == "groups?" {
		path = "groups"
	}
	resp, err := keycloakAdminGetFunc(path)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakCreateGroup(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "groups", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	if resp.StatusCode == http.StatusCreated {
		loc := resp.Header.Get("Location")
		if loc != "" {
			w.Header().Set("Location", loc)
		}
	}
	io.Copy(w, resp.Body)
}

func keycloakGetGroup(w http.ResponseWriter, r *http.Request, id string) {
	resp, err := keycloakAdminGetFunc("groups/" + id)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakUpdateGroup(w http.ResponseWriter, r *http.Request, id string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPut, "groups/"+id, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakDeleteGroup(w http.ResponseWriter, r *http.Request, id string) {
	resp, err := keycloakAdminDelete("groups/" + id)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakGetGroupRoleMappings(w http.ResponseWriter, r *http.Request, groupID string) {
	resp, err := keycloakAdminGetFunc("groups/" + groupID + "/role-mappings")
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakGetGroupRealmRoles(w http.ResponseWriter, r *http.Request, groupID string) {
	resp, err := keycloakAdminGetFunc("groups/" + groupID + "/role-mappings/realm")
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakAddGroupRealmRoles(w http.ResponseWriter, r *http.Request, groupID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "groups/"+groupID+"/role-mappings/realm", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakRemoveGroupRealmRoles(w http.ResponseWriter, r *http.Request, groupID string) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodDelete, "groups/"+groupID+"/role-mappings/realm", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakGetGroupClientRoles(w http.ResponseWriter, r *http.Request, groupID, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	resp, err := keycloakAdminGetFunc("groups/" + groupID + "/role-mappings/clients/" + clientUUID)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakAddGroupClientRoles(w http.ResponseWriter, r *http.Request, groupID, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "groups/"+groupID+"/role-mappings/clients/"+clientUUID, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode >= 400 {
		log.Printf("[Keycloak] addGroupClientRoles group=%s client=%s status=%d body=%s", groupID, clientID, resp.StatusCode, string(respBody))
	}
	w.WriteHeader(resp.StatusCode)
	if len(respBody) > 0 {
		w.Write(respBody)
	}
}

func keycloakRemoveGroupClientRoles(w http.ResponseWriter, r *http.Request, groupID, clientID string) {
	clientUUID, err := getClientUUIDFunc(clientID)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, err.Error())
		return
	}
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodDelete, "groups/"+groupID+"/role-mappings/clients/"+clientUUID, body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakListRealmRoles(w http.ResponseWriter, r *http.Request) {
	resp, err := keycloakAdminGetFunc("roles")
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakCreateRealmRole(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid body")
		return
	}
	resp, err := keycloakAdminRequest(http.MethodPost, "roles", body)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func keycloakGetRealmRoleByName(w http.ResponseWriter, r *http.Request, name string) {
	resp, err := keycloakAdminGetFunc("roles/" + name)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	forwardResponse(w, resp)
}

func keycloakDeleteRealmRoleByName(w http.ResponseWriter, r *http.Request, name string) {
	// Keycloak DELETE realm role by name: DELETE /admin/realms/{realm}/roles/{role-name}
	resp, err := keycloakAdminDelete("roles/" + name)
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

// keycloakListPermissions returns realm roles and client roles (for KEYCLOAK_CLIENT_ID) as "permissions".
func keycloakListPermissions(w http.ResponseWriter, r *http.Request) {
	// Realm roles
	realmResp, err := keycloakAdminGetFunc("roles")
	if err != nil {
		writeJSONError(w, http.StatusBadGateway, err.Error())
		return
	}
	defer realmResp.Body.Close()
	var realmRoles []map[string]interface{}
	if err := json.NewDecoder(realmResp.Body).Decode(&realmRoles); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "decode realm roles: "+err.Error())
		return
	}

	// Client ID for app roles/group mappings (KEYCLOAK_ROLES_CLIENT_ID fallback to KEYCLOAK_CLIENT_ID).
	clientID := getAppClientID()

	// Get client UUID
	clientsResp, err := keycloakAdminGetFunc("clients?clientId=" + clientID)
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"realmRoles":  realmRoles,
			"clientRoles": []interface{}{},
			"clientId":    clientID,
		})
		return
	}
	defer clientsResp.Body.Close()
	var clients []map[string]interface{}
	if err := json.NewDecoder(clientsResp.Body).Decode(&clients); err != nil || len(clients) == 0 {
		writeJSON(w, map[string]interface{}{
			"realmRoles":  realmRoles,
			"clientRoles": []interface{}{},
			"clientId":    clientID,
		})
		return
	}
	clientUUID, _ := clients[0]["id"].(string)
	if clientUUID == "" {
		writeJSON(w, map[string]interface{}{
			"realmRoles":  realmRoles,
			"clientRoles": []interface{}{},
			"clientId":    clientID,
		})
		return
	}

	rolesResp, err := keycloakAdminGetFunc("clients/" + clientUUID + "/roles")
	if err != nil {
		writeJSON(w, map[string]interface{}{
			"realmRoles":  realmRoles,
			"clientRoles": []interface{}{},
			"clientId":    clientID,
		})
		return
	}
	defer rolesResp.Body.Close()
	var clientRoles []map[string]interface{}
	json.NewDecoder(rolesResp.Body).Decode(&clientRoles)

	writeJSON(w, map[string]interface{}{
		"realmRoles":  realmRoles,
		"clientRoles": clientRoles,
		"clientId":    clientID,
	})
}

func forwardResponse(w http.ResponseWriter, resp *http.Response) {
	for k, v := range resp.Header {
		if k == "Content-Length" {
			continue
		}
		for _, vv := range v {
			w.Header().Add(k, vv)
		}
	}
	w.WriteHeader(resp.StatusCode)
	io.Copy(w, resp.Body)
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(v)
}
