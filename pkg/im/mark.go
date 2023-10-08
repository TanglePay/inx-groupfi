package im

import (
	"bytes"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

type Mark struct {
	Address string
	// group id
	GroupId [GroupIdLen]byte
	// timestamp
	Timestamp [4]byte
}

// newMark creates a new Mark.
func NewMark(address string, groupId [GroupIdLen]byte, timestamp [4]byte) *Mark {
	return &Mark{
		Address:   address,
		GroupId:   groupId,
		Timestamp: timestamp,
	}
}

// key = prefix + groupid + timestamp + addressSha256Hash. value = empty
func (im *Manager) MarkKey(mark *Mark) []byte {
	key := make([]byte, 1+GroupIdLen+TimestampLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupMark
	index++
	copy(key[index:], mark.GroupId[:])
	index += GroupIdLen
	copy(key[index:], mark.Timestamp[:])
	index += TimestampLen
	copy(key[index:], Sha256Hash(mark.Address))
	return key
}

// store mark
func (im *Manager) StoreMark(mark *Mark, logger *logger.Logger) error {
	key := im.MarkKey(mark)
	value := []byte(mark.Address)
	return im.imStore.Set(key, value)
}

// delete mark
func (im *Manager) DeleteMark(mark *Mark, logger *logger.Logger) error {
	key := im.MarkKey(mark)
	return im.imStore.Delete(key)
}

// MarkKeyPrefix returns the prefix for the given group id.
func (im *Manager) MarkKeyPrefix(groupId [GroupIdLen]byte) []byte {
	key := make([]byte, 1+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupMark
	index++
	copy(key[index:], groupId[:])
	return key
}

// MarkKeyToMark
func (im *Manager) MarkKeyAndValueToMark(key kvstore.Key, value kvstore.Value) *Mark {
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1:1+GroupIdLen])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], key[1+GroupIdLen:1+GroupIdLen+TimestampLen])
	var addressSha256 [Sha256HashLen]byte
	copy(addressSha256[:], key[1+GroupIdLen+TimestampLen:])
	return NewMark(string(value), groupId, timestamp)
}

// get marks from group id
func (im *Manager) GetMarksFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) ([]*Mark, error) {
	prefix := im.MarkKeyPrefix(groupId)
	marks := make([]*Mark, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// log found mark with key and value
		logger.Infof("Found mark,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
		mark := im.MarkKeyAndValueToMark(key, value)
		marks = append(marks, mark)
		return true
	})
	return marks, err
}

// get group member addresses from group id, get all marks from group id, and get all nfts from group id, nft contains address btw,
// then the intersection of two address sets is the result
func (im *Manager) GetGroupMemberAddressesFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) ([]string, error) {
	marks, err := im.GetMarksFromGroupId(groupId, logger)
	if err != nil {
		return nil, err
	}
	// markAddresses map[string]bool
	markAddresses := make(map[string]bool)
	for _, mark := range marks {
		markAddresses[mark.Address] = true
	}
	nfts, err := im.ReadNFTsFromGroupId(groupId[:])
	if err != nil {
		return nil, err
	}
	var groupMemberAddresses []string
	for _, nft := range nfts {
		nftAddress := string(nft.OwnerAddress)
		if markAddresses[nftAddress] {
			groupMemberAddresses = append(groupMemberAddresses, nftAddress)
		}
	}
	return groupMemberAddresses, nil
}

// deserialized using func ReadBytesWithUint16Len(bytes []byte, idx *int, providedLength ...int) ([]byte, error) {
func (im *Manager) DeserializeUserMarkedGroupIds(address string, data []byte) ([]*Mark, error) {
	marks := make([]*Mark, 0)
	idx := 1
	for idx < len(data) {
		groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
		if err != nil {
			return nil, err
		}
		var groupIdBytes [GroupIdLen]byte
		copy(groupIdBytes[:], groupId)
		timestamp, err := ReadBytesWithUint16Len(data, &idx, TimestampLen)
		if err != nil {
			return nil, err
		}
		var timestampBytes [TimestampLen]byte
		copy(timestampBytes[:], timestamp)
		marks = append(marks, NewMark(address, groupIdBytes, timestampBytes))
	}
	return marks, nil
}

// get unlock address and []*Mark from BasicOutput
func (im *Manager) GetMarksFromBasicOutput(output *iotago.BasicOutput) ([]*Mark, error) {
	unlockConditionSet := output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)
	featureSet := output.FeatureSet()
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil, errors.New("meta is nil")
	}
	marks, err := im.DeserializeUserMarkedGroupIds(ownerAddress, meta.Data)
	if err != nil {
		return nil, err
	}
	return marks, nil
}

// handle group mark basic output created
func (im *Manager) HandleGroupMarkBasicOutputCreated(output *iotago.BasicOutput, logger *logger.Logger) {
	marks, err := im.GetMarksFromBasicOutput(output)
	if err != nil {
		// log error
		logger.Infof("HandleGroupMarkBasicOutputCreated ... err:%s", err.Error())
		return
	}
	if len(marks) == 0 {
		// log zero marks
		logger.Infof("HandleGroupMarkBasicOutputCreated ... zero marks")
		return
	}
	for _, mark := range marks {
		err := im.StoreMark(mark, logger)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputCreated ... err:%s", err.Error())
			return
		}
	}
}

// handle group mark basic output consumed
func (im *Manager) HandleGroupMarkBasicOutputConsumed(output *iotago.BasicOutput, logger *logger.Logger) {
	marks, err := im.GetMarksFromBasicOutput(output)
	if err != nil {
		// log error
		logger.Infof("HandleGroupMarkBasicOutputConsumed ... err:%s", err.Error())
		return
	}
	if len(marks) == 0 {
		return
	}
	for _, mark := range marks {
		err := im.DeleteMark(mark, logger)
		if err != nil {
			return
		}
	}
}

var markTagRawStr = "GROUPFIMARKV2"
var markTag = []byte(markTagRawStr)
var MarkTagStr = iotago.EncodeHex(markTag)

// filter mark output by tag
func (im *Manager) FilterMarkOutput(output iotago.Output, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	return im.FilterOutputByTag(output, markTag, logger)
}

// filter mark output from ledger output
func (im *Manager) FilterMarkOutputFromLedgerOutput(output *inx.LedgerOutput, logger *logger.Logger) (*iotago.BasicOutput, bool) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, false
	}
	return im.FilterMarkOutput(iotaOutput, logger)
}

func (im *Manager) FilterOutputByTag(output iotago.Output, targetTag []byte, logger *logger.Logger) (*iotago.BasicOutput, bool) {

	// Ignore anything other than BasicOutputs
	if output.Type() != iotago.OutputBasic {
		return nil, false
	}

	featureSet := output.FeatureSet()
	tag := featureSet.TagFeature()
	meta := featureSet.MetadataFeature()
	if tag == nil || meta == nil {
		return nil, false
	}
	tagPayload := tag.Tag
	// log found output, with tag, and tag which is looking for
	logger.Infof("Found output,payload len:%d,tag len:%d,tag:%s,targetTag:%s", len(tagPayload), len(targetTag), string(tagPayload), string(targetTag))
	if !bytes.Equal(tagPayload, targetTag) {
		return nil, false
	}
	return output.(*iotago.BasicOutput), true
}