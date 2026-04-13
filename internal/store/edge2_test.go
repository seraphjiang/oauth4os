package store

import (
	"path/filepath"
	"sync"
	"testing"
)

func TestEdge_ConcurrentSetGet(t *testing.T) {
	f, _ := NewFile(filepath.Join(t.TempDir(), "test.json"))
	defer f.Close()
	var wg sync.WaitGroup
	for i := 0; i < 20; i++ {
		wg.Add(2)
		k := string(rune('a' + i%10))
		go func() {
			defer wg.Done()
			f.Set(k, []byte(`"val"`))
		}()
		go func() {
			defer wg.Done()
			f.Get(k)
		}()
	}
	wg.Wait()
}

func TestEdge_PersistenceAcrossReopen(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "persist.json")
	f, _ := NewFile(p)
	f.Set("key", []byte(`"saved"`))
	f.Close()

	f2, _ := NewFile(p)
	defer f2.Close()
	v, err := f2.Get("key")
	if err != nil {
		t.Fatalf("should find key after reopen: %v", err)
	}
	if string(v) != `"saved"` {
		t.Errorf("expected '\"saved\"', got %q", string(v))
	}
}

func TestEdge_SetOverwrite(t *testing.T) {
	f, _ := NewFile(filepath.Join(t.TempDir(), "test.json"))
	defer f.Close()
	f.Set("k", []byte(`"v1"`))
	f.Set("k", []byte(`"v2"`))
	v, _ := f.Get("k")
	if string(v) != `"v2"` {
		t.Errorf("overwrite should return latest, got %q", string(v))
	}
}
