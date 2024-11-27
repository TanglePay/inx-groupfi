package im

import (
	"encoding/binary"
	"fmt"
	"time"

	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
)

// Constants
const (
	RecentOutputIdsLimit = 7 // Maximum number of recent output IDs to store
)

// GetRecentConsumedOutputKey generates the key for a recent consumed output ID.
// The key format is: [Prefix][AddressHash][Timestamp][OutputID]
func GetRecentConsumedOutputKey(addressSha256Hash [Sha256HashLen]byte, timestamp int64, outputID [OutputIdLen]byte) []byte {
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))
	return ConcatByteSlices(
		[]byte{ImStoreKeyPrefixRecentConsumedOutputIds},
		addressSha256Hash[:],
		timestampBytes,
		outputID[:],
	)
}

// StoreRecentConsumedOutputId stores a recent consumed outputID with a timestamp,
// ensuring that only the most recent 7 are kept by deleting older entries during iteration.
func StoreRecentConsumedOutputId(addressSha256Hash [Sha256HashLen]byte, outputID [OutputIdLen]byte, im *Manager) error {
	// log
	Logger.Infof("StoreRecentConsumedOutputId ... address:%s, outputID:%s", iotago.EncodeHex(addressSha256Hash[:]), iotago.EncodeHex(outputID[:]))
	// Get current timestamp in nanoseconds
	currentTimestamp := time.Now().UnixNano()

	// Generate the storage key
	key := GetRecentConsumedOutputKey(addressSha256Hash, currentTimestamp, outputID)

	// Store the outputID in the KV store with an empty value or relevant data
	err := im.imStore.Set(key, nil)
	if err != nil {
		return fmt.Errorf("failed to store recent consumed output ID: %w", err)
	}

	// Now, ensure that only the most recent 7 outputIDs are kept
	// We need to iterate over the keys with the given prefix in ascending order (oldest first)
	keyPrefix := ConcatByteSlices([]byte{ImStoreKeyPrefixRecentConsumedOutputIds}, addressSha256Hash[:])

	count := 0

	err = im.imStore.Iterate(keyPrefix, func(key, value []byte) bool {
		count++
		if count > RecentOutputIdsLimit {
			// Delete the key immediately during iteration
			delErr := im.imStore.Delete(key)
			if delErr != nil {
				// Handle deletion error (log it and continue, or stop iteration)
				// Here, we'll stop iteration and return false to indicate failure
				fmt.Printf("Error deleting key %x: %v\n", key, delErr)
				return false // Stop iteration on error
			}
			// Continue iteration after deletion
			return true
		}
		// Continue iteration
		return true
	}, kvstore.IterDirectionBackward)
	// log count
	Logger.Infof("StoreRecentConsumedOutputId ... address:%s, count:%d", iotago.EncodeHex(addressSha256Hash[:]), count)
	if err != nil {
		return fmt.Errorf("failed to iterate recent consumed output IDs: %w", err)
	}

	return nil
}

// GetRecentConsumedOutputIds retrieves the most recent 7 consumed outputIDs for a given address.
func GetRecentConsumedOutputIds(addressSha256Hash [Sha256HashLen]byte, im *Manager) ([][OutputIdLen]byte, error) {
	// Generate the key prefix to iterate over
	keyPrefix := ConcatByteSlices([]byte{ImStoreKeyPrefixRecentConsumedOutputIds}, addressSha256Hash[:])

	// Collect outputIDs in descending order (newest first)
	var recentOutputIDs [][OutputIdLen]byte

	err := im.imStore.Iterate(keyPrefix, func(key, value []byte) bool {
		// Extract the OutputID from the key, outputId is last outputIdLen bytes
		var oid [OutputIdLen]byte
		copy(oid[:], key[len(key)-OutputIdLen:])
		recentOutputIDs = append(recentOutputIDs, oid)
		return true // Continue iteration
	}, kvstore.IterDirectionBackward)

	if err != nil {
		return nil, fmt.Errorf("failed to iterate recent consumed output IDs: %w", err)
	}

	return recentOutputIDs, nil
}
