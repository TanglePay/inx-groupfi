package im

import (
	"math"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

type UserMuteGroupMember struct {
	GroupId             [GroupIdLen]byte
	MuteAddrSha256Hash  [Sha256HashLen]byte
	MutedAddrSha256Hash [Sha256HashLen]byte
}

// NewUserMuteGroupMember creates a new UserMuteGroupMember.
func NewUserMuteGroupMember(groupId [GroupIdLen]byte, muteAddrSha256Hash [Sha256HashLen]byte, mutedAddrSha256Hash [Sha256HashLen]byte) *UserMuteGroupMember {
	return &UserMuteGroupMember{
		GroupId:             groupId,
		MuteAddrSha256Hash:  muteAddrSha256Hash,
		MutedAddrSha256Hash: mutedAddrSha256Hash,
	}
}

// key = prefix + groupid + mutedAddrSha256Hash + muteAddrSha256Hash, value = empty
func (im *Manager) UserMuteGroupMemberKey(userMuteGroupMember *UserMuteGroupMember) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixUserMuteGroupMember
	index++
	copy(key[index:], userMuteGroupMember.GroupId[:])
	index += GroupIdLen
	copy(key[index:], userMuteGroupMember.MutedAddrSha256Hash[:])
	index += Sha256HashLen
	copy(key[index:], userMuteGroupMember.MuteAddrSha256Hash[:])
	return key
}

// address mute key = prefix + muteAddrSha256Hash + groupid + mutedAddrSha256Hash, value = empty
func (im *Manager) AddressMuteKey(userMuteGroupMember *UserMuteGroupMember) []byte {
	var key []byte
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixAddressMute}, false)
	AppendBytesWithUint16Len(&key, &index, userMuteGroupMember.MuteAddrSha256Hash[:], false)
	AppendBytesWithUint16Len(&key, &index, userMuteGroupMember.GroupId[:], false)
	AppendBytesWithUint16Len(&key, &index, userMuteGroupMember.MutedAddrSha256Hash[:], false)
	return key
}

// check if user has group member
func (im *Manager) UserHasGroupMember(userMuteGroupMember *UserMuteGroupMember) (bool, error) {

	exists, err := im.GroupMemberExistsFromGroupIdAndAddressSha256Hash(userMuteGroupMember.GroupId, userMuteGroupMember.MutedAddrSha256Hash)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// store user mute group member, check if user has group member, if not, return error
func (im *Manager) StoreUserMuteGroupMember(userMuteGroupMember *UserMuteGroupMember, logger *logger.Logger) error {
	exists, err := im.UserHasGroupMember(userMuteGroupMember)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("user has no group member")
	}
	logger.Infof("StoreUserMuteGroupMember: groupId=%s, mutedAddrSha256Hash=%s, muteAddrSha256Hash=%s",
		iotago.EncodeHex(userMuteGroupMember.GroupId[:]),
		iotago.EncodeHex(userMuteGroupMember.MutedAddrSha256Hash[:]),
		iotago.EncodeHex(userMuteGroupMember.MuteAddrSha256Hash[:]),
	)

	key := im.UserMuteGroupMemberKey(userMuteGroupMember)
	addressKey := im.AddressMuteKey(userMuteGroupMember)
	value := []byte{}
	err = im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	err = im.imStore.Set(addressKey, value)
	if err != nil {
		return err
	}
	err = im.UpdateGroupBlacklist(userMuteGroupMember)
	if err != nil {
		return err
	}
	return nil
}

