package im

import (
	"container/list"
	"hash/fnv"
	"sync"
)

// Number of shards. This can be tuned based on expected concurrency.
const shardCount = 512

// Cache is a thread-safe sharded LRU cache.
type Cache struct {
	shards []*cacheShard
}

// cacheShard represents a single shard of the sharded cache.
// It is renamed from Cache to avoid naming conflicts.
type cacheShard struct {
	capacity int
	mu       sync.Mutex
	cache    map[string]*list.Element
	ll       *list.List
}

// entry represents a key-value pair stored in the cache.
type entry struct {
	key   string
	value interface{}
}

// NewCache creates a new sharded LRU Cache with the given total capacity.
// The capacity is divided equally among all shards.
func NewCache(totalCapacity int) *Cache {
	if totalCapacity <= 0 {
		panic("total capacity must be greater than 0")
	}

	shards := make([]*cacheShard, shardCount)
	perShardCapacity := totalCapacity / shardCount
	if perShardCapacity == 0 {
		perShardCapacity = 1
	}

	for i := 0; i < shardCount; i++ {
		shards[i] = newCacheShard(perShardCapacity)
	}

	return &Cache{
		shards: shards,
	}
}

// newCacheShard creates a new cacheShard with the given capacity.
func newCacheShard(capacity int) *cacheShard {
	return &cacheShard{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		ll:       list.New(),
	}
}

// getShard returns the shard corresponding to the given key.
func (c *Cache) getShard(key string) *cacheShard {
	hash := fnv32(key)
	return c.shards[hash%uint32(shardCount)]
}

// Get retrieves a value from the cache by key.
// It returns the value and a boolean indicating whether the key was found.
func (c *Cache) Get(key string) (interface{}, bool) {
	shard := c.getShard(key)
	return shard.Get(key)
}

// Put adds a key-value pair to the cache.
// If the key already exists, it updates the value and moves it to the front.
// If the cache is at capacity, it removes the least recently used item.
func (c *Cache) Put(key string, value interface{}) {
	shard := c.getShard(key)
	shard.Put(key, value)
}

// Remove deletes a key from the cache.
func (c *Cache) Remove(key string) {
	shard := c.getShard(key)
	shard.Remove(key)
}

// Len returns the total number of items in the cache across all shards.
func (c *Cache) Len() int {
	total := 0
	for _, shard := range c.shards {
		total += shard.Len()
	}
	return total
}

// Clear removes all items from the cache across all shards.
func (c *Cache) Clear() {
	var wg sync.WaitGroup
	wg.Add(shardCount)
	for _, shard := range c.shards {
		go func(s *cacheShard) {
			defer wg.Done()
			s.Clear()
		}(shard)
	}
	wg.Wait()
}

// fnv32 computes the FNV-1a hash of a string and returns a uint32 hash value.
func fnv32(key string) uint32 {
	h := fnv.New32a()
	h.Write([]byte(key))
	return h.Sum32()
}

// ------------------- Renamed cacheShard Implementation -------------------

// Get retrieves a value from the shard by key.
// It returns the value and a boolean indicating whether the key was found.
func (s *cacheShard) Get(key string) (interface{}, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.cache[key]; ok {
		s.ll.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return nil, false
}

// Put adds a key-value pair to the shard.
// If the key already exists, it updates the value and moves it to the front.
// If the shard is at capacity, it removes the least recently used item.
func (s *cacheShard) Put(key string, value interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.cache[key]; ok {
		s.ll.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}

	if s.ll.Len() >= s.capacity {
		s.removeOldest()
	}

	newEntry := &entry{key, value}
	elem := s.ll.PushFront(newEntry)
	s.cache[key] = elem
}

// Remove deletes a key from the shard.
func (s *cacheShard) Remove(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if elem, ok := s.cache[key]; ok {
		s.removeElement(elem)
	}
}

// Len returns the number of items in the shard.
func (s *cacheShard) Len() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.ll.Len()
}

// removeOldest removes the least recently used item from the shard.
func (s *cacheShard) removeOldest() {
	elem := s.ll.Back()
	if elem != nil {
		s.removeElement(elem)
	}
}

// removeElement removes a given list element from the shard and the linked list.
func (s *cacheShard) removeElement(elem *list.Element) {
	s.ll.Remove(elem)
	kv := elem.Value.(*entry)
	delete(s.cache, kv.key)
}

// Clear removes all items from the shard.
func (s *cacheShard) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ll.Init()
	s.cache = make(map[string]*list.Element)
}
