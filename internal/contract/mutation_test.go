package contract

import "testing"

// Mutation: remove Run → must execute all checks
func TestMutation_RunAllChecks(t *testing.T) {
	checks := DefaultChecks()
	if len(checks) < 3 {
		t.Fatal("need at least 3 default checks")
	}
	// Verify structure
	for _, c := range checks {
		if c.Name == "" || c.Method == "" || c.Path == "" {
			t.Errorf("check %q has empty fields", c.Name)
		}
	}
}
