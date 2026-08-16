package store

import (
	"context"
	"database/sql"
	"time"

	"learningcrm/pkg/model"
)

// ─── Leads (P26) ─────────────────────────────────────────────────────────────

func CreateLead(ctx context.Context, l *model.Lead) error {
	l.ID = newID()
	l.CreatedAt = nowT()
	if l.Stage == "" {
		l.Stage = "lead"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO leads (id, tenant_id, name, email, company, source, stage, value, owner_id, converted, converted_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		l.ID, l.TenantID, l.Name, l.Email, l.Company, l.Source, l.Stage, l.Value, l.OwnerID, l.Converted, l.ConvertedAt, l.CreatedAt)
	return err
}

func scanLead(row row) (*model.Lead, error) {
	var l model.Lead
	var conv sql.NullTime
	if err := row.Scan(&l.ID, &l.TenantID, &l.Name, &l.Email, &l.Company, &l.Source, &l.Stage, &l.Value, &l.OwnerID, &l.Converted, &conv, &l.CreatedAt); err != nil {
		return nil, err
	}
	if conv.Valid {
		l.ConvertedAt = &conv.Time
	}
	return &l, nil
}

const leadCols = `id, tenant_id, name, email, company, source, stage, value, owner_id, converted, converted_at, created_at`

func ListLeads(ctx context.Context, tenantID, stage string) ([]model.Lead, error) {
	query := `SELECT ` + leadCols + ` FROM leads WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if stage != "" {
		query += ` AND stage=$2`
		args = append(args, stage)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Lead
	for rows.Next() {
		l, err := scanLead(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *l)
	}
	return out, rows.Err()
}

func GetLead(ctx context.Context, id string) (*model.Lead, error) {
	return scanLead(db.QueryRowContext(ctx, `SELECT `+leadCols+` FROM leads WHERE id=$1`, id))
}

func UpdateLead(ctx context.Context, l *model.Lead) error {
	_, err := db.ExecContext(ctx, `
		UPDATE leads SET name=$2, email=$3, company=$4, source=$5, stage=$6, value=$7, owner_id=$8, converted=$9, converted_at=$10
		WHERE id=$1`,
		l.ID, l.Name, l.Email, l.Company, l.Source, l.Stage, l.Value, l.OwnerID, l.Converted, l.ConvertedAt)
	return err
}

// ConvertLead marks a lead won/converted.
func ConvertLead(ctx context.Context, id string) (*model.Lead, error) {
	l, err := GetLead(ctx, id)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	l.Converted = true
	l.ConvertedAt = &now
	l.Stage = "won"
	err = UpdateLead(ctx, l)
	return l, err
}

// ─── Accounts (P26) ──────────────────────────────────────────────────────────

func CreateAccount(ctx context.Context, a *model.Account) error {
	a.ID = newID()
	a.CreatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		INSERT INTO accounts (id, tenant_id, name, type, industry, website, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		a.ID, a.TenantID, a.Name, a.Type, a.Industry, a.Website, a.CreatedAt)
	return err
}

const accountCols = `id, tenant_id, name, type, industry, website, created_at`

func ListAccounts(ctx context.Context, tenantID string) ([]model.Account, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+accountCols+` FROM accounts WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Account
	for rows.Next() {
		var a model.Account
		if err := rows.Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Industry, &a.Website, &a.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func GetAccount(ctx context.Context, id string) (*model.Account, error) {
	var a model.Account
	err := db.QueryRowContext(ctx, `SELECT `+accountCols+` FROM accounts WHERE id=$1`, id).
		Scan(&a.ID, &a.TenantID, &a.Name, &a.Type, &a.Industry, &a.Website, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &a, nil
}

func DeleteAccount(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM accounts WHERE id=$1`, id)
	return err
}

// ─── Opportunities (P26) ─────────────────────────────────────────────────────

func CreateOpportunity(ctx context.Context, o *model.Opportunity) error {
	o.ID = newID()
	o.CreatedAt = nowT()
	if o.Stage == "" {
		o.Stage = "discovery"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO opportunities (id, tenant_id, account_id, lead_id, name, stage, amount, owner_id, expected_close, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		o.ID, o.TenantID, o.AccountID, o.LeadID, o.Name, o.Stage, o.Amount, o.OwnerID, o.ExpectedClose, o.CreatedAt)
	return err
}

func scanOpportunity(row row) (*model.Opportunity, error) {
	var o model.Opportunity
	var ec sql.NullTime
	if err := row.Scan(&o.ID, &o.TenantID, &o.AccountID, &o.LeadID, &o.Name, &o.Stage, &o.Amount, &o.OwnerID, &ec, &o.CreatedAt); err != nil {
		return nil, err
	}
	if ec.Valid {
		o.ExpectedClose = &ec.Time
	}
	return &o, nil
}

const oppCols = `id, tenant_id, account_id, lead_id, name, stage, amount, owner_id, expected_close, created_at`

func ListOpportunities(ctx context.Context, tenantID, stage string) ([]model.Opportunity, error) {
	query := `SELECT ` + oppCols + ` FROM opportunities WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if stage != "" {
		query += ` AND stage=$2`
		args = append(args, stage)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Opportunity
	for rows.Next() {
		o, err := scanOpportunity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *o)
	}
	return out, rows.Err()
}

func GetOpportunity(ctx context.Context, id string) (*model.Opportunity, error) {
	return scanOpportunity(db.QueryRowContext(ctx, `SELECT `+oppCols+` FROM opportunities WHERE id=$1`, id))
}

func UpdateOpportunity(ctx context.Context, o *model.Opportunity) error {
	_, err := db.ExecContext(ctx, `
		UPDATE opportunities SET account_id=$2, lead_id=$3, name=$4, stage=$5, amount=$6, owner_id=$7, expected_close=$8
		WHERE id=$1`,
		o.ID, o.AccountID, o.LeadID, o.Name, o.Stage, o.Amount, o.OwnerID, o.ExpectedClose)
	return err
}

func DeleteOpportunity(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM opportunities WHERE id=$1`, id)
	return err
}

// ─── Partners (P26) ──────────────────────────────────────────────────────────

func CreatePartner(ctx context.Context, p *model.Partner) error {
	p.ID = newID()
	p.CreatedAt = nowT()
	if p.Status == "" {
		p.Status = "active"
	}
	if p.Kind == "" {
		p.Kind = "reseller"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO partners (id, tenant_id, name, kind, status, contact_email, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7)`,
		p.ID, p.TenantID, p.Name, p.Kind, p.Status, p.ContactEmail, p.CreatedAt)
	return err
}

const partnerCols = `id, tenant_id, name, kind, status, contact_email, created_at`

func ListPartners(ctx context.Context, tenantID string) ([]model.Partner, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+partnerCols+` FROM partners WHERE tenant_id=$1 ORDER BY created_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Partner
	for rows.Next() {
		var p model.Partner
		if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Kind, &p.Status, &p.ContactEmail, &p.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

func GetPartner(ctx context.Context, id string) (*model.Partner, error) {
	var p model.Partner
	err := db.QueryRowContext(ctx, `SELECT `+partnerCols+` FROM partners WHERE id=$1`, id).
		Scan(&p.ID, &p.TenantID, &p.Name, &p.Kind, &p.Status, &p.ContactEmail, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func UpdatePartner(ctx context.Context, p *model.Partner) error {
	_, err := db.ExecContext(ctx, `
		UPDATE partners SET name=$2, kind=$3, status=$4, contact_email=$5 WHERE id=$1`,
		p.ID, p.Name, p.Kind, p.Status, p.ContactEmail)
	return err
}

func DeletePartner(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM partners WHERE id=$1`, id)
	return err
}

// ─── Service Requests (P26) ──────────────────────────────────────────────────

func CreateServiceRequest(ctx context.Context, sr *model.ServiceRequest) error {
	sr.ID = newID()
	sr.CreatedAt = nowT()
	sr.UpdatedAt = sr.CreatedAt
	if sr.Status == "" {
		sr.Status = "open"
	}
	if sr.Priority == "" {
		sr.Priority = "medium"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO service_requests (id, tenant_id, title, kind, requester, priority, status, assignee, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`,
		sr.ID, sr.TenantID, sr.Title, sr.Kind, sr.Requester, sr.Priority, sr.Status, sr.Assignee, sr.CreatedAt, sr.UpdatedAt)
	return err
}

const srCols = `id, tenant_id, title, kind, requester, priority, status, assignee, created_at, updated_at`

func ListServiceRequests(ctx context.Context, tenantID, status string) ([]model.ServiceRequest, error) {
	query := `SELECT ` + srCols + ` FROM service_requests WHERE tenant_id=$1`
	args := []interface{}{tenantID}
	if status != "" {
		query += ` AND status=$2`
		args = append(args, status)
	}
	query += ` ORDER BY created_at DESC`
	rows, err := db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.ServiceRequest
	for rows.Next() {
		var sr model.ServiceRequest
		if err := rows.Scan(&sr.ID, &sr.TenantID, &sr.Title, &sr.Kind, &sr.Requester, &sr.Priority, &sr.Status, &sr.Assignee, &sr.CreatedAt, &sr.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, sr)
	}
	return out, rows.Err()
}

func GetServiceRequest(ctx context.Context, id string) (*model.ServiceRequest, error) {
	var sr model.ServiceRequest
	err := db.QueryRowContext(ctx, `SELECT `+srCols+` FROM service_requests WHERE id=$1`, id).
		Scan(&sr.ID, &sr.TenantID, &sr.Title, &sr.Kind, &sr.Requester, &sr.Priority, &sr.Status, &sr.Assignee, &sr.CreatedAt, &sr.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &sr, nil
}

func UpdateServiceRequest(ctx context.Context, sr *model.ServiceRequest) error {
	sr.UpdatedAt = nowT()
	_, err := db.ExecContext(ctx, `
		UPDATE service_requests SET title=$2, kind=$3, requester=$4, priority=$5, status=$6, assignee=$7, updated_at=$8
		WHERE id=$1`,
		sr.ID, sr.Title, sr.Kind, sr.Requester, sr.Priority, sr.Status, sr.Assignee, sr.UpdatedAt)
	return err
}

// ─── Invoices & Billing Sync (P26) ───────────────────────────────────────────

func CreateInvoice(ctx context.Context, in *model.Invoice) error {
	in.ID = newID()
	in.CreatedAt = nowT()
	if in.Status == "" {
		in.Status = "draft"
	}
	if in.Currency == "" {
		in.Currency = "UGX"
	}
	_, err := db.ExecContext(ctx, `
		INSERT INTO invoices (id, tenant_id, number, account_id, partner_id, currency, amount, status, synced, issued_at, due_at, paid_at, created_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		in.ID, in.TenantID, in.Number, in.AccountID, in.PartnerID, in.Currency, in.Amount, in.Status, in.Synced, in.IssuedAt, in.DueAt, in.PaidAt, in.CreatedAt)
	return err
}

const invoiceCols = `id, tenant_id, number, account_id, partner_id, currency, amount, status, synced, issued_at, due_at, paid_at, created_at`

func ListInvoices(ctx context.Context, tenantID string) ([]model.Invoice, error) {
	rows, err := db.QueryContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE tenant_id=$1 ORDER BY issued_at DESC`, tenantID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Invoice
	for rows.Next() {
		var in model.Invoice
		var due, paid sql.NullTime
		if err := rows.Scan(&in.ID, &in.TenantID, &in.Number, &in.AccountID, &in.PartnerID, &in.Currency, &in.Amount, &in.Status, &in.Synced, &in.IssuedAt, &due, &paid, &in.CreatedAt); err != nil {
			return nil, err
		}
		if due.Valid {
			in.DueAt = &due.Time
		}
		if paid.Valid {
			in.PaidAt = &paid.Time
		}
		out = append(out, in)
	}
	return out, rows.Err()
}

func GetInvoice(ctx context.Context, id string) (*model.Invoice, error) {
	var in model.Invoice
	var due, paid sql.NullTime
	err := db.QueryRowContext(ctx, `SELECT `+invoiceCols+` FROM invoices WHERE id=$1`, id).
		Scan(&in.ID, &in.TenantID, &in.Number, &in.AccountID, &in.PartnerID, &in.Currency, &in.Amount, &in.Status, &in.Synced, &in.IssuedAt, &due, &paid, &in.CreatedAt)
	if err != nil {
		return nil, err
	}
	if due.Valid {
		in.DueAt = &due.Time
	}
	if paid.Valid {
		in.PaidAt = &paid.Time
	}
	return &in, nil
}

func UpdateInvoice(ctx context.Context, in *model.Invoice) error {
	_, err := db.ExecContext(ctx, `
		UPDATE invoices SET number=$2, account_id=$3, partner_id=$4, currency=$5, amount=$6, status=$7, synced=$8, issued_at=$9, due_at=$10, paid_at=$11
		WHERE id=$1`,
		in.ID, in.Number, in.AccountID, in.PartnerID, in.Currency, in.Amount, in.Status, in.Synced, in.IssuedAt, in.DueAt, in.PaidAt)
	return err
}

func DeleteInvoice(ctx context.Context, id string) error {
	_, err := db.ExecContext(ctx, `DELETE FROM invoices WHERE id=$1`, id)
	return err
}

// MarkInvoiceSynced flags an invoice as synchronized with the finance backend.
func MarkInvoiceSynced(ctx context.Context, id string) (*model.Invoice, error) {
	in, err := GetInvoice(ctx, id)
	if err != nil {
		return nil, err
	}
	in.Synced = true
	err = UpdateInvoice(ctx, in)
	return in, err
}