package im

import "github.com/iotaledger/hive.go/core/kvstore"

type AddressGroup struct {
	// address group id
	GroupId       []byte
	AddressSha256 []byte
}

// new address group from address in bytes not hashed yet and group id
func NewAddressGroup(address []byte, groupId []byte) *AddressGroup {
	return &AddressGroup{
		GroupId:       groupId,
		AddressSha256: Sha256HashBytes(address),
	}
}

// key = prefix + addressSha256Hash + groupid
func (im *Manager) AddressGroupKey(addressGroup *AddressGroup) []byte {
	key := make([]byte, 1+Sha256HashLen+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressGroup
	index++
	copy(key[index:], addressGroup.AddressSha256)
	index += Sha256HashLen
	copy(key[index:], addressGroup.GroupId)
	return key
}

// prefix = prefix + addressSha256Hash
func (im *Manager) AddressGroupKeyPrefix(addressSha256 []byte) []byte {
	key := make([]byte, 1+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressGroup
	index++
	copy(key[index:], addressSha256)
	return key
}
func (im *Manager) StoreAddressGroup(addressGroup *AddressGroup) error {
	key := im.AddressGroupKey(addressGroup)
	return im.imStore.Set(key, nil)
}

// delete address group
func (im *Manager) DeleteAddressGroup(addressGroup *AddressGroup) error {
	key := im.AddressGroupKey(addressGroup)
	return im.imStore.Delete(key)
}

// get groupIds from address
func (im *Manager) GetGroupIdsFromAddress(addressSha256 []byte) ([][]byte, error) {
	prefix := im.AddressGroupKeyPrefix(addressSha256)
	groupIds := make([][]byte, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		groupId := key[1+Sha256HashLen:]
		groupIds = append(groupIds, groupId)
		return true
	})
	return groupIds, err
}
