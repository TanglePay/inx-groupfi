package im

import (
	"crypto/sha256"
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
func Sha256Hash(str string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(str))
	return hasher.Sum(nil)
}
func Sha256HashBytes(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

func ConcatByteSlices(slices ...[]byte) []byte {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}
	result := make([]byte, totalLen)

	var offset int
	for _, s := range slices {
		copy(result[offset:], s)
		offset += len(s)
	}
	return result
}
