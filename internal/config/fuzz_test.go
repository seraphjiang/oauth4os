package config

import (
	"os"
	"testing"
)

// FuzzLoad ensures config loading never panics on arbitrary YAML.
func FuzzLoad(f *testing.F) {
	f.Add("upstream: https://es.example.com\nlisten: :8443\n")
	f.Add("")
	f.Add("upstream: \"\"\n")
	f.Add("{{{invalid")
	f.Add("providers:\n  - name: test\n    issuer: https://idp\n")
	f.Fuzz(func(t *testing.T, content string) {
		tmp, err := os.CreateTemp("", "cfg-*.yaml")
		if err != nil {
			return
		}
		tmp.WriteString(content)
		tmp.Close()
		defer os.Remove(tmp.Name())
		Load(tmp.Name()) // must not panic
	})
}
