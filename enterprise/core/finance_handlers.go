package main

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// ═══════════════════════════════════════════════════════════════════
// PHASE 14 — FINANCIAL MANAGEMENT, GRANTS, PROCUREMENT & ASSETS
// ═══════════════════════════════════════════════════════════════════

// FinancialGrant represents a multi-donor financial grant agreement.
type FinancialGrant struct {
	ID              string    `json:"id"`
	Title           string    `json:"title"`
	GrantCode       string    `json:"grant_code"`        // e.g. "USAID-STAT-2026-01"
	DonorName       string    `json:"donor_name"`        // e.g. "USAID", "World Bank", "FCDO", "UNDP"
	TotalBudget     float64   `json:"total_budget"`
	DisbursedAmount float64   `json:"disbursed_amount"`
	Currency        string    `json:"currency"`          // "USD", "EUR", "TZS", "KES"
	StartDate       string    `json:"start_date"`
	EndDate         string    `json:"end_date"`
	ProjectID       string    `json:"project_id,omitempty"`
	Status          string    `json:"status"`            // "Active", "Pending_Disbursement", "Closed", "Audited"
	ComplianceRules []string  `json:"compliance_rules,omitempty"`
	CreatedTime     time.Time `json:"created_time"`
}


// Budget represents an allocated institutional budget line.
type Budget struct {
	ID              string    `json:"id"`
	FiscalYear      string    `json:"fiscal_year"`       // "FY2025/2026"
	CostCenterCode  string    `json:"cost_center_code"`  // "CC-STAT-01"
	CostCenterName  string    `json:"cost_center_name"`  // "Directorate of Statistical Sampling"
	AllocatedAmount float64   `json:"allocated_amount"`
	CommittedAmount float64   `json:"committed_amount"`
	SpentAmount     float64   `json:"spent_amount"`
	RemainingAmount float64   `json:"remaining_amount"`
	Currency        string    `json:"currency"`
	Status          string    `json:"status"`            // "Approved", "Draft", "Frozen"
	CreatedTime     time.Time `json:"created_time"`
}

// PurchaseRequest represents a procurement requisition.
type PurchaseRequest struct {
	ID                string    `json:"id"`
	RequisitionNumber string    `json:"requisition_number"` // "PR-2026-084"
	RequestorName     string    `json:"requestor_name"`
	Department        string    `json:"department"`
	ItemDescription   string    `json:"item_description"`
	EstimatedTotal    float64   `json:"estimated_total"`
	Currency          string    `json:"currency"`
	Justification     string    `json:"justification"`
	Status            string    `json:"status"`             // "Submitted", "Approved", "In_Tender", "PO_Issued"
	CreatedTime       time.Time `json:"created_time"`
}

// PurchaseOrder represents an awarded procurement order.
type PurchaseOrder struct {
	ID             string    `json:"id"`
	PONumber       string    `json:"po_number"`          // "PO-2026-019"
	VendorName     string    `json:"vendor_name"`
	RequisitionID  string    `json:"requisition_id"`
	TotalAmount    float64   `json:"total_amount"`
	Currency       string    `json:"currency"`
	DeliveryStatus string    `json:"delivery_status"`    // "Pending", "Delivered", "Inspected", "Invoiced"
	PaymentTerms   string    `json:"payment_terms"`      // "Net 30", "Advance 50%", "On Delivery"
	IssuedDate     string    `json:"issued_date"`
	CreatedTime    time.Time `json:"created_time"`
}

// AssetItem represents a tracked capital asset.
type AssetItem struct {
	ID                 string    `json:"id"`
	AssetCode          string    `json:"asset_code"`         // "AST-SRV-2026-042"
	Name               string    `json:"name"`
	Category           string    `json:"category"`           // "IT_Equipment", "Vehicles", "Laboratory", "Survey_Tablets"
	AcquisitionDate    string    `json:"acquisition_date"`
	OriginalCost       float64   `json:"original_cost"`
	CurrentBookValue   float64   `json:"current_book_value"`
	DepreciationMethod string    `json:"depreciation_method"` // "Straight-Line 20%", "Reducing Balance"
	CustodianName      string    `json:"custodian_name"`
	Location           string    `json:"location"`           // "HQ Server Room", "Arusha Field Office"
	Condition          string    `json:"condition"`          // "Excellent", "Good", "Maintenance_Required", "Disposed"
	CreatedTime        time.Time `json:"created_time"`
}

