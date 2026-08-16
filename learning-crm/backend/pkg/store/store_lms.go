package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"time"

	"learningcrm/pkg/model"
)

type row interface{ Scan(...interface{}) error }

func itoa(n int) string { return strconv.Itoa(n) }

// ─── Courses (P24) ───────────────────────────────────────────────────────────

func CreateCourse(ctx context.Context, c *model.Course) error {
	c.ID = newID()
	c.CreatedAt = nowT()
	c.UpdatedAt = c.CreatedAt
	if c.Status == "" {
		c.Status = "draft"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO courses (id, tenant_id, title, slug, description, category, language, credits, status, created_by, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		c.ID, c.TenantID, c.Title, c.Slug, c.Description, c.Category, c.Language, c.Credits, c.Status, c.CreatedBy, c.CreatedAt, c.UpdatedAt)
	return err
}

func scanCourse(row row) (*model.Course, error) {
	var c model.Course
	if err := row.Scan(&c.ID, &c.TenantID, &c.Title, &c.Slug, &c.Description, &c.Category, &c.Language, &c.Credits, &c.Status, &c.CreatedBy, &c.CreatedAt, &c.UpdatedAt); err != nil {
		return nil, err
	}
	return &c, nil
}

const courseCols = `id, tenant_id, title, slug, description, category, language, credits, status, created_by, created_at, updated_at`

func ListCourses(ctx context.Context, tenantID, status, q string) ([]model.Course, error) {
	query := `SELECT ` + courseCols + ` FROM courses WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	if status != "" {
		argc++
		args = append(args, status)
		query += ` AND status=$` + itoa(argc)
	}
	if q != "" {
		argc++
		args = append(args, "%"+lower(q)+"%")
		query += ` AND (lower(title) LIKE $` + itoa(argc) + ` OR lower(description) LIKE $` + itoa(argc) + `)`
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Course
	for rows.Next() {
		c, err := scanCourse(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *c)
	}
	return out, rows.Err()
}

func GetCourse(ctx context.Context, id string) (*model.Course, error) {
	return scanCourse(db.QueryRowContext(ctx, `SELECT `+courseCols+` FROM courses WHERE id=$1`, id))
}

func UpdateCourse(ctx context.Context, c *model.Course) error {
	c.UpdatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		UPDATE courses SET title=$2, slug=$3, description=$4, category=$5, language=$6, credits=$7, status=$8, updated_at=$9
		WHERE id=$1`,
		c.ID, c.Title, c.Slug, c.Description, c.Category, c.Language, c.Credits, c.Status, c.UpdatedAt)
	return err
}

func DeleteCourse(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM courses WHERE id=$1`, id)
	return err
}

// ─── Modules (P24) ───────────────────────────────────────────────────────────

func CreateModule(ctx context.Context, m *model.Module) error {
	m.ID = newID()
	m.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO modules (id, tenant_id, course_id, title, order_index, content, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		m.ID, m.TenantID, m.CourseID, m.Title, m.OrderIndex, jsonB(m.Content), m.CreatedAt)
	return err
}

func scanModule(row row) (*model.Module, error) {
	var m model.Module
	var content []byte
	if err := row.Scan(&m.ID, &m.TenantID, &m.CourseID, &m.Title, &m.OrderIndex, &content, &m.CreatedAt); err != nil {
		return nil, err
	}
	if len(content) > 0 {
		_ = json.Unmarshal(content, &m.Content)
	}
	return &m, nil
}

const moduleCols = `id, tenant_id, course_id, title, order_index, content, created_at`

func ListModules(ctx context.Context, courseID string) ([]model.Module, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+moduleCols+` FROM modules WHERE course_id=$1 ORDER BY order_index`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Module
	for rows.Next() {
		m, err := scanModule(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *m)
	}
	return out, rows.Err()
}

var _ = sql.ErrNoRows

// ─── Enrollments (P24) ───────────────────────────────────────────────────────

func CreateEnrollment(ctx context.Context, e *model.Enrollment) error {
	e.ID = newID()
	e.CreatedAt = nowT()
	if e.Status == "" {
		e.Status = "enrolled"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO enrollments (id, tenant_id, course_id, user_id, status, progress, completed_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT (tenant_id, course_id, user_id) DO UPDATE SET status='enrolled'`,
		e.ID, e.TenantID, e.CourseID, e.UserID, e.Status, e.Progress, e.CompletedAt, e.CreatedAt)
	return err
}

