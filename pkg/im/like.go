package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

type UserLikeGroupMember struct {
	GroupId             [GroupIdLen]byte
	LikerAddrSha256Hash [Sha256HashLen]byte
	LikedAddrSha256Hash [Sha256HashLen]byte
}

// NewUserLikeGroupMember creates a new UserLikeGroupMember.
func NewUserLikeGroupMember(groupId [GroupIdLen]byte, likerAddrSha256Hash [Sha256HashLen]byte, likedAddrSha256Hash [Sha256HashLen]byte) *UserLikeGroupMember {
	return &UserLikeGroupMember{
		GroupId:             groupId,
		LikerAddrSha256Hash: likerAddrSha256Hash,
		LikedAddrSha256Hash: likedAddrSha256Hash,
	}
}

// key = prefix + groupid + likedAddrSha256Hash + likerAddrSha256Hash, value = empty
func (im *Manager) UserLikeGroupMemberKey(userLikeGroupMember *UserLikeGroupMember) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixUserLikeGroupMember
	index++
	copy(key[index:], userLikeGroupMember.GroupId[:])
	index += GroupIdLen
	copy(key[index:], userLikeGroupMember.LikedAddrSha256Hash[:])
	index += Sha256HashLen
	copy(key[index:], userLikeGroupMember.LikerAddrSha256Hash[:])
	return key
}

// address like key = prefix + likerAddrSha256Hash + groupid + likedAddrSha256Hash, value = empty
func (im *Manager) AddressLikeKey(userLikeGroupMember *UserLikeGroupMember) []byte {
	var key []byte
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixAddressLike}, false)
	AppendBytesWithUint16Len(&key, &index, userLikeGroupMember.LikerAddrSha256Hash[:], false)
	AppendBytesWithUint16Len(&key, &index, userLikeGroupMember.GroupId[:], false)
	AppendBytesWithUint16Len(&key, &index, userLikeGroupMember.LikedAddrSha256Hash[:], false)
	return key
}

// check if user has group member
func (im *Manager) UserHasGroupMemberLike(userLikeGroupMember *UserLikeGroupMember) (bool, error) {

	exists, err := im.GroupMemberExistsFromGroupIdAndAddressSha256Hash(userLikeGroupMember.GroupId, userLikeGroupMember.LikedAddrSha256Hash)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// store user like group member, check if user has group member, if not, return error
func (im *Manager) StoreUserLikeGroupMember(userLikeGroupMember *UserLikeGroupMember, logger *logger.Logger) error {
	exists, err := im.UserHasGroupMemberLike(userLikeGroupMember)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user has no group member")
	}
	logger.Infof("StoreUserLikeGroupMember: groupId=%s, likedAddrSha256Hash=%s, likerAddrSha256Hash=%s",
		iotago.EncodeHex(userLikeGroupMember.GroupId[:]),
		iotago.EncodeHex(userLikeGroupMember.LikedAddrSha256Hash[:]),
		iotago.EncodeHex(userLikeGroupMember.LikerAddrSha256Hash[:]),
	)

	key := im.UserLikeGroupMemberKey(userLikeGroupMember)
	addressKey := im.AddressLikeKey(userLikeGroupMember)
	value := []byte{}
	err = im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	err = im.imStore.Set(addressKey, value)
	if err != nil {
		return err
	}
	err = im.UpdateGroupWhitelist(userLikeGroupMember)
	if err != nil {
		return err
	}
	return nil
}

// calculate reputation score, then update group whitelist accordingly
func (im *Manager) UpdateGroupWhitelist(userLikeGroupMember *UserLikeGroupMember) error {
	reputationScore, err := im.CalculateReputationScore(userLikeGroupMember.GroupId, userLikeGroupMember.LikedAddrSha256Hash)
	if err != nil {
		return err
	}
	// store user group reputation
	userGroupReputation := NewUserGroupReputation(userLikeGroupMember.GroupId, userLikeGroupMember.LikedAddrSha256Hash, reputationScore)
	err = im.StoreGroupUserReputation(userGroupReputation)
	if err != nil {
		return err
	}
	err = im.StoreUserGroupReputation(userGroupReputation)
	if err != nil {
		return err
	}
	// if reputation score < 60, add liked user to group black list
	if reputationScore < 60 {
		err = im.AddAddressToGroupBlacklist(userLikeGroupMember.LikedAddrSha256Hash, userLikeGroupMember.GroupId)
		if err != nil {
			return err
		}
	} else {
		err = im.RemoveAddressFromGroupBlacklist(userLikeGroupMember.LikedAddrSha256Hash, userLikeGroupMember.GroupId)
		if err != nil {
			return err
		}
	}
	return nil
}

