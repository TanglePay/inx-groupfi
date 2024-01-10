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
	// log
	logger.Infof("StoreUserMuteGroupMember: groupId=%s, mutedAddrSha256Hash=%s, muteAddrSha256Hash=%s",
		iotago.EncodeHex(userMuteGroupMember.GroupId[:]),
		iotago.EncodeHex(userMuteGroupMember.MutedAddrSha256Hash[:]),
		iotago.EncodeHex(userMuteGroupMember.MuteAddrSha256Hash[:]),
	)

	key := im.UserMuteGroupMemberKey(userMuteGroupMember)
	value := []byte{}
	err = im.imStore.Set(key, value)
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
	err := im.imStore.Delete(key)
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
func (im *Manager) CalculateReputationScore(groupId [GroupIdLen]byte, mutedAddrSha256Hash [Sha256HashLen]byte) (float32, error) {
	mutedTimes, err := im.CountMutedTimes(groupId, mutedAddrSha256Hash)
	if err != nil {
		return 0, err
	}
	addresses, err := im.GetGroupMembers(groupId)
	if err != nil {
		return 0, err
	}
	groupMemberCount := len(addresses)
	// reputation score = 100 - 150/sqrt(groupMemberCount) * mutedTimes
	reputationScore := float32(100) - float32(150)/float32(math.Sqrt(float64(groupMemberCount)))*float32(mutedTimes)

	return reputationScore, nil
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
func (im *Manager) deserializeUserMuteGroupMember(muteAddress string, data []byte) []*UserMuteGroupMember {
	userMuteGroupMembers := make([]*UserMuteGroupMember, 0)
	idx := 1
	for idx < len(data) {
		groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
		if err != nil {
			return nil
		}
		var groupIdBytes [GroupIdLen]byte
		copy(groupIdBytes[:], groupId)
		mutedAddrSha256Hash, err := ReadBytesWithUint16Len(data, &idx, Sha256HashLen)
		if err != nil {
			return nil
		}
		var mutedAddrSha256HashBytes [Sha256HashLen]byte
		copy(mutedAddrSha256HashBytes[:], mutedAddrSha256Hash)
		var muteAddrSha256HashBytes [Sha256HashLen]byte
		copy(muteAddrSha256HashBytes[:], Sha256Hash(muteAddress))
		userMuteGroupMember := NewUserMuteGroupMember(groupIdBytes, muteAddrSha256HashBytes, mutedAddrSha256HashBytes)
		userMuteGroupMembers = append(userMuteGroupMembers, userMuteGroupMember)
	}
	return userMuteGroupMembers
}

// get user mute group members from basicoutput
func (im *Manager) GetUserMuteGroupMembersFromBasicOutput(output *iotago.BasicOutput) []*UserMuteGroupMember {
	unlockConditionSet := output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	featureSet := output.FeatureSet()
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil
	}
	userMuteGroupMembers := im.deserializeUserMuteGroupMember(ownerAddress, meta.Data)
	return userMuteGroupMembers
}

// handle user mute group member basic output created
func (im *Manager) HandleUserMuteGroupMemberBasicOutputCreated(output *iotago.BasicOutput, logger *logger.Logger) {
	userMuteGroupMembers := im.GetUserMuteGroupMembersFromBasicOutput(output)
	if len(userMuteGroupMembers) == 0 {
		return
	}
	for _, userMuteGroupMember := range userMuteGroupMembers {
		err := im.StoreUserMuteGroupMember(userMuteGroupMember, logger)
		if err != nil {
			return
		}
	}
}

// handle user mute group member basic output consumed
func (im *Manager) HandleUserMuteGroupMemberBasicOutputConsumed(output *iotago.BasicOutput) {
	userMuteGroupMembers := im.GetUserMuteGroupMembersFromBasicOutput(output)
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
