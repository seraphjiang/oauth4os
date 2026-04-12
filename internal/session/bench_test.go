package session

import (
	"strconv"
	"testing"
)

func BenchmarkCreate(b *testing.B) {
	m := New(nil)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Create("s-"+strconv.Itoa(i), "app", "t-"+strconv.Itoa(i), "1.2.3.4")
	}
}

func BenchmarkTouch(b *testing.B) {
	m := New(nil)
	m.Create("s-0", "app", "t-0", "1.2.3.4")
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Touch("s-0")
	}
}

func BenchmarkCount(b *testing.B) {
	m := New(nil)
	for i := 0; i < 100; i++ {
		m.Create("s-"+strconv.Itoa(i), "app", "t-"+strconv.Itoa(i), "1.2.3.4")
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		m.Count("app")
	}
}
