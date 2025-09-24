package agecache

import (
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestInvalidCapacity(t *testing.T) {
	assert.Panics(t, func() {
		New(Config{Capacity: 0})
	})
}

func TestInvalidMaxAge(t *testing.T) {
	assert.Panics(t, func() {
		New(Config{Capacity: 1, MaxAge: -1 * time.Hour})
	})
}

func TestInvalidMinAge(t *testing.T) {
	assert.Panics(t, func() {
		New(Config{Capacity: 1, MinAge: -1 * time.Hour})
	})

	assert.Panics(t, func() {
		New(Config{
			Capacity: 1,
			MaxAge:   time.Hour,
			MinAge:   2 * time.Hour,
		})
	})
}

func TestInvalidRefreshInterval(t *testing.T) {
	assert.Panics(t, func() {
		New(Config{Capacity: 1, RefreshInterval: -1 * time.Hour})
	})
}

func TestBasicSetGet(t *testing.T) {
	cache := New(Config{Capacity: 2})
	cache.Set("foo", 1)
	cache.Set("bar", 2)

	val, ok := cache.Get("foo")
	assert.True(t, ok)
	assert.Equal(t, 1, val)

	val, ok = cache.Get("bar")
	assert.True(t, ok)
	assert.Equal(t, 2, val)
}

func TestBasicSetOverwrite(t *testing.T) {
	cache := New(Config{Capacity: 2})
	cache.Set("foo", 1)
	evict := cache.Set("foo", 2)
	val, ok := cache.Get("foo")

	assert.False(t, evict)
	assert.True(t, ok)
	assert.Equal(t, 2, val)
}

func TestEviction(t *testing.T) {
	var k, v interface{}

	cache := New(Config{
		Capacity: 2,
		OnEviction: func(key, value interface{}) {
			k = key
			v = value
		},
	})

	cache.Set("foo", 1)
	cache.Set("bar", 2)
	evict := cache.Set("baz", 3)
	val, ok := cache.Get("foo")

	assert.True(t, evict)
	assert.False(t, ok)
	assert.Nil(t, val)
	assert.Equal(t, "foo", k)
	assert.Equal(t, 1, v)
}

func TestExpiration(t *testing.T) {
	var k, v interface{}
	var eviction bool

	cache := New(Config{
		Capacity: 1,
		MaxAge:   time.Millisecond,
		OnExpiration: func(key, value interface{}) {
			k = key
			v = value
		},
		OnEviction: func(key, value interface{}) {
			eviction = true
		},
	})

	cache.Set("foo", 1)
	<-time.After(time.Millisecond * 2)

	val, ok := cache.Get("foo")
	assert.False(t, ok)
	assert.Nil(t, val)
	assert.Equal(t, "foo", k)
	assert.Equal(t, 1, v)
	assert.False(t, eviction)
}

func TestCacheBackgroundRefresh(t *testing.T) {
	count := 0
	cache := New(Config{
		Capacity:        1,
		RefreshInterval: 3 * time.Second,
		OnRefresh: func() map[interface{}]interface{} {
			count++
			return map[interface{}]interface{}{"key": count}
		},
	})

	value, ok := cache.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, value)

	time.Sleep(4 * time.Second) // wait for the refresh loop to run

	value, ok = cache.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, 2, value)

	time.Sleep(4 * time.Second)
	value, ok = cache.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, value)

}

func TestCacheBackgroundRefreshForNilData(t *testing.T) {
	count := 0
	cache := New(Config{
		Capacity:        1,
		RefreshInterval: 3 * time.Second,
		OnRefresh: func() map[interface{}]interface{} {
			count++

			if count == 2 {
				return nil
			}
			return map[interface{}]interface{}{"key": count}
		},
	})

	value, ok := cache.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, value)

	time.Sleep(4 * time.Second)

	// Prevent refresh when the OnRefresh call back returns nil
	value, ok = cache.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, 1, value)

	time.Sleep(4 * time.Second)
	value, ok = cache.Get("key")
	assert.Equal(t, true, ok)
	assert.Equal(t, 3, value)

}

type MockRandGenerator struct {
	startAt int64
	incr    int64
	state   int64
}

