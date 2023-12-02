package im

import (
	"sync"
	"time"
)

type Debouncer struct {
	timers map[string]*time.Timer
	mutex  sync.Mutex
}

var (
	instance *Debouncer
	once     sync.Once
)

// GetDebouncer returns the singleton instance of Debouncer. It initializes the instance if it hasn't been created yet.
func GetDebouncer() *Debouncer {
	once.Do(func() {
		instance = &Debouncer{
			timers: make(map[string]*time.Timer),
		}
	})
	return instance
}

// Debounce debounces a function call based on the specified key with the given delay.
func (d *Debouncer) Debounce(key string, delay time.Duration, f func()) {
	d.mutex.Lock()
	defer d.mutex.Unlock()

	if timer, exists := d.timers[key]; exists {
		timer.Stop()
	}

	d.timers[key] = time.AfterFunc(delay, func() {
		d.mutex.Lock()
		delete(d.timers, key)
		d.mutex.Unlock()
		f()
	})
}
