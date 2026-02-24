package plugin

import (
	"fmt"
	"sync"

	"github.com/liuerfire/sieve/internal/storage"
)

type Plugin interface {
	Execute(item *storage.Item) (*storage.Item, error)
}

var (
	registry = make(map[string]Plugin)
	mu       sync.RWMutex
)

func Register(name string, p Plugin) {
	mu.Lock()
	defer mu.Unlock()
	registry[name] = p
}

func Get(name string) (Plugin, error) {
	mu.RLock()
	defer mu.RUnlock()
	p, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("plugin '%s' not found", name)
	}
	return p, nil
}

// NopPlugin does nothing
type NopPlugin struct{}

func (p *NopPlugin) Execute(item *storage.Item) (*storage.Item, error) {
	return item, nil
}

func init() {
	nop := &NopPlugin{}
	Register("nop", nop)
	Register("fetch_content", nop)              // Placeholder
	Register("fetch_meta", nop)                 // Placeholder
	Register("cnbeta_fetch_content", nop)       // Placeholder
	Register("hn_fetch_comments", nop)          // Placeholder
	Register("zaihuapd_clean_description", nop) // Placeholder
}
