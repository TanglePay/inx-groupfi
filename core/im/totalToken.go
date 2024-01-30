package im

import (
	"math/big"
	"sync"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

var (
	totalMapOnce sync.Once
	lock         sync.Mutex
	totalMap     map[string]*TokenTotal
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
	// init totalMap using once
	totalMapOnce.Do(func() {
		totalMap = make(map[string]*TokenTotal)
	})
	tt.rwLock.RLock()
	defer tt.rwLock.RUnlock()
	return tt.totalAmount
}

func GetTokenTotal(tokenIdHash [im.Sha256HashLen]byte) *TokenTotal {
	key := iotago.EncodeHex(tokenIdHash[:])
	lock.Lock()
	defer lock.Unlock()
	total, ok := totalMap[key]
	if !ok {
		total = NewTokenTotal()
		totalMap[key] = total
	}
	return total
}

// new token total
func NewTokenTotal() *TokenTotal {
	return &TokenTotal{
		totalAmount: big.NewInt(0),
		rwLock:      sync.RWMutex{},
	}
}
