package todo

import (
	"fmt"
	"time"
)

// Todo represents a single todo item.
type Todo struct {
	ID        int       `json:"id"`
	Title     string    `json:"title"`
	Done      bool      `json:"done"`
	CreatedAt time.Time `json:"created_at"`
	DoneAt    time.Time `json:"done_at,omitempty"`
}

// String returns a human-readable representation of a todo item.
func (t Todo) String() string {
	status := " "
	if t.Done {
		status = "✓"
	}
	return fmt.Sprintf("[%s] %d: %s", status, t.ID, t.Title)
}
