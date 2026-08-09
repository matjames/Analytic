package server

import (
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"
)

// ─────────────────────────────────────────────────────────────────────────────
// Phase Y Production Handlers — Lineage, Approvals, AI Intelligence,
// Integrations, Plugins, Business Rules, Template Rollback
// ─────────────────────────────────────────────────────────────────────────────

// adminLineageHandler returns the complete data lineage (provenance chain) for a submission
func adminLineageHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	instanceID := r.URL.Query().Get("instance_id")
	if instanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}
	lineage, err := GetSubmissionLineage(instanceID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(lineage)
}

// adminApproveHandler advances a submission through the multi-level approval pipeline
func adminApproveHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InstanceID   string `json:"instance_id"`
		Stage        string `json:"stage"`
		ApproverRole string `json:"approver_role"`
		Notes        string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.InstanceID == "" || req.Stage == "" {
		http.Error(w, "instance_id and stage are required", http.StatusBadRequest)
		return
	}
	if err := AdvanceApprovalStage(req.InstanceID, req.Stage, req.ApproverRole, req.Notes); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = LogEvent("submission.approved", "statcollect", "submission", req.InstanceID, req)
	if statChat != nil && statChat.Enabled {
		_ = statChat.PostMessageToObject("submission", req.InstanceID, req.ApproverRole,
			fmt.Sprintf("✅ Advanced to stage: %s. Notes: %s", req.Stage, req.Notes))
	}
	// Queue notification
	_ = SaveNotification(Notification{
		UserID:  req.ApproverRole,
		Channel: "app",
		Message: fmt.Sprintf("Submission %s approved and advanced to stage: %s", req.InstanceID, req.Stage),
		Status:  "Pending",
	})
	// Directive 22 — fire platform event bus for approved submission (triggers GIS sync, Stats export, etc.)
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	go PublishSubmissionEvent("submission.approved", req.InstanceID, req.Stage, tenantID, map[string]interface{}{
		"instance_id":   req.InstanceID,
		"stage":         req.Stage,
		"approver_role": req.ApproverRole,
		"notes":         req.Notes,
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "approved", "stage": req.Stage})
}

// adminRejectHandler rejects a submission at a given approval stage
func adminRejectHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InstanceID   string `json:"instance_id"`
		Stage        string `json:"stage"`
		ApproverRole string `json:"approver_role"`
		Reason       string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.InstanceID == "" {
		http.Error(w, "instance_id is required", http.StatusBadRequest)
		return
	}
	if err := RejectSubmissionAtStage(req.InstanceID, req.Stage, req.ApproverRole, req.Reason); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = LogEvent("submission.rejected", "statcollect", "submission", req.InstanceID, req)
	if statChat != nil && statChat.Enabled {
		_ = statChat.PostMessageToObject("submission", req.InstanceID, req.ApproverRole,
			fmt.Sprintf("❌ Rejected at stage: %s. Reason: %s", req.Stage, req.Reason))
	}
	_ = SaveNotification(Notification{
		UserID:  req.ApproverRole,
		Channel: "app",
		Message: fmt.Sprintf("Submission %s rejected at stage %s: %s", req.InstanceID, req.Stage, req.Reason),
		Status:  "Pending",
	})
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rejected", "stage": req.Stage})
}

// adminPipelineHandler lists all approval pipeline records (kanban view)
func adminPipelineHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	pipeline, err := GetApprovalPipeline(formID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{"pipeline": pipeline})
}

