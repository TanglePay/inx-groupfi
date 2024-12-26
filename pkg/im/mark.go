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

	OutputId iotago.OutputID

	MilestoneIndex uint32
	// timestamp
	MilestoneTimestamp uint32
}

// newMark creates a new Mark.
func NewMark(address string, groupId [GroupIdLen]byte, milestoneIndex uint32, milestoneTimestamp uint32) *Mark {
	return &Mark{
		Address:            address,
		GroupId:            groupId,
		MilestoneIndex:     milestoneIndex,
		MilestoneTimestamp: milestoneTimestamp,
	}
}

// key = prefix + groupid + addressSha256Hash. value = timestamp + address
func (im *Manager) MarkKey(mark *Mark) []byte {
	key := make([]byte, 1+GroupIdLen+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixGroupMark
	index++
	copy(key[index:], mark.GroupId[:])
	index += GroupIdLen
	copy(key[index:], Sha256HashAddress(mark.Address))
	return key
}

// address mark key
// key = prefix + addressSha256Hash + groupid. value = timestamp + address
func (im *Manager) AddressMarkKey(mark *Mark) []byte {
	key := make([]byte, 1+Sha256HashLen+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressMark
	index++
	copy(key[index:], Sha256HashAddress(mark.Address))
	index += Sha256HashLen
	copy(key[index:], mark.GroupId[:])
	return key
}

// store mark value = timestamp + address
func (im *Manager) StoreMark(mark *Mark, isActuallyMarked bool, logger *logger.Logger) error {
	key := im.MarkKey(mark)
	addressKey := im.AddressMarkKey(mark)
	value := make([]byte, 4+len(mark.Address))
	index := 0
	timeBytes := Uint32ToBytes(mark.MilestoneTimestamp)
	copy(value[index:], timeBytes)
	index += 4
	copy(value[index:], mark.Address)
	// log mark key and value
	logger.Infof("StoreMark,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	err = im.imStore.Set(addressKey, value)
	if err != nil {
		return err
	}

	// check if group qualification exists, if so, store group member
	exists, err := im.GroupQualificationExists(mark.GroupId, mark.Address, logger)
	if err != nil {
		return err
	}
	// log group qualification GroupId, address, exists
	logger.Infof("StoreMark,group qualification exists,groupId:%s,address:%s,exists:%t", iotago.EncodeHex(mark.GroupId[:]), mark.Address, exists)
	if exists && isActuallyMarked {

		groupMember := NewGroupMember(mark.GroupId, mark.Address, mark.MilestoneIndex, mark.MilestoneTimestamp)

		_, err = im.StoreGroupMember(groupMember, logger)
		if err != nil {
			return err
		}

	} else {
		// only mark changed, push mark changed event
		if isActuallyMarked {
			// push mark changed event
			err := GenAndPushMarkChangedEvent(mark, true, im, logger)
			if err != nil {
				return err
			}

		}
	}
	return nil
}

// delete mark
func (im *Manager) DeleteMark(mark *Mark, isActuallyUnmarked bool, logger *logger.Logger) error {
	key := im.MarkKey(mark)
	addressKey := im.AddressMarkKey(mark)
	// log mark key
	logger.Infof("DeleteMark,key:%s", iotago.EncodeHex(key))
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	err = im.imStore.Delete(addressKey)
	if err != nil {
		return err
	}
	// delete group member as well
	groupMember := NewGroupMember(mark.GroupId, mark.Address, mark.MilestoneIndex, mark.MilestoneTimestamp)
	isActuallyDeleted, err := im.DeleteGroupMember(groupMember, logger)
	if err != nil {
		return err
	}
	// delete group shared when previous group member is actually deleted and is actually unmarked
	if isActuallyDeleted && isActuallyUnmarked {
		err = im.DeleteSharedFromGroupId(mark.GroupId)
		if err != nil {
			return err
		}
	} else {
		// only mark changed, push mark changed event
		if isActuallyUnmarked {
			// push mark changed event
			err := GenAndPushMarkChangedEvent(mark, false, im, logger)
			if err != nil {
				return err
			}
		}

	}

	return nil
}

// check if mark exists, input is group id and address
func (im *Manager) MarkExists(groupId [GroupIdLen]byte, address string) (bool, error) {
	key := im.MarkKey(NewMark(address, groupId, 0, 0))
	return im.imStore.Has(key)
}

// GetMark returns a mark for the given group ID and address
func (im *Manager) GetMark(groupId [GroupIdLen]byte, address string) (*Mark, error) {
	key := im.MarkKey(NewMark(address, groupId, 0, 0))
	value, err := im.imStore.Get(key)
	if err != nil {
		return nil, err
	}

	timestampUint32 := BytesToUint32(value[:4])
	return NewMark(address, groupId, 0, timestampUint32), nil
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

// address mark key prefix
func (im *Manager) AddressMarkKeyPrefix(address string) []byte {
	key := make([]byte, 1+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressMark
	index++
	copy(key[index:], Sha256HashAddress(address))
	return key
}

// MarkKeyToMark, key = prefix + groupid + addressSha256Hash. value = timestamp + address
func (im *Manager) MarkKeyAndValueToMark(key kvstore.Key, value kvstore.Value) *Mark {
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1:1+GroupIdLen])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], value[:TimestampLen])
	address := string(value[TimestampLen:])
	timestampUint32 := BytesToUint32(value[4:])
	return NewMark(address, groupId, 0, timestampUint32)
}

// address mark key to mark
func (im *Manager) AddressMarkKeyAndValueToMark(key kvstore.Key, value kvstore.Value) *Mark {
	// key is prefix + addressSha256Hash + groupid
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1+Sha256HashLen:])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], value[:TimestampLen])
	address := string(value[TimestampLen:])
	timestampUint32 := BytesToUint32(value[:4])
	return NewMark(address, groupId, 0, timestampUint32)
}

