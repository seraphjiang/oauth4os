package federation

import "testing"

// FuzzExtractIndex ensures index extraction never panics.
func FuzzExtractIndex(f *testing.F) {
	f.Add("/logs-2024/_search")
	f.Add("/_cluster/health")
	f.Add("")
	f.Add("/")
	f.Add("////////")
	f.Fuzz(func(t *testing.T, path string) {
		extractIndex(path) // must not panic
	})
}

// FuzzGlobMatch ensures glob matching never panics.
func FuzzGlobMatch(f *testing.F) {
	f.Add("logs-*", "logs-2024")
	f.Add("*", "anything")
	f.Add("", "")
	f.Add("a*b*c", "axbxc")
	f.Fuzz(func(t *testing.T, pattern, value string) {
		globMatch(pattern, value) // must not panic
	})
}
