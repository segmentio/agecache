package agecache

import (
	"sort"
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

type MockRandGenerator struct{}

func (mock *MockRandGenerator) Intn(n int) int {
	// Always return max value
	return n
}

func TestJitter(t *testing.T) {
	cache := New(Config{
		Capacity: 1,
		MaxAge:   time.Millisecond * 3,
		MinAge:   time.Millisecond,
	})

	cache.rand = &MockRandGenerator{}

	cache.Set("foo", "bar")

	<-time.After(time.Millisecond * 2)
	_, ok := cache.Get("foo")
	assert.True(t, ok)

	<-time.After(time.Millisecond * 3)
	_, ok = cache.Get("foo")
	assert.False(t, ok)
}

func TestHas(t *testing.T) {
	cache := New(Config{Capacity: 1, MaxAge: time.Millisecond})
	cache.Set("foo", "bar")
	ok := cache.Has("foo")
	assert.True(t, ok)
	<-time.After(time.Millisecond * 2)

	ok = cache.Has("foo")
	assert.False(t, ok)
}

func TestPeek(t *testing.T) {
	cache := New(Config{Capacity: 1, MaxAge: time.Millisecond})
	cache.Set("foo", "bar")
	val, ok := cache.Peek("foo")
	assert.True(t, ok)
	assert.Equal(t, "bar", val)
	<-time.After(time.Millisecond * 2)

	val, ok = cache.Peek("foo")
	assert.False(t, ok)
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
