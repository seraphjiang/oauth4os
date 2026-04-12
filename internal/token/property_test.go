package token

import (
	"os"
	"path/filepath"
	"testing"
)

// Property: Save→Load round-trip preserves all registered clients
func TestProperty_SaveLoadRoundTrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clients.json")

	mgr := NewManager()
	mgr.RegisterClient("app-1", "secret-1", []string{"read"}, []string{"http://localhost/cb"})
	mgr.RegisterClient("app-2", "secret-2", []string{"read", "write"}, []string{"http://localhost/cb2"})

	store, err := NewClientStore(path, mgr)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(mgr); err != nil {
		t.Fatal(err)
	}

	// Load into fresh manager
	mgr2 := NewManager()
	_, err = NewClientStore(path, mgr2)
	if err != nil {
		t.Fatal(err)
	}

	clients := mgr2.Clients()
	if len(clients) < 2 {
		t.Fatalf("expected at least 2 clients after load, got %d", len(clients))
	}

	// Verify auth still works
	if err := mgr2.AuthenticateClient("app-1", "secret-1"); err != nil {
		t.Error("app-1 auth should work after round-trip")
	}

	os.Remove(path)
}
