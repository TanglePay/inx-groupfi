package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
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

// store vote
func (im *Manager) StoreVote(vote *Vote) error {
	key := im.VoteKey(vote)
	value := []byte{vote.Vote}
	return im.imStore.Set(key, value)
}

// delete vote
func (im *Manager) DeleteVote(vote *Vote) error {
	key := im.VoteKey(vote)
	return im.imStore.Delete(key)
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

// count votes for group
func (im *Manager) CountVotesForGroup(groupId [GroupIdLen]byte) (int, int, error) {
	prefix := im.VoteKeyPrefix(groupId)
	publicCt := 0
	privateCt := 0
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		if value[0] == VotePublic {
			publicCt++
		} else {
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
	address := unlock.Address().Address.Bech32(iotago.PrefixShimmer)
	feature := output.FeatureSet()
	meta := feature.MetadataFeature()
	return im.deserializeUserVoteGroup(address, meta.Data)
}

// handle user vote group basic output created
func (im *Manager) HandleUserVoteGroupBasicOutputCreated(output *iotago.BasicOutput) {
	userVoteGroups := im.GetUserVoteGroupsFromBasicOutput(output)
	if len(userVoteGroups) == 0 {
		return
	}
	for _, userVoteGroup := range userVoteGroups {
		err := im.StoreVote(userVoteGroup)
		if err != nil {
			return
		}
	}
}

// handle user vote group basic output consumed
func (im *Manager) HandleUserVoteGroupBasicOutputConsumed(output *iotago.BasicOutput) {
	userVoteGroups := im.GetUserVoteGroupsFromBasicOutput(output)
	if len(userVoteGroups) == 0 {
		return
	}
	for _, userVoteGroup := range userVoteGroups {
		err := im.DeleteVote(userVoteGroup)
		if err != nil {
			return
		}
	}
}

var voteTagRawStr = "GROUPFIVOTEV1"
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