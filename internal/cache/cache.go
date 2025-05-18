package cache

import (
	"log/slog"
	"sync"
)

type Set struct {
	set    map[string]struct{}
	mu     sync.Mutex
	logger *slog.Logger
}

func NewSet(logger *slog.Logger) *Set {
	return &Set{
		set:    make(map[string]struct{}),
		logger: logger,
	}
}
func (c *Set) Add(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.set[key] = struct{}{}
	c.logger.Debug("Добавлена ссылка", "key", key)
}
func (c *Set) Has(key string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	_, exists := c.set[key]
	return exists
}
