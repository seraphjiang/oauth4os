package config

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoad ensures Load+Validate never panics on arbitrary YAML.
func FuzzLoad(f *testing.F) {
	f.Add([]byte(`upstream:
  engine: http://localhost:9200
listen: ":8443"
`))
	f.Add([]byte(`{}`))
	f.Add([]byte(``))
	f.Add([]byte(`upstream: null`))
	f.Add([]byte("upstream:\n  engine: !!binary aGVsbG8="))
	f.Add([]byte("providers:\n  - name: x\n    issuer: http://x\n    audience: [a, b]"))
	f.Add([]byte("rate_limits:\n  admin: 999999999"))
	f.Add([]byte("sigv4:\n  region: us-west-2"))
	f.Add([]byte("clusters:\n  c1:\n    engine: http://x\n    prefixes: ['*']"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		os.WriteFile(path, data, 0644)
		cfg, err := Load(path)
		if err != nil {
			return // parse error is fine
		}
		cfg.Validate() // must not panic
	})
}

// FuzzValidate ensures Validate never panics on any Config struct.
func FuzzValidate(f *testing.F) {
	f.Add("", "", "", false)
	f.Add("http://localhost:9200", ":8443", "us-west-2", true)
	f.Add("://invalid", ":0", "", false)

	f.Fuzz(func(t *testing.T, engine, listen, region string, sigv4 bool) {
		c := &Config{
			Upstream: Upstream{Engine: engine},
			Listen:   listen,
		}
		if sigv4 {
			c.Upstream.SigV4 = &SigV4Config{Region: region, Service: "aoss"}
		}
		c.Validate() // must not panic
	})
}