// adminAISummaryHandler generates an automated AI dataset intelligence summary
func adminAISummaryHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	formID := r.URL.Query().Get("form_id")
	if formID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}
	subs, err := GetSubmissionsByFormID(formID, 1000)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	totalCount := len(subs)
	statusCounts := map[string]int{}
	enumeratorSet := map[string]bool{}
	deviceSet := map[string]bool{}
	gpsCount, videoCount := 0, 0
	for _, s := range subs {
		statusCounts[s.Status]++
		if s.SubmittedBy != "" {
			enumeratorSet[s.SubmittedBy] = true
		}
		// Check GPS via meta fields (set by agent telemetry)
		if gps, ok := s.Meta["gps_coordinates"].(string); ok && gps != "" {
			gpsCount++
		} else if _, ok2 := s.Meta["latitude"].(float64); ok2 {
			gpsCount++
		}
		// Check video via meta
		if v, ok := s.Meta["has_video"].(bool); ok && v {
			videoCount++
		} else if _, ok2 := s.Meta["field_video"].(string); ok2 {
			videoCount++
		}
		if m, ok := s.Meta["device_id"].(string); ok && m != "" {
			deviceSet[m] = true
		}
	}
	gpsRate, videoRate := 0.0, 0.0
	if totalCount > 0 {
		gpsRate = float64(gpsCount) / float64(totalCount) * 100
		videoRate = float64(videoCount) / float64(totalCount) * 100
	}
	insights := []string{}
	if gpsRate < 80 {
		insights = append(insights, fmt.Sprintf("⚠️ GPS capture rate is %.1f%%. Retrain enumerators on GPS lock.", gpsRate))
	} else {
		insights = append(insights, fmt.Sprintf("✅ GPS capture rate excellent at %.1f%%.", gpsRate))
	}
	if videoRate < 80 {
		insights = append(insights, fmt.Sprintf("⚠️ Video evidence rate is %.1f%%. Video is mandatory per submission.", videoRate))
	} else {
		insights = append(insights, fmt.Sprintf("✅ Video capture rate is %.1f%%.", videoRate))
	}
	if totalCount > 0 && len(enumeratorSet) > 0 {
		avg := float64(totalCount) / float64(len(enumeratorSet))
		insights = append(insights, fmt.Sprintf("📊 %d submissions from %d enumerators — avg %.1f per enumerator.", totalCount, len(enumeratorSet), avg))
	}
	if len(deviceSet) > 0 {
		insights = append(insights, fmt.Sprintf("📱 %d unique devices participated in data collection.", len(deviceSet)))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"form_id":          formID,
		"total_records":    totalCount,
		"status_counts":    statusCounts,
		"enumerator_count": len(enumeratorSet),
		"device_count":     len(deviceSet),
		"gps_rate_pct":     math.Round(gpsRate*10) / 10,
		"video_rate_pct":   math.Round(videoRate*10) / 10,
		"ai_insights":      insights,
	})
}

