package im

import (
	"encoding/binary"
	"sort"
	"time"

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

	// milestone index
	MilestoneIndex uint32

	// timestamp
	Timestamp uint32
}

// newGroupMember creates a new GroupMember.
func NewGroupMember(groupId [GroupIdLen]byte, address string, milestoneIndex uint32, timestamp uint32) *GroupMember {
	return &GroupMember{
		GroupId:        groupId,
		Address:        address,
		MilestoneIndex: milestoneIndex,
		Timestamp:      timestamp,
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

// member group, key = prefix + addressSha256Hash + groupid
func (im *Manager) MemberGroupKey(groupMember *GroupMember) []byte {
	key := make([]byte, 1+Sha256HashLen+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixMemberGroup
	index++
	copy(key[index:], Sha256Hash(groupMember.Address))
	index += Sha256HashLen
	copy(key[index:], groupMember.GroupId[:])
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
func (im *Manager) StoreGroupMember(groupMember *GroupMember, logger *logger.Logger) (bool, error) {
	// check if group member exists
	exists, err := im.GroupMemberExists(groupMember.GroupId, groupMember.Address)
	if err != nil {
		return false, err
	}
	isActuallyStored := !exists
	key := im.GroupMemberKey(groupMember)
	value := make([]byte, 4+len(groupMember.Address))
	index := 0
	binary.LittleEndian.PutUint32(value[index:], groupMember.Timestamp)
	index += 4
	copy(value[index:], groupMember.Address)

	// log group member key and value
	logger.Infof("StoreGroupMember,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
	err = im.imStore.Set(key, value)
	if err != nil {
		return isActuallyStored, err
	}
	// store member group
	err = im.StoreMemberGroup(groupMember, logger)
	if err != nil {
		return isActuallyStored, err
	}

	if isActuallyStored && !IsIniting {
		debouncer := GetDebouncer()
		key := "EventGroupMemberChanged:" + iotago.EncodeHex(groupMember.GroupId[:])
		debouncer.Debounce(key, 100*time.Millisecond, func() {
			// log
			logger.Infof("GroupMemberChangedEvent, debouncer.Debounce, key:%s", key)
			// get group members
			groupMembers, err := im.GetGroupMembers(groupMember.GroupId)
			if err != nil {
				logger.Errorf("GroupMemberChangedEvent, debouncer.Debounce, err:%s", err)
				return
			}
			// get group member addresses
			addresses := make([]string, len(groupMembers))
			for i, groupMember := range groupMembers {
				addresses[i] = groupMember.Address
			}

			// create group member changed event
			groupMemberChangedEvent := NewGroupMemberChangedEvent(groupMember.GroupId, groupMember.MilestoneIndex, groupMember.Timestamp)
			// store group member changed event to inbox
			for _, address := range addresses {
				addressSha256Hash := Sha256Hash(address)
				im.PushInbox(groupMemberChangedEvent.ToPushTopic(), groupMemberChangedEvent.ToPushPayload(), logger)
				err = im.StoreGroupMemberChangedEventToInbox(addressSha256Hash, groupMemberChangedEvent, logger)
				if err != nil {
					logger.Errorf("GroupMemberChangedEvent, debouncer.Debounce, err:%s", err)
					return
				}
			}
		})
	}
	return isActuallyStored, nil
}

// store member group
func (im *Manager) StoreMemberGroup(groupMember *GroupMember, logger *logger.Logger) error {
	key := im.MemberGroupKey(groupMember)
	// value is empty
	value := []byte{}
	// log group member key and value
	logger.Infof("StoreMemberGroup,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
	return im.imStore.Set(key, value)
}

// delete group member
func (im *Manager) DeleteGroupMember(groupMember *GroupMember, logger *logger.Logger) (bool, error) {
	// check if group member exists
	exists, err := im.GroupMemberExists(groupMember.GroupId, groupMember.Address)
	isActuallyDeleted := exists
	if err != nil {
		return false, err
	}

	key := im.GroupMemberKey(groupMember)
	// log group member key
	logger.Infof("DeleteGroupMember,key:%s", iotago.EncodeHex(key))
	err = im.imStore.Delete(key)
	if err != nil {
		return isActuallyDeleted, err
	}
	// delete member group as well
	err = im.DeleteMemberGroup(groupMember, logger)
	if err != nil {
		return isActuallyDeleted, err
	}
	if isActuallyDeleted && !IsIniting {
		debouncer := GetDebouncer()
		key := "EventGroupMemberChanged:" + iotago.EncodeHex(groupMember.GroupId[:])
		debouncer.Debounce(key, 100*time.Millisecond, func() {
			// log
			logger.Infof("GroupMemberChangedEvent, debouncer.Debounce, key:%s", key)
			// get group members
			groupMembers, err := im.GetGroupMembers(groupMember.GroupId)
			if err != nil {
				logger.Errorf("GroupMemberChangedEvent, debouncer.Debounce, err:%s", err)
				return
			}
			// get group member addresses
			addresses := make([]string, len(groupMembers))
			for i, groupMember := range groupMembers {
				addresses[i] = groupMember.Address
			}

			// create group member changed event
			groupMemberChangedEvent := NewGroupMemberChangedEvent(groupMember.GroupId, groupMember.MilestoneIndex, groupMember.Timestamp)
			// store group member changed event to inbox
			for _, address := range addresses {
				addressSha256Hash := Sha256Hash(address)
				im.PushInbox(groupMemberChangedEvent.ToPushTopic(), groupMemberChangedEvent.ToPushPayload(), logger)
				err = im.StoreGroupMemberChangedEventToInbox(addressSha256Hash, groupMemberChangedEvent, logger)
				if err != nil {
					logger.Errorf("GroupMemberChangedEvent, debouncer.Debounce, err:%s", err)
					return
				}
			}
		})
	}
	return isActuallyDeleted, nil
}

// delete member group
func (im *Manager) DeleteMemberGroup(groupMember *GroupMember, logger *logger.Logger) error {
	key := im.MemberGroupKey(groupMember)
	// log group member key
	logger.Infof("DeleteMemberGroup,key:%s", iotago.EncodeHex(key))
	return im.imStore.Delete(key)
}

// check if group member exists, input is group id and address
func (im *Manager) GroupMemberExists(groupId [GroupIdLen]byte, address string) (bool, error) {
	key := im.GroupMemberKey(NewGroupMember(groupId, address, 0, 0))
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

// member group key prefix
func (im *Manager) MemberGroupKeyPrefix(address string) []byte {
	key := make([]byte, 1+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixMemberGroup
	index++
	copy(key[index:], Sha256Hash(address))
	return key
}

// GetGroupMemberFromKeyAndValue
func (im *Manager) GetGroupMemberFromKeyAndValue(key kvstore.Key, value kvstore.Value) *GroupMember {
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1:1+GroupIdLen])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], value[:TimestampLen])
	address := string(value[TimestampLen:])
	return NewGroupMember(groupId, address, 0, BytesToUint32(timestamp[:]))
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

// get all member groups, return group ids in string
func (im *Manager) GetMemberGroups(address string) ([]string, error) {
	prefix := im.MemberGroupKeyPrefix(address)
	groupIds := make([]string, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		groupId := key[1+Sha256HashLen:]
		groupIds = append(groupIds, iotago.EncodeHex(groupId))
		return true
	})
	if err != nil {
		return nil, err
	}
	return groupIds, nil
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
