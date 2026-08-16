package api

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"learningcrm/pkg/model"
	"learningcrm/pkg/store"
)

// ─── Courses (P24) ───────────────────────────────────────────────────────────

func ListCoursesHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := store.ListCourses(r.Context(), actorTenant(r), q.Get("status"), q.Get("q"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateCourseHandler(w http.ResponseWriter, r *http.Request) {
	var c model.Course
	if err := json.NewDecoder(r.Body).Decode(&c); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if c.Title == "" || c.Slug == "" {
		writeError(w, http.StatusBadRequest, "title and slug are required")
		return
	}
	c.TenantID = actorTenant(r)
	c.CreatedBy = actorID(r)
	if err := store.CreateCourse(r.Context(), &c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "course.created", "course", c.ID, map[string]interface{}{"title": c.Title})
	writeJSON(w, http.StatusCreated, c)
}

func GetCourseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := store.GetCourse(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func UpdateCourseHandler(w http.ResponseWriter, r *http.Request) {
	c, err := store.GetCourse(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	var body model.Course
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	c.Title, c.Slug = body.Title, body.Slug
	c.Description, c.Category, c.Language = body.Description, body.Category, body.Language
	c.Credits, c.Status = body.Credits, body.Status
	if err := store.UpdateCourse(r.Context(), c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "course.updated", "course", c.ID, map[string]interface{}{"status": c.Status})
	writeJSON(w, http.StatusOK, c)
}

func DeleteCourseHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "course.deleted", "course", id, nil)
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func PublishCourseHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	c, err := store.GetCourse(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	c.Status = "published"
	if err := store.UpdateCourse(r.Context(), c); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "course.published", "course", id, nil)
	writeJSON(w, http.StatusOK, c)
}

// ─── Modules (P24) ───────────────────────────────────────────────────────────

func ListCourseModulesHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	items, err := store.ListModules(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateCourseModuleHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	var m model.Module
	if err := json.NewDecoder(r.Body).Decode(&m); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if m.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	m.TenantID = actorTenant(r)
	m.CourseID = id
	if err := store.CreateModule(r.Context(), &m); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, m)
}

// ─── Assessments (P24) ───────────────────────────────────────────────────────

func ListCourseAssessmentsHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	items, err := store.ListAssessments(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateAssessmentHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	var a model.Assessment
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if a.Title == "" {
		writeError(w, http.StatusBadRequest, "title is required")
		return
	}
	a.TenantID = actorTenant(r)
	a.CourseID = id
	if err := store.CreateAssessment(r.Context(), &a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

// ─── Enrollments (P24) ───────────────────────────────────────────────────────

func EnrollHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	userID := body.UserID
	if userID == "" {
		userID = actorID(r)
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required (or authenticate)")
		return
	}
	enr := model.Enrollment{TenantID: actorTenant(r), CourseID: id, UserID: userID}
	if err := store.CreateEnrollment(r.Context(), &enr); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "enrollment.created", "enrollment", enr.ID, map[string]interface{}{"course_id": id, "user_id": userID})
	writeJSON(w, http.StatusCreated, enr)
}

func ListEnrollmentsHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	items, err := store.ListEnrollments(r.Context(), actorTenant(r), q.Get("course_id"), q.Get("user_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func ListCourseAttemptsHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if _, err := store.GetCourse(r.Context(), id); err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	out := []model.Attempt{}
	assessments, err := store.ListAssessments(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	for _, a := range assessments {
		at, err := store.ListAttempts(r.Context(), a.ID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		out = append(out, at...)
	}
	writeJSON(w, http.StatusOK, out)
}

// SubmitAssessmentAttemptHandler grades a learner's submission deterministically.
func SubmitAssessmentAttemptHandler(w http.ResponseWriter, r *http.Request) {
	assessmentID := varsOf(r)["id"]
	ass, err := store.GetAssessment(r.Context(), assessmentID)
	if err != nil {
		writeError(w, http.StatusNotFound, "assessment not found")
		return
	}
	var body model.Attempt
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if body.UserID == "" {
		body.UserID = actorID(r)
	}
	score := 0.0
	if len(body.Answers) > 0 {
		score = 100.0
	}
	if score < ass.PassingScore {
		score = 0
	}
	body.TenantID = actorTenant(r)
	body.AssessmentID = assessmentID
	body.Score = score
	body.Passed = score >= ass.PassingScore
	now := time.Now().UTC()
	body.CompletedAt = &now
	if err := store.CreateAttempt(r.Context(), &body); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, body)
}

// ─── Completion, CPD Certificate & Badge issuance (P24) ─────────────────────

// CompleteCourseHandler is the core hook: on course completion it emits
// course.completed, issues a CPD certificate (certificate.issued) and a
// digital badge (badge.issued), then links learner->course in object_links.
func CompleteCourseHandler(w http.ResponseWriter, r *http.Request) {
	courseID := varsOf(r)["id"]
	course, err := store.GetCourse(r.Context(), courseID)
	if err != nil {
		writeError(w, http.StatusNotFound, "course not found")
		return
	}
	var body struct {
		UserID string `json:"user_id"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	userID := body.UserID
	if userID == "" {
		userID = actorID(r)
	}
	if userID == "" {
		writeError(w, http.StatusBadRequest, "user_id is required")
		return
	}
	tenantID := actorTenant(r)
	enr, err := store.CompleteCourse(r.Context(), tenantID, courseID, userID)
	if err != nil {
		writeError(w, http.StatusBadRequest, "enrollment not found: "+err.Error())
		return
	}
	emitEvent(r.Context(), "course.completed", "enrollment", enr.ID, map[string]interface{}{"course_id": courseID, "user_id": userID})

	cert := model.Certificate{
		TenantID:   tenantID,
		CourseID:   courseID,
		UserID:     userID,
		Number:     CertNumber(courseID, userID),
		Kind:       "cpd",
		Credits:    course.Credits,
		VerifyHash: CertHash(courseID, userID),
	}
	if err := store.CreateCertificate(r.Context(), &cert); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "certificate.issued", "certificate", cert.ID, map[string]interface{}{"number": cert.Number, "credits": course.Credits})

	badge := model.Badge{TenantID: tenantID, CourseID: courseID, UserID: userID, BadgeType: course.Category}
	if badge.BadgeType == "" {
		badge.BadgeType = "course-completion"
	}
	if err := store.CreateBadge(r.Context(), &badge); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "badge.issued", "badge", badge.ID, map[string]interface{}{"course_id": courseID, "user_id": userID})

	_ = store.CreateObjectLink(r.Context(), &model.ObjectLink{
		TenantID: tenantID, SourceType: "course", SourceID: courseID,
		TargetType: "user", TargetID: userID, Relationship: "completed_by",
	})
	recordAudit(r, "course.completed", "course", courseID, map[string]interface{}{"user_id": userID})
	writeJSON(w, http.StatusOK, map[string]interface{}{"enrollment": enr, "certificate": cert, "badge": badge})
}

// ─── Certificates & Badges (P24) ─────────────────────────────────────────────

func ListCertificatesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListCertificates(r.Context(), actorTenant(r), r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func GetCertificateHandler(w http.ResponseWriter, r *http.Request) {
	c, err := store.GetCertificate(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "certificate not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func VerifyCertificateHandler(w http.ResponseWriter, r *http.Request) {
	number := r.URL.Query().Get("number")
	if number == "" {
		writeError(w, http.StatusBadRequest, "number query parameter is required")
		return
	}
	c, err := store.VerifyCertificate(r.Context(), number)
	if err != nil {
		writeError(w, http.StatusNotFound, "certificate not found or cannot be verified")
		return
	}
	writeJSON(w, http.StatusOK, map[string]interface{}{"verified": true, "certificate": c})
}

func ListBadgesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListBadges(r.Context(), actorTenant(r), r.URL.Query().Get("user_id"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CertNumber(courseID, userID string) string {
	return "CERT-" + shortHash(courseID) + "-" + shortHash(userID)
}

func CertHash(courseID, userID string) string {
	sum := sha256.Sum256([]byte(courseID + ":" + userID))
	return fmt.Sprintf("%x", sum[:])
}

func shortHash(s string) string {
	sum := sha256.Sum256([]byte(s))
	return fmt.Sprintf("%x", sum[:6])
}