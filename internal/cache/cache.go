package cache

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/arne/motd/internal/block"
	"github.com/arne/motd/internal/module"
)

type Entry struct {
	Text       string         `json:"text"`
	Status     *block.Status  `json:"status,omitempty"`
	FetchedAt  time.Time      `json:"fetched_at"`
}

type Cache struct {
	dir string
	mu  sync.Mutex
}

func New(dir string) (*Cache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Cache{dir: dir}, nil
}

func (c *Cache) path(name string) string {
	return filepath.Join(c.dir, name+".json")
}

// Clear removes every cached entry, including any leftover .json.tmp files
// from interrupted writes. Returns the number of entries removed.
func (c *Cache) Clear() (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return 0, nil
		}
		return 0, err
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".json") && !strings.HasSuffix(name, ".json.tmp") {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, name)); err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}

func (c *Cache) read(name string) (*Entry, error) {
	data, err := os.ReadFile(c.path(name))
	if err != nil {
		return nil, err
	}
	var e Entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, err
	}
	return &e, nil
}

func (c *Cache) write(name string, e Entry) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	tmp := c.path(name) + ".tmp"
	data, err := json.Marshal(e)
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, c.path(name))
}

// Resolve returns the block for a module, using cache if fresh, running it if stale.
func (c *Cache) Resolve(ctx context.Context, m module.Module, ttl time.Duration) block.Block {
	name := m.Name()
	if ttl > 0 {
		if e, err := c.read(name); err == nil {
			if time.Since(e.FetchedAt) < ttl {
				return block.Block{Text: e.Text, Status: e.Status}
			}
		}
	}
	b, err := m.Render(ctx)
	if err != nil {
		// Fall back to stale cache if available
		if e, err2 := c.read(name); err2 == nil {
			return block.Block{Text: e.Text, Status: e.Status}
		}
		_ = err
		return block.Block{}
	}
	if ttl > 0 {
		_ = c.write(name, Entry{Text: b.Text, Status: b.Status, FetchedAt: time.Now()})
	}
	return b
}

func DefaultDir() string {
	if v := os.Getenv("XDG_CACHE_HOME"); v != "" {
		return filepath.Join(v, "motd")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".cache", "motd")
	}
	return filepath.Join(os.TempDir(), "motd")
}

var ErrNotCached = errors.New("not cached")
