package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

type Vote struct {
	// group id
	GroupId       [GroupIdLen]byte
	AddressSha256 [Sha256HashLen]byte
	Vote          uint8
}

const (
	VotePublic = iota
	VotePrivate
	VoteUnvote
)

// newVote creates a new Vote.
func NewVote(groupId [GroupIdLen]byte, addressSha256 [Sha256HashLen]byte, vote uint8) *Vote {
	return &Vote{
		GroupId:       groupId,
		AddressSha256: addressSha256,
		Vote:          vote,
	}
}

// key = prefix + groupid + addressSha256Hash, value = empty
func (im *Manager) VoteKey(vote *Vote) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixVote
	index++
	copy(key[index:], vote.GroupId[:])
	index += GroupIdLen
	copy(key[index:], vote.AddressSha256[:])
	return key
}

// address vote key = prefix + addressSha256Hash + groupid, value = vote value
func (im *Manager) AddressVoteKey(vote *Vote) []byte {
	var key []byte
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixAddressVote}, false)
	AppendBytesWithUint16Len(&key, &index, vote.AddressSha256[:], false)
	AppendBytesWithUint16Len(&key, &index, vote.GroupId[:], false)
	return key
}

// store vote, check if user has group member, if not, return error
func (im *Manager) StoreVote(vote *Vote, logger *logger.Logger) error {
	if !IsIniting {
		exists, err := im.GroupMemberExistsFromGroupIdAndAddressSha256Hash(vote.GroupId, vote.AddressSha256)
		if err != nil {
			return err
		}
		if !exists {
			return errors.New("user has no group member")
		}
	}
	key := im.VoteKey(vote)
	addressKey := im.AddressVoteKey(vote)
	value := []byte{vote.Vote}
	// log vote key and value
	logger.Infof("StoreVote,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	err = im.imStore.Set(addressKey, value)
	groupId := vote.GroupId
	im.TryCalculateIfGroupIsPublic(groupId, logger)
	return err
}

// delete vote
func (im *Manager) DeleteVote(vote *Vote, logger *logger.Logger) error {
	key := im.VoteKey(vote)
	addressKey := im.AddressVoteKey(vote)
	// log vote key
	logger.Infof("DeleteVote,key:%s", iotago.EncodeHex(key))
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	err = im.imStore.Delete(addressKey)
	groupId := vote.GroupId
	im.TryCalculateIfGroupIsPublic(groupId, logger)
	return err
}

// VoteKeyPrefix
func (im *Manager) VoteKeyPrefix(groupId [GroupIdLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixVote
	index++
	copy(key[index:], groupId[:])
	return key
}

// address vote key prefix
func (im *Manager) AddressVoteKeyPrefix(addressSha256 [Sha256HashLen]byte) []byte {
	var key []byte
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixAddressVote}, false)
	AppendBytesWithUint16Len(&key, &index, addressSha256[:], false)
	return key
}

// get Vote from key and value
func (im *Manager) GetVoteFromKeyAndValue(key kvstore.Key, value kvstore.Value) *Vote {
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1:1+GroupIdLen])
	var addressSha256 [Sha256HashLen]byte
	copy(addressSha256[:], key[1+GroupIdLen:])
	return NewVote(groupId, addressSha256, value[0])
}

// get vote from address key and value
// address vote key = prefix + addressSha256Hash + groupid,
func (im *Manager) GetVoteFromAddressKeyAndValue(key kvstore.Key, value kvstore.Value) *Vote {
	var addressSha256 [Sha256HashLen]byte
	copy(addressSha256[:], key[1:1+Sha256HashLen])
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1+Sha256HashLen:])
	return NewVote(groupId, addressSha256, value[0])
}

// get all votes from group id
func (im *Manager) GetAllVotesFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) ([]*Vote, error) {
	prefix := im.VoteKeyPrefix(groupId)
	votes := make([]*Vote, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		votes = append(votes, im.GetVoteFromKeyAndValue(key, value))
		return true
	})
	return votes, err
}

