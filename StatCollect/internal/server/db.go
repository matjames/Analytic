package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

var dbPool *pgxpool.Pool

func InitDB(dsn string) error {
	if dsn == "" {
		return nil
	}
	// Retry up to 5 times with exponential back-off — the DB container
	// may not be ready immediately when StatCollect starts.
	var pool *pgxpool.Pool
	var lastErr error
	for attempt := 1; attempt <= 5; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		var err error
		pool, err = pgxpool.New(ctx, dsn)
		cancel()
		if err != nil {
			lastErr = err
			log.Printf("DB connect attempt %d/5 failed: %v — retrying in %ds", attempt, err, attempt*2)
			time.Sleep(time.Duration(attempt*2) * time.Second)
			continue
		}
		pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
		pingErr := pool.Ping(pingCtx)
		pingCancel()
		if pingErr != nil {
			pool.Close()
			lastErr = pingErr
			log.Printf("DB ping attempt %d/5 failed: %v — retrying in %ds", attempt, pingErr, attempt*2)
			time.Sleep(time.Duration(attempt*2) * time.Second)
			continue
		}
		lastErr = nil
		break
	}
	if lastErr != nil {
		return fmt.Errorf("database unavailable after 5 attempts: %w", lastErr)
	}
	dbPool = pool
	// run migrations if present
	if err := runMigrations(pool); err != nil {
		return fmt.Errorf("migrations: %w", err)
	}
	return nil
}

func CloseDB() {
	if dbPool != nil {
		dbPool.Close()
	}
}

// SaveSubmissionToDB persists a submission with tenant and status metadata.
// Kept for backward compatibility with callers that don't have GPS/QA data.
func SaveSubmissionToDB(instanceID, formID string, meta map[string]interface{}, xml string) error {
	return SaveSubmissionFullToDB(instanceID, formID, meta, xml, "", 0, 0, "", nil)
}

// SaveSubmissionFullToDB is the canonical save function. It persists GPS
// coordinates, the submitting user identity, and any QA flags in a single
// INSERT so the data is always consistent.
func SaveSubmissionFullToDB(
	instanceID, formID string,
	meta map[string]interface{},
	xmlDoc string,
	submittedBy string,
	gpsLat, gpsLng float64,
	gpsRaw string,
	qaFlags []map[string]string,
) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	// Marshal QA flags to JSON
	qaJSON, _ := json.Marshal(qaFlags)
	if qaJSON == nil {
		qaJSON = []byte("[]")
	}
	// GPS null handling: store NULL when coordinates are zero (not captured)
	var latPtr, lngPtr *float64
	if gpsLat != 0 || gpsLng != 0 {
		latPtr = &gpsLat
		lngPtr = &gpsLng
	}
	_, err := dbPool.Exec(ctx, `
		INSERT INTO submissions
		  (instance_id, form_id, received_at, meta, xml, tenant_id, status,
		   submitted_by, gps_lat, gps_lng, gps_raw, qa_flags)
		VALUES ($1,$2,$3,$4,$5,$6,'received',$7,$8,$9,$10,$11)
		ON CONFLICT (instance_id) DO UPDATE SET
		  form_id        = EXCLUDED.form_id,
		  received_at    = EXCLUDED.received_at,
		  meta           = EXCLUDED.meta,
		  submitted_by   = COALESCE(EXCLUDED.submitted_by, submissions.submitted_by),
		  gps_lat        = COALESCE(EXCLUDED.gps_lat, submissions.gps_lat),
		  gps_lng        = COALESCE(EXCLUDED.gps_lng, submissions.gps_lng),
		  gps_raw        = COALESCE(EXCLUDED.gps_raw, submissions.gps_raw),
		  qa_flags       = EXCLUDED.qa_flags,
		  updated_at     = now()`,
		instanceID, formID, time.Now(), meta, xmlDoc, tenantID,
		submittedBy, latPtr, lngPtr, gpsRaw, qaJSON)
	if err != nil {
		return fmt.Errorf("insert submission: %w", err)
	}
	return nil
}