// delete user like group member
func (im *Manager) DeleteUserLikeGroupMember(userLikeGroupMember *UserLikeGroupMember) error {
	key := im.UserLikeGroupMemberKey(userLikeGroupMember)
	addressKey := im.AddressLikeKey(userLikeGroupMember)
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	err = im.imStore.Delete(addressKey)
	if err != nil {
		return err
	}
	err = im.UpdateGroupWhitelist(userLikeGroupMember)
	if err != nil {
		return err
	}
	return nil
}

// UserLikeGroupMemberKeyPrefix
func (im *Manager) UserLikeGroupMemberKeyPrefix(groupId [GroupIdLen]byte, likedAddrSha256Hash [Sha256HashLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixUserLikeGroupMember
	index++
	copy(key[index:], groupId[:])
	index += GroupIdLen
	copy(key[index:], likedAddrSha256Hash[:])
	return key
}

// Address Like Key Prefix
func (im *Manager) AddressLikeKeyPrefix(likerAddrSha256Hash [Sha256HashLen]byte) []byte {
	var key []byte
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixAddressLike}, false)
	AppendBytesWithUint16Len(&key, &index, likerAddrSha256Hash[:], false)
	return key
}

// like from address key and value to struct
func (im *Manager) GetUserLikeGroupMemberFromAddressKeyAndValue(key kvstore.Key, value kvstore.Value) *UserLikeGroupMember {
	idx := 0
	// prefix
	idx++
	// likerAddrSha256Hash
	likerAddrSha256Hash, err := ReadBytesWithUint16Len(key, &idx, Sha256HashLen)
	if err != nil {
		return nil
	}
	likerAddrSha256HashFixed := BytesToFixedSha256HashLenBytes(likerAddrSha256Hash)
	// group id
	groupId, err := ReadBytesWithUint16Len(key, &idx, GroupIdLen)
	if err != nil {
		return nil
	}
	groupIdFixed := BytesToFixedSha256HashLenBytes(groupId)
	// likedAddrSha256Hash
	likedAddrSha256Hash, err := ReadBytesWithUint16Len(key, &idx, Sha256HashLen)
	if err != nil {
		return nil
	}
	likedAddrSha256HashFixed := BytesToFixedSha256HashLenBytes(likedAddrSha256Hash)
	// log likedAddrSha256HashFixed
	Logger.Infof("GetUserLikeGroupMemberFromAddressKeyAndValue ... likedAddrSha256Hash=%s", iotago.EncodeHex(likedAddrSha256HashFixed[:]))
	return NewUserLikeGroupMember(groupIdFixed, likerAddrSha256HashFixed, likedAddrSha256HashFixed)
}

// get all like group members from an address
func (im *Manager) GetAllLikeGroupMembersFromAddress(likerAddrSha256Hash [Sha256HashLen]byte, logger *logger.Logger) ([]*UserLikeGroupMember, error) {
	prefix := im.AddressLikeKeyPrefix(likerAddrSha256Hash)
	var likeGroupMembers []*UserLikeGroupMember
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		likeGroupMember := im.GetUserLikeGroupMemberFromAddressKeyAndValue(key, value)
		if likeGroupMember != nil {
			likeGroupMembers = append(likeGroupMembers, likeGroupMember)
		}
		return true
	})
	return likeGroupMembers, err
}

// count times user get liked in group, and compute reputation score
func (im *Manager) CountLikedTimes(groupId [GroupIdLen]byte, likedAddrSha256Hash [Sha256HashLen]byte) (uint16, error) {
	prefix := im.UserLikeGroupMemberKeyPrefix(groupId, likedAddrSha256Hash)
	count := uint16(0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		count++
		return true
	})
	return count, err
}