func (mock *MockRandGenerator) Int63n(n int64) int64 {
	if mock.state == 0 {
		mock.state = mock.startAt
	}
	ret := mock.state
	mock.state += mock.incr
	return ret
}

func TestJitter(t *testing.T) {
	cache := New(Config{
		Capacity: 1,
		MaxAge:   350 * time.Millisecond,
		MinAge:   time.Millisecond,
	})

	cache.rand = &MockRandGenerator{
		startAt: (300 * time.Millisecond).Nanoseconds(),
		incr:    (-50 * time.Millisecond).Nanoseconds(),
	}

	cache.Set("foo", "bar") // 300ms
	_, ok := cache.Get("foo")
	assert.True(t, ok)

	time.Sleep(50 * time.Millisecond)
	_, ok = cache.Get("foo")
	assert.False(t, ok)

	cache.Set("foo", "bar") // 250ms
	time.Sleep(100 * time.Millisecond)
	_, ok = cache.Get("foo")
	assert.False(t, ok)

	cache.Set("foo", "bar") // 200ms
	time.Sleep(50 * time.Millisecond)
	_, ok = cache.Get("foo")
	assert.True(t, ok)

	cache.Set("foo", "bar") // 150ms
	time.Sleep(100 * time.Millisecond)
	_, ok = cache.Get("foo")
	assert.True(t, ok)

	time.Sleep(100 * time.Millisecond)
	_, ok = cache.Get("foo")
	assert.False(t, ok)
}

func TestHas(t *testing.T) {
	cache := New(Config{Capacity: 1, MaxAge: time.Millisecond})
	cache.Set("foo", "bar")
	<-time.After(time.Millisecond * 2)

	ok := cache.Has("foo")
	assert.True(t, ok)
}

func TestPeek(t *testing.T) {
	cache := New(Config{Capacity: 1, MaxAge: time.Millisecond})
	cache.Set("foo", "bar")
	<-time.After(time.Millisecond * 2)

	val, ok := cache.Peek("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", val)
}