// calculate reputation score, then update group blacklist accordingly
func (im *Manager) UpdateGroupBlacklist(userMuteGroupMember *UserMuteGroupMember) error {
	reputationScore, err := im.CalculateReputationScore(userMuteGroupMember.GroupId, userMuteGroupMember.MutedAddrSha256Hash)
	if err != nil {
		return err
	}
	// store user group reputation
	userGroupReputation := NewUserGroupReputation(userMuteGroupMember.GroupId, userMuteGroupMember.MutedAddrSha256Hash, reputationScore)
	err = im.StoreGroupUserReputation(userGroupReputation)
	if err != nil {
		return err
	}
	err = im.StoreUserGroupReputation(userGroupReputation)
	if err != nil {
		return err
	}
	// if reputation score < 60, add muted user to group blacklist
	if reputationScore < 60 {
		err = im.AddAddressToGroupBlacklist(userMuteGroupMember.MutedAddrSha256Hash, userMuteGroupMember.GroupId)
		if err != nil {
			return err
		}
	} else {
		err = im.RemoveAddressFromGroupBlacklist(userMuteGroupMember.MutedAddrSha256Hash, userMuteGroupMember.GroupId)
		if err != nil {
			return err
		}
	}
	return nil
}

// delete user mute group member
func (im *Manager) DeleteUserMuteGroupMember(userMuteGroupMember *UserMuteGroupMember) error {
	key := im.UserMuteGroupMemberKey(userMuteGroupMember)
	addressKey := im.AddressMuteKey(userMuteGroupMember)
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	err = im.imStore.Delete(addressKey)
	if err != nil {
		return err
	}
	err = im.UpdateGroupBlacklist(userMuteGroupMember)
	if err != nil {
		return err
	}
	return nil
}

// UserMuteGroupMemberKeyPrefix
func (im *Manager) UserMuteGroupMemberKeyPrefix(groupId [GroupIdLen]byte, mutedAddrSha256Hash [Sha256HashLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixUserMuteGroupMember
	index++
	copy(key[index:], groupId[:])
	index += GroupIdLen
	copy(key[index:], mutedAddrSha256Hash[:])
	return key
}

// Address Mute Key Prefix
func (im *Manager) AddressMuteKeyPrefix(muteAddrSha256Hash [Sha256HashLen]byte) []byte {
	var key []byte
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixAddressMute}, false)
	AppendBytesWithUint16Len(&key, &index, muteAddrSha256Hash[:], false)
	return key
}

// vote from address key and value to struct
func (im *Manager) GetUserMuteGroupMemberFromAddressKeyAndValue(key kvstore.Key, value kvstore.Value) *UserMuteGroupMember {
	idx := 0
	// prefix
	idx++
	// muteAddrSha256Hash
	muteAddrSha256Hash, err := ReadBytesWithUint16Len(key, &idx, Sha256HashLen)
	if err != nil {
		return nil
	}
	muteAddrSha256HashFixed := BytesToFixedSha256HashLenBytes(muteAddrSha256Hash)
	// group id
	groupId, err := ReadBytesWithUint16Len(key, &idx, GroupIdLen)
	if err != nil {
		return nil
	}
	groupIdFixed := BytesToFixedSha256HashLenBytes(groupId)
	// mutedAddrSha256Hash
	mutedAddrSha256Hash, err := ReadBytesWithUint16Len(key, &idx, Sha256HashLen)
	if err != nil {
		return nil
	}
	mutedAddrSha256HashFixed := BytesToFixedSha256HashLenBytes(mutedAddrSha256Hash)
	return NewUserMuteGroupMember(groupIdFixed, muteAddrSha256HashFixed, mutedAddrSha256HashFixed)
}

// get all mute group members from an address
func (im *Manager) GetAllMuteGroupMembersFromAddress(muteAddrSha256Hash [Sha256HashLen]byte, logger *logger.Logger) ([]*UserMuteGroupMember, error) {
	prefix := im.AddressMuteKeyPrefix(muteAddrSha256Hash)
	var muteGroupMembers []*UserMuteGroupMember
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		muteGroupMember := im.GetUserMuteGroupMemberFromAddressKeyAndValue(key, value)
		if muteGroupMember != nil {
			muteGroupMembers = append(muteGroupMembers, muteGroupMember)
		}
		return true
	})
	return muteGroupMembers, err
}

