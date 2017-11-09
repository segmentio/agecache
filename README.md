# agecache

Thread-safe LRU cache supporting max age and expiration. Supports cache
statistics, as well as eviction and expiration callbacks. Differs from
some implementations in that OnEviction is only invoked when an entry
is removed as a result of the LRU eviction policy - not when you explicitly
delete it or when it expires. OnExpiration is available and invoked when an
item expires. Expiration can be passively enforced when performing a Get,
or actively enforced by iterating over all keys with an interval.

``` go
cache := agecache.New(agecache.Config{
	Capacity: 100,
	MaxAge:   time.Hour,
	OnExpiration: func(key, value interface{}) {
		// Handle expiration
	},
	OnEviction: func(key, value interface{}) {
		// Handle eviction
	},
})

cache.Set("foo", "bar")
```

## Documentation

Full docs are available on [Godoc][godoc].

[godoc]: https://godoc.org/github.com/segmentio/agecache
