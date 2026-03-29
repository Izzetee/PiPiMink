package llm

import (
	"container/list"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"PiPiMink/internal/config"
	"PiPiMink/internal/models"
)

type decisionCache struct {
	enabled          bool
	ttl              time.Duration
	maxEntries       int
	statsLogInterval time.Duration

	mu      sync.Mutex
	entries map[string]*cacheItem
	order   *list.List

	hits                 uint64
	misses               uint64
	expired              uint64
	sets                 uint64
	evictions            uint64
	lastStatsLogUnixNano int64
}

type cacheLookupStatus string

const (
	cacheStatusHit      cacheLookupStatus = "hit"
	cacheStatusMiss     cacheLookupStatus = "miss"
	cacheStatusExpired  cacheLookupStatus = "expired"
	cacheStatusDisabled cacheLookupStatus = "disabled"
)

type cacheStats struct {
	Hits      uint64
	Misses    uint64
	Expired   uint64
	Sets      uint64
	Evictions uint64
}

type cacheItem struct {
	model     string
	expiresAt time.Time
	element   *list.Element
}

func newDecisionCache(cfg *config.Config) *decisionCache {
	cache := &decisionCache{
		enabled:          true,
		ttl:              2 * time.Minute,
		maxEntries:       1000,
		statsLogInterval: time.Minute,
		entries:          make(map[string]*cacheItem),
		order:            list.New(),
	}

	if cfg == nil {
		return cache
	}

	cache.enabled = cfg.SelectionCacheEnabled
	if cfg.SelectionCacheTTL > 0 {
		cache.ttl = cfg.SelectionCacheTTL
	}
	if cfg.SelectionCacheMaxEntries > 0 {
		cache.maxEntries = cfg.SelectionCacheMaxEntries
	}
	if cfg.SelectionCacheStatsLogInterval >= 0 {
		cache.statsLogInterval = cfg.SelectionCacheStatsLogInterval
	}

	return cache
}

func (c *decisionCache) getWithStatus(key string) (string, bool, cacheLookupStatus) {
	if c == nil || !c.enabled {
		return "", false, cacheStatusDisabled
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.entries[key]
	if !ok {
		atomic.AddUint64(&c.misses, 1)
		return "", false, cacheStatusMiss
	}

	if now.After(item.expiresAt) {
		c.removeItem(key, item)
		atomic.AddUint64(&c.expired, 1)
		atomic.AddUint64(&c.misses, 1)
		return "", false, cacheStatusExpired
	}

	c.order.MoveToFront(item.element)
	atomic.AddUint64(&c.hits, 1)
	return item.model, true, cacheStatusHit
}

func (c *decisionCache) set(key, model string) {
	if c == nil || !c.enabled {
		return
	}

	now := time.Now()
	c.mu.Lock()
	defer c.mu.Unlock()

	if item, ok := c.entries[key]; ok {
		item.model = model
		item.expiresAt = now.Add(c.ttl)
		c.order.MoveToFront(item.element)
		atomic.AddUint64(&c.sets, 1)
		return
	}

	element := c.order.PushFront(key)
	c.entries[key] = &cacheItem{
		model:     model,
		expiresAt: now.Add(c.ttl),
		element:   element,
	}
	atomic.AddUint64(&c.sets, 1)

	for len(c.entries) > c.maxEntries {
		back := c.order.Back()
		if back == nil {
			break
		}
		oldKey, ok := back.Value.(string)
		if !ok {
			c.order.Remove(back)
			continue
		}
		if oldItem, exists := c.entries[oldKey]; exists {
			c.removeItem(oldKey, oldItem)
			atomic.AddUint64(&c.evictions, 1)
		}
	}
}

func (c *decisionCache) statsSnapshot() cacheStats {
	if c == nil {
		return cacheStats{}
	}

	return cacheStats{
		Hits:      atomic.LoadUint64(&c.hits),
		Misses:    atomic.LoadUint64(&c.misses),
		Expired:   atomic.LoadUint64(&c.expired),
		Sets:      atomic.LoadUint64(&c.sets),
		Evictions: atomic.LoadUint64(&c.evictions),
	}
}

func (c *decisionCache) maybeStatsSummary() (cacheStats, bool) {
	if c == nil || !c.enabled || c.statsLogInterval <= 0 {
		return cacheStats{}, false
	}

	now := time.Now().UnixNano()
	last := atomic.LoadInt64(&c.lastStatsLogUnixNano)
	if last > 0 && now-last < c.statsLogInterval.Nanoseconds() {
		return cacheStats{}, false
	}

	if !atomic.CompareAndSwapInt64(&c.lastStatsLogUnixNano, last, now) {
		return cacheStats{}, false
	}

	return c.statsSnapshot(), true
}

func (c *decisionCache) removeItem(key string, item *cacheItem) {
	delete(c.entries, key)
	if item != nil && item.element != nil {
		c.order.Remove(item.element)
	}
}

func buildDecisionCacheKey(message string, availableModels map[string]models.ModelInfo) (string, error) {
	normalizedMessage := strings.Join(strings.Fields(strings.TrimSpace(message)), " ")

	type cacheModel struct {
		Name         string `json:"name"`
		Source       string `json:"source"`
		Tags         string `json:"tags"`
		Enabled      bool   `json:"enabled"`
		HasReasoning bool   `json:"has_reasoning"`
	}

	items := make([]cacheModel, 0, len(availableModels))
	for name, info := range availableModels {
		items = append(items, cacheModel{
			Name:         name,
			Source:       info.Source,
			Tags:         info.Tags,
			Enabled:      info.Enabled,
			HasReasoning: info.HasReasoning,
		})
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].Name < items[j].Name
	})

	payload, err := json.Marshal(struct {
		Message string       `json:"message"`
		Models  []cacheModel `json:"models"`
	}{
		Message: normalizedMessage,
		Models:  items,
	})
	if err != nil {
		return "", fmt.Errorf("error building cache key payload: %w", err)
	}

	h := sha256.Sum256(payload)
	return hex.EncodeToString(h[:]), nil
}
