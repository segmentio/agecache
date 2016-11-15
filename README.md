
# agecache

Creates an LRU cache with a set max age. The main difference between
this and a simple LRU cache is that it will make sure to age out old entries.

The Agecache doesn't aim to be thread-safe (though the underlying
implementation is). Instead, it aims to give fast access, maintaining the
most recently used items, but expiring them after a certain time.

## Documentation

Full docs are available on [Godoc][godoc].

[godoc]: https://godoc.org/github.com/segmentio/agecache