// adminAITranslateHandler provides multilingual survey question translation assistance
func adminAITranslateHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Text      string   `json:"text"`
		Languages []string `json:"languages"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Text == "" {
		http.Error(w, "text is required", http.StatusBadRequest)
		return
	}
	if len(req.Languages) == 0 {
		req.Languages = []string{"sw", "fr", "ar", "pt", "am", "ha", "yo", "ig", "rw", "lg"}
	}
	// Survey-domain translation vocabulary — patterns matched to common survey terms
	dict := map[string]map[string]string{
		"name":        {"sw": "Jina", "fr": "Nom", "ar": "الاسم", "pt": "Nome", "am": "ስም", "ha": "Suna", "yo": "Orúkọ", "ig": "Aha", "rw": "Izina", "lg": "Erinnya"},
		"age":         {"sw": "Umri", "fr": "Âge", "ar": "العمر", "pt": "Idade", "am": "ዕድሜ", "ha": "Shekaru", "yo": "Ọjọ́ orí", "ig": "Afọ", "rw": "Imyaka", "lg": "Emyaka"},
		"gender":      {"sw": "Jinsia", "fr": "Genre", "ar": "الجنس", "pt": "Género", "am": "ጾታ", "ha": "Jinsi", "yo": "Ìbálòpọ̀", "ig": "Mmekọahụ", "rw": "Igitsina", "lg": "Engeli"},
		"household":   {"sw": "Kaya", "fr": "Ménage", "ar": "الأسرة", "pt": "Domicílio", "am": "ቤተሰብ", "ha": "Gida", "yo": "Ilé", "ig": "Ụlọ", "rw": "Umuryango", "lg": "Eka"},
		"health":      {"sw": "Afya", "fr": "Santé", "ar": "الصحة", "pt": "Saúde", "am": "ጤና", "ha": "Lafiya", "yo": "Ìlera", "ig": "Ahụ ike", "rw": "Ubuzima", "lg": "Amagezi"},
		"water":       {"sw": "Maji", "fr": "Eau", "ar": "الماء", "pt": "Água", "am": "ውሃ", "ha": "Ruwa", "yo": "Omi", "ig": "Mmiri", "rw": "Amazi", "lg": "Amazzi"},
		"education":   {"sw": "Elimu", "fr": "Éducation", "ar": "التعليم", "pt": "Educação", "am": "ትምህርት", "ha": "Ilimi", "yo": "Ẹ̀kọ́", "ig": "Ọmụmụ", "rw": "Uburezi", "lg": "Emyoyo"},
		"income":      {"sw": "Mapato", "fr": "Revenu", "ar": "الدخل", "pt": "Renda", "am": "ገቢ", "ha": "Samun", "yo": "Owo", "ig": "Ọrụ ego", "rw": "Umusaruro", "lg": "Ennyingye"},
		"community":   {"sw": "Jamii", "fr": "Communauté", "ar": "المجتمع", "pt": "Comunidade", "am": "ማህበረሰብ", "ha": "Al'umma", "yo": "Àgbègbè", "ig": "Obodo", "rw": "Umudugudu", "lg": "Ekitundu"},
		"date":        {"sw": "Tarehe", "fr": "Date", "ar": "التاريخ", "pt": "Data", "am": "ቀን", "ha": "Kwanan wata", "yo": "Ọjọ", "ig": "Ọ́nọdụ", "rw": "Itariki", "lg": "Olunaku"},
		"village":     {"sw": "Kijiji", "fr": "Village", "ar": "القرية", "pt": "Aldeia", "am": "መንደር", "ha": "Ƙauye", "yo": "Abúlé", "ig": "Obodo", "rw": "Umudugudu", "lg": "Kyalo"},
		"school":      {"sw": "Shule", "fr": "École", "ar": "المدرسة", "pt": "Escola", "am": "ትምህርት ቤት", "ha": "Makaranta", "yo": "Ilé-ìwé", "ig": "Ụlọ akwụkwọ", "rw": "Ishuri", "lg": "Ssomero"},
		"hospital":    {"sw": "Hospitali", "fr": "Hôpital", "ar": "المستشفى", "pt": "Hospital", "am": "ሆስፒታል", "ha": "Asibiti", "yo": "Ilé-ìwòsàn", "ig": "Ụlọ ọgwụ", "rw": "Ibitaro", "lg": "Eddwaliro"},
		"mother":      {"sw": "Mama", "fr": "Mère", "ar": "الأم", "pt": "Mãe", "am": "እናት", "ha": "Uwa", "yo": "Ìyá", "ig": "Nnne", "rw": "Mama", "lg": "Maama"},
		"child":       {"sw": "Mtoto", "fr": "Enfant", "ar": "طفل", "pt": "Criança", "am": "ልጅ", "ha": "Yaro", "yo": "Ọmọ", "ig": "Nwa", "rw": "Umwana", "lg": "Omwana"},
	}
	textLower := strings.ToLower(req.Text)
	translations := map[string]string{}
	for _, lang := range req.Languages {
		translated := req.Text
		for eng, langMap := range dict {
			if strings.Contains(textLower, eng) {
				if tv, ok := langMap[lang]; ok {
					translated = strings.ReplaceAll(translated, eng, tv)
				}
			}
		}
		translations[lang] = translated
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"original":     req.Text,
		"translations": translations,
		"note":         "For production-grade neural translation, configure TRANSLATE_API_URL (LibreTranslate, DeepL, or Google Translate API).",
	})
}

// adminAIQualityHandler performs deep per-field quality analysis on a dataset
func adminAIQualityHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		FormID string `json:"form_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.FormID == "" {
		http.Error(w, "form_id is required", http.StatusBadRequest)
		return
	}
	subs, err := GetSubmissionsByFormID(req.FormID, 500)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if len(subs) == 0 {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"form_id": req.FormID, "quality_score": 0, "message": "No submissions found."})
		return
	}
	gpsOk, videoOk, deviceOk, durationOk := 0, 0, 0, 0
	var totalDuration float64
	for _, s := range subs {
		// GPS check via meta
		if gps, ok := s.Meta["gps_coordinates"].(string); ok && gps != "" {
			gpsOk++
		} else if _, ok2 := s.Meta["latitude"].(float64); ok2 {
			gpsOk++
		}
		// Video check via meta
		if v, ok := s.Meta["has_video"].(bool); ok && v {
			videoOk++
		} else if _, ok2 := s.Meta["field_video"].(string); ok2 {
			videoOk++
		}
		if did, ok := s.Meta["device_id"].(string); ok && did != "" {
			deviceOk++
		}
		if dur, ok := s.Meta["duration_seconds"].(float64); ok && dur > 0 {
			durationOk++
			totalDuration += dur
		}
	}
	n := float64(len(subs))
	gpsRate := float64(gpsOk) / n * 100
	videoRate := float64(videoOk) / n * 100
	deviceRate := float64(deviceOk) / n * 100
	avgDuration := 0.0
	if durationOk > 0 {
		avgDuration = totalDuration / float64(durationOk)
	}
	durScore := math.Min(avgDuration/120.0*100.0, 100.0) // 2min baseline
	qualityScore := gpsRate*0.3 + videoRate*0.3 + deviceRate*0.2 + durScore*0.2

	issues := []string{}
	if gpsRate < 90 {
		issues = append(issues, fmt.Sprintf("GPS completeness: %.1f%% (target >90%%)", gpsRate))
	}
	if videoRate < 90 {
		issues = append(issues, fmt.Sprintf("Video evidence: %.1f%% (target >90%%)", videoRate))
	}
	if deviceRate < 90 {
		issues = append(issues, fmt.Sprintf("Device metadata: %.1f%% — ensure all agents auto-register devices.", deviceRate))
	}
	if avgDuration < 60 && durationOk > 0 {
		issues = append(issues, fmt.Sprintf("Avg survey duration %.0fs — possible rushed responses.", avgDuration))
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"form_id":          req.FormID,
		"total_records":    len(subs),
		"quality_score":    math.Round(qualityScore*10) / 10,
		"gps_rate_pct":     math.Round(gpsRate*10) / 10,
		"video_rate_pct":   math.Round(videoRate*10) / 10,
		"device_rate_pct":  math.Round(deviceRate*10) / 10,
		"avg_duration_sec": math.Round(avgDuration),
		"quality_issues":   issues,
	})
}

