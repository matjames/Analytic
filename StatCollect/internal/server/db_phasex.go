package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Phase Y Production DB Helpers — Integration Hub, Plugins, Business Rules,
// Approval Pipeline, Data Lineage, Notification Dispatcher, Template Rollback
// ─────────────────────────────────────────────────────────────────────────────

// ─── Structs ───

type Integration struct {
	ID           int64                  `json:"id"`
	Name         string                 `json:"name"`
	SystemType   string                 `json:"system_type"`
	BaseURL      string                 `json:"base_url"`
	Credentials  map[string]interface{} `json:"credentials"`
	FieldMapping map[string]interface{} `json:"field_mapping"`
	Status       string                 `json:"status"`
	LastPushAt   *time.Time             `json:"last_push_at,omitempty"`
	PushCount    int                    `json:"push_count"`
	TenantID     string                 `json:"tenant_id"`
	CreatedAt    time.Time              `json:"created_at"`
	UpdatedAt    time.Time              `json:"updated_at"`
}

type Plugin struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Version     string                 `json:"version"`
	Description string                 `json:"description,omitempty"`
	EntryPoint  string                 `json:"entry_point"`
	PluginType  string                 `json:"plugin_type"`
	Config      map[string]interface{} `json:"config"`
	Enabled     bool                   `json:"enabled"`
	CreatedAt   time.Time              `json:"created_at"`
}

type BusinessRule struct {
	ID            int64     `json:"id"`
	TemplateID    string    `json:"template_id,omitempty"`
	Name          string    `json:"name"`
	Description   string    `json:"description,omitempty"`
	RuleType      string    `json:"rule_type"`
	ConditionExpr string    `json:"condition_expr"`
	ActionExpr    string    `json:"action_expr"`
	Priority      int       `json:"priority"`
	Enabled       bool      `json:"enabled"`
	CreatedAt     time.Time `json:"created_at"`
}

// ─── Integration Hub DB Helpers ───

func SaveIntegration(ig Integration) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	creds, _ := json.Marshal(ig.Credentials)
	fm, _ := json.Marshal(ig.FieldMapping)
	_, err := dbPool.Exec(ctx, `INSERT INTO integrations (name, system_type, base_url, credentials, field_mapping, status, tenant_id, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7, now())
		ON CONFLICT DO NOTHING`,
		ig.Name, ig.SystemType, ig.BaseURL, creds, fm, ig.Status, ig.TenantID)
	return err
}

func ListIntegrations() ([]Integration, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, name, system_type, base_url, credentials, field_mapping, status, last_push_at, push_count, tenant_id, created_at, updated_at FROM integrations ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Integration
	for rows.Next() {
		var ig Integration
		var creds, fm []byte
		if err := rows.Scan(&ig.ID, &ig.Name, &ig.SystemType, &ig.BaseURL, &creds, &fm, &ig.Status, &ig.LastPushAt, &ig.PushCount, &ig.TenantID, &ig.CreatedAt, &ig.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(creds, &ig.Credentials)
		_ = json.Unmarshal(fm, &ig.FieldMapping)
		out = append(out, ig)
	}
	return out, nil
}

func DeleteIntegration(id int64) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `DELETE FROM integrations WHERE id=$1`, id)
	return err
}

// ─── Plugin Registry DB Helpers ───

func SavePlugin(p Plugin) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	config, _ := json.Marshal(p.Config)
	version := p.Version
	if version == "" {
		version = "1.0"
	}
	_, err := dbPool.Exec(ctx, `INSERT INTO plugins (id, name, version, description, entry_point, plugin_type, config, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name,
			version=EXCLUDED.version,
			description=EXCLUDED.description,
			entry_point=EXCLUDED.entry_point,
			config=EXCLUDED.config,
			enabled=EXCLUDED.enabled`,
		p.ID, p.Name, version, p.Description, p.EntryPoint, p.PluginType, config, p.Enabled)
	return err
}

