package main

import (
	"fmt"
	"log"
	"time"

	"github.com/segmentio/agecache"
)

func main() {
	c, err := agecache.New(100, time.Second)
	if err != nil {
		log.Fatalf("new: %s", err)
	}

	c.Add("a", "1")

	prev := c.Stats()
	tick := time.Tick(time.Millisecond * 100)

	for range tick {
		c.Get("a") // hit
		c.Get("b") // miss
		c.Get("a") // hit

		stats := c.Stats().Sub(prev)
		fmt.Println("hits", stats.Hits)
		fmt.Println("misses", stats.Misses)

		prev = c.Stats()
	}
}
