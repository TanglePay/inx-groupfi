package im

import (
	"bytes"
	"encoding/binary"

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
	// timestamp
	Timestamp [TimestampLen]byte
}

// newMark creates a new Mark.
func NewMark(address string, groupId [GroupIdLen]byte, timestamp [4]byte) *Mark {
	return &Mark{
		Address:   address,
		GroupId:   groupId,
		Timestamp: timestamp,
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
	copy(key[index:], Sha256Hash(mark.Address))
	return key
}

// address mark key
// key = prefix + addressSha256Hash + groupid. value = timestamp + address
func (im *Manager) AddressMarkKey(mark *Mark) []byte {
	key := make([]byte, 1+Sha256HashLen+GroupIdLen)
	index := 0
	key[index] = ImStoreKeyPrefixAddressMark
	index++
	copy(key[index:], Sha256Hash(mark.Address))
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
	binary.LittleEndian.PutUint32(value[index:], binary.LittleEndian.Uint32(mark.Timestamp[:]))
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
	exists, err := im.GroupQualificationExists(mark.GroupId, mark.Address)
	if err != nil {
		return err
	}
	// log group qualification GroupId, address, exists
	logger.Infof("StoreMark,group qualification exists,groupId:%s,address:%s,exists:%t", iotago.EncodeHex(mark.GroupId[:]), mark.Address, exists)
	if exists {
		outputId := mark.OutputId
		resp, err := NodeHTTPAPIClient.OutputMetadataByID(ListeningCtx, outputId)
		if err != nil {
			return err
		}

		groupMember := NewGroupMember(mark.GroupId, mark.Address, resp.MilestoneIndexBooked, resp.MilestoneTimestampBooked)

		_, err = im.StoreGroupMember(groupMember, logger)
		if err != nil {
			return err
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
	groupMember := NewGroupMember(mark.GroupId, mark.Address, CurrentMilestoneIndex, CurrentMilestoneTimestamp)
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
	}

	return nil
}

// check if mark exists, input is group id and address
func (im *Manager) MarkExists(groupId [GroupIdLen]byte, address string) (bool, error) {
	key := im.MarkKey(NewMark(address, groupId, [4]byte{}))
	return im.imStore.Has(key)
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
	copy(key[index:], Sha256Hash(address))
	return key
}

// MarkKeyToMark, key = prefix + groupid + addressSha256Hash. value = timestamp + address
func (im *Manager) MarkKeyAndValueToMark(key kvstore.Key, value kvstore.Value) *Mark {
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1:1+GroupIdLen])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], value[:TimestampLen])
	address := string(value[TimestampLen:])
	return NewMark(address, groupId, timestamp)
}

// address mark key to mark
func (im *Manager) AddressMarkKeyAndValueToMark(key kvstore.Key, value kvstore.Value) *Mark {
	// key is prefix + addressSha256Hash + groupid
	var groupId [GroupIdLen]byte
	copy(groupId[:], key[1+Sha256HashLen:])
	var timestamp [TimestampLen]byte
	copy(timestamp[:], value[:TimestampLen])
	address := string(value[TimestampLen:])
	return NewMark(address, groupId, timestamp)
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
func (im *Manager) GetMarksFromBasicOutput(output *OutputAndOutputId) ([]*Mark, error) {
	unlockConditionSet := output.Output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	featureSet := output.Output.FeatureSet()
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil, errors.New("meta is nil")
	}
	outputId := output.OutputId
	marks, err := im.DeserializeUserMarkedGroupIds(ownerAddress, meta.Data)
	if err != nil {
		return nil, err
	}
	for _, mark := range marks {
		mark.OutputId = outputId
	}
	return marks, nil
}

// handle group mark basic output created
func (im *Manager) HandleGroupMarkBasicOutputConsumedAndCreated(consumedOutput *OutputAndOutputId, createdOutput *OutputAndOutputId, logger *logger.Logger) {

	// log entering
	logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ...")
	var consumedMarks []*Mark
	var createdMarks []*Mark
	if consumedOutput != nil {
		_consumedMarks, err := im.GetMarksFromBasicOutput(consumedOutput)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
			return
		}
		consumedMarks = _consumedMarks
	}
	if createdOutput != nil {
		_createdMarks, err := im.GetMarksFromBasicOutput(createdOutput)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
			return
		}
		createdMarks = _createdMarks
	}
	// map consumed marks to map[GroupId]true
	consumedMarksMap := make(map[[GroupIdLen]byte]bool)
	for _, mark := range consumedMarks {
		consumedMarksMap[mark.GroupId] = true
	}
	// map created marks to map[GroupId]true
	createdMarksMap := make(map[[GroupIdLen]byte]bool)
	for _, mark := range createdMarks {
		createdMarksMap[mark.GroupId] = true
	}
	// filter created marks out of consumed marks
	// loop through consumed marks, if groupId is in created marks, delete it
	var filteredConsumedMarks []*Mark
	for _, mark := range consumedMarks {
		_, ok := createdMarksMap[mark.GroupId]
		if ok {
			continue
		}
		filteredConsumedMarks = append(filteredConsumedMarks, mark)
	}
	// filter consumed marks out of created marks
	// loop through created marks, if groupId is in consumed marks, delete it
	var filteredCreatedMarks []*Mark
	for _, mark := range createdMarks {
		_, ok := consumedMarksMap[mark.GroupId]
		if ok {
			continue
		}
		filteredCreatedMarks = append(filteredCreatedMarks, mark)
	}
	// store filtered consumed marks

	for _, mark := range filteredConsumedMarks {
		err := im.DeleteMark(mark, true, logger)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
			return
		}
	}
	// store filtered created marks
	for _, mark := range filteredCreatedMarks {
		err := im.StoreMark(mark, true, logger)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
			return
		}
	}

	// store filtered created marks
	for _, mark := range filteredCreatedMarks {
		err := im.StoreMark(mark, true, logger)
		if err != nil {
			// log error
			logger.Infof("HandleGroupMarkBasicOutputConsumedAndCreated ... err:%s", err.Error())
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