// ExpenseClaim represents a travel authorization & expense reimbursement.
type ExpenseClaim struct {
	ID             string    `json:"id"`
	ClaimNumber    string    `json:"claim_number"`       // "EXP-TRV-2026-09"
	StaffName      string    `json:"staff_name"`
	Destination    string    `json:"destination"`        // "Dodoma Field Mission"
	PerDiemAmount  float64   `json:"per_diem_amount"`
	AdvancePaid    float64   `json:"advance_paid"`
	ActualExpenses float64   `json:"actual_expenses"`
	NetBalance     float64   `json:"net_balance"`        // (Actual - Advance)
	Currency       string    `json:"currency"`
	Status         string    `json:"status"`             // "Pending_Review", "Approved", "Reconciled", "Disbursed"
	SubmittedDate  string    `json:"submitted_date"`
	CreatedTime    time.Time `json:"created_time"`
}

// In-memory seeds for Phase 14 Financial Management
var defaultFinancialGrants = []FinancialGrant{

	{
		ID:              "grant-001",
		Title:           "National Welfare Monitoring & Census Modernization",
		GrantCode:       "WB-STAT-2026-09",
		DonorName:       "World Bank IDA",
		TotalBudget:     4850000.00,
		DisbursedAmount: 3200000.00,
		Currency:        "USD",
		StartDate:       "2025-01-01",
		EndDate:         "2027-12-31",
		Status:          "Active",
		ComplianceRules: []string{"IATI Format Publishing", "Annual Independent Audit", "Environmental Safeguards"},
		CreatedTime:     time.Now().Add(-180 * 24 * time.Hour),
	},
	{
		ID:              "grant-002",
		Title:           "Sub-National Health Information & Geospatial Surveillance",
		GrantCode:       "GF-HLTH-2025-04",
		DonorName:       "Global Fund",
		TotalBudget:     1750000.00,
		DisbursedAmount: 1100000.00,
		Currency:        "USD",
		StartDate:       "2025-06-01",
		EndDate:         "2026-12-31",
		Status:          "Active",
		ComplianceRules: []string{"Quarterly Milestone Reports", "Zero Procurement Kickback Policy"},
		CreatedTime:     time.Now().Add(-120 * 24 * time.Hour),
	},
	{
		ID:              "grant-003",
		Title:           "Open Science & Clinical Bioethics Data Fabric",
		GrantCode:       "WT-RES-2026-11",
		DonorName:       "Wellcome Trust",
		TotalBudget:     890000.00,
		DisbursedAmount: 450000.00,
		Currency:        "USD",
		StartDate:       "2026-02-01",
		EndDate:         "2028-01-31",
		Status:          "Active",
		ComplianceRules: []string{"FAIR Data Principles", "Open Access Repository Deposit within 6 Months"},
		CreatedTime:     time.Now().Add(-45 * 24 * time.Hour),
	},
}

var defaultBudgets = []Budget{
	{
		ID:              "bgt-001",
		FiscalYear:      "FY2025/2026",
		CostCenterCode:  "CC-STAT-01",
		CostCenterName:  "National Census & Household Surveys",
		AllocatedAmount: 1200000000.00,
		CommittedAmount: 450000000.00,
		SpentAmount:     580000000.00,
		RemainingAmount: 170000000.00,
		Currency:        "TZS",
		Status:          "Approved",
		CreatedTime:     time.Now().Add(-90 * 24 * time.Hour),
	},
	{
		ID:              "bgt-002",
		FiscalYear:      "FY2025/2026",
		CostCenterCode:  "CC-RMS-02",
		CostCenterName:  "Institutional Research & Bioethics Secretariat",
		AllocatedAmount: 450000000.00,
		CommittedAmount: 120000000.00,
		SpentAmount:     210000000.00,
		RemainingAmount: 120000000.00,
		Currency:        "TZS",
		Status:          "Approved",
		CreatedTime:     time.Now().Add(-90 * 24 * time.Hour),
	},
	{
		ID:              "bgt-003",
		FiscalYear:      "FY2025/2026",
		CostCenterCode:  "CC-GIS-03",
		CostCenterName:  "Geospatial Information & Remote Sensing Unit",
		AllocatedAmount: 680000000.00,
		CommittedAmount: 230000000.00,
		SpentAmount:     310000000.00,
		RemainingAmount: 140000000.00,
		Currency:        "TZS",
		Status:          "Approved",
		CreatedTime:     time.Now().Add(-90 * 24 * time.Hour),
	},
}

