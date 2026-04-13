package session

import "testing"

// FuzzCreate ensures Create never panics on arbitrary input
func FuzzCreate(f *testing.F) {
	f.Add("sid", "client", "token", "1.2.3.4")
	f.Add("", "", "", "")
	f.Add("a\x00b", "c\x00d", "e", "::1")
	f.Fuzz(func(t *testing.T, sid, client, token, ip string) {
		m := New(nil)
		m.Create(sid, client, token, ip)
		m.Touch(sid)
		m.Remove(sid)
	})
}
