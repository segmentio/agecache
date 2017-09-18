package agecache

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAgeOut(t *testing.T) {
	c, err := New(100, time.Millisecond)
	assert.Nil(t, err)

	c.Add("foo", "bar")
	<-time.After(time.Millisecond * 2)

	val, ok := c.Get("foo")
	assert.False(t, ok)
	assert.Nil(t, val)
}

func TestTooManyItems(t *testing.T) {
	c, err := New(1, time.Minute)
	assert.Nil(t, err)

	c.Add("foo", "bar")
	c.Add("bar", "baz")
	val, ok := c.Get("foo")
	assert.False(t, ok)
	assert.Nil(t, val)

	val, ok = c.Get("bar")
	assert.True(t, ok)
	assert.Equal(t, val, "baz")
}

func TestError(t *testing.T) {
	_, err := New(-1, time.Minute)
	assert.Error(t, err)
}

func TestStats(t *testing.T) {
	t.Run("increments hits", func(t *testing.T) {
		c, err := New(100, time.Second)
		if err != nil {
			t.Fatalf("new: %s", err)
		}

		c.Add("foo", "bar")
		c.Get("foo")

		assert.Equal(t, Stats{Hits: 1}, c.Stats())
	})

	t.Run("increments misses", func(t *testing.T) {
		c, err := New(100, time.Second)
		if err != nil {
			t.Fatalf("new: %s", err)
		}

		c.Get("foo")

		assert.Equal(t, Stats{Misses: 1}, c.Stats())
	})

	t.Run("sub stats", func(t *testing.T) {
		c, err := New(100, time.Second)
		if err != nil {
			t.Fatalf("new: %s", err)
		}

		c.Add("a", "1")

		prev := c.Stats()

		for i := 0; i < 10; i++ {
			c.Get("a") // hit
			c.Get("b") // miss
			c.Get("a") // hit

			stats := c.Stats().Sub(prev)
			assert.Equal(t, Stats{Hits: 2, Misses: 1}, stats)

			prev = c.Stats()
		}
	})

	t.Run("copy", func(t *testing.T) {
		c, err := New(100, time.Second)
		if err != nil {
			t.Fatalf("new: %s", err)
		}

		stats := c.Stats()
		stats.Hits++
		stats.Misses++

		assert.Equal(t, Stats{}, c.Stats())
	})
}

func BenchmarkCache(b *testing.B) {
	c, err := New(100, time.Second)
	if err != nil {
		b.Fatalf("new: %s", err)
	}

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			c.Add("a", "b")
			c.Get("a")
		}
	})
}
