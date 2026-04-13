package store

import (
	"path/filepath"
	"testing"
)

func TestEdge_NewFileCreates(t *testing.T) {
	f, err := NewFile(filepath.Join(t.TempDir(), "test.json"))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
}

func TestEdge_SetGetRoundTrip(t *testing.T) {
	f, _ := NewFile(filepath.Join(t.TempDir(), "test.json"))
	defer f.Close()
	f.Set("k1", []byte(`"v1"`))
	v, err := f.Get("k1")
	if err != nil {
		t.Fatal(err)
	}
	if string(v) != `"v1"` {
		t.Errorf("got %q", string(v))
	}
}

func TestEdge_GetMissing(t *testing.T) {
	f, _ := NewFile(filepath.Join(t.TempDir(), "test.json"))
	defer f.Close()
	_, err := f.Get("nope")
	if err == nil {
		t.Error("missing key should error")
	}
}

func TestEdge_DeleteRemoves(t *testing.T) {
	f, _ := NewFile(filepath.Join(t.TempDir(), "test.json"))
	defer f.Close()
	f.Set("k1", []byte(`"v1"`))
	f.Delete("k1")
	_, err := f.Get("k1")
	if err == nil {
		t.Error("deleted key should error")
	}
}
