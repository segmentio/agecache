// LRU largely inspired by https://github.com/golang/groupcache
package agecache

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// Stats hold cache statistics.
type Stats struct {
	Capacity  int64 // Gauge, maximum capacity for the cache
	Count     int64 // Gauge, number of items in the cache
	Sets      int64 // Counter, number of sets
	Gets      int64 // Counter, number of gets
	Hits      int64 // Counter, number of cache hits from Get operations
	Misses    int64 // Counter, number of cache misses from Get operations
	Evictions int64 // Counter, number of evictions
}

// Delta returns a Stats object such that all counters are calculated as the
// difference since the previous.
func (stats Stats) Delta(previous Stats) Stats {
	return Stats{
		Capacity:  stats.Capacity,
		Count:     stats.Count,
		Sets:      stats.Sets - previous.Sets,
		Gets:      stats.Gets - previous.Gets,
		Hits:      stats.Hits - previous.Hits,
		Misses:    stats.Misses - previous.Misses,
		Evictions: stats.Evictions - previous.Evictions,
	}
}

// Configuration for the Cache.
type Config struct {
	// Maximum number of items in the cache
	Capacity int
	// Optional duration before an item expires. If zero, expiration is disabled
	MaxAge time.Duration
	// Optional callback invoked when an item is evicted due to the LRU policy
	OnEviction func(key, value interface{})
	// Optional callback invoked when an item expired
	OnExpiration func(key, value interface{})
}

// Entry pointed to by each list.Element
type cacheEntry struct {
	key     interface{}
	value   interface{}
	created time.Time
}

// LRU implements a thread-safe fixed-capacity LRU cache.
type Cache struct {
	// Fields defined by configuration
	capacity     int
	maxAge       time.Duration
	onEviction   func(key, value interface{})
	onExpiration func(key, value interface{})

	// Cache statistics
	sets      int64
	gets      int64
	hits      int64
	misses    int64
	evictions int64

	items        map[interface{}]*list.Element
	evictionList *list.List
	mutex        sync.RWMutex
}

// New constructs an LRU Cache with the given Config object. config.Capacity
// must be a positive int, and config.MaxAge a zero or positive duration. A
// duration of zero disables item expiration. Panics given an invalid
// config.Capacity or config.MaxAge.
func New(config Config) *Cache {
	if config.Capacity <= 0 {
		panic("Must supply a positive config.Capacity")
	}

	if config.MaxAge < 0 {
		panic("Must supply a zero or positive config.MaxAge")
	}

	cache := &Cache{
		capacity:     config.Capacity,
		maxAge:       config.MaxAge,
		onEviction:   config.OnEviction,
		onExpiration: config.OnExpiration,
		items:        make(map[interface{}]*list.Element),
		evictionList: list.New(),
	}

	return cache
}

// Set updates a key:value pair in the cache. Returns true if an eviction
// occurrred, and subsequently invokes the OnEviction callback.
func (cache *Cache) Set(key, value interface{}) bool {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.sets++
	created := time.Now()

	if element, ok := cache.items[key]; ok {
		cache.evictionList.MoveToFront(element)
		entry := element.Value.(*cacheEntry)
		entry.value = value
		entry.created = created
		return false
	}

	entry := &cacheEntry{key, value, created}
	element := cache.evictionList.PushFront(entry)
	cache.items[key] = element

	evict := cache.evictionList.Len() > cache.capacity
	if evict {
		cache.evictOldest()
	}
	return evict
}

// Get returns the value stored at `key`. The boolean value reports whether or
// not the value was found. The OnExpiration callback is invoked if the value
// had expired on access
func (cache *Cache) Get(key interface{}) (interface{}, bool) {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.gets++

	if element, ok := cache.items[key]; ok {
		entry := element.Value.(*cacheEntry)
		if cache.maxAge == 0 || time.Since(entry.created) <= cache.maxAge {
			cache.evictionList.MoveToFront(element)
			cache.hits++
			return entry.value, true
		}

		// Entry expired
		cache.deleteElement(element)
		cache.misses++
		if cache.onExpiration != nil {
			cache.onExpiration(entry.key, entry.value)
		}
		return nil, false
	}

	cache.misses++
	return nil, false
}

// Has returns whether or not the `key` is in the cache without updating
// how recently it was accessed or deleting it for having expired.
func (cache *Cache) Has(key interface{}) bool {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	_, ok := cache.items[key]
	return ok
}

// Peek returns the value at the specified key and a boolean specifying whether
// or not it was found, without updating how recently it was accessed or
// deleting it for having expired.
func (cache *Cache) Peek(key interface{}) (interface{}, bool) {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	if element, ok := cache.items[key]; ok {
		return element.Value.(*cacheEntry).value, true
	}

	return nil, false
}

// Remove removes the provided key from the cache, returning a bool indicating
// whether or not it existed.
func (cache *Cache) Remove(key interface{}) bool {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	if element, ok := cache.items[key]; ok {
		cache.deleteElement(element)
		return true
	}

	return false
}

// Len returns the number of items in the cache.
func (cache *Cache) Len() int {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	return cache.evictionList.Len()
}

// Clear empties the cache.
func (cache *Cache) Clear() {
	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	for _, val := range cache.items {
		cache.deleteElement(val)
	}
	cache.evictionList.Init()
}

// Keys returns all keys in the cache.
func (cache *Cache) Keys() []interface{} {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	keys := make([]interface{}, len(cache.items))
	i := 0

	for key := range cache.items {
		keys[i] = key
		i++
	}

	return keys
}

// OrderedKeys returns all keys in the cache, ordered from oldest to newest.
func (cache *Cache) OrderedKeys() []interface{} {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	keys := make([]interface{}, len(cache.items))
	i := 0

	for element := cache.evictionList.Back(); element != nil; element = element.Prev() {
		keys[i] = element.Value.(*cacheEntry).key
		i++
	}

	return keys
}

// SetMaxAge updates the max age for items in the cache. A duration of zero
// disables expiration. A negative duration results in an error.
func (cache *Cache) SetMaxAge(maxAge time.Duration) error {
	if maxAge < 0 {
		return errors.New("Must supply a zero or positive maxAge")
	}

	cache.mutex.Lock()
	defer cache.mutex.Unlock()

	cache.maxAge = maxAge

	return nil
}

// Stats returns cache stats.
func (cache *Cache) Stats() Stats {
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()

	return Stats{
		Capacity:  int64(cache.capacity),
		Count:     int64(cache.evictionList.Len()),
		Sets:      cache.sets,
		Gets:      cache.gets,
		Hits:      cache.hits,
		Misses:    cache.misses,
		Evictions: cache.evictions,
	}
}

func (cache *Cache) evictOldest() {
	element := cache.evictionList.Back()
	if element == nil {
		return
	}

	cache.evictions++
	entry := cache.deleteElement(element)
	if cache.onEviction != nil {
		cache.onEviction(entry.key, entry.value)
	}
}

func (cache *Cache) deleteElement(element *list.Element) *cacheEntry {
	cache.evictionList.Remove(element)
	entry := element.Value.(*cacheEntry)
	delete(cache.items, entry.key)
	return entry
}
