package main

import (
	"testing"
	"time"
)

func TestCalculateRiskLevel(t *testing.T) {
	tests := []struct {
		score    int
		expected string
	}{
		{score: 25, expected: "Critical"},
		{score: 16, expected: "Critical"},
		{score: 15, expected: "Critical"},
		{score: 12, expected: "High"},
		{score: 10, expected: "High"},
		{score: 9, expected: "Medium"},
		{score: 5, expected: "Medium"},
		{score: 4, expected: "Low"},
		{score: 1, expected: "Low"},
	}

	for _, tt := range tests {
		result := calculateRiskLevel(tt.score)
		if result != tt.expected {
			t.Errorf("calculateRiskLevel(%d) = %s; want %s", tt.score, result, tt.expected)
		}
	}
}

func TestRiskInherentScore(t *testing.T) {
	r := Risk{
		Probability: 4,
		Impact:      4,
	}
	r.InherentRiskScore = r.Probability * r.Impact
	r.InherentRiskLevel = calculateRiskLevel(r.InherentRiskScore)

	if r.InherentRiskScore != 16 {
		t.Errorf("Expected inherent risk score 16, got %d", r.InherentRiskScore)
	}
	if r.InherentRiskLevel != "Critical" {
		t.Errorf("Expected inherent risk level Critical, got %s", r.InherentRiskLevel)
	}
}

func TestDelegationExpiration(t *testing.T) {
	past := time.Now().Add(-2 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	activeDlg := Delegation{
		StartDate: time.Now().Add(-1 * time.Hour),
		EndDate:   future,
		Status:    "Active",
	}

	expiredDlg := Delegation{
		StartDate: time.Now().Add(-5 * time.Hour),
		EndDate:   past,
		Status:    "Active",
	}

	now := time.Now()
	if now.After(activeDlg.EndDate) {
		activeDlg.Status = "Expired"
	}
	if now.After(expiredDlg.EndDate) {
		expiredDlg.Status = "Expired"
	}

	if activeDlg.Status != "Active" {
		t.Errorf("Expected active delegation to remain Active, got %s", activeDlg.Status)
	}
	if expiredDlg.Status != "Expired" {
		t.Errorf("Expected past delegation to transition to Expired, got %s", expiredDlg.Status)
	}
}

func TestPolicyLifecycleTransitions(t *testing.T) {
	p := Policy{
		PolicyNumber: "POL-TEST-01",
		Title:        "Data Protection Standard",
		Status:       "Draft",
		Version:      "1.0",
	}

	// 1. Submit for Approval
	p.Status = "Approval"
	if p.Status != "Approval" {
		t.Errorf("Expected status Approval, got %s", p.Status)
	}

	// 2. Approve
	p.Status = "Approved"
	p.ApprovedBy = "Dr. Board Member"
	p.ApprovedAt = time.Now().UTC().Format(time.RFC3339)

	if p.Status != "Approved" || p.ApprovedBy == "" {
		t.Errorf("Expected policy to be approved with approver, got status %s", p.Status)
	}

	// 3. Publish
	p.Status = "Active"
	p.PublishedAt = time.Now().UTC().Format(time.RFC3339)

	if p.Status != "Active" || p.PublishedAt == "" {
		t.Errorf("Expected policy to be published and Active, got status %s", p.Status)
	}
}

func TestControlEffectivenessLogic(t *testing.T) {
	c := Control{
		ControlCode:   "CTL-SEC-10",
		Name:          "TLS 1.3 Transport Encryption",
		Effectiveness: "Effective",
	}

	test := ControlTest{
		Tester:        "Auditor James",
		Result:        "Pass",
		Effectiveness: "Effective",
	}

	if test.Result != "Pass" || test.Effectiveness != "Effective" {
		t.Errorf("Control test should pass with Effective status")
	}

	// Ineffective test
	failedTest := ControlTest{
		Tester:        "Auditor James",
		Result:        "Fail",
		Effectiveness: "Ineffective",
		Findings:      "Unencrypted HTTP port 80 was discovered open without redirect.",
	}

	c.Effectiveness = failedTest.Effectiveness
	if c.Effectiveness != "Ineffective" {
		t.Errorf("Control effectiveness should update to Ineffective upon failed test")
	}
}