// get marks from group id
func (im *Manager) GetMarksFromGroupId(groupId [GroupIdLen]byte, logger *logger.Logger) ([]*Mark, error) {
	prefix := im.MarkKeyPrefix(groupId)
	// log group id and prefix
	logger.Infof("GetMarksFromGroupId,groupId:%s,prefix:%s", iotago.EncodeHex(groupId[:]), iotago.EncodeHex(prefix))
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

// get marks from address
func (im *Manager) GetMarksFromAddress(address string, logger *logger.Logger) ([]*Mark, error) {
	prefix := im.AddressMarkKeyPrefix(address)
	// log address and prefix
	logger.Infof("GetMarksFromAddress,address:%s,prefix:%s", address, iotago.EncodeHex(prefix))
	marks := make([]*Mark, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// log found mark with key and value
		logger.Infof("Found mark,key:%s,value:%s", iotago.EncodeHex(key), iotago.EncodeHex(value))
		mark := im.AddressMarkKeyAndValueToMark(key, value)
		marks = append(marks, mark)
		return true
	})
	return marks, err
}

// deserialized using func ReadBytesWithUint16Len(bytes []byte, idx *int, providedLength ...int) ([]byte, error) {
func (im *Manager) DeserializeUserMarkedGroupIds(address string, data []byte) ([]*Mark, string, error) {
	marks := make([]*Mark, 0)
	idx := 0
	commonHeader, err := DeserializeCommonHeader(data, &idx)
	if err != nil {
		return nil, "", err
	}
	if !commonHeader.IsActAsSelf {
		address = im.ConvertAddressToActualAddress(address)
	}
	for idx < len(data) {
		groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
		if err != nil {
			return nil, "", err
		}
		var groupIdBytes [GroupIdLen]byte
		copy(groupIdBytes[:], groupId)
		timestamp, err := ReadBytesWithUint16Len(data, &idx, TimestampLen)
		if err != nil {
			return nil, "", err
		}
		timestamp32 := BytesToUint32(timestamp)
		marks = append(marks, NewMark(address, groupIdBytes, 0, timestamp32))
	}
	return marks, address, nil
}

// get unlock address and []*Mark from BasicOutput
func (im *Manager) GetMarksFromBasicOutput(output *OutputAndOutputIdAndMilestoneIndexAndMilestoneTimestamp) ([]*Mark, string, error) {
	unlockConditionSet := output.Output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	featureSet := output.Output.FeatureSet()
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil, "", errors.New("meta is nil")
	}
	outputId := output.OutputId
	marks, address, err := im.DeserializeUserMarkedGroupIds(ownerAddress, meta.Data)
	if err != nil {
		return nil, "", err
	}
	for i, mark := range marks {
		mark.OutputId = outputId

		if i == len(marks)-1 {
			mark.MilestoneIndex = output.MilestoneIndex
			mark.MilestoneTimestamp = output.MilestoneTimestamp
		}
		// mark.MilestoneTimestamp can not be greater than im.CurrentMilestoneTimestamp

		if mark.MilestoneTimestamp > CurrentMilestoneTimestamp {
			mark.MilestoneTimestamp = CurrentMilestoneTimestamp
		}
	}
	return marks, address, nil
}

