package draft_test

import (
	"testing"

	"github.com/awch-D/ForgeX/forgex-agent/protocol"
	"github.com/awch-D/ForgeX/forgex-cognition/draft"
)

func TestDraftStore_SetAndGet(t *testing.T) {
	s := draft.NewStore()

	s.Set(protocol.RoleCoder, "plan", "write code")
	val, ok := s.Get(protocol.RoleCoder, "plan")
	if !ok {
		t.Fatal("expected key to exist")
	}
	if val != "write code" {
		t.Errorf("expected 'write code', got %v", val)
	}
}

func TestDraftStore_RoleIsolation(t *testing.T) {
	s := draft.NewStore()

	s.Set(protocol.RoleCoder, "data", "coder secret")
	s.Set(protocol.RoleTester, "data", "tester secret")

	coderVal, _ := s.Get(protocol.RoleCoder, "data")
	testerVal, _ := s.Get(protocol.RoleTester, "data")

	if coderVal != "coder secret" {
		t.Errorf("coder data mismatch: got %v", coderVal)
	}
	if testerVal != "tester secret" {
		t.Errorf("tester data mismatch: got %v", testerVal)
	}
}

func TestDraftStore_GetMissing(t *testing.T) {
	s := draft.NewStore()

	_, ok := s.Get(protocol.RoleCoder, "nonexistent")
	if ok {
		t.Error("expected key to not exist")
	}

	_, ok = s.Get("unknown_role", "anything")
	if ok {
		t.Error("expected unknown role to not exist")
	}
}

func TestDraftStore_GetString(t *testing.T) {
	s := draft.NewStore()

	s.Set(protocol.RoleCoder, "name", "Alice")
	str := s.GetString(protocol.RoleCoder, "name")
	if str != "Alice" {
		t.Errorf("expected 'Alice', got %q", str)
	}

	// Non-string value
	s.Set(protocol.RoleCoder, "count", 42)
	str = s.GetString(protocol.RoleCoder, "count")
	if str != "" {
		t.Errorf("expected empty string for non-string value, got %q", str)
	}

	// Missing key
	str = s.GetString(protocol.RoleCoder, "missing")
	if str != "" {
		t.Errorf("expected empty string for missing key, got %q", str)
	}
}

func TestDraftStore_Clear(t *testing.T) {
	s := draft.NewStore()

	s.Set(protocol.RoleCoder, "a", "1")
	s.Set(protocol.RoleCoder, "b", "2")
	s.Set(protocol.RoleTester, "c", "3")

	s.Clear(protocol.RoleCoder)

	_, ok := s.Get(protocol.RoleCoder, "a")
	if ok {
		t.Error("coder data should be cleared")
	}

	_, ok = s.Get(protocol.RoleTester, "c")
	if !ok {
		t.Error("tester data should NOT be cleared")
	}
}

func TestDraftStore_ClearAll(t *testing.T) {
	s := draft.NewStore()

	s.Set(protocol.RoleCoder, "a", "1")
	s.Set(protocol.RoleTester, "b", "2")

	s.ClearAll()

	_, ok1 := s.Get(protocol.RoleCoder, "a")
	_, ok2 := s.Get(protocol.RoleTester, "b")
	if ok1 || ok2 {
		t.Error("all data should be cleared")
	}
}
