package im

import (
	"container/list"
	"sync"
)

// Cache is a thread-safe LRU cache
type Cache struct {
	capacity int
	mu       sync.Mutex
	cache    map[string]*list.Element
	ll       *list.List
}

// entry represents a key-value pair stored in the cache
type entry struct {
	key   string
	value interface{}
}

// NewCache creates a new LRU Cache with the given capacity
func NewCache(capacity int) *Cache {
	if capacity <= 0 {
		panic("capacity must be greater than 0")
	}
	return &Cache{
		capacity: capacity,
		cache:    make(map[string]*list.Element),
		ll:       list.New(),
	}
}

// Get retrieves a value from the cache by key.
// It returns the value and a boolean indicating whether the key was found.
func (c *Cache) Get(key string) (interface{}, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		return elem.Value.(*entry).value, true
	}
	return nil, false
}

// Put adds a key-value pair to the cache.
// If the key already exists, it updates the value and moves it to the front.
// If the cache is at capacity, it removes the least recently used item.
func (c *Cache) Put(key string, value interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.ll.MoveToFront(elem)
		elem.Value.(*entry).value = value
		return
	}

	if c.ll.Len() >= c.capacity {
		c.removeOldest()
	}

	newEntry := &entry{key, value}
	elem := c.ll.PushFront(newEntry)
	c.cache[key] = elem
}

// Remove deletes a key from the cache.
func (c *Cache) Remove(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if elem, ok := c.cache[key]; ok {
		c.removeElement(elem)
	}
}

// Len returns the number of items in the cache.
func (c *Cache) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.ll.Len()
}

// removeOldest removes the least recently used item from the cache.
func (c *Cache) removeOldest() {
	elem := c.ll.Back()
	if elem != nil {
		c.removeElement(elem)
	}
}

// removeElement removes a given list element from the cache and the linked list.
func (c *Cache) removeElement(elem *list.Element) {
	c.ll.Remove(elem)
	kv := elem.Value.(*entry)
	delete(c.cache, kv.key)
}

// Clear removes all items from the cache.
func (c *Cache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ll.Init()
	c.cache = make(map[string]*list.Element)
}