// SaveAttachmentToDB persists an attachment record with inferred content type.
func SaveAttachmentToDB(instanceID, filename, path string, size int64) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	// Use filepath.Ext for safe extension extraction
	contentType := ""
	switch strings.ToLower(filepath.Ext(filename)) {
	case ".jpg", ".jpeg":
		contentType = "image/jpeg"
	case ".png":
		contentType = "image/png"
	case ".gif":
		contentType = "image/gif"
	case ".pdf":
		contentType = "application/pdf"
	case ".xml":
		contentType = "application/xml"
	case ".mp4":
		contentType = "video/mp4"
	case ".mov":
		contentType = "video/quicktime"
	case ".avi":
		contentType = "video/avi"
	case ".json":
		contentType = "application/json"
	case ".csv":
		contentType = "text/csv"
	}
	_, err := dbPool.Exec(ctx, `INSERT INTO attachments (submission_instance_id, filename, path, size, content_type, created_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		instanceID, filename, path, size, contentType, time.Now())
	if err != nil {
		return fmt.Errorf("insert attachment: %w", err)
	}
	return nil
}

// ListAttachments returns all attachment metadata records for a submission.
func ListAttachments(instanceID string) ([]map[string]interface{}, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `
		SELECT id, filename, path, size, COALESCE(content_type, ''), created_at
		FROM attachments WHERE submission_instance_id=$1
		ORDER BY created_at ASC`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var id, size int64
		var fn, path, ct string
		var cat time.Time
		if err := rows.Scan(&id, &fn, &path, &size, &ct, &cat); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"id":           id,
			"filename":     fn,
			"path":         path,
			"size":         size,
			"content_type": ct,
			"created_at":   cat,
		})
	}
	return out, nil
}

// DeleteSubmission deletes a submission and its associated cascade records.
func DeleteSubmission(instanceID string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `DELETE FROM submissions WHERE instance_id=$1`, instanceID)
	return err
}

func SubmissionExists(instanceID string) (bool, error) {
	if dbPool == nil {
		return false, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var cnt int
	err := dbPool.QueryRow(ctx, `SELECT COUNT(1) FROM submissions WHERE instance_id=$1`, instanceID).Scan(&cnt)
	if err != nil {
		return false, err
	}
	return cnt > 0, nil
}

type SubmissionSummary struct {
	InstanceID  string                 `json:"instance_id"`
	FormID      string                 `json:"form_id"`
	ReceivedAt  time.Time              `json:"received_at"`
	Meta        map[string]interface{} `json:"meta"`
	TenantID    string                 `json:"tenant_id"`
	Status      string                 `json:"status"`
	SubmittedBy string                 `json:"submitted_by,omitempty"`
	GPSLat      *float64               `json:"gps_lat,omitempty"`
	GPSLng      *float64               `json:"gps_lng,omitempty"`
	GPSRaw      string                 `json:"gps_raw,omitempty"`
	QAFlags     []map[string]string    `json:"qa_flags,omitempty"`
}

func ListSubmissions(limit, offset int) ([]SubmissionSummary, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `
		SELECT instance_id, form_id, received_at, meta, tenant_id, status,
		       COALESCE(submitted_by,''), gps_lat, gps_lng, COALESCE(gps_raw,''),
		       COALESCE(qa_flags::text, '[]')
		FROM submissions
		ORDER BY received_at DESC
		LIMIT $1 OFFSET $2`, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubmissionSummary
	for rows.Next() {
		var s SubmissionSummary
		var metaData map[string]interface{}
		var qaFlagsJSON string
		if err := rows.Scan(&s.InstanceID, &s.FormID, &s.ReceivedAt, &metaData,
			&s.TenantID, &s.Status, &s.SubmittedBy, &s.GPSLat, &s.GPSLng, &s.GPSRaw, &qaFlagsJSON); err != nil {
			return nil, err
		}
		s.Meta = metaData
		if qaFlagsJSON != "" && qaFlagsJSON != "null" {
			_ = json.Unmarshal([]byte(qaFlagsJSON), &s.QAFlags)
		}
		out = append(out, s)
	}
	return out, nil
}

func GetSubmission(instanceID string) (*SubmissionSummary, string, error) {
	if dbPool == nil {
		return nil, "", fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var s SubmissionSummary
	var xmlDoc string
	var metaData map[string]interface{}
	var qaFlagsJSON string
	err := dbPool.QueryRow(ctx, `
		SELECT instance_id, form_id, received_at, meta, xml, tenant_id, status,
		       COALESCE(submitted_by,''), gps_lat, gps_lng, COALESCE(gps_raw,''),
		       COALESCE(qa_flags::text, '[]')
		FROM submissions WHERE instance_id=$1`, instanceID).Scan(
		&s.InstanceID, &s.FormID, &s.ReceivedAt, &metaData, &xmlDoc,
		&s.TenantID, &s.Status, &s.SubmittedBy, &s.GPSLat, &s.GPSLng, &s.GPSRaw, &qaFlagsJSON)
	if err != nil {
		return nil, "", err
	}
	s.Meta = metaData
	if qaFlagsJSON != "" && qaFlagsJSON != "null" {
		_ = json.Unmarshal([]byte(qaFlagsJSON), &s.QAFlags)
	}
	return &s, xmlDoc, nil
}

// ── Object Linkage (StatGate Object Connectivity) ──

// CreateObjectLink links a submission to another StatGate platform object.
func CreateObjectLink(sourceType, sourceID, targetType, targetID, relationship string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	_, err := dbPool.Exec(ctx, `INSERT INTO object_links (source_type, source_id, target_type, target_id, relationship, tenant_id)
		VALUES ($1,$2,$3,$4,$5,$6)
		ON CONFLICT (source_type, source_id, target_type, target_id, relationship) DO NOTHING`,
		sourceType, sourceID, targetType, targetID, relationship, tenantID)
	if err != nil {
		return fmt.Errorf("create object link: %w", err)
	}
	return nil
}

// ListObjectLinks returns all links for a given object.
func ListObjectLinks(objectType, objectID string) ([]map[string]interface{}, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT source_type, source_id, target_type, target_id, relationship, created_at
		FROM object_links
		WHERE (source_type=$1 AND source_id=$2) OR (target_type=$1 AND target_id=$2)
		ORDER BY created_at DESC`, objectType, objectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var st, si, tt, ti, rel string
		var created time.Time
		if err := rows.Scan(&st, &si, &tt, &ti, &rel, &created); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"source_type":  st,
			"source_id":    si,
			"target_type":  tt,
			"target_id":    ti,
			"relationship": rel,
			"created_at":   created,
		})
	}
	return out, nil
}