func TestRemove(t *testing.T) {
	var eviction bool

	cache := New(Config{
		Capacity: 1,
		OnEviction: func(key, value interface{}) {
			eviction = true
		},
	})

	cache.Set("foo", "bar")
	ok := cache.Remove("foo")

	assert.True(t, ok)
	assert.False(t, eviction)

	val, ok := cache.Get("foo")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestEvictOldest(t *testing.T) {
	var eviction bool

	cache := New(Config{
		Capacity: 1,
		OnEviction: func(key, value interface{}) {
			eviction = true
		},
	})

	cache.Set("foo", "bar")
	ok := cache.EvictOldest()

	assert.True(t, ok)
	assert.True(t, eviction)

	val, ok := cache.Get("foo")
	assert.False(t, ok)
	assert.Nil(t, val)

	eviction = false
	ok = cache.EvictOldest()
	assert.False(t, ok)
	assert.False(t, eviction)
}

func TestLen(t *testing.T) {
	cache := New(Config{Capacity: 10})
	for i := 0; i <= 9; i++ {
		evict := cache.Set(i, i)
		assert.False(t, evict)
	}

	assert.Equal(t, 10, cache.Len())
}

func TestClear(t *testing.T) {
	cache := New(Config{Capacity: 10})
	for i := 0; i <= 9; i++ {
		evict := cache.Set(i, i)
		assert.False(t, evict)
	}

	cache.Clear()

	for i := 0; i <= 9; i++ {
		_, ok := cache.Get(i)
		assert.False(t, ok)
	}
	assert.Equal(t, 0, cache.Len())
}

func TestRefreshCache(t *testing.T) {
	cache := New(Config{Capacity: 10})
	cache.Set("foo", 1)
	cache.Set("bar", 2)

	refreshedCacheEntries := map[interface{}]interface{}{}
	for i := 0; i <= 9; i++ {
		refreshedCacheEntries[i] = i
	}
	cache.RefreshCache(refreshedCacheEntries)

	assert.False(t, cache.Has("foo"))
	assert.False(t, cache.Has("bar"))

	for i := 0; i <= 9; i++ {
		_, ok := cache.Get(i)
		assert.True(t, ok)
	}

	assert.Equal(t, 10, cache.Len())

}

func TestKeys(t *testing.T) {
	cache := New(Config{Capacity: 10})
	cache.Set("foo", 1)
	cache.Set("bar", 2)

	// key order isn't guarenteed
	keys := cache.Keys()
	sortedKeys := []string{keys[0].(string), keys[1].(string)}
	sort.Strings(sortedKeys)

	assert.Equal(t, 2, len(sortedKeys))
	assert.Equal(t, "bar", sortedKeys[0])
	assert.Equal(t, "foo", sortedKeys[1])
}

func TestOrderedKeys(t *testing.T) {
	cache := New(Config{Capacity: 10})
	cache.Set("foo", 1)
	cache.Set("bar", 2)

	keys := cache.OrderedKeys()

	assert.Equal(t, 2, len(keys))
	assert.Equal(t, "foo", keys[0])
	assert.Equal(t, "bar", keys[1])
}

func TestSetMaxAge(t *testing.T) {
	cache := New(Config{Capacity: 10})
	err := cache.SetMaxAge(-1 * time.Hour)
	assert.Error(t, err)

	err = cache.SetMaxAge(time.Second)
	assert.NoError(t, err)
}

func TestSetMinAge(t *testing.T) {
	cache := New(Config{Capacity: 10, MaxAge: time.Hour})
	err := cache.SetMinAge(-1 * time.Hour)
	assert.Error(t, err)

	err = cache.SetMinAge(time.Second)
	assert.NoError(t, err)
}

func TestOnEviction(t *testing.T) {
	var eviction bool

	cache := New(Config{Capacity: 1})
	cache.OnEviction(func(key, value interface{}) {
		eviction = true
	})

	cache.Set("foo", 1)
	cache.Set("bar", 2)

	assert.True(t, eviction)
}

func TestOnExpiration(t *testing.T) {
	var expiration bool

	cache := New(Config{
		Capacity: 1,
		MaxAge:   time.Millisecond,
	})
	cache.OnExpiration(func(key, value interface{}) {
		expiration = true
	})

	cache.Set("foo", 1)
	<-time.After(time.Millisecond * 2)
	cache.Get("foo")

	assert.True(t, expiration)
}

func TestActiveExpiration(t *testing.T) {
	invoked := make(chan bool)

	cache := New(Config{
		Capacity:       1,
		MaxAge:         time.Millisecond,
		ExpirationType: ActiveExpiration,
	})

	cache.OnExpiration(func(key, value interface{}) {
		invoked <- true
	})

	cache.Set("foo", 1)
	start := time.Now()
	<-invoked
	duration := time.Now().Sub(start)

	assert.True(t, duration < time.Millisecond*2)
}

func TestResize(t *testing.T) {
	cache := New(Config{
		Capacity: 2,
	})
	cache.Set("a", 1)
	cache.Set("b", 1)

	cache.Resize(2) // no-op

	assert.True(t, cache.Has("a"))
	assert.True(t, cache.Has("b"))

	cache.Set("c", 1)
	assert.False(t, cache.Has("a"))
	assert.True(t, cache.Has("b"))
	assert.True(t, cache.Has("c"))

	cache.Resize(1)
	assert.False(t, cache.Has("a"))
	assert.False(t, cache.Has("b"))
	assert.True(t, cache.Has("c"))

	cache.Resize(2)
	cache.Set("d", 1)

	assert.False(t, cache.Has("a"))
	assert.False(t, cache.Has("b"))
	assert.True(t, cache.Has("c"))
	assert.True(t, cache.Has("d"))
}

func TestStats(t *testing.T) {
	t.Run("reports capacity", func(t *testing.T) {
		cache := New(Config{Capacity: 100})
		assert.Equal(t, int64(100), cache.Stats().Capacity)
	})

	t.Run("reports count", func(t *testing.T) {
		cache := New(Config{Capacity: 100})
		for i := 0; i < 10; i++ {
			cache.Set(i, i)
		}
		assert.Equal(t, int64(10), cache.Stats().Count)
	})

	t.Run("increments sets", func(t *testing.T) {
		cache := New(Config{Capacity: 100, MaxAge: time.Second})
		for i := 0; i < 10; i++ {
			cache.Set("foo", "bar")
		}
		assert.Equal(t, int64(10), cache.Stats().Sets)
	})

	t.Run("increments gets", func(t *testing.T) {
		cache := New(Config{Capacity: 100, MaxAge: time.Second})
		for i := 0; i < 10; i++ {
			cache.Get("foo")
		}
		assert.Equal(t, int64(10), cache.Stats().Gets)
	})

	t.Run("increments hits", func(t *testing.T) {
		cache := New(Config{Capacity: 100, MaxAge: time.Second})
		cache.Set("foo", "bar")
		cache.Get("foo")
		assert.Equal(t, int64(1), cache.Stats().Gets)
	})

	t.Run("increments misses", func(t *testing.T) {
		cache := New(Config{Capacity: 100, MaxAge: time.Second})
		cache.Get("foo")
		assert.Equal(t, int64(1), cache.Stats().Misses)
	})

	t.Run("increments evictions", func(t *testing.T) {
		cache := New(Config{Capacity: 1, MaxAge: time.Second})
		for i := 0; i < 10; i++ {
			cache.Set(i, i)
		}
		assert.Equal(t, int64(9), cache.Stats().Evictions)
	})

	t.Run("delta stats", func(t *testing.T) {
		cache := New(Config{Capacity: 100, MaxAge: time.Second})
		cache.Set("a", "1")
		prev := cache.Stats()

		for i := 0; i < 10; i++ {
			cache.Get("a") // hit
			cache.Get("b") // miss
			cache.Get("a") // hit

			stats := cache.Stats().Delta(prev)
			assert.Equal(t, int64(100), stats.Capacity)
			assert.Equal(t, int64(1), stats.Count)
			assert.Equal(t, int64(0), stats.Sets)
			assert.Equal(t, int64(3), stats.Gets)
			assert.Equal(t, int64(2), stats.Hits)
			assert.Equal(t, int64(1), stats.Misses)
			assert.Equal(t, int64(0), stats.Evictions)

			prev = cache.Stats()
		}
	})

	t.Run("copy", func(t *testing.T) {
		cache := New(Config{Capacity: 100, MaxAge: time.Second})
		stats := cache.Stats()
		stats.Hits++
		stats.Misses++

		assert.Equal(t, Stats{Capacity: 100}, cache.Stats())
	})
}

func BenchmarkCache(b *testing.B) {
	cache := New(Config{Capacity: 100, MaxAge: time.Second})

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			cache.Set("a", "b")
			cache.Get("a")
		}
	})
}

