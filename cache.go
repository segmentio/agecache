package agecache

import (
	"time"

	"github.com/hashicorp/golang-lru"
)

// LRU implements a non-thread safe fixed size LRU cache
type Cache struct {
	lru    *lru.Cache
	maxAge time.Duration
}

type entry struct {
	value   interface{}
	created time.Time
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

// Add adds an additional key/value pair to our cache
func (c *Cache) Add(key, value interface{}) bool {
	return c.lru.Add(key, entry{
		value:   value,
		created: time.Now(),
	})
}

func (c *Cache) Get(key interface{}) (interface{}, bool) {
	val, ok := c.lru.Get(key)
	if !ok {
		return nil, ok
	}

	e := val.(entry)
	if time.Since(e.created) <= c.maxAge {
		return e.value, true
	}

	c.lru.Remove(key)
	return nil, false
}
