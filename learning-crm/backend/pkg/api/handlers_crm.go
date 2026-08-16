package api

import (
	"encoding/json"
	"net/http"
	"time"

	"learningcrm/pkg/model"
	"learningcrm/pkg/store"
)

// ─── Leads (P26) ─────────────────────────────────────────────────────────────

func ListLeadsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListLeads(r.Context(), actorTenant(r), r.URL.Query().Get("stage"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateLeadHandler(w http.ResponseWriter, r *http.Request) {
	var l model.Lead
	if err := json.NewDecoder(r.Body).Decode(&l); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if l.Name == "" || l.Email == "" {
		writeError(w, http.StatusBadRequest, "name and email are required")
		return
	}
	l.TenantID = actorTenant(r)
	l.OwnerID = actorID(r)
	if err := store.CreateLead(r.Context(), &l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	recordAudit(r, "lead.created", "lead", l.ID, map[string]interface{}{"email": l.Email})
	writeJSON(w, http.StatusCreated, l)
}

func GetLeadHandler(w http.ResponseWriter, r *http.Request) {
	l, err := store.GetLead(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	writeJSON(w, http.StatusOK, l)
}

func UpdateLeadHandler(w http.ResponseWriter, r *http.Request) {
	l, err := store.GetLead(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	var body model.Lead
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	l.Name, l.Email = body.Name, body.Email
	l.Company, l.Source, l.Stage = body.Company, body.Source, body.Stage
	l.Value, l.OwnerID = body.Value, body.OwnerID
	if err := store.UpdateLead(r.Context(), l); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, l)
}

// ConvertLeadHandler marks a lead converted and emits lead.converted.
func ConvertLeadHandler(w http.ResponseWriter, r *http.Request) {
	l, err := store.ConvertLead(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "lead not found")
		return
	}
	emitEvent(r.Context(), "lead.converted", "lead", l.ID, map[string]interface{}{"email": l.Email, "value": l.Value})
	_ = store.CreateObjectLink(r.Context(), &model.ObjectLink{
		TenantID: l.TenantID, SourceType: "lead", SourceID: l.ID, TargetType: "account", TargetID: l.Company, Relationship: "converted_to",
	})
	recordAudit(r, "lead.converted", "lead", l.ID, map[string]interface{}{"email": l.Email})
	writeJSON(w, http.StatusOK, l)
}

// ─── Accounts (P26) ──────────────────────────────────────────────────────────

func ListAccountsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListAccounts(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateAccountHandler(w http.ResponseWriter, r *http.Request) {
	var a model.Account
	if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if a.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	a.TenantID = actorTenant(r)
	if err := store.CreateAccount(r.Context(), &a); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, a)
}

func GetAccountHandler(w http.ResponseWriter, r *http.Request) {
	a, err := store.GetAccount(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "account not found")
		return
	}
	writeJSON(w, http.StatusOK, a)
}

func DeleteAccountHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteAccount(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Opportunities (P26) ─────────────────────────────────────────────────────

func ListOpportunitiesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListOpportunities(r.Context(), actorTenant(r), r.URL.Query().Get("stage"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateOpportunityHandler(w http.ResponseWriter, r *http.Request) {
	var o model.Opportunity
	if err := json.NewDecoder(r.Body).Decode(&o); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if o.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	o.TenantID = actorTenant(r)
	o.OwnerID = actorID(r)
	if err := store.CreateOpportunity(r.Context(), &o); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, o)
}

func GetOpportunityHandler(w http.ResponseWriter, r *http.Request) {
	o, err := store.GetOpportunity(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "opportunity not found")
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func UpdateOpportunityHandler(w http.ResponseWriter, r *http.Request) {
	o, err := store.GetOpportunity(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "opportunity not found")
		return
	}
	var body model.Opportunity
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	o.Name, o.Stage, o.Amount = body.Name, body.Stage, body.Amount
	o.AccountID, o.LeadID, o.OwnerID, o.ExpectedClose = body.AccountID, body.LeadID, body.OwnerID, body.ExpectedClose
	if err := store.UpdateOpportunity(r.Context(), o); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, o)
}

func DeleteOpportunityHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteOpportunity(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Partners (P26) ──────────────────────────────────────────────────────────

func ListPartnersHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListPartners(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreatePartnerHandler(w http.ResponseWriter, r *http.Request) {
	var p model.Partner
	if err := json.NewDecoder(r.Body).Decode(&p); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if p.Name == "" {
		writeError(w, http.StatusBadRequest, "name is required")
		return
	}
	p.TenantID = actorTenant(r)
	if err := store.CreatePartner(r.Context(), &p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "partner.registered", "partner", p.ID, map[string]interface{}{"name": p.Name, "kind": p.Kind})
	writeJSON(w, http.StatusCreated, p)
}

func GetPartnerHandler(w http.ResponseWriter, r *http.Request) {
	p, err := store.GetPartner(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "partner not found")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func UpdatePartnerHandler(w http.ResponseWriter, r *http.Request) {
	p, err := store.GetPartner(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "partner not found")
		return
	}
	var body model.Partner
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	p.Name, p.Kind, p.Status, p.ContactEmail = body.Name, body.Kind, body.Status, body.ContactEmail
	if err := store.UpdatePartner(r.Context(), p); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, p)
}

func DeletePartnerHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeletePartner(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// ─── Service Requests (P26) ──────────────────────────────────────────────────

func ListServiceRequestsHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListServiceRequests(r.Context(), actorTenant(r), r.URL.Query().Get("status"))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateServiceRequestHandler(w http.ResponseWriter, r *http.Request) {
	var sr model.ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&sr); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if sr.Title == "" || sr.Requester == "" {
		writeError(w, http.StatusBadRequest, "title and requester are required")
		return
	}
	sr.TenantID = actorTenant(r)
	if err := store.CreateServiceRequest(r.Context(), &sr); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	emitEvent(r.Context(), "service_request.created", "service_request", sr.ID, map[string]interface{}{"title": sr.Title, "priority": sr.Priority})
	writeJSON(w, http.StatusCreated, sr)
}

func GetServiceRequestHandler(w http.ResponseWriter, r *http.Request) {
	sr, err := store.GetServiceRequest(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "service request not found")
		return
	}
	writeJSON(w, http.StatusOK, sr)
}

func UpdateServiceRequestHandler(w http.ResponseWriter, r *http.Request) {
	sr, err := store.GetServiceRequest(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "service request not found")
		return
	}
	var body model.ServiceRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	sr.Title, sr.Kind, sr.Priority = body.Title, body.Kind, body.Priority
	sr.Status, sr.Assignee = body.Status, body.Assignee
	if err := store.UpdateServiceRequest(r.Context(), sr); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, sr)
}

// ─── Invoices & Billing Sync (P26) ───────────────────────────────────────────

func ListInvoicesHandler(w http.ResponseWriter, r *http.Request) {
	items, err := store.ListInvoices(r.Context(), actorTenant(r))
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func CreateInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	var in model.Invoice
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	if in.Number == "" || in.Amount <= 0 {
		writeError(w, http.StatusBadRequest, "number and positive amount are required")
		return
	}
	in.TenantID = actorTenant(r)
	if in.IssuedAt.IsZero() {
		in.IssuedAt = time.Now().UTC()
	}
	if err := store.CreateInvoice(r.Context(), &in); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, in)
}

func GetInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	in, err := store.GetInvoice(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func UpdateInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	in, err := store.GetInvoice(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	var body model.Invoice
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeError(w, http.StatusBadRequest, "invalid body: "+err.Error())
		return
	}
	in.Status, in.Amount, in.Currency = body.Status, body.Amount, body.Currency
	in.DueAt, in.PaidAt = body.DueAt, body.PaidAt
	if err := store.UpdateInvoice(r.Context(), in); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, in)
}

func DeleteInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	id := varsOf(r)["id"]
	if err := store.DeleteInvoice(r.Context(), id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

// SyncInvoiceHandler marks an invoice synchronized with the finance/ERP app.
func SyncInvoiceHandler(w http.ResponseWriter, r *http.Request) {
	in, err := store.MarkInvoiceSynced(r.Context(), varsOf(r)["id"])
	if err != nil {
		writeError(w, http.StatusNotFound, "invoice not found")
		return
	}
	recordAudit(r, "invoice.synced", "invoice", in.ID, map[string]interface{}{"number": in.Number})
	writeJSON(w, http.StatusOK, in)
}