package cedar

// TenantEngine holds per-issuer Cedar policies with a global fallback.
type TenantEngine struct {
	global  *Engine
	tenants map[string]*Engine // issuer → engine
}

// NewTenantEngine creates a multi-tenant Cedar engine.
func NewTenantEngine(globalPolicies []Policy) *TenantEngine {
	return &TenantEngine{
		global:  NewEngine(globalPolicies),
		tenants: make(map[string]*Engine),
	}
}

// AddTenant registers Cedar policies for a specific issuer.
func (te *TenantEngine) AddTenant(issuer string, policies []Policy) {
	te.tenants[issuer] = NewEngine(policies)
}

// Evaluate runs tenant-specific policies if available, else global.
func (te *TenantEngine) Evaluate(issuer string, req Request) Decision {
	if eng, ok := te.tenants[issuer]; ok {
		return eng.Evaluate(req)
	}
	return te.global.Evaluate(req)
}
