package im

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/core/kvstore"
)

// get key for ttl store, key = prefix + timetoexpire + hash of data value
func GetTtlStoreKey(val kvstore.Value, ttl uint32) []byte {
	ttlKey := make([]byte, 1+4+Sha256HashLen)
	ttlKey[0] = ImStoreKeyPrefixTtl
	timeBytes := Uint32ToBytes(CurrentMilestoneTimestamp + ttl)
	copy(ttlKey[1:], timeBytes)
	copy(ttlKey[1+4:], Sha256HashBytes(val))
	return ttlKey
}

// store key of data to ttl store as value, key = prefix + timetoexpire + hash of data value
func StoreKeyAndValueToTtlStore(key kvstore.Key, value kvstore.Value, ttl uint32, im *Manager) error {
	ttlKey := GetTtlStoreKey(value, ttl)

	return im.imStore.Set(ttlKey, key)
}

// get key for ttl store, with given timestamp as expire time, key = prefix + timestamp + hash of data value, hash of data value use zero value
func GetTimestampKey(timestamp uint32) []byte {
	ttlKey := make([]byte, 1+4+Sha256HashLen)
	ttlKey[0] = ImStoreKeyPrefixTtl
	timeBytes := Uint32ToBytes(timestamp)
	copy(ttlKey[1:], timeBytes)
	return ttlKey
}

// get prefix for ttl store, key = prefix
func GetTtlPrefix() []byte {
	return []byte{ImStoreKeyPrefixTtl}
}

// iterate ttl store, get all keys that expired, and delete them
func CleanTtlStoreUntilNow(im *Manager) error {
	keyForCurrentTime := GetTimestampKey(CurrentMilestoneTimestamp)
	// iterate ttl store
	prefix := GetTtlPrefix()
	return im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// if key is expired, delete it, compare key with keyForCurrentTime
		if bytes.Compare(key, keyForCurrentTime) < 0 {
			im.imStore.Delete(key)
			return true
		} else {
			return false
		}
	})
}

func CleanTtlStore(ctx context.Context, im *Manager) error {
	ticker := time.NewTicker(2 * time.Second) // Set the timer for 2 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			fmt.Println("Clean Ttl Timer stopped:", ctx.Err())
			return nil
		case <-ticker.C:
			CleanTtlStoreUntilNow(im)
		}
	}
}
