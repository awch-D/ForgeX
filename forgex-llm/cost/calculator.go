// Package cost provides token tracking and pricing calculations.
package cost

import (
	"sync"
	"time"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Ledger tracks total tokens and estimated cost in USD.
// This is thread-safe for concurrent agent usage.
type Ledger struct {
	mu           sync.Mutex
	PromptTokens int
	OutputTokens int
	TotalTokens  int
	TotalCostUSD float64

	modelPrices map[string]Pricing
}

// Pricing defines the cost per 1M tokens.
type Pricing struct {
	InputPer1M  float64
	OutputPer1M float64
}

// Global defaults for cost calculation. Actual prices can be updated later.
var defaultPricing = map[string]Pricing{
	"gpt-4o":               {InputPer1M: 5.0, OutputPer1M: 15.0},
	"claude-3-5-sonnet":    {InputPer1M: 3.0, OutputPer1M: 15.0},
	"deepseek-coder-v2":    {InputPer1M: 0.14, OutputPer1M: 0.28},
	"ollama":               {InputPer1M: 0.0, OutputPer1M: 0.0},
}

var (
	globalLedger *Ledger
	once         sync.Once
)

// Global returns a thread-safe global cost ledger.
func Global() *Ledger {
	once.Do(func() {
		globalLedger = &Ledger{
			modelPrices: defaultPricing,
		}
	})
	return globalLedger
}

// Add records token usage and calculates cost for a specific model.
func (l *Ledger) Add(model string, promptTokens, outputTokens int) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.PromptTokens += promptTokens
	l.OutputTokens += outputTokens
	l.TotalTokens += promptTokens + outputTokens

	price, ok := l.modelPrices[model]
	if !ok {
		// Fallback to a default generic pricing if unknown model
		price = Pricing{InputPer1M: 1.0, OutputPer1M: 3.0}
	}

	cost := (float64(promptTokens)/1_000_000.0)*price.InputPer1M +
		(float64(outputTokens)/1_000_000.0)*price.OutputPer1M
	
	l.TotalCostUSD += cost

	go func(c float64) {
	    logger.L().Debugw("Token usage recorded",
		    "model", model,
		    "prompt", promptTokens,
		    "output", outputTokens,
		    "cost_usd", c,
	    )
    }(cost)
}

// Summary returns a snapshot of current usage.
func (l *Ledger) Summary() (int, float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.TotalTokens, l.TotalCostUSD
}

// Reset clears the ledger (useful for beginning a new top-level task).
func (l *Ledger) Reset() {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.PromptTokens = 0
	l.OutputTokens = 0
	l.TotalTokens = 0
	l.TotalCostUSD = 0
	logger.L().Infow("Cost ledger reset", "time", time.Now().Format(time.RFC3339))
}