func (im *Manager) deserializeUserLikeGroupMember(likeAddress string, data []byte) ([]*UserLikeGroupMember, string) {
	userLikeGroupMembers := make([]*UserLikeGroupMember, 0)
	idx := 0
	commonHeader, err := DeserializeCommonHeader(data, &idx)
	if err != nil {
		return nil, ""
	}
	if !commonHeader.IsActAsSelf {
		likeAddress = im.ConvertAddressToActualAddress(likeAddress)
	}

	for idx < len(data) {
		groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
		if err != nil {
			return nil, ""
		}
		var groupIdBytes [GroupIdLen]byte
		copy(groupIdBytes[:], groupId)
		likedAddrSha256Hash, err := ReadBytesWithUint16Len(data, &idx, Sha256HashLen)
		if err != nil {
			return nil, ""
		}
		var likedAddrSha256HashBytes [Sha256HashLen]byte
		copy(likedAddrSha256HashBytes[:], likedAddrSha256Hash)
		var likerAddrSha256HashBytes [Sha256HashLen]byte
		copy(likerAddrSha256HashBytes[:], Sha256Hash(likeAddress))
		userLikeGroupMember := NewUserLikeGroupMember(groupIdBytes, likerAddrSha256HashBytes, likedAddrSha256HashBytes)
		userLikeGroupMembers = append(userLikeGroupMembers, userLikeGroupMember)
	}
	return userLikeGroupMembers, likeAddress
}

// get user like group members from basicoutput
func (im *Manager) GetUserLikeGroupMembersFromBasicOutput(output *iotago.BasicOutput) ([]*UserLikeGroupMember, string) {
	unlockConditionSet := output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	featureSet := output.FeatureSet()
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil, ""
	}
	userLikeGroupMembers, adderss := im.deserializeUserLikeGroupMember(ownerAddress, meta.Data)
	return userLikeGroupMembers, adderss
}

// handle user like group member basic output created
func (im *Manager) HandleUserLikeGroupMemberBasicOutputCreated(output *iotago.BasicOutput, logger *logger.Logger) {
	getKey := func(userLikeGroupMember *UserLikeGroupMember) string {
		joined := iotago.EncodeHex(userLikeGroupMember.GroupId[:]) + "-" + iotago.EncodeHex(userLikeGroupMember.LikedAddrSha256Hash[:])
		return joined
	}
	createdUserLikeGroupMembers, address := im.GetUserLikeGroupMembersFromBasicOutput(output)
	addressSha256Hash := Sha256HashFixed(address)
	existingUserLikeGroupMembers, err := im.GetAllLikeGroupMembersFromAddress(addressSha256Hash, logger)
	if err != nil {
		return
	}
	toCreate, toDelete := CalculateDiff(createdUserLikeGroupMembers, existingUserLikeGroupMembers, getKey)

	// create
	for _, userLikeGroupMember := range toCreate {
		err := im.StoreUserLikeGroupMember(userLikeGroupMember, logger)
		if err != nil {
			// log error then continue
			logger.Infof("HandleUserLikeGroupMemberBasicOutputCreated ... err:%s", err.Error())
			continue
		}
		GenAndPushLikeChangedEvent(addressSha256Hash, userLikeGroupMember.GroupId, true, CurrentMilestoneTimestamp, im, logger)
	}

	// delete
	for _, userLikeGroupMember := range toDelete {
		err := im.DeleteUserLikeGroupMember(userLikeGroupMember)
		if err != nil {
			// log error then continue
			logger.Infof("HandleUserLikeGroupMemberBasicOutputCreated ... err:%s", err.Error())
			continue
		}
		GenAndPushLikeChangedEvent(addressSha256Hash, userLikeGroupMember.GroupId, false, CurrentMilestoneTimestamp, im, logger)
	}
}

// handle user like group member basic output consumed
func (im *Manager) HandleUserLikeGroupMemberBasicOutputConsumed(output *iotago.BasicOutput) {
	userLikeGroupMembers, _ := im.GetUserLikeGroupMembersFromBasicOutput(output)
	if len(userLikeGroupMembers) == 0 {
		return
	}
	for _, userLikeGroupMember := range userLikeGroupMembers {
		err := im.DeleteUserLikeGroupMember(userLikeGroupMember)
		if err != nil {
			return
		}
	}
}

var likeTagRawStr = "GROUPFILIKEV1"
var likeTag = []byte(likeTagRawStr)
var LikeTagStr = iotago.EncodeHex(likeTag)

// filter out like output from output
func (im *Manager) FilterLikeOutput(output iotago.Output, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	return im.FilterOutputByTag(output, likeTag, logger)
}

// filter out like output from LedgerOutput
func (im *Manager) FilterLikeOutputFromLedgerOutput(output *inx.LedgerOutput, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, false
	}
	return im.FilterLikeOutput(iotaOutput, logger)
}