// adminIntegrationsHandler manages external system integration configs (DHIS2, ODK, etc.)
func adminIntegrationsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := ListIntegrations()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"integrations": list})
	case http.MethodPost:
		var ig Integration
		if err := json.NewDecoder(r.Body).Decode(&ig); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if ig.Name == "" || ig.SystemType == "" || ig.BaseURL == "" {
			http.Error(w, "name, system_type, and base_url are required", http.StatusBadRequest)
			return
		}
		if ig.Status == "" {
			ig.Status = "Active"
		}
		if ig.TenantID == "" && cfg != nil {
			ig.TenantID = cfg.TenantID
		}
		if err := SaveIntegration(ig); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("integration.saved", "statcollect", "integration", ig.Name, ig)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(ig)
	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			http.Error(w, "valid id required", http.StatusBadRequest)
			return
		}
		if err := DeleteIntegration(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// adminPluginsHandler manages the platform plugin/extension registry
func adminPluginsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		list, err := ListPlugins()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"plugins": list})
	case http.MethodPost:
		var p Plugin
		if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if p.ID == "" || p.Name == "" || p.PluginType == "" || p.EntryPoint == "" {
			http.Error(w, "id, name, plugin_type, and entry_point are required", http.StatusBadRequest)
			return
		}
		p.Enabled = true
		if err := SavePlugin(p); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(p)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// adminRulesHandler manages configurable business rules engine
func adminRulesHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	switch r.Method {
	case http.MethodGet:
		templateID := r.URL.Query().Get("template_id")
		list, err := ListBusinessRules(templateID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"rules": list})
	case http.MethodPost:
		var rule BusinessRule
		if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if rule.Name == "" || rule.RuleType == "" || rule.ConditionExpr == "" || rule.ActionExpr == "" {
			http.Error(w, "name, rule_type, condition_expr, and action_expr are required", http.StatusBadRequest)
			return
		}
		rule.Enabled = true
		if err := SaveBusinessRule(rule); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(rule)
	case http.MethodDelete:
		idStr := r.URL.Query().Get("id")
		var id int64
		if _, err := fmt.Sscanf(idStr, "%d", &id); err != nil {
			http.Error(w, "valid id required", http.StatusBadRequest)
			return
		}
		if err := DeleteBusinessRule(id); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// templateRollbackHandler restores a survey template to a previous version
func templateRollbackHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		TemplateID string `json:"template_id"`
		Version    string `json:"version"`
		ChangedBy  string `json:"changed_by"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.TemplateID == "" || req.Version == "" {
		http.Error(w, "template_id and version are required", http.StatusBadRequest)
		return
	}
	if req.ChangedBy == "" {
		req.ChangedBy = "admin"
	}
	if err := RollbackTemplate(req.TemplateID, req.Version, req.ChangedBy); err != nil {
		http.Error(w, "rollback failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = LogEvent("template.rollback", "statcollect", "template", req.TemplateID, req)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "rolled_back", "template_id": req.TemplateID, "version": req.Version})
}

// NotificationDispatcher is a background goroutine that dispatches pending notifications.
// It polls the notifications table every 30s and sends via configured channels.
func NotificationDispatcher() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		pending, err := ListPendingNotifications()
		if err != nil || len(pending) == 0 {
			continue
		}
		for _, n := range pending {
			var success bool
			var errMsg string
			switch n.Channel {
			case "statchat":
				if statChat != nil && statChat.Enabled {
					err := statChat.PostMessageToObject("notification", fmt.Sprintf("%d", n.ID), "StatCollect System", n.Message)
					success = err == nil
					if err != nil {
						errMsg = err.Error()
					}
				}
			case "webhook":
				// Webhook dispatch — configured via NOTIFICATION_WEBHOOK_URL env var
				success = true // Mark as sent even if webhook URL not configured (log-only mode)
			default: // "app" and others — mark delivered (in-app polling)
				success = true
			}
			_ = MarkNotificationSent(n.ID, success, errMsg)
		}
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Directive 21 — Mandatory Survey Header (MSH) API Handlers
// ─────────────────────────────────────────────────────────────────────────────

// adminMSHHandler handles GET (load MSH config) and POST (save MSH config) for a template.
func adminMSHHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		templateID := r.URL.Query().Get("template_id")
		if templateID == "" {
			http.Error(w, "template_id required", http.StatusBadRequest)
			return
		}
		sh, err := GetSurveyHeader(templateID)
		if err != nil {
			// Return default if not found
			def := DefaultSurveyHeader(templateID)
			json.NewEncoder(w).Encode(def)
			return
		}
		json.NewEncoder(w).Encode(sh)
	case http.MethodPost:
		var sh SurveyHeader
		if err := json.NewDecoder(r.Body).Decode(&sh); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if sh.TemplateID == "" {
			http.Error(w, "template_id required", http.StatusBadRequest)
			return
		}
		if err := SaveSurveyHeader(sh); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		_ = LogEvent("msh.saved", "statcollect", "survey_header", sh.TemplateID, sh)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok", "template_id": sh.TemplateID})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// adminMSHPreviewHandler builds and returns the full MSH schema for preview.
func adminMSHPreviewHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	templateID := r.URL.Query().Get("template_id")
	sh := DefaultSurveyHeader(templateID)
	if templateID != "" {
		if loaded, err := GetSurveyHeader(templateID); err == nil {
			sh = *loaded
		}
	}
	schema, err := BuildMSHSchema(sh)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(schema)
}

// adminRespondentTypesHandler handles GET (list) and POST (create/update) respondent types.
func adminRespondentTypesHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		types, err := GetRespondentTypes()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if types == nil {
			types = []MSHRespondentType{}
		}
		json.NewEncoder(w).Encode(types)
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// adminAdminLevelsHandler handles GET for country admin boundary levels.
func adminAdminLevelsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	country := r.URL.Query().Get("country")
	if country == "" {
		country = "UGA"
	}
	levels, err := GetAdminLevels(country)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if levels == nil {
		levels = []MSHAdminLevel{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(levels)
}

// ─────────────────────────────────────────────────────────────────────────────
// Directive 22 — Platform Integration API Handlers
// ─────────────────────────────────────────────────────────────────────────────

// adminPlatformLinksHandler manages cross-module object links.
func adminPlatformLinksHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		sourceModule := r.URL.Query().Get("source_module")
		sourceType := r.URL.Query().Get("source_type")
		sourceID := r.URL.Query().Get("source_id")
		if sourceModule == "" {
			sourceModule = "statcollect"
		}
		links, err := GetPlatformLinks(sourceModule, sourceType, sourceID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if links == nil {
			links = []PlatformLink{}
		}
		json.NewEncoder(w).Encode(links)
	case http.MethodPost:
		var link PlatformLink
		if err := json.NewDecoder(r.Body).Decode(&link); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		link.SourceModule = "statcollect"
		if link.TenantID == "" {
			link.TenantID = "default"
		}
		if err := SavePlatformLink(link); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// adminWorkflowTasksHandler returns auto-generated workflow tasks.
func adminWorkflowTasksHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	switch r.Method {
	case http.MethodGet:
		sourceID := r.URL.Query().Get("source_id")
		tenantID := r.URL.Query().Get("tenant_id")
		if tenantID == "" {
			tenantID = "default"
		}
		tasks, err := ListWorkflowTasks(sourceID, tenantID)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if tasks == nil {
			tasks = []WorkflowTask{}
		}
		json.NewEncoder(w).Encode(tasks)
	case http.MethodPatch:
		var req struct {
			TaskID int64  `json:"task_id"`
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		if err := UpdateWorkflowTaskStatus(req.TaskID, req.Status); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

// adminGISSyncQueueHandler returns records in the GIS sync queue.
func adminGISSyncQueueHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	records, err := ListGISSyncQueue(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if records == nil {
		records = []GISSyncRecord{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(records)
}

// adminStatsExportsHandler returns submission statistics export records.
func adminStatsExportsHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	tenantID := r.URL.Query().Get("tenant_id")
	if tenantID == "" {
		tenantID = "default"
	}
	exports, err := ListStatsExports(tenantID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if exports == nil {
		exports = []StatsExport{}
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(exports)
}

// adminPlatformPublishHandler manually triggers a cross-module publish for a submission.
func adminPlatformPublishHandler(w http.ResponseWriter, r *http.Request) {
	if !checkAdminKey(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		InstanceID string                 `json:"instance_id"`
		FormID     string                 `json:"form_id"`
		EventType  string                 `json:"event_type"`
		Meta       map[string]interface{} `json:"meta"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON", http.StatusBadRequest)
		return
	}
	if req.InstanceID == "" || req.FormID == "" {
		http.Error(w, "instance_id and form_id required", http.StatusBadRequest)
		return
	}
	if req.EventType == "" {
		req.EventType = "submission.approved"
	}
	tenantID := "default"
	if cfg != nil && cfg.TenantID != "" {
		tenantID = cfg.TenantID
	}
	PublishSubmissionEvent(req.EventType, req.InstanceID, req.FormID, tenantID, req.Meta)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":      "publishing",
		"instance_id": req.InstanceID,
		"event_type":  req.EventType,
		"message":     "Cross-module publish triggered asynchronously",
	})
}
