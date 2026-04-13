package keyring

import (
	"testing"
	"time"
)

func TestEdge_CurrentNonNil(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k := r.Current()
	if k == nil {
		t.Error("Current should return non-nil key")
	}
}

func TestEdge_CurrentHasID(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	k := r.Current()
	if k.ID == "" {
		t.Error("key should have non-empty ID")
	}
}

func TestEdge_RotateChangesKey(t *testing.T) {
	r, err := New(2048, time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Stop()
	id1 := r.Current().ID
	r.Rotate()
	id2 := r.Current().ID
	if id1 == id2 {
		t.Error("Rotate should produce new key ID")
	}
}
