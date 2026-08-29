package main

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	statgatetenant "github.com/matjames/statgate-lib/tenant"

)

type API struct {
	ID, Name, Version, BaseURL, Protocol, Status, Owner, CreatedAt string
	Metadata                                                       map[string]any `json:"metadata"`
}
type Connector struct {
	ID, Name, Type, Status, Owner, CreatedAt string
	Config                                   map[string]any `json:"config"`
}
type Schema struct {
	ID, Name, Version, Format, Owner, CreatedAt string
	Definition                                  map[string]any `json:"definition"`
}
type Webhook struct {
	ID, Name, TargetURL, Status, Owner, CreatedAt string
	Events                                        []string `json:"events"`
	Secret                                        string   `json:"-"`
}
type APIKey struct{ ID, Name, Prefix, Scopes, Status, Owner, CreatedAt string }
type Plugin struct {
	ID, Name, Version, Status, Publisher, CreatedAt string
	Capabilities                                    []string       `json:"capabilities"`
	Manifest                                        map[string]any `json:"manifest"`
}
type Component struct {
	ID, Name, Category, Version, Status, Owner, CreatedAt string
	Schema                                                map[string]any `json:"schema"`
}
type GeneratedApp struct {
	ID, Name, Slug, Status, Owner, CreatedAt, DeploymentURL string
	Definition                                              map[string]any `json:"definition"`
}
type IntegrationEvent struct {
	ID, Type, Source, Subject, TenantID, OccurredAt string
	Payload                                         map[string]any `json:"payload"`
}
type ConnectedSystem struct {
	ID, Name, BaseURL, HealthURL, Status string
}

var db *sql.DB

var connectedSystems = []ConnectedSystem{
	{ID: "enterprise-core", Name: "Enterprise Core", BaseURL: "http://statgate-enterprise-core:8096", HealthURL: "http://statgate-enterprise-core:8096/health"},
	{ID: "pms", Name: "Projects Management System", BaseURL: "http://statgate-pms-api:8080", HealthURL: "http://statgate-pms-api:8080/health"},
	{ID: "rms", Name: "Research Management System", BaseURL: "http://statgate-rms-api:8080", HealthURL: "http://statgate-rms-api:8080/health"},
	{ID: "registry", Name: "Field Operations Registry", BaseURL: "http://statgate-registry-api:9090", HealthURL: "http://statgate-registry-api:9090/health"},
	{ID: "statchat", Name: "StatChat", BaseURL: "http://statchat-backend:4000", HealthURL: "http://statchat-backend:4000/health"},
	{ID: "helpdesk", Name: "Operations Helpdesk", BaseURL: "http://statgate-helpdesk-api:5000", HealthURL: "http://statgate-helpdesk-api:5000/health"},
	{ID: "governance", Name: "StatGovernance", BaseURL: "http://statgate-governance-api:8080", HealthURL: "http://statgate-governance-api:8080/health"},
	{ID: "analytics", Name: "StatGate Analytics", BaseURL: "http://statgate-analytics:5000", HealthURL: "http://statgate-analytics:5000/health"},
}

func configureConnectedSystems() {
	urlVars := map[string]string{"enterprise-core": "ENTERPRISE_CORE_URL", "pms": "PMS_API_URL", "rms": "RMS_API_URL", "registry": "REGISTRY_API_URL", "statchat": "STATCHAT_API_URL", "helpdesk": "HELPDESK_API_URL", "governance": "GOVERNANCE_API_URL", "analytics": "ANALYTICS_API_URL"}
	for index := range connectedSystems {
		if configured := strings.TrimRight(env(urlVars[connectedSystems[index].ID], connectedSystems[index].BaseURL), "/"); configured != "" {
			connectedSystems[index].BaseURL = configured
			connectedSystems[index].HealthURL = configured + "/health"
		}
	}
}

