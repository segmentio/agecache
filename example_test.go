package agecache

import (
	"fmt"
	"time"
)

func ExampleNew() {
	// Create a new cache of type string, that expires after 10 mintues
	cache := NewGeneric(Config[string]{
		Capacity:           10,
		ExpirationInterval: time.Minute * 10,
	})

	cache.Set("key", "value")
	value, ok := cache.Get("key")
	fmt.Printf("%v: %s\n", ok, *value)

	// Output: true: value
}