// handle group mark basic output created
func (im *Manager) HandleGroupMarkBasicOutputConsumedAndCreated(createdOutput *OutputAndOutputIdAndMilestoneIndexAndMilestoneTimestamp, logger *logger.Logger) {

	// log entering
	logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ...")
	var createdMarkGroupIds []string
	var createdMarks []*Mark
	var address string
	if createdOutput != nil {
		_createdMarks, _address, err := im.GetMarksFromBasicOutput(createdOutput)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
			return
		}
		createdMarks = _createdMarks
		address = _address
	}
	// log address
	logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... address:%s", address)
	if address == "" {
		return
	}
	for _, mark := range createdMarks {
		createdMarkGroupIds = append(createdMarkGroupIds, iotago.EncodeHex(mark.GroupId[:]))
	}
	var existingMarkGroupIds []string
	var existingMarks []*Mark
	err := im.imStore.Iterate(im.AddressMarkKeyPrefix(address), func(key kvstore.Key, value kvstore.Value) bool {
		mark := im.AddressMarkKeyAndValueToMark(key, value)
		existingMarks = append(existingMarks, mark)
		existingMarkGroupIds = append(existingMarkGroupIds, iotago.EncodeHex(mark.GroupId[:]))
		return true
	})
	if err != nil {
		// log error
		logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
		return
	}
	// log existingMarkGroupIds, createdMarkGroupIds
	logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... existingMarkGroupIds:%v,createdMarkGroupIds:%v", existingMarkGroupIds, createdMarkGroupIds)
	// calculate difference
	var unmarkedMarkGroupIds []string
	for _, existingMarkGroupId := range existingMarkGroupIds {
		found := false
		for _, createdMarkGroupId := range createdMarkGroupIds {
			if existingMarkGroupId == createdMarkGroupId {
				found = true
				break
			}
		}
		if !found {
			unmarkedMarkGroupIds = append(unmarkedMarkGroupIds, existingMarkGroupId)
		}
	}
	var markedMarkGroupIds []string
	for _, createdMarkGroupId := range createdMarkGroupIds {
		found := false
		for _, existingMarkGroupId := range existingMarkGroupIds {
			if existingMarkGroupId == createdMarkGroupId {
				found = true
				break
			}
		}
		if !found {
			markedMarkGroupIds = append(markedMarkGroupIds, createdMarkGroupId)
		}
	}
	// log unmarkedMarkGroupIds, markedMarkGroupIds
	logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... unmarkedMarkGroupIds:%v,markedMarkGroupIds:%v", unmarkedMarkGroupIds, markedMarkGroupIds)
	// delete unmarked marks
	for _, mark := range existingMarks {
		found := false
		for _, unmarkedMarkGroupId := range unmarkedMarkGroupIds {
			if iotago.EncodeHex(mark.GroupId[:]) == unmarkedMarkGroupId {
				found = true
				break
			}
		}
		if found {
			err := im.DeleteMark(mark, true, logger)
			if err != nil {
				// log error
				logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
				return
			}
		}
	}

	// store marked marks
	for _, mark := range createdMarks {
		found := false
		for _, markedMarkGroupId := range markedMarkGroupIds {
			if iotago.EncodeHex(mark.GroupId[:]) == markedMarkGroupId {
				found = true
				break
			}
		}
		if found {
			err := im.StoreMark(mark, true, logger)
			if err != nil {
				// log error
				logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
				return
			}
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
func (im *Manager) FilterMarkOutputFromLedgerOutput(output *inx.LedgerOutput, logger *logger.Logger) (*iotago.BasicOutput, iotago.OutputID, bool) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, iotago.OutputID{}, false
	}
	outputId := output.UnwrapOutputID()
	outputFiltered, is := im.FilterMarkOutput(iotaOutput, logger)
	return outputFiltered, outputId, is
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