var defaultPurchaseRequests = []PurchaseRequest{
	{
		ID:                "pr-001",
		RequisitionNumber: "PR-2026-084",
		RequestorName:     "Dr. Beatrice Mvula",
		Department:        "Health Demographics",
		ItemDescription:   "150x Rugged Survey Tablets with GPS & Solar Power Banks",
		EstimatedTotal:    67500.00,
		Currency:          "USD",
		Justification:     "Field enumeration for National Health Survey round 2",
		Status:            "Approved",
		CreatedTime:       time.Now().Add(-14 * 24 * time.Hour),
	},
	{
		ID:                "pr-002",
		RequisitionNumber: "PR-2026-085",
		RequestorName:     "Simon Ndosi",
		Department:        "IT Infrastructure",
		ItemDescription:   "High-Memory PostGIS Cluster Server Node (256GB RAM, NVMe)",
		EstimatedTotal:    14200.00,
		Currency:          "USD",
		Justification:     "Geospatial analytical acceleration for Phase 10 spatial queries",
		Status:            "In_Tender",
		CreatedTime:       time.Now().Add(-5 * 24 * time.Hour),
	},
}

var defaultPurchaseOrders = []PurchaseOrder{
	{
		ID:             "po-001",
		PONumber:       "PO-2026-019",
		VendorName:     "AfriTech Hardware Solutions Ltd",
		RequisitionID:  "pr-001",
		TotalAmount:    64800.00,
		Currency:       "USD",
		DeliveryStatus: "Delivered",
		PaymentTerms:   "Net 30 Days after Inspection",
		IssuedDate:     "2026-07-28",
		CreatedTime:    time.Now().Add(-18 * 24 * time.Hour),
	},
}

var defaultAssets = []AssetItem{
	{
		ID:                 "ast-001",
		AssetCode:          "AST-TAB-2026-104",
		Name:               "Samsung Galaxy Active Pro Survey Tablet",
		Category:           "Survey_Tablets",
		AcquisitionDate:    "2026-02-15",
		OriginalCost:       450.00,
		CurrentBookValue:   360.00,
		DepreciationMethod: "Straight-Line 20%",
		CustodianName:      "Fatuma Ally (Field Team 4)",
		Location:           "Mwanza Field Office",
		Condition:          "Good",
		CreatedTime:        time.Now().Add(-180 * 24 * time.Hour),
	},
	{
		ID:                 "ast-002",
		AssetCode:          "AST-SRV-2025-012",
		Name:               "Dell PowerEdge R750 Enterprise Server",
		Category:           "IT_Equipment",
		AcquisitionDate:    "2025-08-10",
		OriginalCost:       12500.00,
		CurrentBookValue:   10000.00,
		DepreciationMethod: "Straight-Line 20%",
		CustodianName:      "Systems Administrator",
		Location:           "National Data Center - Dodoma",
		Condition:          "Excellent",
		CreatedTime:        time.Now().Add(-360 * 24 * time.Hour),
	},
}

var defaultExpenseClaims = []ExpenseClaim{
	{
		ID:             "exp-001",
		ClaimNumber:    "EXP-TRV-2026-042",
		StaffName:      "Emanuel Mrema",
		Destination:    "Morogoro Enumeration Supervision Mission",
		PerDiemAmount:  1400.00,
		AdvancePaid:    1400.00,
		ActualExpenses: 1520.00,
		NetBalance:     120.00, // Reimburse employee $120
		Currency:       "USD",
		Status:         "Reconciled",
		SubmittedDate:  "2026-08-10",
		CreatedTime:    time.Now().Add(-5 * 24 * time.Hour),
	},
}

// ─── HTTP Handlers ──────────────────────────────────────────────────

