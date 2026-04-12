package token

import (
	"os"
	"path/filepath"
	"testing"
)

func TestClientStore_SaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clients.json")

	mgr := NewManager()
	mgr.RegisterClient("app1", "secret1", []string{"read:logs"}, nil)

	store, err := NewClientStore(path, mgr)
	if err != nil {
		t.Fatalf("NewClientStore: %v", err)
	}
	if err := store.Save(mgr); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Load into fresh manager
	mgr2 := NewManager()
	_, err = NewClientStore(path, mgr2)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	clients := mgr2.Clients()
	if len(clients) != 1 || clients[0].ID != "app1" {
		t.Fatalf("expected 1 client 'app1', got %+v", clients)
	}
}

func TestClientStore_BackupCreated(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clients.json")

	mgr := NewManager()
	mgr.RegisterClient("c1", "s1", nil, nil)
	store, _ := NewClientStore(path, mgr)
	store.Save(mgr)

	// Second save should create .bak
	mgr.RegisterClient("c2", "s2", nil, nil)
	store.Save(mgr)

	if _, err := os.Stat(path + ".bak"); err != nil {
		t.Fatal("expected .bak file after second save")
	}
}

func TestClientStore_CorruptionRecovery(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clients.json")

	// Write valid backup
	mgr := NewManager()
	mgr.RegisterClient("backup-client", "s", []string{"read"}, nil)
	store, _ := NewClientStore(path, mgr)
	store.Save(mgr)

	// Corrupt the main file
	os.WriteFile(path, []byte("NOT JSON{{{"), 0644)

	// Load should recover from .bak
	mgr2 := NewManager()
	_, err := NewClientStore(path, mgr2)
	if err != nil {
		t.Fatalf("should recover from backup: %v", err)
	}
	clients := mgr2.Clients()
	if len(clients) != 1 || clients[0].ID != "backup-client" {
		t.Fatalf("expected backup-client, got %+v", clients)
	}
}

func TestClientStore_EmptyFileStartsFresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clients.json")
	os.WriteFile(path, []byte(""), 0644)

	mgr := NewManager()
	_, err := NewClientStore(path, mgr)
	if err != nil {
		t.Fatalf("empty file should start fresh: %v", err)
	}
	if len(mgr.Clients()) != 0 {
		t.Fatal("expected 0 clients from empty file")
	}
}

func TestClientStore_MissingFileStartsFresh(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nonexistent", "clients.json")

	mgr := NewManager()
	_, err := NewClientStore(path, mgr)
	if err != nil {
		t.Fatalf("missing file should start fresh: %v", err)
	}
}

func TestClientStore_AtomicWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "clients.json")

	mgr := NewManager()
	mgr.RegisterClient("c1", "s1", nil, nil)
	store, _ := NewClientStore(path, mgr)
	store.Save(mgr)

	// No .tmp files should remain
	entries, _ := os.ReadDir(dir)
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".tmp" {
			t.Fatalf("temp file should be cleaned up: %s", e.Name())
		}
	}
}
