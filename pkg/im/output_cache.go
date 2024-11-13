package im

import (
	"sync"
)

// Initialize a global cache with capacity 1000
var (
	OutputCache     *Cache
	outputCacheOnce sync.Once
)

// InitializeCache initializes the global OutputCache with a given capacity.
// It uses sync.Once to ensure that the cache is only initialized once.
func InitializeOutputCache(capacity int) {
	outputCacheOnce.Do(func() {
		OutputCache = NewCache(capacity)
	})
}
