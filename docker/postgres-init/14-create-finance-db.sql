-- ═══════════════════════════════════════════════════════════════════
-- PHASE 14 — FINANCIAL MANAGEMENT, GRANTS, PROCUREMENT & ASSET MANAGEMENT
-- ═══════════════════════════════════════════════════════════════════

-- Grants & Multi-Donor Funding
CREATE TABLE IF NOT EXISTS financial_grants (
    id VARCHAR(64) PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    grant_code VARCHAR(64) UNIQUE NOT NULL,
    donor_name VARCHAR(255) NOT NULL,
    total_budget NUMERIC(15, 2) NOT NULL,
    disbursed_amount NUMERIC(15, 2) DEFAULT 0.00,
    currency VARCHAR(10) DEFAULT 'USD',
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    project_id VARCHAR(64),
    status VARCHAR(50) DEFAULT 'Active',
    compliance_rules JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Institutional Budget & Cost Centers
CREATE TABLE IF NOT EXISTS budgets (
    id VARCHAR(64) PRIMARY KEY,
    fiscal_year VARCHAR(32) NOT NULL,
    cost_center_code VARCHAR(64) NOT NULL,
    cost_center_name VARCHAR(255) NOT NULL,
    allocated_amount NUMERIC(15, 2) NOT NULL,
    committed_amount NUMERIC(15, 2) DEFAULT 0.00,
    spent_amount NUMERIC(15, 2) DEFAULT 0.00,
    remaining_amount NUMERIC(15, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'TZS',
    status VARCHAR(50) DEFAULT 'Approved',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Procurement Requisitions
CREATE TABLE IF NOT EXISTS purchase_requests (
    id VARCHAR(64) PRIMARY KEY,
    requisition_number VARCHAR(64) UNIQUE NOT NULL,
    requestor_name VARCHAR(255) NOT NULL,
    department VARCHAR(255) NOT NULL,
    item_description TEXT NOT NULL,
    estimated_total NUMERIC(15, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    justification TEXT,
    status VARCHAR(50) DEFAULT 'Submitted',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Purchase Orders & Contracts
CREATE TABLE IF NOT EXISTS purchase_orders (
    id VARCHAR(64) PRIMARY KEY,
    po_number VARCHAR(64) UNIQUE NOT NULL,
    vendor_name VARCHAR(255) NOT NULL,
    requisition_id VARCHAR(64) REFERENCES purchase_requests(id) ON DELETE SET NULL,
    total_amount NUMERIC(15, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    delivery_status VARCHAR(50) DEFAULT 'Pending',
    payment_terms VARCHAR(255),
    issued_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Capital Asset Registry & Depreciation
CREATE TABLE IF NOT EXISTS asset_registry (
    id VARCHAR(64) PRIMARY KEY,
    asset_code VARCHAR(64) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100) NOT NULL,
    acquisition_date DATE NOT NULL,
    original_cost NUMERIC(15, 2) NOT NULL,
    current_book_value NUMERIC(15, 2) NOT NULL,
    depreciation_method VARCHAR(100) DEFAULT 'Straight-Line 20%',
    custodian_name VARCHAR(255),
    location VARCHAR(255),
    condition VARCHAR(50) DEFAULT 'Good',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Travel Advances & Expense Claims
CREATE TABLE IF NOT EXISTS expense_claims (
    id VARCHAR(64) PRIMARY KEY,
    claim_number VARCHAR(64) UNIQUE NOT NULL,
    staff_name VARCHAR(255) NOT NULL,
    destination VARCHAR(255) NOT NULL,
    per_diem_amount NUMERIC(12, 2) DEFAULT 0.00,
    advance_paid NUMERIC(12, 2) DEFAULT 0.00,
    actual_expenses NUMERIC(12, 2) NOT NULL,
    net_balance NUMERIC(12, 2) NOT NULL,
    currency VARCHAR(10) DEFAULT 'USD',
    status VARCHAR(50) DEFAULT 'Pending_Review',
    submitted_date DATE DEFAULT CURRENT_DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);