// LRU Sampling Tests

func TestInvalidLRUSamplingRate(t *testing.T) {
	assert.Panics(t, func() {
		New(Config{Capacity: 1, LRUSamplingRate: -0.1})
	})

	assert.Panics(t, func() {
		New(Config{Capacity: 1, LRUSamplingRate: 1.1})
	})
}

func TestDefaultLRUSamplingBehaviorUnchanged(t *testing.T) {
	// Test that default (zero value) config behaves identically to 100% sampling
	defaultCache := New(Config{Capacity: 3})
	explicitCache := New(Config{Capacity: 3, LRUSamplingRate: 1.0})

	keys := []string{"a", "b", "c", "d"}

	// Fill both caches identically
	for _, key := range keys[:3] {
		defaultCache.Set(key, key+"_value")
		explicitCache.Set(key, key+"_value")
	}

	// Access items in same pattern
	for i := 0; i < 10; i++ {
		defaultCache.Get("a")
		explicitCache.Get("a")
		defaultCache.Get("b")
		explicitCache.Get("b")
	}

	// Add new item to trigger eviction
	defaultCache.Set("d", "d_value")
	explicitCache.Set("d", "d_value")

	// Both should evict "c" (least recently used)
	_, foundInDefault := defaultCache.Get("c")
	_, foundInExplicit := explicitCache.Get("c")

	assert.False(t, foundInDefault)
	assert.False(t, foundInExplicit)

	// Both should still have "a", "b", "d"
	for _, key := range []string{"a", "b", "d"} {
		_, foundInDefault := defaultCache.Get(key)
		_, foundInExplicit := explicitCache.Get(key)
		assert.True(t, foundInDefault)
		assert.True(t, foundInExplicit)
	}
}

