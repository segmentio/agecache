package main

import (
	"fmt"
	"time"

	"github.com/segmentio/agecache"
)

func main() {
	cache := agecache.New(agecache.Config[string, string]{
		Capacity: 100,
		MaxAge:   time.Second,
	})

	cache.Set("a", "1")

	prev := cache.Stats()
	tick := time.Tick(time.Millisecond * 100)

	for range tick {
		cache.Get("a") // hit
		cache.Get("b") // miss
		cache.Get("a") // hit

		stats := cache.Stats().Delta(prev)
		fmt.Println("hits", stats.Hits)
		fmt.Println("misses", stats.Misses)

		prev = cache.Stats()
	}
}