func env(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}
func id(prefix string) string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		panic(err)
	}
	return prefix + "_" + hex.EncodeToString(b)
}
func now() string { return time.Now().UTC().Format(time.RFC3339) }
func actor(c *gin.Context) string {
	if value := c.GetHeader("X-User-ID"); value != "" {
		return value
	}
	return "integration-system"
}

func initDB() {
	host := env("INTEGRATION_DB_HOST", env("ENTERPRISE_DB_HOST", ""))
	if host == "" {
		return
	}
	port := env("INTEGRATION_DB_PORT", env("ENTERPRISE_DB_PORT", "5432"))
	name := env("INTEGRATION_DB_NAME", env("ENTERPRISE_DB_NAME", "statgate_ml_staging"))
	user := env("INTEGRATION_DB_USER", env("ENTERPRISE_DB_USER", ""))
	password := env("INTEGRATION_DB_PASSWORD", env("ENTERPRISE_DB_PASSWORD", ""))
	dsn := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s", host, port, name, user, password, env("INTEGRATION_DB_SSLMODE", "disable"))
	var err error
	db, err = sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("integration database disabled: %v", err)
		return
	}
	if err = db.Ping(); err != nil {
		log.Printf("integration database unavailable: %v", err)
		db.Close()
		db = nil
		return
	}
	_, err = db.Exec(`CREATE SCHEMA IF NOT EXISTS integration;
CREATE TABLE IF NOT EXISTS integration.resources (kind TEXT NOT NULL, id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, body JSONB NOT NULL, created_at TIMESTAMPTZ NOT NULL DEFAULT now(), updated_at TIMESTAMPTZ NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS integration_resources_kind_tenant ON integration.resources(kind, tenant_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS integration.events (id TEXT PRIMARY KEY, tenant_id TEXT NOT NULL, event_type TEXT NOT NULL, source TEXT NOT NULL, subject TEXT, payload JSONB NOT NULL, occurred_at TIMESTAMPTZ NOT NULL);
CREATE INDEX IF NOT EXISTS integration_events_tenant_time ON integration.events(tenant_id, occurred_at DESC);`)
	if err != nil {
		log.Printf("integration schema initialization failed: %v", err)
		db.Close()
		db = nil
		return
	}
	bootstrapConnectedSystems()
}

func bootstrapConnectedSystems() {
	if db == nil {
		return
	}
	for _, system := range connectedSystems {
		body, err := json.Marshal(system)
		if err != nil {
			continue
		}
		if _, err = db.Exec(`INSERT INTO integration.resources(kind,id,tenant_id,body) VALUES('system',$1,'default',$2) ON CONFLICT(id) DO UPDATE SET body=EXCLUDED.body, updated_at=now()`, system.ID, body); err != nil {
			log.Printf("system registry write failed for %s: %v", system.ID, err)
		}
	}
}

