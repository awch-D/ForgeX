// Package pool manages Agent goroutine lifecycles.
package pool

import (
	"context"
	"fmt"
	"sync"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Agent is the interface all agents must implement.
type Agent interface {
	// Role returns the agent's role identifier.
	Role() protocol.AgentRole
	// Run starts the agent's main loop. It should block until ctx is cancelled
	// or the agent completes its work. Errors are reported via the event bus.
	Run(ctx context.Context) error
}

// Pool manages a set of agent goroutines.
type Pool struct {
	mu     sync.Mutex
	agents []Agent
	bus    *protocol.EventBus
	wg     sync.WaitGroup
	cancel context.CancelFunc
}

// NewPool creates a new agent pool with the given event bus.
func NewPool(bus *protocol.EventBus) *Pool {
	return &Pool{
		bus: bus,
	}
}

// Register adds an agent to the pool. Must be called before Start.
func (p *Pool) Register(agent Agent) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.agents = append(p.agents, agent)
	logger.L().Infow("🏊 Pool: agent registered", "role", agent.Role())
}

// Start launches all registered agents in separate goroutines.
func (p *Pool) Start(parentCtx context.Context) {
	ctx, cancel := context.WithCancel(parentCtx)
	p.cancel = cancel

	for _, a := range p.agents {
		p.wg.Add(1)
		go p.runAgent(ctx, a)
	}

	logger.L().Infow("🏊 Pool: all agents launched", "count", len(p.agents))
}

// Wait blocks until all agents finish.
func (p *Pool) Wait() {
	p.wg.Wait()
}

// Shutdown signals all agents to stop and waits for them.
func (p *Pool) Shutdown() {
	if p.cancel != nil {
		p.cancel()
	}
	p.wg.Wait()
	logger.L().Info("🏊 Pool: all agents stopped")
}

func (p *Pool) runAgent(ctx context.Context, a Agent) {
	defer p.wg.Done()
	defer func() {
		if r := recover(); r != nil {
			logger.L().Errorw("💥 Agent panicked!", "role", a.Role(), "panic", fmt.Sprint(r))
			// Publish error to bus so Supervisor knows
			p.bus.Publish(ctx, protocol.Message{
				Sender:  a.Role(),
				Type:    protocol.MsgError,
				Payload: fmt.Sprintf("agent %s panicked: %v", a.Role(), r),
			})
		}
	}()

	logger.L().Infow("▶️  Agent started", "role", a.Role())
	if err := a.Run(ctx); err != nil {
		logger.L().Errorw("Agent exited with error", "role", a.Role(), "error", err)
		p.bus.Publish(ctx, protocol.Message{
			Sender:  a.Role(),
			Type:    protocol.MsgError,
			Payload: err.Error(),
		})
	} else {
		logger.L().Infow("✅ Agent completed", "role", a.Role())
	}
}