func handleFinanceSummary(c *gin.Context) {
	var totalGrants float64
	var totalDisbursed float64
	for _, g := range defaultFinancialGrants {
		totalGrants += g.TotalBudget
		totalDisbursed += g.DisbursedAmount
	}

	var totalBudgetAllocated float64
	var totalBudgetSpent float64
	for _, b := range defaultBudgets {
		totalBudgetAllocated += b.AllocatedAmount
		totalBudgetSpent += b.SpentAmount
	}

	c.JSON(http.StatusOK, gin.H{
		"total_grant_commitments_usd": totalGrants,
		"total_grant_disbursed_usd":   totalDisbursed,
		"grant_burn_rate_percentage":  (totalDisbursed / totalGrants) * 100,
		"total_institutional_budget":  totalBudgetAllocated,
		"total_budget_spent":          totalBudgetSpent,
		"active_grants_count":         len(defaultFinancialGrants),
		"active_cost_centers_count":   len(defaultBudgets),
		"pending_procurements_count":  len(defaultPurchaseRequests),
		"total_tracked_assets_count":  len(defaultAssets),
		"pending_expense_claims":      len(defaultExpenseClaims),
		"last_financial_sync":         time.Now().Format(time.RFC3339),
	})
}

func handleListFinancialGrants(c *gin.Context) {
	c.JSON(http.StatusOK, defaultFinancialGrants)
}

func handleCreateFinancialGrant(c *gin.Context) {
	var req FinancialGrant
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("grant-%d", time.Now().Unix())
	req.CreatedTime = time.Now()
	if req.Status == "" {
		req.Status = "Active"
	}
	defaultFinancialGrants = append([]FinancialGrant{req}, defaultFinancialGrants...)
	c.JSON(http.StatusCreated, req)
}


func handleListBudgets(c *gin.Context) {
	c.JSON(http.StatusOK, defaultBudgets)
}

func handleCreateBudget(c *gin.Context) {
	var req Budget
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("bgt-%d", time.Now().Unix())
	req.CreatedTime = time.Now()
	req.RemainingAmount = req.AllocatedAmount - (req.CommittedAmount + req.SpentAmount)
	defaultBudgets = append([]Budget{req}, defaultBudgets...)
	c.JSON(http.StatusCreated, req)
}

func handleListPurchaseRequests(c *gin.Context) {
	c.JSON(http.StatusOK, defaultPurchaseRequests)
}

func handleCreatePurchaseRequest(c *gin.Context) {
	var req PurchaseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("pr-%d", time.Now().Unix())
	req.RequisitionNumber = fmt.Sprintf("PR-2026-%03d", len(defaultPurchaseRequests)+1)
	req.CreatedTime = time.Now()
	req.Status = "Submitted"
	defaultPurchaseRequests = append([]PurchaseRequest{req}, defaultPurchaseRequests...)
	c.JSON(http.StatusCreated, req)
}

func handleListPurchaseOrders(c *gin.Context) {
	c.JSON(http.StatusOK, defaultPurchaseOrders)
}

func handleListAssets(c *gin.Context) {
	c.JSON(http.StatusOK, defaultAssets)
}

func handleCreateAsset(c *gin.Context) {
	var req AssetItem
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("ast-%d", time.Now().Unix())
	req.CreatedTime = time.Now()
	if req.CurrentBookValue == 0 {
		req.CurrentBookValue = req.OriginalCost
	}
	defaultAssets = append([]AssetItem{req}, defaultAssets...)
	c.JSON(http.StatusCreated, req)
}

func handleListExpenses(c *gin.Context) {
	c.JSON(http.StatusOK, defaultExpenseClaims)
}

func handleCreateExpense(c *gin.Context) {
	var req ExpenseClaim
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	req.ID = fmt.Sprintf("exp-%d", time.Now().Unix())
	req.ClaimNumber = fmt.Sprintf("EXP-TRV-2026-%03d", len(defaultExpenseClaims)+1)
	req.NetBalance = req.ActualExpenses - req.AdvancePaid
	req.CreatedTime = time.Now()
	req.Status = "Pending_Review"
	defaultExpenseClaims = append([]ExpenseClaim{req}, defaultExpenseClaims...)
	c.JSON(http.StatusCreated, req)
}