// get all votes from address sha256 hash
func (im *Manager) GetAllVotesFromAddressSha256Hash(addressSha256 [Sha256HashLen]byte, logger *logger.Logger) ([]*Vote, error) {
	prefix := im.AddressVoteKeyPrefix(addressSha256)
	var votes []*Vote
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		votes = append(votes, im.GetVoteFromAddressKeyAndValue(key, value))
		return true
	})
	return votes, err
}

// count votes for group
func (im *Manager) CountVotesForGroup(groupId [GroupIdLen]byte) (int, int, error) {
	prefix := im.VoteKeyPrefix(groupId)
	publicCt := 0
	privateCt := 0
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		if value[0] == VotePublic {
			publicCt++
		} else if value[0] == VotePrivate {
			privateCt++
		}
		return true
	})
	return publicCt, privateCt, err
}

func (im *Manager) deserializeUserVoteGroup(address string, data []byte) []*Vote {
	userVoteGroups := make([]*Vote, 0)
	idx := 1
	for idx < len(data) {
		groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
		if err != nil {
			return nil
		}
		var groupIdBytes [GroupIdLen]byte
		copy(groupIdBytes[:], groupId)
		vote, err := ReadBytesWithUint16Len(data, &idx, 1)
		if err != nil {
			return nil
		}
		var addressSha256Bytes [Sha256HashLen]byte
		copy(addressSha256Bytes[:], Sha256Hash(address))
		userVoteGroups = append(userVoteGroups, NewVote(groupIdBytes, addressSha256Bytes, vote[0]))
	}
	return userVoteGroups
}

// get user vote groups from basic output
func (im *Manager) GetUserVoteGroupsFromBasicOutput(output *iotago.BasicOutput) []*Vote {
	unlock := output.UnlockConditionSet()
	address := unlock.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	// ConvertAddressToActualAddress
	address = im.ConvertAddressToActualAddress(address)
	feature := output.FeatureSet()
	meta := feature.MetadataFeature()
	return im.deserializeUserVoteGroup(address, meta.Data)
}

// handle user vote group basic output created
func (im *Manager) HandleUserVoteGroupBasicOutputCreated(output *iotago.BasicOutput, logger *logger.Logger) {
	// log entering
	logger.Infof("HandleUserVoteGroupBasicOutputCreated ...")
	userVoteGroups := im.GetUserVoteGroupsFromBasicOutput(output)
	if len(userVoteGroups) == 0 {
		return
	}
	for _, userVoteGroup := range userVoteGroups {
		err := im.StoreVote(userVoteGroup, logger)
		if err != nil {
			// log error
			logger.Infof("HandleUserVoteGroupBasicOutputCreated ... err:%s", err.Error())
			continue
		}
	}
}

// handle user vote group basic output consumed
func (im *Manager) HandleUserVoteGroupBasicOutputConsumed(output *iotago.BasicOutput, logger *logger.Logger) {
	// log entering
	logger.Infof("HandleUserVoteGroupBasicOutputConsumed ...")
	userVoteGroups := im.GetUserVoteGroupsFromBasicOutput(output)
	if len(userVoteGroups) == 0 {
		return
	}
	for _, userVoteGroup := range userVoteGroups {
		err := im.DeleteVote(userVoteGroup, logger)
		if err != nil {
			// log error
			logger.Infof("HandleUserVoteGroupBasicOutputConsumed ... err:%s", err.Error())
			return
		}
	}
}

var voteTagRawStr = "GROUPFIVOTEV2"
var voteTag = []byte(voteTagRawStr)
var VoteTagStr = iotago.EncodeHex(voteTag)

// filter vote basic output from output
func (im *Manager) FilterVoteOutput(output iotago.Output, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	return im.FilterOutputByTag(output, voteTag, logger)
}

// filter vote output from ledger output
func (im *Manager) FilterVoteOutputFromLedgerOutput(output *inx.LedgerOutput, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, false
	}
	return im.FilterVoteOutput(iotaOutput, logger)
}
