package im

import (
	"bytes"

	"github.com/iotaledger/hive.go/core/kvstore"
)

// key for group message count, key = prefix + groupId + timestampForHour
func KeyForGroupMessageCount(groupId [GroupIdLen]byte, timestampForHour uint32) []byte {
	key := make([]byte, 1+GroupIdLen+4)
	key[0] = ImStoreKeyPrefixGroupMessageCount
	copy(key[1:], groupId[:])
	timeBytes := Uint32ToBytes(timestampForHour)
	copy(key[1+GroupIdLen:], timeBytes)
	return key
}

// key for time group message count, key = prefix + timestampForHour + groupId
func KeyForTimeGroupMessageCount(timestampForHour uint32, groupId [GroupIdLen]byte) []byte {
	key := make([]byte, 1+4+GroupIdLen)
	key[0] = ImStoreKeyPrefixTimeGroupMessageCount
	timeBytes := Uint32ToBytes(timestampForHour)
	copy(key[1:], timeBytes)
	copy(key[1+4:], groupId[:])
	return key
}

// increment message count by groupId, given groupId, using CurrentMilestoneTimestamp and func StartOfHour(epochTimestamp uint32) uint32
// get update then store to kvstore
func IncrementGroupMessageCount(groupId [GroupIdLen]byte, im *Manager) error {
	timestampForHour := StartOfHour(CurrentMilestoneTimestamp)
	groupKey := KeyForGroupMessageCount(groupId, timestampForHour)
	timeGroupKey := KeyForTimeGroupMessageCount(timestampForHour, groupId)

	// Increment the count for both keys
	if err := incrementCountForKey(im.imStore, groupKey); err != nil {
		return err
	}

	if err := incrementCountForKey(im.imStore, timeGroupKey); err != nil {
		return err
	}

	return nil
}

// helper function to increment the count for a given key
func incrementCountForKey(store kvstore.KVStore, key kvstore.Key) error {
	// Retrieve the current count
	value, err := store.Get(key)
	if err != nil && err != kvstore.ErrKeyNotFound {
		return err
	}

	var count uint32
	if len(value) > 0 {
		count = BytesToUint32(value)
	}

	// Increment the count
	count++

	// Convert back to bytes and store
	countBytes := Uint32ToBytes(count)
	return store.Set(key, countBytes)
}

// Struct to hold groupId, timestampOfHour, and messageCount
type GroupMessageInfo struct {
	GroupId         [GroupIdLen]byte
	TimestampOfHour uint32
	MessageCount    uint32
}

// iterate function for KeyForTimeGroupMessageCount to return a list of {groupId, timestampOfHour, messageCount} after a given timestampForHour
func GetGroupMessagesAfterTimestamp(timestampForHour uint32, im *Manager) ([]GroupMessageInfo, error) {
	var result []GroupMessageInfo

	// Start key (inclusive)
	startKey := KeyForTimeGroupMessageCount(timestampForHour, [GroupIdLen]byte{})

	// Iterate over the keys starting from the given timestamp
	err := im.imStore.Iterate(startKey, func(key kvstore.Key, value kvstore.Value) bool {
		// Extract the timestamp and groupId from the key
		extractedTimestamp := BytesToUint32(key[1:(1 + 4)])
		var extractedGroupId [GroupIdLen]byte
		copy(extractedGroupId[:], key[1+4:])

		// Get the message count from the value
		messageCount := BytesToUint32(value)

		// Append to the result list
		result = append(result, GroupMessageInfo{
			GroupId:         extractedGroupId,
			TimestampOfHour: extractedTimestamp,
			MessageCount:    messageCount,
		})

		return true // Continue iteration
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetMessageCountForGroupInRange retrieves the message count for a given groupId within an optional time range.
func GetMessageCountForGroupInRange(groupId [GroupIdLen]byte, startTimestamp, endTimestamp uint32, im *Manager) (uint32, error) {
	var totalCount uint32

	// Use the earliest possible timestamp if startTimestamp is not provided
	if startTimestamp == 0 {
		startTimestamp = 0
	}

	// Use the current timestamp as the end if endTimestamp is not provided
	if endTimestamp == 0 {
		endTimestamp = uint32(CurrentMilestoneTimestamp)
	}

	// Start key (inclusive) for iteration
	startKey := KeyForGroupMessageCount(groupId, startTimestamp)

	// End key (inclusive) for iteration, add 1 to make it exclusive
	endKey := KeyForGroupMessageCount(groupId, endTimestamp+1)

	// Iterate over the keys within the range
	err := im.imStore.Iterate(startKey, func(key kvstore.Key, value kvstore.Value) bool {
		// Stop iteration if the key exceeds the end key
		if bytes.Compare(key, endKey) >= 0 {
			return false
		}

		// Add the count to the total
		if len(value) > 0 {
			count := BytesToUint32(value)
			totalCount += count
		}
		return true // Continue iteration
	})

	if err != nil {
		return 0, err
	}

	return totalCount, nil
}
