# agecache

Thread-safe LRU cache supporting expiration and jitter. Supports cache
statistics, as well as eviction and expiration callbacks. Differs from
some implementations in that OnEviction is only invoked when an entry
is removed as a result of the LRU eviction policy - not when you explicitly
delete it or when it expires. OnExpiration is available and invoked when an
item expires. Expiration can be passively enforced when performing a Get,
or actively enforced by iterating over all keys with an interval.

``` go
cache := agecache.New(agecache.Config{
	Capacity: 100,
	MaxAge:   70 * time.Minute,
	MinAge:   60 * time.Minute,
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

## LRU Sampling (advanced)

`agecache` supports reducing lock contention on hot `Get` paths by sampling
how often LRU positions are updated.

- Configure with `Config.LRUSamplingRate` in the range `[0.0, 1.0]`.
  - `0.0` or the zero value behaves like `1.0` (traditional LRU update on every `Get`).
  - Lower values (e.g., `0.2`–`0.25`) can significantly improve throughput under high concurrency, at the cost of approximate LRU ordering.
- To keep stats inexpensive but useful, you can enable `Config.SampleStats`.
  - When `SampleStats` is `true`, stats counters (Gets/Hits/Misses) are updated only when an LRU update would happen, and are scaled by approximately `1/LRUSamplingRate` so they estimate the unsampled totals.
  - When `SampleStats` is `false` (default), stats are exact but may add contention in very hot paths.

Notes:
- Sampling changes eviction accuracy. For many workloads a rate around `0.2–0.25` is a good starting point; benchmark for your use case.
- With `SampleStats: true`, values are estimates and may vary slightly; with `false`, they are exact.
