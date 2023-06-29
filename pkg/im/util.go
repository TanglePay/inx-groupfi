package im

import (
	"sync"
)

var (
	incrementerOnce sync.Once
	incrementer     *Incrementer
)

type Incrementer struct {
	index   uint32
	counter uint32
	mu      sync.Mutex
}

func GetIncrementer() *Incrementer {
	incrementerOnce.Do(func() {
		incrementer = &Incrementer{}
	})
	return incrementer
}

func (inc *Incrementer) Increment(index uint32) uint32 {
	inc.mu.Lock()
	defer inc.mu.Unlock()

	if index != inc.index {
		inc.index = index
		inc.counter = 0
	}

	inc.counter++
	return inc.counter
}
