package todo

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"time"
)

const defaultFile = ".todos.json"

// Store manages persistence of todo items to a JSON file.
type Store struct {
	path  string
	Todos []Todo `json:"todos"`
}

// NewStore creates a store backed by the default JSON file.
func NewStore() *Store {
	return &Store{path: defaultFile}
}

// Load reads todos from disk. If the file doesn't exist, starts empty.
func (s *Store) Load() error {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.Todos = []Todo{}
			return nil
		}
		return fmt.Errorf("reading store: %w", err)
	}
	if err := json.Unmarshal(data, &s.Todos); err != nil {
		return fmt.Errorf("parsing store: %w", err)
	}
	return nil
}

// Save writes the current todos to disk.
func (s *Store) Save() error {
	data, err := json.MarshalIndent(s.Todos, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling store: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0644); err != nil {
		return fmt.Errorf("writing store: %w", err)
	}
	return nil
}

// nextID returns the next available ID.
func (s *Store) nextID() int {
	max := 0
	for _, t := range s.Todos {
		if t.ID > max {
			max = t.ID
		}
	}
	return max + 1
}

// Add creates a new todo with the given title.
func (s *Store) Add(title string) Todo {
	t := Todo{
		ID:        s.nextID(),
		Title:     title,
		Done:      false,
		CreatedAt: time.Now(),
	}
	s.Todos = append(s.Todos, t)
	return t
}

// Done marks the todo with the given ID as completed.
func (s *Store) Done(id int) error {
	for i := range s.Todos {
		if s.Todos[i].ID == id {
			if s.Todos[i].Done {
				return fmt.Errorf("todo %d is already done", id)
			}
			s.Todos[i].Done = true
			s.Todos[i].DoneAt = time.Now()
			return nil
		}
	}
	return fmt.Errorf("todo %d not found", id)
}

// Delete removes the todo with the given ID.
func (s *Store) Delete(id int) error {
	for i, t := range s.Todos {
		if t.ID == id {
			s.Todos = append(s.Todos[:i], s.Todos[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("todo %d not found", id)
}