// count times user get muted in group, and compute reputation score
func (im *Manager) CountMutedTimes(groupId [GroupIdLen]byte, mutedAddrSha256Hash [Sha256HashLen]byte) (uint16, error) {
	prefix := im.UserMuteGroupMemberKeyPrefix(groupId, mutedAddrSha256Hash)
	count := uint16(0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		count++
		return true
	})
	return count, err
}

// calculate reputation score
func (im *Manager) CalculateReputationScore(groupId [GroupIdLen]byte, likedOrMutedAddrSha256Hash [Sha256HashLen]byte) (float32, error) {
	likedTimes, err := im.CountLikedTimes(groupId, likedOrMutedAddrSha256Hash)
	if err != nil {
		return 0, err
	}
	mutedTimes, err := im.CountMutedTimes(groupId, likedOrMutedAddrSha256Hash)
	if err != nil {
		return 0, err
	}
	addresses, err := im.GetGroupMembers(groupId)
	if err != nil {
		return 0, err
	}
	var likedTimesInt int = int(likedTimes)
	var mutedTimesInt int = int(mutedTimes)
	count := likedTimesInt - mutedTimesInt
	groupMemberCount := len(addresses)
	// reputation score = 100 + 150/sqrt(groupMemberCount+42) * (likedTimes - mutedTimes)
	// Calculate the denominator separately
	denominator := math.Sqrt(float64(groupMemberCount + 42))

	tmp1 := 150.0 / denominator
	tmp11 := float64(count * 1.0)
	tmp2 := tmp1 * tmp11
	// Perform the division and the rest of the calculation
	reputationScore := 100.0 + tmp2
	reputationScore = math.Round(reputationScore*10) / 10

	// log reputation score, likedTimes, mutedTimes, groupMemberCount, adderss，denominator,tmp1,tmp11,tmp2
	Logger.Infof("CalculateReputationScore: reputationScore=%f, likedTimes=%d, mutedTimes=%d, groupMemberCount=%d, adderss=%s, denominator=%f, tmp1=%f, tmp11=%f, tmp2=%f",
		reputationScore, likedTimes, mutedTimes, groupMemberCount, iotago.EncodeHex(likedOrMutedAddrSha256Hash[:]), denominator, tmp1, tmp11, tmp2)
	return float32(reputationScore), nil
}

/*
	export function deserializeUserMuteGroupMember(reader: ReadStream): IMUserMuteGroupMemberIntermediate[] {
	    const list: IMUserMuteGroupMemberIntermediate[] = [];
	    while (reader.hasRemaining(1)) {
	        const groupId = reader.readBytes("groupId", 32);
	        const addrSha256Hash = reader.readBytes("addrSha256Hash", 32);
	        list.push({
	            groupId,
	            addrSha256Hash
	        });
	    }
	    return list;
	}
*/
func (im *Manager) deserializeUserMuteGroupMember(muteAddress string, data []byte) ([]*UserMuteGroupMember, string) {
	userMuteGroupMembers := make([]*UserMuteGroupMember, 0)
	idx := 0
	commonHeader, err := DeserializeCommonHeader(data, &idx)
	if err != nil {
		return nil, ""
	}
	if !commonHeader.IsActAsSelf {
		muteAddress = im.ConvertAddressToActualAddress(muteAddress)
	}

	for idx < len(data) {
		groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
		if err != nil {
			return nil, ""
		}
		var groupIdBytes [GroupIdLen]byte
		copy(groupIdBytes[:], groupId)
		mutedAddrSha256Hash, err := ReadBytesWithUint16Len(data, &idx, Sha256HashLen)
		if err != nil {
			return nil, ""
		}
		var mutedAddrSha256HashBytes [Sha256HashLen]byte
		copy(mutedAddrSha256HashBytes[:], mutedAddrSha256Hash)
		var muteAddrSha256HashBytes [Sha256HashLen]byte
		copy(muteAddrSha256HashBytes[:], Sha256Hash(muteAddress))
		userMuteGroupMember := NewUserMuteGroupMember(groupIdBytes, muteAddrSha256HashBytes, mutedAddrSha256HashBytes)
		userMuteGroupMembers = append(userMuteGroupMembers, userMuteGroupMember)
	}
	return userMuteGroupMembers, muteAddress
}

