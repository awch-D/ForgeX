// Package e2e_test provides end-to-end integration tests for the multi-agent system.
// It wires up Pool + EventBus + mock agents to verify the full collaboration flow.
package e2e_test

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/awch-D/ForgeX/forgex-agent/pool"
	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-cognition/draft"
)

// --- Mock Agents for e2e testing ---

// supervisorAgent decomposes a task and dispatches sub-tasks to coder.
type supervisorAgent struct {
	bus   *protocol.EventBus
	inbox <-chan protocol.Message
}

func (a *supervisorAgent) Role() protocol.AgentRole { return protocol.RoleSupervisor }
func (a *supervisorAgent) Run(ctx context.Context) error {
	// Step 1: Decompose task into 2 sub-tasks
	a.bus.Publish(ctx, protocol.Message{
		ID:       "task-1",
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
		Payload: protocol.TaskPayload{
			TaskID:      "task-1",
			Description: "Create main.go",
			Priority:    1,
		},
	})

	a.bus.Publish(ctx, protocol.Message{
		ID:       "task-2",
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
		Payload: protocol.TaskPayload{
			TaskID:      "task-2",
			Description: "Create handler.go",
			Priority:    2,
		},
	})

	// Step 2: Wait for 2 results
	received := 0
	for received < 2 {
		select {
		case msg := <-a.inbox:
			if msg.Type == protocol.MsgResult {
				received++
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

	// Step 3: Request test
	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleTester,
		Type:     protocol.MsgTest,
	})

	// Wait for test result
	select {
	case <-a.inbox:
	case <-ctx.Done():
		return ctx.Err()
	}

	// Step 4: Signal completion
	a.bus.Publish(ctx, protocol.Message{
		Sender: protocol.RoleSupervisor,
		Type:   protocol.MsgComplete,
	})
	return nil
}

// coderAgent receives tasks and produces results.
type coderAgent struct {
	bus          *protocol.EventBus
	inbox        <-chan protocol.Message
	drafts       *draft.Store
	filesCreated []string
	mu           sync.Mutex
}

func (a *coderAgent) Role() protocol.AgentRole { return protocol.RoleCoder }
func (a *coderAgent) Run(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if msg.Type == protocol.MsgTask {
				payloadJSON, _ := json.Marshal(msg.Payload)
				var task protocol.TaskPayload
				json.Unmarshal(payloadJSON, &task)

				// Simulate writing file
				fileName := task.TaskID + ".go"
				a.mu.Lock()
				a.filesCreated = append(a.filesCreated, fileName)
				a.mu.Unlock()

				// Store intermediate work in draft
				a.drafts.Set(protocol.RoleCoder, task.TaskID, "code for "+task.Description)

				// Report result
				a.bus.Publish(ctx, protocol.Message{
					Sender:   protocol.RoleCoder,
					Receiver: protocol.RoleSupervisor,
					Type:     protocol.MsgResult,
					Payload: protocol.ResultPayload{
						TaskID:       task.TaskID,
						FilesCreated: []string{fileName},
						Summary:      "Created " + fileName,
						Success:      true,
					},
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// testerAgent receives test requests and reports success.
type testerAgent struct {
	bus   *protocol.EventBus
	inbox <-chan protocol.Message
}

func (a *testerAgent) Role() protocol.AgentRole { return protocol.RoleTester }
func (a *testerAgent) Run(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if msg.Type == protocol.MsgTest {
				a.bus.Publish(ctx, protocol.Message{
					Sender:   protocol.RoleTester,
					Receiver: protocol.RoleSupervisor,
					Type:     protocol.MsgTest,
					Payload: protocol.TestPayload{
						TaskID:      "test-all",
						Passed:      true,
						TotalTests:  2,
						FailedTests: 0,
						Output:      "all tests passed",
					},
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// --- End-to-End Tests ---

func TestMultiAgent_FullWorkflow(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	drafts := draft.NewStore()

	// Track all messages for verification
	allMsgs := bus.SubscribeAll(100)

	// Create agents
	supInbox := bus.Subscribe(protocol.RoleSupervisor, 50)
	sup := &supervisorAgent{bus: bus, inbox: supInbox}

	coderInbox := bus.Subscribe(protocol.RoleCoder, 50)
	coder := &coderAgent{bus: bus, inbox: coderInbox, drafts: drafts}

	testerInbox := bus.Subscribe(protocol.RoleTester, 50)
	tester := &testerAgent{bus: bus, inbox: testerInbox}

	// Setup pool
	p := pool.NewPool(bus)
	p.Register(sup)
	p.Register(coder)
	p.Register(tester)

	// Run with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.Start(ctx)

	// Wait for supervisor to finish (it drives the workflow)
	p.WaitForRole(protocol.RoleSupervisor)
	p.Shutdown()

	// === Verification ===

	// 1. Check coder produced files
	coder.mu.Lock()
	if len(coder.filesCreated) != 2 {
		t.Errorf("expected 2 files created, got %d: %v", len(coder.filesCreated), coder.filesCreated)
	}
	coder.mu.Unlock()

	// 2. Check draft store has coder's intermediate work
	val1 := drafts.GetString(protocol.RoleCoder, "task-1")
	val2 := drafts.GetString(protocol.RoleCoder, "task-2")
	if val1 == "" || val2 == "" {
		t.Errorf("expected draft entries for both tasks, got %q and %q", val1, val2)
	}

	// 3. Check message flow: task → result → test → complete
	var msgTypes []protocol.MsgType
	for {
		select {
		case msg := <-allMsgs:
			msgTypes = append(msgTypes, msg.Type)
		default:
			goto verify
		}
	}
verify:

	// Should have at least: 2 tasks + 2 results + 1 test request + 1 test response + 1 complete
	if len(msgTypes) < 7 {
		t.Errorf("expected at least 7 messages, got %d: %v", len(msgTypes), msgTypes)
	}

	// Verify completion message exists
	hasComplete := false
	for _, mt := range msgTypes {
		if mt == protocol.MsgComplete {
			hasComplete = true
		}
	}
	if !hasComplete {
		t.Error("expected MsgComplete in message flow")
	}
}

func TestMultiAgent_TimeoutHandling(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	// Create an agent that blocks forever
	blockerInbox := bus.Subscribe(protocol.RoleSupervisor, 10)
	blocker := &supervisorAgent{bus: bus, inbox: blockerInbox}

	p := pool.NewPool(bus)
	p.Register(blocker)

	// Very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	p.Start(ctx)

	// Should complete without hanging
	done := make(chan struct{})
	go func() {
		p.Wait()
		close(done)
	}()

	select {
	case <-done:
		// OK : agents exited on context cancellation
	case <-time.After(2 * time.Second):
		t.Fatal("agents did not exit on context cancellation")
	}
}

// --- Reviewer Agent mock ---

type reviewerAgent struct {
	bus   *protocol.EventBus
	inbox <-chan protocol.Message
}

func (a *reviewerAgent) Role() protocol.AgentRole { return protocol.RoleReviewer }
func (a *reviewerAgent) Run(ctx context.Context) error {
	for {
		select {
		case msg, ok := <-a.inbox:
			if !ok {
				return nil
			}
			if msg.Type == protocol.MsgReview {
				a.bus.Publish(ctx, protocol.Message{
					Sender:   protocol.RoleReviewer,
					Receiver: protocol.RoleSupervisor,
					Type:     protocol.MsgReview,
					Payload: protocol.ReviewPayload{
						TaskID:   "review-all",
						Score:    85,
						Passed:   true,
						Feedback: "Code quality is good, minor style issues.",
					},
				})
			}
		case <-ctx.Done():
			return nil
		}
	}
}

// --- Supervisor with Reviewer support ---

type supervisorWithReviewerAgent struct {
	bus            *protocol.EventBus
	inbox          <-chan protocol.Message
	reviewReceived bool
	mu             sync.Mutex
}

func (a *supervisorWithReviewerAgent) Role() protocol.AgentRole { return protocol.RoleSupervisor }
func (a *supervisorWithReviewerAgent) Run(ctx context.Context) error {
	// Step 1: Dispatch task to coder
	a.bus.Publish(ctx, protocol.Message{
		ID:       "task-review-1",
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleCoder,
		Type:     protocol.MsgTask,
		Payload: protocol.TaskPayload{
			TaskID:      "task-review-1",
			Description: "Create service.go",
			Priority:    1,
		},
	})

	// Step 2: Wait for coder result
	for {
		select {
		case msg := <-a.inbox:
			if msg.Type == protocol.MsgResult {
				goto testPhase
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

testPhase:
	// Step 3: Request test
	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleTester,
		Type:     protocol.MsgTest,
	})

	for {
		select {
		case msg := <-a.inbox:
			if msg.Type == protocol.MsgTest {
				goto reviewPhase
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

reviewPhase:
	// Step 4: Request review (NEW - tests Reviewer integration)
	a.bus.Publish(ctx, protocol.Message{
		Sender:   protocol.RoleSupervisor,
		Receiver: protocol.RoleReviewer,
		Type:     protocol.MsgReview,
	})

	for {
		select {
		case msg := <-a.inbox:
			if msg.Type == protocol.MsgReview {
				a.mu.Lock()
				a.reviewReceived = true
				a.mu.Unlock()
				goto complete
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}

complete:
	a.bus.Publish(ctx, protocol.Message{
		Sender: protocol.RoleSupervisor,
		Type:   protocol.MsgComplete,
	})
	return nil
}

func TestMultiAgent_ReviewerIntegration(t *testing.T) {
	bus := protocol.NewEventBus()
	defer bus.Close()

	drafts := draft.NewStore()
	allMsgs := bus.SubscribeAll(100)

	supInbox := bus.Subscribe(protocol.RoleSupervisor, 50)
	sup := &supervisorWithReviewerAgent{bus: bus, inbox: supInbox}

	coderInbox := bus.Subscribe(protocol.RoleCoder, 50)
	cdr := &coderAgent{bus: bus, inbox: coderInbox, drafts: drafts}

	testerInbox := bus.Subscribe(protocol.RoleTester, 50)
	tstr := &testerAgent{bus: bus, inbox: testerInbox}

	reviewerInbox := bus.Subscribe(protocol.RoleReviewer, 50)
	rvw := &reviewerAgent{bus: bus, inbox: reviewerInbox}

	p := pool.NewPool(bus)
	p.Register(sup)
	p.Register(cdr)
	p.Register(tstr)
	p.Register(rvw)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	p.Start(ctx)
	p.WaitForRole(protocol.RoleSupervisor)
	p.Shutdown()

	// Verify Supervisor received review
	sup.mu.Lock()
	if !sup.reviewReceived {
		t.Error("expected supervisor to receive review feedback from reviewer")
	}
	sup.mu.Unlock()

	// Verify message flow includes MsgReview
	var hasReview, hasComplete bool
	for {
		select {
		case msg := <-allMsgs:
			if msg.Type == protocol.MsgReview {
				hasReview = true
			}
			if msg.Type == protocol.MsgComplete {
				hasComplete = true
			}
		default:
			goto done
		}
	}
done:
	if !hasReview {
		t.Error("expected MsgReview in message flow")
	}
	if !hasComplete {
		t.Error("expected MsgComplete in message flow")
	}
}

func TestMultiAgent_EventBusClosedGracefully(t *testing.T) {
	bus := protocol.NewEventBus()

	// Subscribe before close
	inbox := bus.Subscribe(protocol.RoleSupervisor, 10)

	bus.Close()

	// After close, channel should be closed and readable
	select {
	case _, ok := <-inbox:
		if ok {
			t.Error("expected channel to be closed after EventBus.Close()")
		}
	case <-time.After(1 * time.Second):
		t.Fatal("channel not closed within timeout")
	}
}
