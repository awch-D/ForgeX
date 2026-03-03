package budget_test

import (
	"errors"
	"testing"

	"github.com/awch-D/ForgeX/forgex-governance/budget"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
)

func TestGuard_WithinBudget(t *testing.T) {
	ledger := cost.Global()
	ledger.Reset()

	guard := budget.NewGuard(1.0, ledger)

	if err := guard.Check(); err != nil {
		t.Errorf("should be within budget, got: %v", err)
	}
}

func TestGuard_ExceedsBudget(t *testing.T) {
	ledger := cost.Global()
	ledger.Reset()

	// Set a very low budget
	guard := budget.NewGuard(0.0001, ledger)

	// Add enough tokens to exceed budget
	ledger.Add("gpt-4o", 10000, 5000)

	err := guard.Check()
	if err == nil {
		t.Fatal("should exceed budget but no error returned")
	}
	if !errors.Is(err, budget.ErrBudgetExceeded) {
		t.Errorf("expected ErrBudgetExceeded, got: %v", err)
	}
}

func TestGuard_Remaining(t *testing.T) {
	ledger := cost.Global()
	ledger.Reset()

	guard := budget.NewGuard(10.0, ledger)

	remaining := guard.Remaining()
	if remaining != 10.0 {
		t.Errorf("expected 10.0 remaining, got %f", remaining)
	}
}

func TestGuard_Usage(t *testing.T) {
	ledger := cost.Global()
	ledger.Reset()

	guard := budget.NewGuard(10.0, ledger)
	usage := guard.Usage()
	if usage == "" {
		t.Error("usage string should not be empty")
	}
}
