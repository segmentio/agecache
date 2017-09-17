package agecache

import (
	"sync/atomic"
	"time"

	"github.com/hashicorp/golang-lru"
)

// Stats hold cache statistics.
type Stats struct {
	Hits   int64 // number of cache hits.
	Misses int64 // number of cache misses
}

// Sub subtracts stats `s` with `o`.
func (s Stats) Sub(o Stats) Stats {
	return Stats{
		Hits:   atomic.LoadInt64(&s.Hits) - o.Hits,
		Misses: atomic.LoadInt64(&s.Misses) - o.Misses,
	}
}

type entry struct {
	value   interface{}
	created time.Time
}

// LRU implements a thread-safe fixed size LRU cache.
type Cache struct {
	lru    *lru.Cache
	maxAge time.Duration
	stats  Stats
}

// New constructs an LRU of the given size and max age to return
// results for
func New(size int, maxAge time.Duration) (*Cache, error) {
	l, err := lru.New(size)
	if err != nil {
		return nil, err
	}

	c := &Cache{
		lru:    l,
		maxAge: maxAge,
	}
	return c, nil
}

// Add adds an additional key/value pair to our cache.
func (c *Cache) Add(key, value interface{}) bool {
	return c.lru.Add(key, entry{
		value:   value,
		created: time.Now(),
	})
}

// Get returns the value stored at `key`.
//
// The boolean value reports if the value was found.
func (c *Cache) Get(key interface{}) (interface{}, bool) {
	val, ok := c.lru.Get(key)
	if !ok {
		atomic.AddInt64(&c.stats.Misses, 1)
		return nil, ok
	}

	e := val.(entry)
	if time.Since(e.created) <= c.maxAge {
		atomic.AddInt64(&c.stats.Hits, 1)
		return e.value, true
	}

	c.lru.Remove(key)
	return nil, false
}

// Stats returns cache stats.
func (c *Cache) Stats() Stats {
	return c.stats
}