// ── Event Log (Audit / Traceability) ──

// LogEvent records a cross-module event for audit and traceability.
func LogEvent(eventType, source, objectType, objectID string, payload interface{}) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	payloadB, _ := json.Marshal(payload)
	_, err := dbPool.Exec(ctx, `INSERT INTO event_log (event_type, source, object_type, object_id, tenant_id, payload)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		eventType, source, objectType, objectID, tenantID, payloadB)
	if err != nil {
		return fmt.Errorf("log event: %w", err)
	}
	return nil
}

// ListEvents returns recent events, optionally filtered by object.
func ListEvents(objectType, objectID string, limit int) ([]map[string]interface{}, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 50
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT event_type, source, object_type, object_id, tenant_id, payload, created_at
		FROM event_log`
	var args []interface{}
	if objectType != "" && objectID != "" {
		query += ` WHERE object_type=$1 AND object_id=$2`
		args = append(args, objectType, objectID)
	}
	query += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprint(len(args)+1)
	args = append(args, limit)

	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var et, src, ot, oid, tid string
		var payload []byte
		var created time.Time
		if err := rows.Scan(&et, &src, &ot, &oid, &tid, &payload, &created); err != nil {
			return nil, err
		}
		var payloadData interface{}
		_ = json.Unmarshal(payload, &payloadData)
		out = append(out, map[string]interface{}{
			"event_type":  et,
			"source":      src,
			"object_type": ot,
			"object_id":   oid,
			"tenant_id":   tid,
			"payload":     payloadData,
			"created_at":  created,
		})
	}
	return out, nil
}

