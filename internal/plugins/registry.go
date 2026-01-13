package plugins

import (
	"fmt"
	"sync"
)

type Registry struct {
	mu      sync.RWMutex
	plugins map[string]AuthPlugin
}

func NewRegistry() *Registry {
	return &Registry{plugins: map[string]AuthPlugin{}}
}

func (r *Registry) Register(p AuthPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.plugins[p.Name()] = p
}

func (r *Registry) Get(name string) (AuthPlugin, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	p := r.plugins[name]
	if p == nil {
		return nil, fmt.Errorf("unknown provider: %s", name)
	}
	return p, nil
}