func scanEnrollment(row row) (*model.Enrollment, error) {
	var e model.Enrollment
	var comp sql.NullTime
	if err := row.Scan(&e.ID, &e.TenantID, &e.CourseID, &e.UserID, &e.Status, &e.Progress, &comp, &e.CreatedAt); err != nil {
		return nil, err
	}
	if comp.Valid {
		e.CompletedAt = &comp.Time
	}
	return &e, nil
}

const enrollmentCols = `id, tenant_id, course_id, user_id, status, progress, completed_at, created_at`

func ListEnrollments(ctx context.Context, tenantID, courseID, userID string) ([]model.Enrollment, error) {
	query := `SELECT ` + enrollmentCols + ` FROM enrollments WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	argc := 1
	if courseID != "" {
		argc++
		args = append(args, courseID)
		query += ` AND course_id=$` + itoa(argc)
	}
	if userID != "" {
		argc++
		args = append(args, userID)
		query += ` AND user_id=$` + itoa(argc)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Enrollment
	for rows.Next() {
		e, err := scanEnrollment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *e)
	}
	return out, rows.Err()
}

func FindEnrollment(ctx context.Context, tenantID, courseID, userID string) (*model.Enrollment, error) {
	return scanEnrollment(db.QueryRowContext(ctx,
		`SELECT `+enrollmentCols+` FROM enrollments WHERE tenant_id=$1 AND course_id=$2 AND user_id=$3`,
		tenantID, courseID, userID))
}

// CompleteCourse marks an enrollment (and therefore the course) completed.
func CompleteCourse(ctx context.Context, tenantID, courseID, userID string) (*model.Enrollment, error) {
	e, err := FindEnrollment(ctx, tenantID, courseID, userID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	e.Status = "completed"
	e.Progress = 100
	e.CompletedAt = &now
	_, err = db.ExecContext(ctx, `UPDATE enrollments SET status='completed', progress=100, completed_at=$2 WHERE id=$1`,
		e.ID, now)
	return e, err
}

// ─── Assessments (P24) ───────────────────────────────────────────────────────

func CreateAssessment(ctx context.Context, a *model.Assessment) error {
	a.ID = newID()
	a.CreatedAt = nowT()
	if a.Kind == "" {
		a.Kind = "quiz"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO assessments (id, tenant_id, course_id, title, kind, config, passing_score, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		a.ID, a.TenantID, a.CourseID, a.Title, a.Kind, jsonB(a.Config), a.PassingScore, a.CreatedAt)
	return err
}

func scanAssessment(row row) (*model.Assessment, error) {
	var a model.Assessment
	var config []byte
	if err := row.Scan(&a.ID, &a.TenantID, &a.CourseID, &a.Title, &a.Kind, &config, &a.PassingScore, &a.CreatedAt); err != nil {
		return nil, err
	}
	if len(config) > 0 {
		_ = json.Unmarshal(config, &a.Config)
	}
	return &a, nil
}

const assessmentCols = `id, tenant_id, course_id, title, kind, config, passing_score, created_at`

func GetAssessment(ctx context.Context, id string) (*model.Assessment, error) {
	return scanAssessment(db.QueryRowContext(ctx, `SELECT `+assessmentCols+` FROM assessments WHERE id=$1`, id))
}

func ListAssessments(ctx context.Context, courseID string) ([]model.Assessment, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+assessmentCols+` FROM assessments WHERE course_id=$1 ORDER BY created_at`, courseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Assessment
	for rows.Next() {
		a, err := scanAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *a)
	}
	return out, rows.Err()
}

// ─── Attempts ────────────────────────────────────────────────────────────────

func CreateAttempt(ctx context.Context, at *model.Attempt) error {
	at.ID = newID()
	at.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO attempts (id, tenant_id, assessment_id, user_id, answers, score, passed, completed_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9)`,
		at.ID, at.TenantID, at.AssessmentID, at.UserID, jsonB(at.Answers), at.Score, at.Passed, at.CompletedAt, at.CreatedAt)
	return err
}