// get user mute group members from basicoutput
func (im *Manager) GetUserMuteGroupMembersFromBasicOutput(output *iotago.BasicOutput) ([]*UserMuteGroupMember, string) {
	unlockConditionSet := output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	featureSet := output.FeatureSet()
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil, ""
	}
	userMuteGroupMembers, adderss := im.deserializeUserMuteGroupMember(ownerAddress, meta.Data)
	return userMuteGroupMembers, adderss
}

// handle user mute group member basic output created
func (im *Manager) HandleUserMuteGroupMemberBasicOutputCreated(output *iotago.BasicOutput, logger *logger.Logger) {
	getKey := func(userMuteGroupMember *UserMuteGroupMember) string {
		joined := iotago.EncodeHex(userMuteGroupMember.GroupId[:]) + "-" + iotago.EncodeHex(userMuteGroupMember.MutedAddrSha256Hash[:])
		return joined
	}
	createdUserMuteGroupMembers, address := im.GetUserMuteGroupMembersFromBasicOutput(output)
	addressSha256Hash := Sha256HashFixed(address)
	existingUserMuteGroupMembers, err := im.GetAllMuteGroupMembersFromAddress(addressSha256Hash, logger)
	if err != nil {
		return
	}
	toCreate, toDelete := CalculateDiff(createdUserMuteGroupMembers, existingUserMuteGroupMembers, getKey)
	// create
	for _, userMuteGroupMember := range toCreate {
		err := im.StoreUserMuteGroupMember(userMuteGroupMember, logger)
		if err != nil {
			// log error then continue
			logger.Infof("HandleUserMuteGroupMemberBasicOutputCreated ... err:%s", err.Error())
			continue
		}
		// GenAndPushMuteChangedEvent, just for address
		GenAndPushMuteChangedEvent(addressSha256Hash, userMuteGroupMember.GroupId, true, im, logger)
	}

	// delete
	for _, userMuteGroupMember := range toDelete {
		err := im.DeleteUserMuteGroupMember(userMuteGroupMember)
		if err != nil {
			// log error then continue
			logger.Infof("HandleUserMuteGroupMemberBasicOutputCreated ... err:%s", err.Error())
			continue
		}
		GenAndPushMuteChangedEvent(addressSha256Hash, userMuteGroupMember.GroupId, false, im, logger)
	}

}

// handle user mute group member basic output consumed
func (im *Manager) HandleUserMuteGroupMemberBasicOutputConsumed(output *iotago.BasicOutput) {
	userMuteGroupMembers, _ := im.GetUserMuteGroupMembersFromBasicOutput(output)
	if len(userMuteGroupMembers) == 0 {
		return
	}
	for _, userMuteGroupMember := range userMuteGroupMembers {
		err := im.DeleteUserMuteGroupMember(userMuteGroupMember)
		if err != nil {
			return
		}
	}
}

var muteTagRawStr = "GROUPFIMUTEV1"
var muteTag = []byte(muteTagRawStr)
var MuteTagStr = iotago.EncodeHex(muteTag)

// filter out mute output from output
func (im *Manager) FilterMuteOutput(output iotago.Output, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	return im.FilterOutputByTag(output, muteTag, logger)
}

// filter out mute output from LedgerOutput
func (im *Manager) FilterMuteOutputFromLedgerOutput(output *inx.LedgerOutput, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, false
	}
	return im.FilterMuteOutput(iotaOutput, logger)
}
