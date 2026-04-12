package store

import "testing"

func TestMultiTenant_Isolation(t *testing.T) {
	mt := NewMultiTenant(func(tenant string) Store { return NewMemory() })

	mt.For("a").Set("key1", []byte("val-a"))
	mt.For("b").Set("key1", []byte("val-b"))

	va, _ := mt.For("a").Get("key1")
	vb, _ := mt.For("b").Get("key1")
	if string(va) != "val-a" || string(vb) != "val-b" {
		t.Errorf("isolation broken: a=%q b=%q", va, vb)
	}
}

func TestMultiTenant_Tenants(t *testing.T) {
	mt := NewMultiTenant(func(tenant string) Store { return NewMemory() })
	mt.For("x")
	mt.For("y")
	if len(mt.Tenants()) != 2 {
		t.Errorf("expected 2 tenants, got %d", len(mt.Tenants()))
	}
}

func TestMultiTenant_CloseAll(t *testing.T) {
	mt := NewMultiTenant(func(tenant string) Store { return NewMemory() })
	mt.For("a").Set("k", []byte("v"))
	if err := mt.CloseAll(); err != nil {
		t.Fatal(err)
	}
}

func TestMultiTenant_SameTenantSameStore(t *testing.T) {
	mt := NewMultiTenant(func(tenant string) Store { return NewMemory() })
	s1 := mt.For("t1")
	s1.Set("k", []byte("v"))
	s2 := mt.For("t1")
	v, err := s2.Get("k")
	if err != nil || string(v) != "v" {
		t.Error("same tenant should return same store")
	}
}
