// Package protocol defines the inter-agent communication primitives.
package protocol

import (
	"context"
	"sync"
	"time"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// MsgType categorizes the message purpose.
type MsgType string

const (
	MsgTask     MsgType = "task"     // A sub-task assignment
	MsgResult   MsgType = "result"   // Code/file generation result
	MsgReview   MsgType = "review"   // Code review request/response
	MsgTest     MsgType = "test"     // Test execution request/response
	MsgStatus   MsgType = "status"   // Progress status update
	MsgError    MsgType = "error"    // Error report
	MsgComplete MsgType = "complete" // Agent signals completion
)

// AgentRole identifies an agent's specialty.
type AgentRole string

const (
	RoleSupervisor AgentRole = "supervisor"
	RoleCoder      AgentRole = "coder"
	RoleTester     AgentRole = "tester"
	RoleReviewer   AgentRole = "reviewer"
)

// Message is the unit of communication between agents.
type Message struct {
	ID        string            `json:"id"`
	Sender    AgentRole         `json:"sender"`
	Receiver  AgentRole         `json:"receiver"` // empty = broadcast
	Type      MsgType           `json:"type"`
	Payload   interface{}       `json:"payload"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Timestamp time.Time         `json:"timestamp"`
}

// TaskPayload carries a sub-task assignment from Supervisor to Coder.
type TaskPayload struct {
	TaskID      string `json:"task_id"`
	Description string `json:"description"`
	Context     string `json:"context,omitempty"` // Extra context like related files
	Priority    int    `json:"priority"`          // 1=highest
}

// ResultPayload carries the output from a Coder back to Supervisor.
type ResultPayload struct {
	TaskID       string   `json:"task_id"`
	FilesCreated []string `json:"files_created"`
	Summary      string   `json:"summary"`
	Success      bool     `json:"success"`
	Error        string   `json:"error,omitempty"`
}

// ReviewPayload carries a code review verdict.
type ReviewPayload struct {
	TaskID   string `json:"task_id"`
	Score    int    `json:"score"`    // 0-100
	Passed   bool   `json:"passed"`
	Feedback string `json:"feedback"`
}

// TestPayload carries test execution results.
type TestPayload struct {
	TaskID     string `json:"task_id"`
	Passed     bool   `json:"passed"`
	TotalTests int    `json:"total_tests"`
	FailedTests int   `json:"failed_tests"`
	Output     string `json:"output"`
}

// EventBus provides publish/subscribe messaging between agents.
// Thread-safe. Multiple goroutines can subscribe and publish concurrently.
type EventBus struct {
	mu          sync.RWMutex
	subscribers map[AgentRole][]chan Message
	allSubs     []chan Message // broadcast subscribers
	closed      bool
}

// NewEventBus creates a new event bus.
func NewEventBus() *EventBus {
	return &EventBus{
		subscribers: make(map[AgentRole][]chan Message),
	}
}

// Subscribe registers a channel to receive messages for a specific role.
// Buffer size determines how many messages can queue before blocking.
func (eb *EventBus) Subscribe(role AgentRole, bufSize int) <-chan Message {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Message, bufSize)
	eb.subscribers[role] = append(eb.subscribers[role], ch)
	logger.L().Infow("📡 EventBus: subscriber registered", "role", role)
	return ch
}

// SubscribeAll registers a channel to receive ALL messages (for UI/logging).
func (eb *EventBus) SubscribeAll(bufSize int) <-chan Message {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	ch := make(chan Message, bufSize)
	eb.allSubs = append(eb.allSubs, ch)
	return ch
}

// Publish sends a message. If Receiver is set, only that role's subscribers
// get it. If Receiver is empty, all role subscribers get it.
// All broadcast subscribers always receive every message.
func (eb *EventBus) Publish(ctx context.Context, msg Message) {
	eb.mu.RLock()
	defer eb.mu.RUnlock()

	if eb.closed {
		return
	}

	msg.Timestamp = time.Now()

	// Send to targeted subscribers
	if msg.Receiver != "" {
		for _, ch := range eb.subscribers[msg.Receiver] {
			select {
			case ch <- msg:
			case <-ctx.Done():
				return
			default:
				logger.L().Warnw("EventBus: message dropped (buffer full)",
					"sender", msg.Sender, "receiver", msg.Receiver, "type", msg.Type)
			}
		}
	} else {
		// Broadcast to all role subscribers
		for _, subs := range eb.subscribers {
			for _, ch := range subs {
				select {
				case ch <- msg:
				default:
				}
			}
		}
	}

	// Always send to broadcast subscribers (UI, logging)
	for _, ch := range eb.allSubs {
		select {
		case ch <- msg:
		default:
		}
	}
}

// Close shuts down the event bus and closes all subscriber channels.
func (eb *EventBus) Close() {
	eb.mu.Lock()
	defer eb.mu.Unlock()

	if eb.closed {
		return
	}
	eb.closed = true

	for _, subs := range eb.subscribers {
		for _, ch := range subs {
			close(ch)
		}
	}
	for _, ch := range eb.allSubs {
		close(ch)
	}
	logger.L().Info("📡 EventBus: closed")
}