func ListPlugins() ([]Plugin, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, name, version, COALESCE(description,''), entry_point, plugin_type, config, enabled, created_at FROM plugins ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Plugin
	for rows.Next() {
		var p Plugin
		var config []byte
		if err := rows.Scan(&p.ID, &p.Name, &p.Version, &p.Description, &p.EntryPoint, &p.PluginType, &config, &p.Enabled, &p.CreatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(config, &p.Config)
		out = append(out, p)
	}
	return out, nil
}

// ─── Business Rules Engine DB Helpers ───

func SaveBusinessRule(rule BusinessRule) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var templateID interface{}
	if rule.TemplateID != "" {
		templateID = rule.TemplateID
	}
	_, err := dbPool.Exec(ctx, `INSERT INTO business_rules (template_id, name, description, rule_type, condition_expr, action_expr, priority, enabled)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		templateID, rule.Name, rule.Description, rule.RuleType, rule.ConditionExpr, rule.ActionExpr, rule.Priority, rule.Enabled)
	return err
}

func ListBusinessRules(templateID string) ([]BusinessRule, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, COALESCE(template_id,''), name, COALESCE(description,''), rule_type, condition_expr, action_expr, priority, enabled, created_at FROM business_rules`
	var args []interface{}
	if templateID != "" {
		query += ` WHERE template_id=$1`
		args = append(args, templateID)
	}
	query += ` ORDER BY priority ASC, created_at ASC`
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []BusinessRule
	for rows.Next() {
		var rule BusinessRule
		if err := rows.Scan(&rule.ID, &rule.TemplateID, &rule.Name, &rule.Description, &rule.RuleType, &rule.ConditionExpr, &rule.ActionExpr, &rule.Priority, &rule.Enabled, &rule.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, rule)
	}
	return out, nil
}

func DeleteBusinessRule(id int64) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `DELETE FROM business_rules WHERE id=$1`, id)
	return err
}

// ─── Multi-Level Approval Pipeline DB Helpers ───

func AdvanceApprovalStage(instanceID, stage, approverRole, notes string) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO submission_validations (instance_id, validator, status, notes, validated_at, stage, approver_role, updated_at)
		VALUES ($1,$2,'approved',$3,now(),$4,$5,now())`,
		instanceID, approverRole, notes, stage, approverRole)
	if err != nil {
		return fmt.Errorf("advance approval stage: %w", err)
	}
	// Update parent submission status
	nextStatus := "approved"
	if stage != "national" && stage != "final" {
		nextStatus = "in_review"
	}
	_, _ = dbPool.Exec(ctx, `UPDATE submissions SET status=$2, updated_at=now() WHERE instance_id=$1`, instanceID, nextStatus)
	return nil
}

func RejectSubmissionAtStage(instanceID, stage, approverRole, reason string) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO submission_validations (instance_id, validator, status, notes, validated_at, stage, approver_role, updated_at)
		VALUES ($1,$2,'rejected',$3,now(),$4,$5,now())`,
		instanceID, approverRole, reason, stage, approverRole)
	if err != nil {
		return fmt.Errorf("reject at stage: %w", err)
	}
	_, _ = dbPool.Exec(ctx, `UPDATE submissions SET status='rejected', updated_at=now() WHERE instance_id=$1`, instanceID)
	return nil
}

