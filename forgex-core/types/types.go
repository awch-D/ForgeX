// Package types provides shared type definitions for ForgeX.
package types

// TaskLevel represents the complexity gear level.
type TaskLevel int

const (
	L1 TaskLevel = iota + 1 // Simple: bug fix, < 3 files
	L2                       // Medium: feature module, 3-20 files
	L3                       // Complex: core architecture, 20+ files
	L4                       // Cross-system: multi-project refactor
)

func (l TaskLevel) String() string {
	switch l {
	case L1:
		return "🟢 L1 (Simple)"
	case L2:
		return "🟡 L2 (Medium)"
	case L3:
		return "🔴 L3 (Complex)"
	case L4:
		return "🟣 L4 (Cross-system)"
	default:
		return "Unknown"
	}
}

// Task represents a high-level task from the user.
type Task struct {
	ID          string    `json:"id"`
	Description string    `json:"description"`
	Level       TaskLevel `json:"level"`
	Status      TaskStatus `json:"status"`
}

// TaskStatus represents the current state of a task.
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusPlanning  TaskStatus = "planning"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusReview    TaskStatus = "review"
	TaskStatusDone      TaskStatus = "done"
	TaskStatusFailed    TaskStatus = "failed"
)

// AgentRole defines the role of an Agent in the pool.
type AgentRole string

const (
	RoleSupervisor AgentRole = "supervisor"
	RoleCoder      AgentRole = "coder"
	RoleTester     AgentRole = "tester"
	RoleReviewer   AgentRole = "reviewer"
)

// SafetyLevel defines the operation safety classification.
type SafetyLevel string

const (
	SafetyGreen  SafetyLevel = "green"  // Auto-approve
	SafetyYellow SafetyLevel = "yellow" // Auto + notify
	SafetyRed    SafetyLevel = "red"    // Require approval
	SafetyBlack  SafetyLevel = "black"  // Double confirmation
)
