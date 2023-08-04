package im

import (
	"math/big"
	"sync"
)

var (
	smrOnce sync.Once
	smr     *TokenTotal
)

type TokenTotal struct {
	totalAmount *big.Int
	rwLock      sync.RWMutex
}

// TokenTotal.Add
func (tt *TokenTotal) Add(amount *big.Int) {
	tt.rwLock.Lock()
	defer tt.rwLock.Unlock()
	tt.totalAmount.Add(tt.totalAmount, amount)
}

// TokenTotal.Sub
func (tt *TokenTotal) Sub(amount *big.Int) {
	tt.rwLock.Lock()
	defer tt.rwLock.Unlock()
	tt.totalAmount.Sub(tt.totalAmount, amount)
}

// TokenTotal.Get
func (tt *TokenTotal) Get() *big.Int {
	tt.rwLock.RLock()
	defer tt.rwLock.RUnlock()
	return tt.totalAmount
}

func GetSmrTokenTotal() *TokenTotal {
	smrOnce.Do(func() {
		smr = NewTokenTotal()
	})
	return smr
}

// new token total
func NewTokenTotal() *TokenTotal {
	return &TokenTotal{
		totalAmount: big.NewInt(0),
		rwLock:      sync.RWMutex{},
	}
}
