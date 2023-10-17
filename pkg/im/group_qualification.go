package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

// struct for group qualification
type GroupQualification struct {
	// group id
	GroupId [GroupIdLen]byte

	Address string

	// group name
	GroupName string

	// group qualification type
	GroupQualifyType int

	// group qualification ipfs link
	IpfsLink string
}

// newGroupQualification creates a new GroupQualification.
func NewGroupQualification(groupId [GroupIdLen]byte, address string, groupName string, qualificationType int, ipfsLink string) *GroupQualification {
	return &GroupQualification{
		GroupId:          groupId,
		Address:          address,
		GroupName:        groupName,
		GroupQualifyType: qualificationType,
		IpfsLink:         ipfsLink,
	}
}

// key = prefix + groupid + addressSha256Hash. value = qualification type + len(address) + address + len(groupname) + groupname + len(ipfslink) + ipfslink
func (im *Manager) GroupQualificationKey(groupQualification *GroupQualification) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupQualification
	index++
	copy(key[index:], groupQualification.GroupId[:])
	index += GroupIdLen
	copy(key[index:], Sha256Hash(groupQualification.Address))
	return key
}

// struct to value, value = qualification type + len(address) + address + len(groupname) + groupname + len(ipfslink) + ipfslink
func (im *Manager) GroupQualificationValue(groupQualification *GroupQualification) []byte {
	// func AppendBytesWithUint16Len(bytes *[]byte, idx *int, slice []byte, appendLength bool) {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte{byte(groupQualification.GroupQualifyType)}, false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(groupQualification.Address), true)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(groupQualification.GroupName), true)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(groupQualification.IpfsLink), true)
	return bytes
}

// key and value to struct
func (im *Manager) ParseGroupQualificationKeyAndValue(key []byte, value []byte) (*GroupQualification, error) {
	groupId := key[1 : 1+GroupIdLen]
	idx := 0
	qualificationType, err := ReadBytesWithUint16Len(value, &idx, 1)
	if err != nil {
		return nil, err
	}
	address, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	groupName, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	ipfsLink, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	var groupId32 [GroupIdLen]byte
	copy(groupId32[:], groupId)
	groupQualification := &GroupQualification{
		GroupId:          groupId32,
		Address:          string(address),
		GroupName:        string(groupName),
		GroupQualifyType: int(qualificationType[0]),
		IpfsLink:         string(ipfsLink),
	}
	return groupQualification, nil
}

// store group qualification
func (im *Manager) StoreGroupQualification(groupQualification *GroupQualification, logger *logger.Logger) error {
	key := im.GroupQualificationKey(groupQualification)
	value := im.GroupQualificationValue(groupQualification)
	// log group qualification key and value
	logger.Infof("StoreGroupQualification,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	// check if mark exists, if so, store group member
	mark := NewMark(groupQualification.Address, groupQualification.GroupId, [4]byte{0, 0, 0, 0})
	exists, err := im.MarkExists(mark.GroupId, mark.Address)
	if err != nil {
		return err
	}
	if exists {
		groupMember := NewGroupMember(groupQualification.GroupId, groupQualification.Address, GetCurrentEpochTimestamp())
		err = im.StoreGroupMember(groupMember, logger)
		if err != nil {
			return err
		}
	}
	return nil
}

// delete group qualification
func (im *Manager) DeleteGroupQualification(groupQualification *GroupQualification, logger *logger.Logger) error {
	key := im.GroupQualificationKey(groupQualification)
	// log group qualification key
	logger.Infof("DeleteGroupQualification,key:%s", iotago.EncodeHex(key))
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	// delete group member as well
	groupMember := NewGroupMember(groupQualification.GroupId, groupQualification.Address, 0)
	err = im.DeleteGroupMember(groupMember, logger)
	if err != nil {
		return err
	}
	return nil
}

// check if group qualification exists, input is group id and address
func (im *Manager) GroupQualificationExists(groupQualification *GroupQualification) (bool, error) {
	key := im.GroupQualificationKey(groupQualification)
	return im.imStore.Has(key)
}

// get prefix from group id
func (im *Manager) GroupQualificationKeyPrefix(groupId [GroupIdLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupQualification
	index++
	copy(key[index:], groupId[:])
	return key
}

// get all group qualifications from group id
func (im *Manager) GetAllGroupQualificationsFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) ([]*GroupQualification, error) {
	prefix := im.GroupQualificationKeyPrefix(groupId)
	groupQualifications := make([]*GroupQualification, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		groupQualification, err := im.ParseGroupQualificationKeyAndValue(key, value)
		if err != nil {
			logger.Errorf("ParseGroupQualificationKeyAndValue error: %s", err)
			return false
		}
		groupQualifications = append(groupQualifications, groupQualification)
		return true
	})
	if err != nil {
		return nil, err
	}
	return groupQualifications, nil
}
