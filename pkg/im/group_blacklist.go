package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

// key = prefix + groupid + addressSha256Hash, value = empty
func (im *Manager) GroupBlacklistKey(groupId [GroupIdLen]byte, addressSha256Hash [Sha256HashLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupBlacklist
	index++
	copy(key[index:], groupId[:])
	index += GroupIdLen
	copy(key[index:], addressSha256Hash[:])
	return key
}

// add address to group blacklist
func (im *Manager) AddAddressToGroupBlacklist(addressSha256Hash [Sha256HashLen]byte, groupId [GroupIdLen]byte) error {
	key := im.GroupBlacklistKey(groupId, addressSha256Hash)
	value := []byte{}
	return im.imStore.Set(key, value)
}

// remove address from group blacklist
func (im *Manager) RemoveAddressFromGroupBlacklist(addressSha256Hash [Sha256HashLen]byte, groupId [GroupIdLen]byte) error {
	key := im.GroupBlacklistKey(groupId, addressSha256Hash)
	return im.imStore.Delete(key)
}

// GroupBlacklistKeyPrefix returns the prefix for the given group id.
func (im *Manager) GroupBlacklistKeyPrefix(groupId [GroupIdLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupBlacklist
	index++
	copy(key[index:], groupId[:])
	return key
}

// get all addresses from group blacklist for group id
func (im *Manager) GetAddresseHashsFromGroupBlacklist(groupId [GroupIdLen]byte, logger *logger.Logger) ([]string, error) {
	prefix := im.GroupBlacklistKeyPrefix(groupId)
	addresses := make([]string, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		address := key[1+GroupIdLen:]
		addresses = append(addresses, iotago.EncodeHex(address))
		return true
	})
	return addresses, err
}