// ── StatChat Object Discussion Links ──

// SaveStatChatLink records the StatChat conversation for a submission.
func SaveStatChatLink(objectType, objectID, conversationID string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	_, err := dbPool.Exec(ctx, `INSERT INTO statchat_links (object_type, object_id, conversation_id, tenant_id)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (object_type, object_id) DO UPDATE SET conversation_id=EXCLUDED.conversation_id`,
		objectType, objectID, conversationID, tenantID)
	if err != nil {
		return fmt.Errorf("save statchat link: %w", err)
	}
	return nil
}

// GetStatChatLink returns the StatChat conversation ID for an object.
func GetStatChatLink(objectType, objectID string) (string, error) {
	if dbPool == nil {
		return "", fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	var convID string
	err := dbPool.QueryRow(ctx, `SELECT conversation_id FROM statchat_links WHERE object_type=$1 AND object_id=$2`,
		objectType, objectID).Scan(&convID)
	if err != nil {
		return "", err
	}
	return convID, nil
}

// ── Submission Validation Workflow ──

// CreateValidation creates a validation record for a submission.
func CreateValidation(instanceID, validator, status, notes string) error {
	if dbPool == nil {
		return fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO submission_validations (instance_id, validator, status, notes, validated_at)
		VALUES ($1,$2,$3,$4, now())`,
		instanceID, validator, status, notes)
	if err != nil {
		return fmt.Errorf("create validation: %w", err)
	}
	// update submission status
	_, err = dbPool.Exec(ctx, `UPDATE submissions SET status=$2, updated_at=now() WHERE instance_id=$1`,
		instanceID, status)
	if err != nil {
		return fmt.Errorf("update submission status: %w", err)
	}
	return nil
}

// ListValidations returns validation records for a submission.
func ListValidations(instanceID string) ([]map[string]interface{}, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT validator, status, notes, validated_at, created_at
		FROM submission_validations WHERE instance_id=$1 ORDER BY created_at DESC`, instanceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var validator, status, notes string
		var validatedAt, createdAt time.Time
		if err := rows.Scan(&validator, &status, &notes, &validatedAt, &createdAt); err != nil {
			return nil, err
		}
		out = append(out, map[string]interface{}{
			"validator":    validator,
			"status":       status,
			"notes":        notes,
			"validated_at": validatedAt,
			"created_at":   createdAt,
		})
	}
	return out, nil
}

// GetSubmissionsByFormID returns all submissions for a given form/template ID.
func GetSubmissionsByFormID(formID string, limit int) ([]SubmissionSummary, error) {
	if dbPool == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	if limit <= 0 {
		limit = 1000
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT instance_id, form_id, received_at, meta, tenant_id, status,
		       COALESCE(submitted_by,''), gps_lat, gps_lng, COALESCE(gps_raw,''),
		       COALESCE(qa_flags::text, '[]')
		FROM submissions`
	var args []interface{}
	if formID != "" {
		query += ` WHERE form_id=$1`
		args = append(args, formID)
	}
	query += ` ORDER BY received_at DESC LIMIT $` + fmt.Sprint(len(args)+1)
	args = append(args, limit)

	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SubmissionSummary
	for rows.Next() {
		var s SubmissionSummary
		var metaData map[string]interface{}
		var qaFlagsJSON string
		if err := rows.Scan(&s.InstanceID, &s.FormID, &s.ReceivedAt, &metaData,
			&s.TenantID, &s.Status, &s.SubmittedBy, &s.GPSLat, &s.GPSLng, &s.GPSRaw, &qaFlagsJSON); err != nil {
			return nil, err
		}
		s.Meta = metaData
		if qaFlagsJSON != "" && qaFlagsJSON != "null" {
			_ = json.Unmarshal([]byte(qaFlagsJSON), &s.QAFlags)
		}
		out = append(out, s)
	}
	return out, nil
}

// ScanSubmissionsQA analyzes submissions for quality flags:
// - MISSING_GPS: submission has no GPS coordinates recorded
// - MISSING_FIELD_VIDEO: no video attachment was uploaded
// - POSSIBLE_DUPLICATE: identical payload signature to another submission
func ScanSubmissionsQA(formID string) ([]map[string]interface{}, error) {
	subs, err := GetSubmissionsByFormID(formID, 500)
	if err != nil {
		return nil, err
	}
	if subs == nil {
		return nil, fmt.Errorf("database not initialized")
	}
	var flags []map[string]interface{}
	seenHashes := make(map[string]string)

	for _, s := range subs {
		qaFlags := []string{}
		metaJSON, _ := json.Marshal(s.Meta)
		metaStr := string(metaJSON)

		// ── GPS CHECK ──
		hasGPS := (s.GPSLat != nil && s.GPSLng != nil) ||
			s.GPSRaw != "" ||
			strings.Contains(metaStr, "gps") ||
			strings.Contains(metaStr, "latitude") ||
			strings.Contains(metaStr, "location")
		if !hasGPS {
			qaFlags = append(qaFlags, "MISSING_GPS_COORDINATES")
		}

		// ── VIDEO EVIDENCE CHECK ──
		hasVideo := strings.Contains(metaStr, "field_video")
		if !hasVideo {
			qaFlags = append(qaFlags, "MISSING_FIELD_VIDEO_EVIDENCE")
		}

		// Also incorporate pre-computed QA flags
		for _, qf := range s.QAFlags {
			if fName, ok := qf["flag"]; ok && fName != "" {
				qaFlags = append(qaFlags, fName)
			}
		}

		// ── DUPLICATE DETECTION ──
		hash := fmt.Sprintf("%x", metaStr)
		if prevID, exists := seenHashes[hash]; exists && len(metaStr) > 20 {
			qaFlags = append(qaFlags, fmt.Sprintf("POSSIBLE_DUPLICATE_OF_%s", prevID))
		} else {
			seenHashes[hash] = s.InstanceID
		}

		if len(qaFlags) > 0 {
			flags = append(flags, map[string]interface{}{
				"instance_id":  s.InstanceID,
				"form_id":      s.FormID,
				"submitted_by": s.SubmittedBy,
				"received_at":  s.ReceivedAt,
				"status":       s.Status,
				"flags":        qaFlags,
			})
		}
	}
	return flags, nil
}

func runMigrations(pool *pgxpool.Pool) error {
	// Look for all .sql files in migrations/ in sorted order
	files, err := os.ReadDir("migrations")
	if err != nil {
		return nil
	}
	for _, fi := range files {
		if fi.IsDir() || !strings.HasSuffix(fi.Name(), ".sql") {
			continue
		}
		path := filepath.Join("migrations", fi.Name())
		b, err := os.ReadFile(path)
		if err != nil {
			log.Printf("warning: could not read migration %s: %v", path, err)
			continue
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		_, err = pool.Exec(ctx, string(b))
		cancel()
		if err != nil {
			log.Printf("migration %s execution warning: %v", fi.Name(), err)
		} else {
			log.Printf("applied migration: %s", fi.Name())
		}
	}
	return nil
}

// ── Phase Y Enterprise Structs & DB Helpers ──

type Device struct {
	DeviceID           string    `json:"device_id"`
	Name               string    `json:"name"`
	Owner              string    `json:"owner"`
	Status             string    `json:"status"` // Active, Suspended, Retired
	BatteryLevel       int       `json:"battery_level"`
	StorageUtilization float64   `json:"storage_utilization"`
	OS                 string    `json:"os"`
	AppVersion         string    `json:"app_version"`
	SecurityCompliant  bool      `json:"security_compliant"`
	LastSyncAt         time.Time `json:"last_sync_at"`
	RegisteredAt       time.Time `json:"registered_at"`
}

type Assignment struct {
	ID                  int64      `json:"id"`
	SurveyID            string     `json:"survey_id"`
	TargetType          string     `json:"target_type"`
	TargetID            string     `json:"target_id"`
	StartDate           *time.Time `json:"start_date,omitempty"`
	EndDate             *time.Time `json:"end_date,omitempty"`
	Deadline            *time.Time `json:"deadline,omitempty"`
	DailyTarget         int        `json:"daily_target"`
	CompletionThreshold float64    `json:"completion_threshold"`
	ReminderCron        string     `json:"reminder_cron,omitempty"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
}

type Registry struct {
	ID               string                 `json:"id"`
	RegistryType     string                 `json:"registry_type"`
	Name             string                 `json:"name"`
	Attributes       map[string]interface{} `json:"attributes"`
	ParentRegistryID string                 `json:"parent_registry_id,omitempty"`
	TenantID         string                 `json:"tenant_id"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

type LongitudinalLink struct {
	ID                   int64     `json:"id"`
	SubmissionInstanceID string    `json:"submission_instance_id"`
	RegistryID           string    `json:"registry_id"`
	VisitNumber          int       `json:"visit_number"`
	Phase                string    `json:"phase"`
	CreatedAt            time.Time `json:"created_at"`
}

type SamplingDesign struct {
	ID          int64           `json:"id"`
	Name        string          `json:"name"`
	Method      string          `json:"method"`
	SampleSize  int             `json:"sample_size"`
	Seed        int64           `json:"seed"`
	FrameData   json.RawMessage `json:"frame_data"`
	SelectedIDs json.RawMessage `json:"selected_ids"`
	AuditTrail  string          `json:"audit_trail,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type Workflow struct {
	ID           int64           `json:"id"`
	SurveyID     string          `json:"survey_id"`
	Stages       json.RawMessage `json:"stages"`
	CurrentStage string          `json:"current_stage"`
	CreatedAt    time.Time       `json:"created_at"`
}

type Schedule struct {
	ID             int64      `json:"id"`
	SurveyID       string     `json:"survey_id"`
	CronExpression string     `json:"cron_expression"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
}

type Comment struct {
	ID         int64     `json:"id"`
	ObjectType string    `json:"object_type"`
	ObjectID   string    `json:"object_id"`
	Author     string    `json:"author"`
	Comment    string    `json:"comment"`
	CreatedAt  time.Time `json:"created_at"`
}

type Notification struct {
	ID        int64      `json:"id"`
	UserID    string     `json:"user_id"`
	Channel   string     `json:"channel"`
	Message   string     `json:"message"`
	Status    string     `json:"status"`
	SentAt    *time.Time `json:"sent_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// ── DB Accessors ──

// Devices queries
func SaveDevice(d Device) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO devices (device_id, name, owner, status, battery_level, storage_utilization, os, app_version, security_compliant, last_sync_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		ON CONFLICT (device_id) DO UPDATE SET
			name=EXCLUDED.name,
			owner=EXCLUDED.owner,
			status=EXCLUDED.status,
			battery_level=EXCLUDED.battery_level,
			storage_utilization=EXCLUDED.storage_utilization,
			os=EXCLUDED.os,
			app_version=EXCLUDED.app_version,
			security_compliant=EXCLUDED.security_compliant,
			last_sync_at=EXCLUDED.last_sync_at`,
		d.DeviceID, d.Name, d.Owner, d.Status, d.BatteryLevel, d.StorageUtilization, d.OS, d.AppVersion, d.SecurityCompliant, d.LastSyncAt)
	return err
}

func ListDevices() ([]Device, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT device_id, name, COALESCE(owner,''), status, battery_level, storage_utilization, COALESCE(os,''), COALESCE(app_version,''), security_compliant, COALESCE(last_sync_at, now()), registered_at FROM devices ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Device
	for rows.Next() {
		var d Device
		if err := rows.Scan(&d.DeviceID, &d.Name, &d.Owner, &d.Status, &d.BatteryLevel, &d.StorageUtilization, &d.OS, &d.AppVersion, &d.SecurityCompliant, &d.LastSyncAt, &d.RegisteredAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, nil
}

func DeleteDevice(deviceID string) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `DELETE FROM devices WHERE device_id=$1`, deviceID)
	return err
}

// Assignments queries
func SaveAssignment(a Assignment) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO assignments (survey_id, target_type, target_id, start_date, end_date, deadline, daily_target, completion_threshold, reminder_cron, status)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		a.SurveyID, a.TargetType, a.TargetID, a.StartDate, a.EndDate, a.Deadline, a.DailyTarget, a.CompletionThreshold, a.ReminderCron, a.Status)
	return err
}

func ListAssignments(surveyID string) ([]Assignment, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, survey_id, target_type, target_id, start_date, end_date, deadline, daily_target, completion_threshold, COALESCE(reminder_cron,''), status, created_at FROM assignments`
	var args []interface{}
	if surveyID != "" {
		query += ` WHERE survey_id=$1`
		args = append(args, surveyID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Assignment
	for rows.Next() {
		var a Assignment
		if err := rows.Scan(&a.ID, &a.SurveyID, &a.TargetType, &a.TargetID, &a.StartDate, &a.EndDate, &a.Deadline, &a.DailyTarget, &a.CompletionThreshold, &a.ReminderCron, &a.Status, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, nil
}

func DeleteAssignment(id int64) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `DELETE FROM assignments WHERE id=$1`, id)
	return err
}

// Registries queries
func SaveRegistry(r Registry) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	attribs, _ := json.Marshal(r.Attributes)
	var parentID interface{}
	if r.ParentRegistryID != "" {
		parentID = r.ParentRegistryID
	}
	tenantID := r.TenantID
	if tenantID == "" {
		tenantID = "default"
	}
	_, err := dbPool.Exec(ctx, `INSERT INTO registries (id, registry_type, name, attributes, parent_registry_id, tenant_id, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6, now())
		ON CONFLICT (id) DO UPDATE SET
			name=EXCLUDED.name,
			attributes=EXCLUDED.attributes,
			parent_registry_id=EXCLUDED.parent_registry_id,
			updated_at=now()`,
		r.ID, r.RegistryType, r.Name, attribs, parentID, tenantID)
	return err
}

func ListRegistries(regType string) ([]Registry, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, registry_type, name, attributes, COALESCE(parent_registry_id, ''), tenant_id, created_at, updated_at FROM registries`
	var args []interface{}
	if regType != "" {
		query += ` WHERE registry_type=$1`
		args = append(args, regType)
	}
	query += ` ORDER BY name`
	rows, err := dbPool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Registry
	for rows.Next() {
		var r Registry
		var attribs []byte
		if err := rows.Scan(&r.ID, &r.RegistryType, &r.Name, &attribs, &r.ParentRegistryID, &r.TenantID, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, err
		}
		_ = json.Unmarshal(attribs, &r.Attributes)
		out = append(out, r)
	}
	return out, nil
}

func GetRegistryLongitudinalHistory(registryID string) ([]map[string]interface{}, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT l.visit_number, l.phase, l.created_at, s.instance_id, s.form_id, s.submitted_by, s.status, s.meta
		FROM longitudinal_links l
		JOIN submissions s ON l.submission_instance_id = s.instance_id
		WHERE l.registry_id = $1
		ORDER BY l.visit_number ASC`, registryID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []map[string]interface{}
	for rows.Next() {
		var visitNum int
		var phase, instanceID, formID, submittedBy, status string
		var createdAt time.Time
		var meta []byte
		if err := rows.Scan(&visitNum, &phase, &createdAt, &instanceID, &formID, &submittedBy, &status, &meta); err != nil {
			return nil, err
		}
		var metaData map[string]interface{}
		_ = json.Unmarshal(meta, &metaData)
		out = append(out, map[string]interface{}{
			"visit_number": visitNum,
			"phase":        phase,
			"created_at":   createdAt,
			"instance_id":  instanceID,
			"form_id":      formID,
			"submitted_by": submittedBy,
			"status":       status,
			"meta":         metaData,
		})
	}
	return out, nil
}

func SaveLongitudinalLink(l LongitudinalLink) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO longitudinal_links (submission_instance_id, registry_id, visit_number, phase)
		VALUES ($1,$2,$3,$4)`,
		l.SubmissionInstanceID, l.RegistryID, l.VisitNumber, l.Phase)
	return err
}

// Sampling queries
func SaveSamplingDesign(s SamplingDesign) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO sampling_designs (name, method, sample_size, seed, frame_data, selected_ids, audit_trail)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		s.Name, s.Method, s.SampleSize, s.Seed, s.FrameData, s.SelectedIDs, s.AuditTrail)
	return err
}

func ListSamplingDesigns() ([]SamplingDesign, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, name, method, sample_size, seed, frame_data, selected_ids, COALESCE(audit_trail,''), created_at FROM sampling_designs ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []SamplingDesign
	for rows.Next() {
		var s SamplingDesign
		if err := rows.Scan(&s.ID, &s.Name, &s.Method, &s.SampleSize, &s.Seed, &s.FrameData, &s.SelectedIDs, &s.AuditTrail, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// Workflows queries
func SaveWorkflow(w Workflow) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO survey_workflows (survey_id, stages, current_stage)
		VALUES ($1,$2,$3)
		ON CONFLICT (survey_id) DO UPDATE SET
			stages=EXCLUDED.stages,
			current_stage=EXCLUDED.current_stage`,
		w.SurveyID, w.Stages, w.CurrentStage)
	return err
}

func GetWorkflow(surveyID string) (*Workflow, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var w Workflow
	err := dbPool.QueryRow(ctx, `SELECT id, survey_id, stages, current_stage, created_at FROM survey_workflows WHERE survey_id=$1`, surveyID).Scan(&w.ID, &w.SurveyID, &w.Stages, &w.CurrentStage, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func UpdateWorkflowStage(surveyID string, stage string) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `UPDATE survey_workflows SET current_stage=$2 WHERE survey_id=$1`, surveyID, stage)
	return err
}

// Schedules queries
func SaveSchedule(s Schedule) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO survey_schedules (survey_id, cron_expression, next_run_at, status)
		VALUES ($1,$2,$3,$4)`,
		s.SurveyID, s.CronExpression, s.NextRunAt, s.Status)
	return err
}

func ListSchedules() ([]Schedule, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, survey_id, cron_expression, next_run_at, status, created_at FROM survey_schedules ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Schedule
	for rows.Next() {
		var s Schedule
		if err := rows.Scan(&s.ID, &s.SurveyID, &s.CronExpression, &s.NextRunAt, &s.Status, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, nil
}

// Comments queries
func SaveComment(c Comment) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO survey_comments (object_type, object_id, author, comment)
		VALUES ($1,$2,$3,$4)`,
		c.ObjectType, c.ObjectID, c.Author, c.Comment)
	return err
}

func ListComments(objType, objID string) ([]Comment, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	rows, err := dbPool.Query(ctx, `SELECT id, object_type, object_id, author, comment, created_at FROM survey_comments WHERE object_type=$1 AND object_id=$2 ORDER BY created_at ASC`, objType, objID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Comment
	for rows.Next() {
		var c Comment
		if err := rows.Scan(&c.ID, &c.ObjectType, &c.ObjectID, &c.Author, &c.Comment, &c.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, nil
}

// Notifications queries
func SaveNotification(n Notification) error {
	if dbPool == nil {
		return nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := dbPool.Exec(ctx, `INSERT INTO notifications (user_id, channel, message, status, sent_at)
		VALUES ($1,$2,$3,$4,$5)`,
		n.UserID, n.Channel, n.Message, n.Status, n.SentAt)
	return err
}

func ListNotifications(userID string) ([]Notification, error) {
	if dbPool == nil {
		return nil, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	query := `SELECT id, user_id, channel, message, status, sent_at, created_at FROM notifications`
	var args []interface{}
	if userID != "" {
		query += ` WHERE user_id=$1`
		args = append(args, userID)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := dbPool.Query(ctx, query, args...)
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