func GetApprovalPipeline(formID string) ([]map[string]interface{}, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT v.instance_id, v.stage, v.status, v.validator, v.notes, v.validated_at, v.approver_role, s.form_id, s.submitted_by
		FROM submission_validations v
		JOIN submissions s ON v.instance_id = s.instance_id`
	var args []interface{}
	if formID != "" {
		query += ` WHERE s.form_id=$1`
		args = append(args, formID)
	}
	query += ` ORDER BY v.validated_at DESC LIMIT 200`
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var instanceID, stage, status, validator, notes, approverRole, fID, submittedBy string
		var validatedAt *time.Time
		if err := rows.Scan(&instanceID, &stage, &status, &validator, &notes, &validatedAt, &approverRole, &fID, &submittedBy); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"instance_id":   instanceID,
			"stage":         stage,
			"status":        status,
			"validator":     validator,
			"notes":         notes,
			"validated_at":  validatedAt,
			"approver_role": approverRole,
			"form_id":       fID,
			"submitted_by":  submittedBy,
		})
	}
	return out, nil
}

// ─── Data Lineage DB Helper ───

func GetSubmissionLineage(instanceID string) (map[string]interface{}, error) {
	if dbPool == nil {
		return map[string]interface{}{"instance_id": instanceID, "note": "database not connected"}, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	lineage := map[string]interface{}{"instance_id": instanceID}

	// Core submission record
	var formID, submittedBy, status string
	var receivedAt time.Time
	var meta []byte
	err := dbPool.QueryRow(ctx, `SELECT form_id, COALESCE(submitted_by,''), status, received_at, COALESCE(meta::text,'{}') FROM submissions WHERE instance_id=$1`, instanceID).Scan(&formID, &submittedBy, &status, &receivedAt, &meta)
	if err != nil {
		return nil, fmt.Errorf("submission not found: %w", err)
	}
	var metaMap map[string]interface{}
	_ = json.Unmarshal(meta, &metaMap)

	lineage["form_id"] = formID
	lineage["submitted_by"] = submittedBy
	lineage["status"] = status
	lineage["received_at"] = receivedAt
	lineage["metadata"] = metaMap

	// Device provenance
	if did, ok := metaMap["device_id"].(string); ok && did != "" {
		lineage["device_id"] = did
		lineage["operating_system"] = metaMap["operating_system"]
		lineage["browser"] = metaMap["browser"]
		lineage["online_status"] = metaMap["online_status"]
		lineage["duration_seconds"] = metaMap["duration_seconds"]
	}

	// Attachments
	rows, _ := dbPool.Query(ctx, `SELECT filename, size, content_type, created_at FROM attachments WHERE submission_instance_id=$1`, instanceID)
	attachments := []map[string]interface{}{}
	if rows != nil {
		defer rows.Close()
		for rows.Next() {
			var fn, ct string
			var sz int64
			var cat time.Time
			_ = rows.Scan(&fn, &sz, &ct, &cat)
			attachments = append(attachments, map[string]interface{}{"filename": fn, "size": sz, "content_type": ct, "created_at": cat})
		}
	}
	lineage["attachments"] = attachments

	// Approval history
	apRows, _ := dbPool.Query(ctx, `SELECT stage, status, validator, notes, validated_at, approver_role FROM submission_validations WHERE instance_id=$1 ORDER BY validated_at ASC`, instanceID)
	approvalHistory := []map[string]interface{}{}
	if apRows != nil {
		defer apRows.Close()
		for apRows.Next() {
			var stage, status2, validator, notes, approverRole string
			var valAt *time.Time
			_ = apRows.Scan(&stage, &status2, &validator, &notes, &valAt, &approverRole)
			approvalHistory = append(approvalHistory, map[string]interface{}{"stage": stage, "status": status2, "validator": validator, "notes": notes, "validated_at": valAt, "approver_role": approverRole})
		}
	}
	lineage["approval_history"] = approvalHistory

	// Longitudinal links
	llRows, _ := dbPool.Query(ctx, `SELECT registry_id, visit_number, phase, created_at FROM longitudinal_links WHERE submission_instance_id=$1`, instanceID)
	longLinks := []map[string]interface{}{}
	if llRows != nil {
		defer llRows.Close()
		for llRows.Next() {
			var regID, phase string
			var visitNum int
			var lat time.Time
			_ = llRows.Scan(&regID, &visitNum, &phase, &lat)
			longLinks = append(longLinks, map[string]interface{}{"registry_id": regID, "visit_number": visitNum, "phase": phase, "created_at": lat})
		}
	}
	lineage["longitudinal_links"] = longLinks

	// Event log (audit trail)
	evRows, _ := dbPool.Query(ctx, `SELECT event_type, source, payload, created_at FROM event_log WHERE object_type='submission' AND object_id=$1 ORDER BY created_at ASC LIMIT 50`, instanceID)
	events := []map[string]interface{}{}
	if evRows != nil {
		defer evRows.Close()
		for evRows.Next() {
			var evType, src string
			var payload []byte
			var evAt time.Time
			_ = evRows.Scan(&evType, &src, &payload, &evAt)
			var payloadMap interface{}
			_ = json.Unmarshal(payload, &payloadMap)
			events = append(events, map[string]interface{}{"event_type": evType, "source": src, "payload": payloadMap, "created_at": evAt})
		}
	}
	lineage["audit_trail"] = events

	// StatChat discussion link
	var convID string
	_ = dbPool.QueryRow(ctx, `SELECT conversation_id FROM statchat_links WHERE object_type='submission' AND object_id=$1`, instanceID).Scan(&convID)
	lineage["statchat_conversation_id"] = convID

	return lineage, nil
}

// ─── Template Rollback DB Helper ───

func RollbackTemplate(templateID, version, changedBy string) error {
	if dbPool == nil {
		return fmt.Errorf("database not connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Find the target version schema
	var schema []byte
	err := dbPool.QueryRow(ctx, `SELECT schema FROM template_versions WHERE template_id=$1 AND version=$2 ORDER BY created_at DESC LIMIT 1`, templateID, version).Scan(&schema)
	if err != nil {
		return fmt.Errorf("version %s not found for template %s: %w", version, templateID, err)
	}

	// Snapshot current state before rollback
	var currentVersion string
	var currentSchema []byte
	_ = dbPool.QueryRow(ctx, `SELECT version, schema FROM templates WHERE id=$1`, templateID).Scan(&currentVersion, &currentSchema)
	if currentVersion != "" && len(currentSchema) > 0 {
		_, _ = dbPool.Exec(ctx, `INSERT INTO template_versions (template_id, version, schema, changed_by, change_note) VALUES ($1,$2,$3,$4,$5)`,
			templateID, currentVersion+"-pre-rollback", currentSchema, changedBy, "Auto-snapshot before rollback to "+version)
	}

	// Apply the rollback
	_, err = dbPool.Exec(ctx, `UPDATE templates SET schema=$2, version=$3, updated_at=now() WHERE id=$1`, templateID, schema, version)
	if err != nil {
		return fmt.Errorf("rollback failed: %w", err)
	}

	// Record the rollback action in version history
	_, _ = dbPool.Exec(ctx, `INSERT INTO template_versions (template_id, version, schema, changed_by, change_note) VALUES ($1,$2,$3,$4,$5)`,
		templateID, version, schema, changedBy, "Rollback to version "+version)

	return nil
}

// ─── Notification Dispatcher DB Helpers ───

func ListPendingNotifications() ([]Notification, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, user_id, channel, message, status, sent_at, created_at FROM notifications WHERE status='Pending' ORDER BY created_at ASC LIMIT 50`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Notification
	for rows.Next() {
		var n Notification
		if err := rows.Scan(&n.ID, &n.UserID, &n.Channel, &n.Message, &n.Status, &n.SentAt, &n.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

func MarkNotificationSent(id int64, success bool, errMsg string) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status := "Sent"
	if !success {
		status = "Failed"
	}
	now := time.Now()
	_, err := dbPool.Exec(ctx, `UPDATE notifications SET status=$2, sent_at=$3 WHERE id=$1`, id, status, now)
	if err != nil {
		log.Printf("warning: could not mark notification %d as %s: %v (err: %s)", id, status, err, errMsg)
	}
	return err
}
