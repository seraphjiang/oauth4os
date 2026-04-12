package ipfilter

import "testing"

func BenchmarkCheck(b *testing.B) {
	r, _ := New(Config{Clients: map[string]*FilterConfig{
		"app": {Allow: []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"}},
	}})
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Check("app", "10.1.2.3:8080")
	}
}
