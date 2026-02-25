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

// FetchContentPlugin is a placeholder for full content fetching logic
type FetchContentPlugin struct{}

func (p *FetchContentPlugin) Execute(item *storage.Item) (*storage.Item, error) {
	// In the future, this will use an HTTP client to fetch the full HTML
	// and extract the main content. For now, we ensure the field exists.
	if item.Content == "" {
		item.Content = item.Description
	}
	return item, nil
}

func init() {
	nop := &NopPlugin{}
	fetcher := &FetchContentPlugin{}

	Register("nop", nop)
	Register("fetch_content", fetcher)
	Register("fetch_meta", nop)
	Register("cnbeta_fetch_content", nop)
	Register("hn_fetch_comments", nop)
	Register("zaihuapd_clean_description", nop)
}