func requireControlPlane() gin.HandlerFunc {
	return func(c *gin.Context) {
		key := env("STATGATE_INTERNAL_API_KEY", "")
		if key != "" && hmac.Equal([]byte(c.GetHeader("X-Internal-API-Key")), []byte(key)) {
			c.Next()
			return
		}
		// The Registry gateway validates bearer tokens before forwarding them. This service
		// deliberately accepts them only behind that gateway; direct deployments must use its key.
		if strings.HasPrefix(c.GetHeader("Authorization"), "Bearer ") && env("INTEGRATION_TRUSTED_GATEWAY", "false") == "true" {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication_required"})
	}
}
func requireStore() gin.HandlerFunc {
	return func(c *gin.Context) {
		if db == nil {
			c.AbortWithStatusJSON(http.StatusServiceUnavailable, gin.H{"error": "integration_store_unavailable", "message": "The Integration Hub requires PostgreSQL before control-plane changes can be made."})
			return
		}
		c.Next()
	}
}
func tenant(c *gin.Context) string {
	if value := c.GetHeader("X-Tenant-ID"); value != "" {
		return value
	}
	return "default"
}
func save(c *gin.Context, kind, resourceID string, body any) error {
	if db == nil {
		return nil
	}
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}
	_, err = db.Exec(`INSERT INTO integration.resources(kind,id,tenant_id,body) VALUES($1,$2,$3,$4) ON CONFLICT(id) DO UPDATE SET body=EXCLUDED.body, updated_at=now()`, kind, resourceID, tenant(c), data)
	return err
}
func list(c *gin.Context, kind string, destination any) error {
	if db == nil {
		return nil
	}
	rows, err := db.Query(`SELECT body FROM integration.resources WHERE kind=$1 AND tenant_id=$2 ORDER BY updated_at DESC`, kind, tenant(c))
	if err != nil {
		return err
	}
	defer rows.Close()
	raw := make([]json.RawMessage, 0)
	for rows.Next() {
		var value json.RawMessage
		if err = rows.Scan(&value); err != nil {
			return err
		}
		raw = append(raw, value)
	}
	data, _ := json.Marshal(raw)
	return json.Unmarshal(data, destination)
}
func remove(c *gin.Context, resourceID string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(`DELETE FROM integration.resources WHERE id=$1 AND tenant_id=$2`, resourceID, tenant(c))
	return err
}
func audit(c *gin.Context, action, resourceID string) {
	log.Printf("integration_audit action=%s resource=%s actor=%s tenant=%s", action, resourceID, actor(c), tenant(c))
}
func emit(c *gin.Context, eventType, source, subject string, payload map[string]any) {
	if db == nil {
		return
	}
	_, err := db.Exec(`INSERT INTO integration.events(id,tenant_id,event_type,source,subject,payload,occurred_at) VALUES($1,$2,$3,$4,$5,$6,$7)`, id("evt"), tenant(c), eventType, source, subject, payload, time.Now().UTC())
	if err != nil {
		log.Printf("event write failed: %v", err)
	}
	forwardToEnterpriseCore(c, eventType, source, subject, payload)
}

func forwardToEnterpriseCore(c *gin.Context, eventType, source, subject string, payload map[string]any) {
	endpoint := strings.TrimRight(env("ENTERPRISE_CORE_URL", "http://statgate-enterprise-core:8096"), "/") + "/api/events"
	event := map[string]any{"id": id("fabric_evt"), "event_type": eventType, "source": source, "object_type": "integration", "object_id": subject, "actor": actor(c), "tenant_id": tenant(c), "payload": payload, "timestamp": now()}
	body, err := json.Marshal(event)
	if err != nil {
		return
	}
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Internal-API-Key", env("STATGATE_INTERNAL_API_KEY", ""))
	req.Header.Set("X-Tenant-ID", tenant(c))
	client := &http.Client{Timeout: 3 * time.Second}
	response, err := client.Do(req)
	if err != nil {
		log.Printf("enterprise event bridge unavailable: %v", err)
		return
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		log.Printf("enterprise event bridge rejected %s: %s", eventType, response.Status)
	}
}

