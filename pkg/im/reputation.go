package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/pkg/errors"
)

type UserGroupRepuation struct {
	GroupId        [GroupIdLen]byte
	AddrSha256Hash [Sha256HashLen]byte
	Reputation     float32
}

// new user group reputation
func NewUserGroupReputation(groupId [GroupIdLen]byte, addrSha256Hash [Sha256HashLen]byte, reputation float32) *UserGroupRepuation {
	return &UserGroupRepuation{
		GroupId:        groupId,
		AddrSha256Hash: addrSha256Hash,
		Reputation:     reputation,
	}
}

// new user group reputation from groupId and address
func NewUserGroupReputationFromGroupIdAndAddress(groupId [GroupIdLen]byte, address string) *UserGroupRepuation {
	addressSha256Hash := Sha256Hash(address)
	var addressSha256HashFixedLen [Sha256HashLen]byte
	copy(addressSha256HashFixedLen[:], addressSha256Hash)
	return &UserGroupRepuation{
		GroupId:        groupId,
		AddrSha256Hash: addressSha256HashFixedLen,
		Reputation:     0,
	}
}

func (im *Manager) GroupUserReputationKey(userGroupReputation *UserGroupRepuation) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupUserReputation
	index++
	copy(key[index:], userGroupReputation.GroupId[:])
	index += GroupIdLen
	copy(key[index:], userGroupReputation.AddrSha256Hash[:])
	return key
}

// key and value to group user reputation
// key = prefix + groupid + addrSha256Hash, value = reputation
func (im *Manager) GroupUserReputationKeyAndValue(key kvstore.Key, value kvstore.Value) (userGroupReputation *UserGroupRepuation) {
	var groupId32 [GroupIdLen]byte
	var addrSha256Hash [Sha256HashLen]byte
	copy(groupId32[:], key[1:1+GroupIdLen])
	copy(addrSha256Hash[:], key[1+GroupIdLen:1+GroupIdLen+Sha256HashLen])
	reputation := BytesToFloat32(value)
	userGroupReputation = &UserGroupRepuation{
		GroupId:        groupId32,
		AddrSha256Hash: addrSha256Hash,
		Reputation:     reputation,
	}
	return userGroupReputation
}

// prefix from groupid
func (im *Manager) GroupUserReputationPrefix(groupId [GroupIdLen]byte) []byte {
	prefix := make([]byte, 1+GroupIdLen)
	index := 0
	prefix[index] = ImStoreKeyPrefixGroupUserReputation
	index++
	copy(prefix[index:], groupId[:])
	return prefix
}
func (im *Manager) StoreGroupUserReputation(userGroupReputation *UserGroupRepuation) error {
	key := im.GroupUserReputationKey(userGroupReputation)
	value := Float32ToBytes(userGroupReputation.Reputation)
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return nil
}

func (im *Manager) GetGroupAllUsersReputation(groupId [GroupIdLen]byte, logger *logger.Logger) ([]*UserGroupRepuation, error) {
	prefix := im.GroupUserReputationPrefix(groupId)
	var userGroupReputations []*UserGroupRepuation
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		userGroupReputation := im.GroupUserReputationKeyAndValue(key, value)
		userGroupReputations = append(userGroupReputations, userGroupReputation)
		return true
	})
	if err != nil {
		return nil, err
	}
	return userGroupReputations, nil
}

// user group reputation key
func (im *Manager) UserGroupReputationKey(userGroupReputation *UserGroupRepuation) []byte {
	key := make([]byte, 1+Sha256HashLen+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixUserGroupReputation
	index++
	copy(key[index:], userGroupReputation.AddrSha256Hash[:])
	index += Sha256HashLen
	copy(key[index:], userGroupReputation.GroupId[:])
	return key
}

// store user group reputation
func (im *Manager) StoreUserGroupReputation(userGroupReputation *UserGroupRepuation) error {
	key := im.UserGroupReputationKey(userGroupReputation)
	value := Float32ToBytes(userGroupReputation.Reputation)
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return nil
}

// key and value to user group reputation
// key = prefix + addrSha256Hash + groupid, value = reputation
func (im *Manager) UserGroupReputationKeyAndValue(key kvstore.Key, value kvstore.Value) (userGroupReputation *UserGroupRepuation) {
	var addrSha256Hash [Sha256HashLen]byte
	var groupId32 [GroupIdLen]byte
	copy(addrSha256Hash[:], key[1:1+Sha256HashLen])
	copy(groupId32[:], key[1+Sha256HashLen:1+Sha256HashLen+GroupIdLen])
	reputation := BytesToFloat32(value)
	userGroupReputation = &UserGroupRepuation{
		GroupId:        groupId32,
		AddrSha256Hash: addrSha256Hash,
		Reputation:     reputation,
	}
	return userGroupReputation
}

// get reputation from groupid and address
func (im *Manager) GetUserGroupReputation(groupId [GroupIdLen]byte, address string, logger *logger.Logger) (*UserGroupRepuation, error) {
	userGroupReputation := NewUserGroupReputationFromGroupIdAndAddress(groupId, address)
	key := im.UserGroupReputationKey(userGroupReputation)
	value, err := im.imStore.Get(key)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return im.UserGroupReputationKeyAndValue(key, value), nil
}
