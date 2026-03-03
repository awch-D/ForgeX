// Package budget provides token and cost budget enforcement.
// It monitors LLM spending and halts execution when limits are exceeded.
package budget

import (
	"fmt"
	"sync"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-llm/cost"
)

// ErrBudgetExceeded is returned when the cost exceeds the configured maximum.
var ErrBudgetExceeded = fmt.Errorf("budget exceeded")

// Guard monitors cost and enforces budget limits.
type Guard struct {
	mu        sync.RWMutex
	maxBudget float64
	ledger    *cost.Ledger
	exceeded  bool
}

// NewGuard creates a new budget guard with the given maximum budget in USD.
func NewGuard(maxBudget float64, ledger *cost.Ledger) *Guard {
	return &Guard{
		maxBudget: maxBudget,
		ledger:    ledger,
	}
}

// Check verifies that the current spending is within budget.
// Returns ErrBudgetExceeded if the budget has been exceeded.
func (g *Guard) Check() error {
	g.mu.RLock()
	if g.exceeded {
		g.mu.RUnlock()
		return ErrBudgetExceeded
	}
	g.mu.RUnlock()

	_, currentCost := g.ledger.Summary()
	if currentCost > g.maxBudget {
		g.mu.Lock()
		g.exceeded = true
		g.mu.Unlock()

		logger.L().Warnw("💰 Budget exceeded!",
			"current", fmt.Sprintf("$%.4f", currentCost),
			"max", fmt.Sprintf("$%.4f", g.maxBudget),
		)
		return fmt.Errorf("%w: current $%.4f exceeds max $%.4f",
			ErrBudgetExceeded, currentCost, g.maxBudget)
	}
	return nil
}

// Remaining returns the remaining budget in USD.
func (g *Guard) Remaining() float64 {
	_, currentCost := g.ledger.Summary()
	remaining := g.maxBudget - currentCost
	if remaining < 0 {
		return 0
	}
	return remaining
}

// Usage returns a formatted string showing current usage vs max.
func (g *Guard) Usage() string {
	_, currentCost := g.ledger.Summary()
	pct := 0.0
	if g.maxBudget > 0 {
		pct = (currentCost / g.maxBudget) * 100
	}
	return fmt.Sprintf("$%.4f / $%.4f (%.1f%%)", currentCost, g.maxBudget, pct)
}

// MaxBudget returns the configured maximum budget.
func (g *Guard) MaxBudget() float64 {
	return g.maxBudget
}
