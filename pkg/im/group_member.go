package im

import (
	"encoding/binary"
	"sort"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

// struct for group member
type GroupMember struct {
	// group id
	GroupId [GroupIdLen]byte
	// address
	Address string
	// timestamp
	Timestamp uint32
}

// newGroupMember creates a new GroupMember.
func NewGroupMember(groupId [GroupIdLen]byte, address string, timestamp uint32) *GroupMember {
	return &GroupMember{
		GroupId:   groupId,
		Address:   address,
		Timestamp: timestamp,
	}
}

// key = prefix + groupid + addressSha256Hash. value = timestamp + address
func (im *Manager) GroupMemberKey(groupMember *GroupMember) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupMember
	index++
	copy(key[index:], groupMember.GroupId[:])
	index += GroupIdLen
	copy(key[index:], Sha256Hash(groupMember.Address))
	return key
}

// group member key from groupid + addressSha256Hash
func (im *Manager) GroupMemberKeyFromGroupIdAndAddressSha256Hash(groupId [GroupIdLen]byte, addressSha256Hash [Sha256HashLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupMember
	index++
	copy(key[index:], groupId[:])
	index += GroupIdLen
	copy(key[index:], addressSha256Hash[:])
	return key
}

// store group member
func (im *Manager) StoreGroupMember(groupMember *GroupMember, logger *logger.Logger) error {
	key := im.GroupMemberKey(groupMember)
	value := make([]byte, 4+len(groupMember.Address))
	index := 0
	binary.LittleEndian.PutUint32(value[index:], groupMember.Timestamp)
	index += 4
	copy(value[index:], groupMember.Address)
	// log group member key and value
	logger.Infof("StoreGroupMember,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
	return im.imStore.Set(key, value)
}

// delete group member
func (im *Manager) DeleteGroupMember(groupMember *GroupMember, logger *logger.Logger) error {
	key := im.GroupMemberKey(groupMember)
	// log group member key
	logger.Infof("DeleteGroupMember,key:%s", iotago.EncodeHex(key))
	return im.imStore.Delete(key)
}

// check if group member exists, input is group id and address
func (im *Manager) GroupMemberExists(groupId [GroupIdLen]byte, address string) (bool, error) {
	key := im.GroupMemberKey(NewGroupMember(groupId, address, 0))
	return im.imStore.Has(key)
}

// check if group member exists, input is group id and addressHash
func (im *Manager) GroupMemberExistsFromGroupIdAndAddressSha256Hash(groupId [GroupIdLen]byte, addressSha256Hash [Sha256HashLen]byte) (bool, error) {
	key := im.GroupMemberKeyFromGroupIdAndAddressSha256Hash(groupId, addressSha256Hash)
	return im.imStore.Has(key)
}

// GroupMemberKeyPrefix returns the prefix for the given group id.
func (im *Manager) GroupMemberKeyPrefix(groupId [GroupIdLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupMember
	index++
	copy(key[index:], groupId[:])
	return key
}

// GetGroupMemberFromKeyAndValue
func (im *Manager) GetGroupMemberFromKeyAndValue(key kvstore.Key, value kvstore.Value) *GroupMember {
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1:1+GroupIdLen])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], value[:TimestampLen])
	address := string(value[TimestampLen:])
	return NewGroupMember(groupId, address, binary.LittleEndian.Uint32(timestamp[:]))
}

// get all group members, input is group id, sorted by timestamp
func (im *Manager) GetGroupMembers(groupId [GroupIdLen]byte) ([]*GroupMember, error) {
	prefix := im.GroupMemberKeyPrefix(groupId)
	groupMembers := make([]*GroupMember, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		groupMember := im.GetGroupMemberFromKeyAndValue(key, value)
		groupMembers = append(groupMembers, groupMember)
		return true
	})
	if err != nil {
		return nil, err
	}
	// sort group members by timestamp
	sort.Slice(groupMembers, func(i, j int) bool {
		return groupMembers[i].Timestamp > groupMembers[j].Timestamp
	})
	return groupMembers, nil
}

func (im *Manager) GetGroupMemberAddressesFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) ([]string, error) {
	groupMembers, err := im.GetGroupMembers(groupId)
	if err != nil {
		return nil, err
	}
	addresses := make([]string, len(groupMembers))
	for i, groupMember := range groupMembers {
		addresses[i] = groupMember.Address
	}
	return addresses, nil
}

// get group member addresses count
func (im *Manager) GetGroupMemberAddressesCountFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) (int, error) {
	groupMembers, err := im.GetGroupMembers(groupId)
	if err != nil {
		return 0, err
	}
	return len(groupMembers), nil
}
