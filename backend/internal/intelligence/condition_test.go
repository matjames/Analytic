package intelligence

import (
	"context"
	"testing"
)

func TestEvaluateConditionOptimal(t *testing.T) {
	calc := NewConditionCalculator()
	cond := calc.EvaluateCondition(context.Background(), "tenant-alpha", 0, 0, 95.0, 92.0, 99.0)

	if cond == nil {
		t.Fatalf("Expected non-nil InstitutionalCondition")
	}
	if cond.Level != LevelOptimal {
		t.Errorf("Expected LevelOptimal, got %s", cond.Level)
	}
	if cond.CompositeScore < 90.0 {
		t.Errorf("Expected composite >= 90.0, got %f", cond.CompositeScore)
	}
	if len(cond.DomainScores) != 4 {
		t.Errorf("Expected 4 domain scores, got %d", len(cond.DomainScores))
	}
}

func TestEvaluateConditionElevated(t *testing.T) {
	calc := NewConditionCalculator()
	cond := calc.EvaluateCondition(context.Background(), "tenant-alpha", 3, 2, 65.0, 70.0, 75.0)

	if cond == nil {
		t.Fatalf("Expected non-nil InstitutionalCondition")
	}
	if cond.Level == LevelOptimal || cond.Level == LevelNominal {
		t.Errorf("Expected Warning/Elevated level on degraded inputs, got %s", cond.Level)
	}
	if len(cond.Recommendations) == 0 {
		t.Errorf("Expected actionable recommendations on elevated condition")
	}
}