func systemsList(c *gin.Context) {
	client := &http.Client{Timeout: 2 * time.Second}
	items := make([]ConnectedSystem, 0, len(connectedSystems))
	for _, system := range connectedSystems {
		item := system
		response, err := client.Get(system.HealthURL)
		if err != nil {
			item.Status = "unreachable"
		} else {
			response.Body.Close()
			if response.StatusCode >= 200 && response.StatusCode < 300 {
				item.Status = "healthy"
			} else {
				item.Status = "degraded"
			}
		}
		items = append(items, item)
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func bind(c *gin.Context, target any) bool {
	if err := c.ShouldBindJSON(target); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_request", "message": err.Error()})
		return false
	}
	return true
}
func apiList(c *gin.Context) {
	records := []API{}
	if err := list(c, "api", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func apiCreate(c *gin.Context) {
	var value API
	if !bind(c, &value) {
		return
	}
	if value.Name == "" || value.BaseURL == "" {
		c.JSON(400, gin.H{"error": "name_and_base_url_required"})
		return
	}
	value.ID = id("api")
	value.Owner = actor(c)
	value.Status = "active"
	value.CreatedAt = now()
	if value.Protocol == "" {
		value.Protocol = "rest"
	}
	if err := save(c, "api", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "api.created", value.ID)
	c.JSON(201, value)
}
func apiUpdate(c *gin.Context) {
	var value API
	if !bind(c, &value) {
		return
	}
	value.ID = c.Param("id")
	value.Owner = actor(c)
	if err := save(c, "api", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "api.updated", value.ID)
	c.JSON(200, value)
}
func apiDelete(c *gin.Context) {
	if err := remove(c, c.Param("id")); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "api.deleted", c.Param("id"))
	c.Status(204)
}
func connectorList(c *gin.Context) {
	records := []Connector{}
	if err := list(c, "connector", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func connectorCreate(c *gin.Context) {
	var value Connector
	if !bind(c, &value) {
		return
	}
	if value.Name == "" || value.Type == "" {
		c.JSON(400, gin.H{"error": "name_and_type_required"})
		return
	}
	value.ID = id("conn")
	value.Owner = actor(c)
	value.Status = "draft"
	value.CreatedAt = now()
	if err := save(c, "connector", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "connector.created", value.ID)
	c.JSON(201, value)
}
func connectorStatus(c *gin.Context) {
	var request struct {
		Status string `json:"status"`
	}
	if !bind(c, &request) {
		return
	}
	if request.Status != "draft" && request.Status != "active" && request.Status != "disabled" {
		c.JSON(400, gin.H{"error": "invalid_status"})
		return
	}
	if db == nil {
		c.JSON(503, gin.H{"error": "connector_store_unavailable"})
		return
	}
	var raw []byte
	err := db.QueryRow(`SELECT body FROM integration.resources WHERE kind='connector' AND id=$1 AND tenant_id=$2`, c.Param("id"), tenant(c)).Scan(&raw)
	if err != nil {
		c.JSON(404, gin.H{"error": "connector_not_found"})
		return
	}
	var value Connector
	if json.Unmarshal(raw, &value) != nil {
		c.JSON(500, gin.H{"error": "connector_decode_failed"})
		return
	}
	value.Status = request.Status
	if err = save(c, "connector", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "connector.status_updated", value.ID)
	c.JSON(200, value)
}
func schemaList(c *gin.Context) {
	records := []Schema{}
	if err := list(c, "schema", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func schemaCreate(c *gin.Context) {
	var value Schema
	if !bind(c, &value) {
		return
	}
	if value.Name == "" || value.Format == "" {
		c.JSON(400, gin.H{"error": "name_and_format_required"})
		return
	}
	value.ID = id("schema")
	value.Owner = actor(c)
	value.CreatedAt = now()
	if err := save(c, "schema", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "schema.created", value.ID)
	c.JSON(201, value)
}
func webhookList(c *gin.Context) {
	records := []Webhook{}
	if err := list(c, "webhook", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	for i := range records {
		records[i].Secret = ""
	}
	c.JSON(200, gin.H{"items": records})
}
func webhookCreate(c *gin.Context) {
	var value Webhook
	if !bind(c, &value) {
		return
	}
	if value.Name == "" {
		c.JSON(400, gin.H{"error": "name_required"})
		return
	}
	value.ID = id("hook")
	value.Owner = actor(c)
	value.Status = "active"
	value.CreatedAt = now()
	secret := make([]byte, 32)
	_, _ = rand.Read(secret)
	value.Secret = hex.EncodeToString(secret)
	if err := save(c, "webhook", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "webhook.created", value.ID)
	c.JSON(201, gin.H{"webhook": value, "signing_secret": value.Secret})
}
func webhookDelete(c *gin.Context) {
	if err := remove(c, c.Param("id")); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "webhook.deleted", c.Param("id"))
	c.Status(204)
}
func webhookIngress(c *gin.Context) { // Lookup is deliberately DB-only so ingress never accidentally accepts an unconfigured endpoint.
	if db == nil {
		c.JSON(503, gin.H{"error": "webhook_store_unavailable"})
		return
	}
	var raw []byte
	err := db.QueryRow(`SELECT body FROM integration.resources WHERE kind='webhook' AND id=$1 AND tenant_id=$2`, c.Param("id"), tenant(c)).Scan(&raw)
	if err != nil {
		c.JSON(404, gin.H{"error": "webhook_not_found"})
		return
	}
	var hook Webhook
	if json.Unmarshal(raw, &hook) != nil || hook.Status != "active" {
		c.JSON(404, gin.H{"error": "webhook_not_found"})
		return
	}
	payload, err := io.ReadAll(io.LimitReader(c.Request.Body, 1024*1024))
	if err != nil {
		c.JSON(400, gin.H{"error": "payload_read_failed"})
		return
	}
	expected := hmac.New(sha256.New, []byte(hook.Secret))
	expected.Write(payload)
	supplied := strings.TrimPrefix(c.GetHeader("X-StatGate-Signature"), "sha256=")
	if !hmac.Equal([]byte(hex.EncodeToString(expected.Sum(nil))), []byte(supplied)) {
		c.JSON(401, gin.H{"error": "invalid_signature"})
		return
	}
	var event map[string]any
	if json.Unmarshal(payload, &event) != nil {
		event = map[string]any{"raw": string(payload)}
	}
	emit(c, "webhook.received", "webhook", hook.ID, event)
	c.JSON(202, gin.H{"accepted": true, "webhook_id": hook.ID})
}
func apiKeyList(c *gin.Context) {
	records := []APIKey{}
	if err := list(c, "api_key", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func apiKeyCreate(c *gin.Context) {
	var request struct {
		Name   string `json:"name"`
		Scopes string `json:"scopes"`
	}
	if !bind(c, &request) {
		return
	}
	if request.Name == "" {
		c.JSON(400, gin.H{"error": "name_required"})
		return
	}
	token := make([]byte, 32)
	_, _ = rand.Read(token)
	plaintext := "sgk_" + hex.EncodeToString(token)
	value := APIKey{ID: id("key"), Name: request.Name, Prefix: plaintext[:12], Scopes: request.Scopes, Status: "active", Owner: actor(c), CreatedAt: now()}
	if err := save(c, "api_key", value.ID, struct {
		APIKey
		Hash string `json:"hash"`
	}{value, fmt.Sprintf("%x", sha256.Sum256([]byte(plaintext)))}); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "api_key.created", value.ID)
	c.JSON(201, gin.H{"api_key": value, "token": plaintext, "notice": "Store this token now; it cannot be retrieved again."})
}
func apiKeyRevoke(c *gin.Context) {
	if db == nil {
		c.JSON(503, gin.H{"error": "api_key_store_unavailable"})
		return
	}
	var raw []byte
	err := db.QueryRow(`SELECT body FROM integration.resources WHERE kind='api_key' AND id=$1 AND tenant_id=$2`, c.Param("id"), tenant(c)).Scan(&raw)
	if err != nil {
		c.JSON(404, gin.H{"error": "api_key_not_found"})
		return
	}
	var stored struct {
		APIKey
		Hash string `json:"hash"`
	}
	if json.Unmarshal(raw, &stored) != nil {
		c.JSON(500, gin.H{"error": "api_key_decode_failed"})
		return
	}
	stored.Status = "revoked"
	if err = save(c, "api_key", stored.ID, stored); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "api_key.revoked", stored.ID)
	c.JSON(200, stored.APIKey)
}
func pluginList(c *gin.Context) {
	records := []Plugin{}
	if err := list(c, "plugin", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func pluginCreate(c *gin.Context) {
	var value Plugin
	if !bind(c, &value) {
		return
	}
	if value.Name == "" || value.Version == "" {
		c.JSON(400, gin.H{"error": "name_and_version_required"})
		return
	}
	value.ID, value.Publisher, value.Status, value.CreatedAt = id("plugin"), actor(c), "draft", now()
	if err := save(c, "plugin", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "plugin.submitted", value.ID)
	c.JSON(201, value)
}
func componentList(c *gin.Context) {
	records := []Component{}
	if err := list(c, "component", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func componentCreate(c *gin.Context) {
	var value Component
	if !bind(c, &value) {
		return
	}
	if value.Name == "" || value.Category == "" {
		c.JSON(400, gin.H{"error": "name_and_category_required"})
		return
	}
	value.ID, value.Owner, value.Status, value.CreatedAt = id("component"), actor(c), "active", now()
	if err := save(c, "component", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "component.registered", value.ID)
	c.JSON(201, value)
}
func appList(c *gin.Context) {
	records := []GeneratedApp{}
	if err := list(c, "generated_app", &records); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	c.JSON(200, gin.H{"items": records})
}
func appCreate(c *gin.Context) {
	var value GeneratedApp
	if !bind(c, &value) {
		return
	}
	if value.Name == "" || value.Slug == "" {
		c.JSON(400, gin.H{"error": "name_and_slug_required"})
		return
	}
	value.ID, value.Owner, value.Status, value.CreatedAt = id("app"), actor(c), "draft", now()
	if err := save(c, "generated_app", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	audit(c, "low_code_app.created", value.ID)
	c.JSON(201, value)
}
func appDeploy(c *gin.Context) {
	if db == nil {
		c.JSON(503, gin.H{"error": "app_store_unavailable"})
		return
	}
	var raw []byte
	if err := db.QueryRow(`SELECT body FROM integration.resources WHERE kind='generated_app' AND id=$1 AND tenant_id=$2`, c.Param("id"), tenant(c)).Scan(&raw); err != nil {
		c.JSON(404, gin.H{"error": "app_not_found"})
		return
	}
	var value GeneratedApp
	if json.Unmarshal(raw, &value) != nil {
		c.JSON(500, gin.H{"error": "app_decode_failed"})
		return
	}
	value.Status = "deployment_requested"
	if err := save(c, "generated_app", value.ID, value); err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	emit(c, "low_code.deployment_requested", "integration-fabric", value.ID, map[string]any{"slug": value.Slug})
	audit(c, "low_code_app.deployment_requested", value.ID)
	c.JSON(http.StatusAccepted, value)
}
func eventList(c *gin.Context) {
	if db == nil {
		c.JSON(200, gin.H{"items": []IntegrationEvent{}})
		return
	}
	rows, err := db.Query(`SELECT id,event_type,source,COALESCE(subject,''),tenant_id,payload,occurred_at::text FROM integration.events WHERE tenant_id=$1 ORDER BY occurred_at DESC LIMIT 100`, tenant(c))
	if err != nil {
		c.JSON(500, gin.H{"error": "storage_error"})
		return
	}
	defer rows.Close()
	items := []IntegrationEvent{}
	for rows.Next() {
		var item IntegrationEvent
		var payload []byte
		if err = rows.Scan(&item.ID, &item.Type, &item.Source, &item.Subject, &item.TenantID, &payload, &item.OccurredAt); err == nil {
			_ = json.Unmarshal(payload, &item.Payload)
			items = append(items, item)
		}
	}
	c.JSON(200, gin.H{"items": items})
}

func portal(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html><html><head><meta name="viewport" content="width=device-width,initial-scale=1"><title>StatGate Integration Fabric</title><style>body{font:16px system-ui;max-width:980px;margin:40px auto;color:#152238;background:#f8fafc;padding:0 20px}header{display:flex;justify-content:space-between;align-items:center}h1{margin-bottom:5px}section{background:white;border:1px solid #dbe3ee;border-radius:12px;padding:20px;margin:18px 0}input,button{padding:9px;border-radius:6px;border:1px solid #9aa9bd}button{background:#075985;color:white;cursor:pointer}li{padding:8px 0;border-bottom:1px solid #edf2f7}.muted{color:#64748b}</style></head><body><header><div><h1>Integration Fabric (EIP)</h1><p class="muted">Phases 16, 18 &amp; 42 · interoperability, ecosystem and low-code control plane</p></div><a href="/health">Service health</a></header><section><h2>Developer access</h2><p>Enter an approved internal API key to browse registered integrations. Keys are held only in this browser tab.</p><input id="key" type="password" placeholder="X-Internal-API-Key" size="38"><button onclick="load()">Connect</button><p id="status" class="muted"></p></section><section><h2>Registered APIs</h2><ul id="apis"><li class="muted">Connect to load the API registry.</li></ul></section><section><h2>Connectors</h2><ul id="connectors"><li class="muted">Connect to load configured connectors.</li></ul></section><script>const out=(id,items,empty)=>{const e=document.getElementById(id);e.replaceChildren();(items.length?items:[{name:empty,status:''}]).forEach(x=>{const li=document.createElement('li');li.textContent=x.name+(x.type?' · '+x.type:'')+(x.version?' · v'+x.version:'')+(x.status?' · '+x.status:'');e.append(li)})};async function get(path){let r=await fetch('/api/v1/'+path,{headers:{'X-Internal-API-Key':document.getElementById('key').value}});if(!r.ok)throw new Error((await r.json()).error||r.status);return r.json()}async function load(){try{let[a,b]=await Promise.all([get('apis'),get('connectors')]);out('apis',a.items,'No APIs are registered.');out('connectors',b.items,'No connectors are configured.');document.getElementById('status').textContent='Connected.'}catch(e){document.getElementById('status').textContent='Unable to load: '+e.message}}</script></body></html>`))
}
func main() {
	_ = godotenv.Load("../../.env")
	_ = godotenv.Load(".env")
	configureConnectedSystems()
	initDB()
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())
	router.GET("/", portal)
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "healthy", "service": "statgate-integration-hub", "version": "16.0.0"})
	})
	router.GET("/ready", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ready", "database": db != nil}) })
	control := router.Group("/api/v1", requireControlPlane(), requireStore(), statgatetenant.GinWorkspaceContext(), statgatetenant.GinWorkspaceMembership("", nil))
	control.GET("/apis", apiList)
	control.POST("/apis", apiCreate)
	control.PUT("/apis/:id", apiUpdate)
	control.DELETE("/apis/:id", apiDelete)
	control.GET("/connectors", connectorList)
	control.POST("/connectors", connectorCreate)
	control.PATCH("/connectors/:id/status", connectorStatus)
	control.GET("/schemas", schemaList)
	control.POST("/schemas", schemaCreate)
	control.GET("/webhooks", webhookList)
	control.POST("/webhooks", webhookCreate)
	control.DELETE("/webhooks/:id", webhookDelete)
	control.GET("/api-keys", apiKeyList)
	control.POST("/api-keys", apiKeyCreate)
	control.POST("/api-keys/:id/revoke", apiKeyRevoke)
	// Phase 18 — Marketplace and plugin submissions.
	control.GET("/marketplace/plugins", pluginList)
	control.POST("/marketplace/plugins", pluginCreate)
	// Phase 42 — Low-code component and generated-app registry.
	control.GET("/components", componentList)
	control.POST("/components", componentCreate)
	control.GET("/apps", appList)
	control.POST("/apps", appCreate)
	control.POST("/apps/:id/deploy", appDeploy)
	control.GET("/systems", systemsList)
	control.GET("/events", eventList)
	router.POST("/hooks/:id", webhookIngress)
	port := env("INTEGRATION_HUB_PORT", "8097")
	log.Printf("StatGate Integration Hub v16.0.0 on :%s", port)
	log.Fatal(router.Run(":" + port))
}
