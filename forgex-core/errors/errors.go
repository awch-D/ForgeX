// Package errors provides a unified error handling system for ForgeX.
package errors

import "fmt"

// Code represents a ForgeX error code.
type Code int

const (
	// General errors
	ErrUnknown       Code = 1000
	ErrInvalidInput  Code = 1001
	ErrNotFound      Code = 1002
	ErrAlreadyExists Code = 1003
	ErrTimeout       Code = 1004

	// LLM errors
	ErrLLMConnection Code = 2001
	ErrLLMRateLimit  Code = 2002
	ErrLLMBadResponse Code = 2003

	// Agent errors
	ErrAgentSpawn    Code = 3001
	ErrAgentTimeout  Code = 3002
	ErrAgentPanic    Code = 3003

	// MCP / Sandbox errors
	ErrMCPToolNotFound Code = 4001
	ErrSandboxFuel     Code = 4002
	ErrSandboxMemory   Code = 4003
	ErrSandboxNetwork  Code = 4004

	// Governance errors
	ErrSafetyBlocked Code = 5001
	ErrBudgetExceeded Code = 5002
)

// ForgeXError is the standard error type across the system.
type ForgeXError struct {
	Code    Code   `json:"code"`
	Message string `json:"message"`
	Cause   error  `json:"-"`
}

func (e *ForgeXError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[FX-%d] %s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[FX-%d] %s", e.Code, e.Message)
}

func (e *ForgeXError) Unwrap() error {
	return e.Cause
}

// New creates a new ForgeXError.
func New(code Code, message string) *ForgeXError {
	return &ForgeXError{Code: code, Message: message}
}

// Wrap creates a new ForgeXError wrapping an existing error.
func Wrap(code Code, message string, cause error) *ForgeXError {
	return &ForgeXError{Code: code, Message: message, Cause: cause}
}
