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
