// Package draft provides an in-memory scratch pad for agents.
// Each agent gets an isolated namespace. Data is ephemeral and cleared after each run.
package draft

import (
	"sync"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
)

// Store is the in-memory draft/scratch pad.
type Store struct {
	mu   sync.RWMutex
	data map[protocol.AgentRole]map[string]interface{}
}

// NewStore creates a new draft store.
func NewStore() *Store {
	return &Store{
		data: make(map[protocol.AgentRole]map[string]interface{}),
	}
}

// Set stores a value in the agent's private namespace.
func (s *Store) Set(role protocol.AgentRole, key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.data[role] == nil {
		s.data[role] = make(map[string]interface{})
	}
	s.data[role][key] = value
}

// Get retrieves a value from the agent's private namespace.
func (s *Store) Get(role protocol.AgentRole, key string) (interface{}, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ns, ok := s.data[role]
	if !ok {
		return nil, false
	}
	val, ok := ns[key]
	return val, ok
}

// GetString is a convenience helper.
func (s *Store) GetString(role protocol.AgentRole, key string) string {
	val, ok := s.Get(role, key)
	if !ok {
		return ""
	}
	str, _ := val.(string)
	return str
}

// Clear wipes all data for a specific agent.
func (s *Store) Clear(role protocol.AgentRole) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.data, role)
}

// ClearAll wipes the entire store.
func (s *Store) ClearAll() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.data = make(map[protocol.AgentRole]map[string]interface{})
}
