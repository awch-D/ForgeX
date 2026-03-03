package pool_test

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/awch-D/ForgeX/forgex-agent/pool"
	"github.com/awch-D/ForgeX/forgex-agent/protocol"
)

// mockAgent implements pool.Agent for testing.
type mockAgent struct {
	role    protocol.AgentRole
	runFunc func(ctx context.Context) error
}

func (m *mockAgent) Role() protocol.AgentRole      { return m.role }
func (m *mockAgent) Run(ctx context.Context) error { return m.runFunc(ctx) }

func TestPool_RegisterAndStart(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()
	p := pool.NewPool(bus)

	var called atomic.Int32
	agent := &mockAgent{
		role: protocol.RoleCoder,
		runFunc: func(ctx context.Context) error {
			called.Add(1)
			return nil
		},
	}

	p.Register(agent)
	p.Start(context.Background())
	p.Wait()

	if called.Load() != 1 {
		t.Errorf("expected agent Run to be called once, got %d", called.Load())
	}
}

func TestPool_MultipleAgents(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()
	p := pool.NewPool(bus)

	var count atomic.Int32
	for _, role := range []protocol.AgentRole{protocol.RoleCoder, protocol.RoleTester, protocol.RoleReviewer} {
		r := role
		p.Register(&mockAgent{
			role: r,
			runFunc: func(ctx context.Context) error {
				count.Add(1)
				return nil
			},
		})
	}

	p.Start(context.Background())
	p.Wait()

	if count.Load() != 3 {
		t.Errorf("expected 3 agents to run, got %d", count.Load())
	}
}

func TestPool_Shutdown(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()
	p := pool.NewPool(bus)

	started := make(chan struct{})
	agent := &mockAgent{
		role: protocol.RoleCoder,
		runFunc: func(ctx context.Context) error {
			close(started)
			<-ctx.Done() // Block until cancelled
			return ctx.Err()
		},
	}

	p.Register(agent)
	p.Start(context.Background())

	<-started // Wait for agent to actually start
	p.Shutdown()
	// If we get here, Shutdown worked without deadlocking
}

func TestPool_PanicRecovery(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	// Subscribe to receive error messages from panicked agents
	allCh := bus.SubscribeAll(10)
	p := pool.NewPool(bus)

	agent := &mockAgent{
		role: protocol.RoleCoder,
		runFunc: func(ctx context.Context) error {
			panic("test panic!")
		},
	}

	p.Register(agent)
	p.Start(context.Background())
	p.Wait() // Should not deadlock despite panic

	// Check that an error message was published
	select {
	case msg := <-allCh:
		if msg.Type != protocol.MsgError {
			t.Errorf("expected MsgError, got %v", msg.Type)
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatal("no error message received after agent panic")
	}
}

func TestPool_WaitForRole(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()
	p := pool.NewPool(bus)

	coderDone := make(chan struct{})
	p.Register(&mockAgent{
		role: protocol.RoleCoder,
		runFunc: func(ctx context.Context) error {
			time.Sleep(50 * time.Millisecond)
			close(coderDone)
			return nil
		},
	})

	// Tester will block until cancelled
	p.Register(&mockAgent{
		role: protocol.RoleTester,
		runFunc: func(ctx context.Context) error {
			<-ctx.Done()
			return nil
		},
	})

	p.Start(context.Background())

	// WaitForRole should return once coder finishes (even though tester still runs)
	p.WaitForRole(protocol.RoleCoder)
	select {
	case <-coderDone:
		// OK
	default:
		t.Fatal("WaitForRole returned before coder finished")
	}

	p.Shutdown()
}

func TestPool_AgentError(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()
	p := pool.NewPool(bus)

	p.Register(&mockAgent{
		role: protocol.RoleCoder,
		runFunc: func(ctx context.Context) error {
			return errors.New("something went wrong")
		},
	})

	p.Start(context.Background())
	p.Wait() // Should not deadlock even with error
}