func TestLRUSamplingFunctionality(t *testing.T) {
	// Test basic functionality with various sampling rates
	testCases := []float64{1.0, 0.5, 0.1}

	for _, samplingRate := range testCases {
		t.Run(fmt.Sprintf("sampling_%.1f", samplingRate), func(t *testing.T) {
			cache := New(Config{Capacity: 10, LRUSamplingRate: samplingRate})

			// Basic set/get should work regardless of sampling rate
			cache.Set("key1", "value1")
			cache.Set("key2", "value2")

			value, found := cache.Get("key1")
			assert.True(t, found)
			assert.Equal(t, "value1", value)

			// Stats should still be accurate
			stats := cache.Stats()
			assert.Equal(t, int64(2), stats.Sets)
			assert.Equal(t, int64(1), stats.Gets)
			assert.Equal(t, int64(1), stats.Hits)
		})
	}
}

func TestSamplingWithExpiration(t *testing.T) {
	// Test that expired item handling works correctly with sampling
	cache := New(Config{Capacity: 10, MaxAge: 50 * time.Millisecond, LRUSamplingRate: 0.1})

	cache.Set("key1", "value1")

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should return miss for expired items regardless of sampling path
	value, found := cache.Get("key1")
	assert.False(t, found)
	assert.Nil(t, value)
}

func TestConcurrentGetConsistency(t *testing.T) {
	// Test that concurrent gets with sampling don't cause race conditions
	cache := New(Config{Capacity: 100, LRUSamplingRate: 0.5})

	// Fill cache
	for i := 0; i < 50; i++ {
		cache.Set(i, i*10)
	}

	var wg sync.WaitGroup
	numGoroutines := 10
	getsPerGoroutine := 50

	// Run concurrent gets
	for g := 0; g < numGoroutines; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < getsPerGoroutine; i++ {
				key := i % 50
				value, found := cache.Get(key)
				if found {
					assert.Equal(t, key*10, value)
				}
			}
		}()
	}

	wg.Wait()
}

func BenchmarkGetConcurrency(b *testing.B) {
	testCases := []struct {
		name         string
		samplingRate float64
	}{
		{"Traditional_100pct", 1.0},
		{"Sampling_10pct", 0.1},
		{"Sampling_1pct", 0.01},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			cache := New(Config{Capacity: 1000, LRUSamplingRate: tc.samplingRate, SampleStats: true})

			// Pre-fill cache
			for i := 0; i < 500; i++ {
				cache.Set(i, i*10)
			}

			b.ResetTimer()
			// Use per-goroutine RNG and precomputed keys to avoid time.Now() inside hot loop
			const keySpace = 500
			const ringSize = 1024 // power-of-two for efficient wrap
			const mask = ringSize - 1

			b.RunParallel(func(pb *testing.PB) {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				keys := make([]int, ringSize)
				for i := range keys {
					keys[i] = r.Intn(keySpace)
				}
				idx := 0
				for pb.Next() {
					key := keys[idx&mask]
					cache.Get(key)
					idx++
				}
			})
		})
	}
}

// BenchmarkGetSamplingSweep measures Get throughput across a range of
// LRUSamplingRate values to find the performance sweet spot.
func BenchmarkGetSamplingSweep(b *testing.B) {
	rates := []float64{
		1.0,
		0.75, 0.5,
		0.33, 0.25, 0.2,
		0.15, 0.125, 0.1,
		0.08, 0.06, 0.05,
		0.04, 0.03, 0.02,
		0.015, 0.01, 0.0075, 0.005,
	}

	for _, rate := range rates {
		b.Run(fmt.Sprintf("rate_%g", rate), func(b *testing.B) {
			cache := New(Config{Capacity: 1000, LRUSamplingRate: rate, SampleStats: true})

			// Pre-fill cache
			const keySpace = 500
			for i := 0; i < keySpace; i++ {
				cache.Set(i, i*10)
			}

			b.ResetTimer()

			// Per-goroutine RNG and precomputed keys to avoid time.Now() in hot loop
			const ringSize = 1024
			const mask = ringSize - 1

			b.RunParallel(func(pb *testing.PB) {
				r := rand.New(rand.NewSource(time.Now().UnixNano()))
				keys := make([]int, ringSize)
				for i := range keys {
					keys[i] = r.Intn(keySpace)
				}
				idx := 0
				for pb.Next() {
					key := keys[idx&mask]
					cache.Get(key)
					idx++
				}
			})
		})
	}
}
