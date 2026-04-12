package store

import "testing"

// Mutation: remove Set → Get must return stored value
func TestMutation_SetGet(t *testing.T) {
	m := NewMemory()
	m.Set("key1", []byte("value1"))
	got, err := m.Get("key1")
	if err != nil || string(got) != "value1" {
		t.Errorf("expected 'value1', got %q, err %v", got, err)
	}
}

// Mutation: remove Delete → key must be removed
func TestMutation_Delete(t *testing.T) {
	m := NewMemory()
	m.Set("key1", []byte("value1"))
	m.Delete("key1")
	_, err := m.Get("key1")
	if err == nil {
		t.Error("deleted key should return error")
	}
}

// Mutation: remove List → must return all keys
func TestMutation_List(t *testing.T) {
	m := NewMemory()
	m.Set("a", []byte("1"))
	m.Set("b", []byte("2"))
	keys, err := m.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(keys) != 2 {
		t.Errorf("expected 2 keys, got %d", len(keys))
	}
}

// Mutation: remove Get miss → missing key must error
func TestMutation_GetMissing(t *testing.T) {
	m := NewMemory()
	_, err := m.Get("nonexistent")
	if err == nil {
		t.Error("missing key should return error")
	}
}
