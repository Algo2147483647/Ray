package factory

import "sync"

// exprEnvPool owns reusable expression environments. It is always shared by
// pointer because sync.Pool must not be copied after first use.
type exprEnvPool struct {
	base      map[string]interface{}
	variables []string
	pool      sync.Pool
}

func newExprEnvPool(constants map[string]float64, variables ...string) *exprEnvPool {
	base := implicitExprBaseEnvWithConstants(constants)
	for _, name := range variables {
		base[name] = 0.0
	}
	p := &exprEnvPool{
		base:      base,
		variables: append([]string(nil), variables...),
	}
	p.pool.New = func() interface{} {
		return cloneImplicitExprEnv(base)
	}
	return p
}

func (p *exprEnvPool) get(values ...float64) map[string]interface{} {
	if p == nil {
		return implicitExprBaseEnv()
	}
	raw := p.pool.Get()
	env, ok := raw.(map[string]interface{})
	if !ok || env == nil {
		env = cloneImplicitExprEnv(p.base)
	}
	for i, name := range p.variables {
		if i < len(values) {
			env[name] = values[i]
		}
	}
	return env
}

func (p *exprEnvPool) put(env map[string]interface{}) {
	if p != nil && env != nil {
		p.pool.Put(env)
	}
}