func ListAttempts(ctx context.Context, assessmentID string) ([]model.Attempt, error) {
	rows, err := db.QueryContext(ctx, `
		SELECT id, tenant_id, assessment_id, user_id, answers, score, passed, completed_at, created_at
		FROM attempts WHERE assessment_id=$1 ORDER BY created_at DESC`, assessmentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Attempt
	for rows.Next() {
		var a model.Attempt
		var answers []byte
		var comp sql.NullTime
		if err := rows.Scan(&a.ID, &a.TenantID, &a.AssessmentID, &a.UserID, &answers, &a.Score, &a.Passed, &comp, &a.CreatedAt); err != nil {
			return nil, err
		}
		if len(answers) > 0 {
			_ = json.Unmarshal(answers, &a.Answers)
		}
		if comp.Valid {
			a.CompletedAt = &comp.Time
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// ─── CPD Certificates & Badges (P24) ─────────────────────────────────────────

func CreateCertificate(ctx context.Context, c *model.Certificate) error {
	c.ID = newID()
	if c.IssuedAt.IsZero() {
		c.IssuedAt = nowT()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO certificates (id, tenant_id, course_id, user_id, number, kind, credits, issued_at, expires_at, verify_hash)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		c.ID, c.TenantID, c.CourseID, c.UserID, c.Number, c.Kind, c.Credits, c.IssuedAt, c.ExpiresAt, c.VerifyHash)
	return err
}

func GetCertificate(ctx context.Context, id string) (*model.Certificate, error) {
	var c model.Certificate
	var exp sql.NullTime
	err := db.QueryRowContext(ctx, `
		SELECT id, tenant_id, course_id, user_id, number, kind, credits, issued_at, expires_at, verify_hash
		FROM certificates WHERE id=$1`, id).
		Scan(&c.ID, &c.TenantID, &c.CourseID, &c.UserID, &c.Number, &c.Kind, &c.Credits, &c.IssuedAt, &exp, &c.VerifyHash)
	if err != nil {
		return nil, err
	}
	if exp.Valid {
		c.ExpiresAt = &exp.Time
	}
	return &c, nil
}

func VerifyCertificate(ctx context.Context, number string) (*model.Certificate, error) {
	var c model.Certificate
	var exp sql.NullTime
	err := db.QueryRowContext(ctx, `
		SELECT id, tenant_id, course_id, user_id, number, kind, credits, issued_at, expires_at, verify_hash
		FROM certificates WHERE number=$1`, number).
		Scan(&c.ID, &c.TenantID, &c.CourseID, &c.UserID, &c.Number, &c.Kind, &c.Credits, &c.IssuedAt, &exp, &c.VerifyHash)
	if err != nil {
		return nil, err
	}
	if exp.Valid {
		c.ExpiresAt = &exp.Time
	}
	return &c, nil
}

func ListCertificates(ctx context.Context, tenantID, userID string) ([]model.Certificate, error) {
	query := `SELECT id, tenant_id, course_id, user_id, number, kind, credits, issued_at, expires_at, verify_hash FROM certificates WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if userID != "" {
		query += ` AND user_id=$2`
		args = append(args, userID)
	}
	query += ` ORDER BY issued_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Certificate
	for rows.Next() {
		var c model.Certificate
		var exp sql.NullTime
		if err := rows.Scan(&c.ID, &c.TenantID, &c.CourseID, &c.UserID, &c.Number, &c.Kind, &c.Credits, &c.IssuedAt, &exp, &c.VerifyHash); err != nil {
			return nil, err
		}
		if exp.Valid {
			c.ExpiresAt = &exp.Time
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// CreateBadge issues a digital badge.
func CreateBadge(ctx context.Context, b *model.Badge) error {
	b.ID = newID()
	if b.IssuedAt.IsZero() {
		b.IssuedAt = nowT()
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO badges (id, tenant_id, course_id, user_id, badge_type, issued_at)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		b.ID, b.TenantID, b.CourseID, b.UserID, b.BadgeType, b.IssuedAt)
	return err
}

func ListBadges(ctx context.Context, tenantID, userID string) ([]model.Badge, error) {
	query := `SELECT id, tenant_id, course_id, user_id, badge_type, issued_at FROM badges WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if userID != "" {
		query += ` AND user_id=$2`
		args = append(args, userID)
	}
	query += ` ORDER BY issued_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Badge
	for rows.Next() {
		var b model.Badge
		if err := rows.Scan(&b.ID, &b.TenantID, &b.CourseID, &b.UserID, &b.BadgeType, &b.IssuedAt); err != nil {
			return nil, err
		}
		out = append(out, b)
	}
	return out, rows.Err()
}