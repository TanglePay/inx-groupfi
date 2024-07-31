package im

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

// constant evm address len = 20
const (
	EvmAddressLen     = 20
	AddressTypeEvm    = 1
	AddressTypeSolana = 2
)

var evmQualifyTagRawStr = "GROUPFIQUALIFYV1"
var evmQualifyTag = []byte(evmQualifyTagRawStr)
var EvmQualifyTagStr = iotago.EncodeHex(evmQualifyTag)

type EvmQualify struct {
	OutputId    [OutputIdLen]byte
	GroupId     [GroupIdLen]byte
	AddressList []string
	Signature   []byte
}

// new evm qualify
func NewEvmQualify(outputId [OutputIdLen]byte, groupId [GroupIdLen]byte, addressList []string, signature []byte) *EvmQualify {
	return &EvmQualify{
		OutputId:    outputId,
		GroupId:     groupId,
		AddressList: addressList,
		Signature:   signature,
	}
}

// unmarsal evm qualify from bytes
func UnmarshalEvmQualify(outputId [OutputIdLen]byte,
	data []byte, logger *logger.Logger) (*EvmQualify, error) {
	// schema(1byte) + signature length  + signature + group id + address list
	idx := 0
	commonHeader, err := DeserializeCommonHeader(data, &idx)
	if err != nil {
		return nil, err
	}
	// log schema
	logger.Infof("UnmarshalEvmQualify schema %d", commonHeader.SchemaVersion)
	signatureBytes, err := ReadBytesWithUint16Len(data, &idx)
	if err != nil {
		return nil, err
	}
	// log signature
	logger.Infof("UnmarshalEvmQualify signature %s", string(signatureBytes))
	groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	// log group id
	logger.Infof("UnmarshalEvmQualify groupId %s", iotago.EncodeHex(groupId))
	groupIdFixed := [GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	addressType := AddressTypeEvm
	if commonHeader.SchemaVersion > 1 {
		addressTypeBytes, err := ReadBytesWithUint16Len(data, &idx, 1)
		if err != nil {
			return nil, err
		}
		addressType = int(addressTypeBytes[0])
		timestampBytes, err := ReadBytesWithUint16Len(data, &idx, 4)
		if err != nil {
			return nil, err
		}
		timestamp := BytesToUint32(timestampBytes)
		// log timestamp
		logger.Infof("UnmarshalEvmQualify timestamp %d", timestamp)
	}
	addressLen := EvmAddressLen
	if addressType == AddressTypeSolana {
		addressLen = SolanaAddressLength
	}
	addressList := make([]string, 0)
	for idx < len(data) {
		if len(data)-idx < addressLen {
			return nil, fmt.Errorf("invalid evm qualify data")
		}
		address, err := ReadBytesWithUint16Len(data, &idx, addressLen)
		if err != nil {
			return nil, err
		}
		addressString := ""
		if addressType == AddressTypeEvm {
			addressString = iotago.EncodeHex(address)
		} else if addressType == AddressTypeSolana {
			solanaAddress, err := UnmarshalSolanaAddress(address)
			if err != nil {
				return nil, err
			}
			addressString = solanaAddress
		}
		// log address
		logger.Infof("UnmarshalEvmQualify address %s", addressString)
		addressList = append(addressList, addressString)
	}
	return NewEvmQualify(outputId,
		groupIdFixed, addressList, signatureBytes), nil
}

// store one evm qualify
func (im *Manager) StoreSingleEvmQualify(evmQualify *EvmQualify, logger *logger.Logger) error {
	// log evm qualify, all fields, including group id, address list, signature
	logger.Infof("StoreSingleEvmQualify ,groupId %s, addressList %v, signature %s", iotago.EncodeHex(evmQualify.GroupId[:]), evmQualify.AddressList, iotago.EncodeHex(evmQualify.Signature))
	// check if group id exist
	if bytes.Equal(evmQualify.GroupId[:], []byte{}) {
		return fmt.Errorf("StoreSingleEvmQualify invalid group id")
	}

	// get group config by group id
	groupConfig, err := ReadGroupConfigMetaFromGroupId(evmQualify.GroupId, im)
	if err != nil {
		return err
	}
	if groupConfig == nil {
		groupIdHex := iotago.EncodeHex(evmQualify.GroupId[:])
		return fmt.Errorf("group config not found for group id %s", groupIdHex)
	}
	// get group qualify type
	groupQualifyType := groupConfig.QualifyType
	// log group qualify type
	logger.Infof("StoreSingleEvmQualify group qualify type %s", groupQualifyType)
	// store if not exist
	for _, addressStr := range evmQualify.AddressList {
		// log address
		logger.Infof("StoreSingleEvmQualify address %s", addressStr)
		exist, err := im.GroupQualificationExists(evmQualify.GroupId, addressStr, logger)
		if err != nil {
			return err
		}
		if exist {
			// log address exist
			logger.Infof("StoreSingleEvmQualify address %s exist", addressStr)
		}

		hash := [Sha256HashLen]byte{}
		copy(hash[:], Sha256Hash(addressStr))
		var qualifyType int
		if groupQualifyType == "nft" {
			qualifyType = GroupQualifyTypeNft
		} else if groupQualifyType == "token" {
			qualifyType = GroupQualifyTypeToken
		} else {
			return fmt.Errorf("invalid group qualify type %s", groupQualifyType)
		}
		qualification := NewGroupQualification(evmQualify.GroupId, addressStr, hash, "", qualifyType, "")
		// log store group qualification
		logger.Infof("StoreSingleEvmQualify store group qualification %s", addressStr)
		// store group qualification
		err = im.StoreGroupQualification(qualification, logger)
		if err != nil {
			return err
		}
		addressGroup := NewAddressGroupNft([]byte(addressStr), evmQualify.GroupId[:], "", "")
		err = im.StoreAddressGroup(addressGroup, logger)
		if err != nil {
			return err
		}
	}
	// go through all qualified address of this group, if not in evm qualify, delete
	qualifiedList, err := im.GetAllGroupQualificationsFromGroupId(evmQualify.GroupId, logger)
	if err != nil {
		return err
	}
	for _, qualified := range qualifiedList {
		addressPreviouslyQualified := qualified.Address
		found := false
		for _, addressStr := range evmQualify.AddressList {
			if addressPreviouslyQualified == addressStr {
				found = true
				break
			}
		}
		if !found {
			// delete group qualification
			err = im.DeleteGroupQualification(qualified, logger)
			if err != nil {
				return err
			}
			addressGroup := NewAddressGroupNft([]byte(addressPreviouslyQualified), evmQualify.GroupId[:], "", "")
			err = im.DeleteAddressGroup(addressGroup)
			if err != nil {
				return err
			}
		}
	}
	// StoreEvmQualifyOutputId
	err = StoreEvmQualifyOutputId(evmQualify.GroupId, evmQualify.OutputId, im, logger)
	if err != nil {
		return err
	}
	GenAndPushEvmQualifyChangedEvent(evmQualify, im, logger)
	return nil
}

// for effecting outputid for group key = prefix + groupid, value = outputid
func GetEvmQualifyOutputIdKey(groupId [GroupIdLen]byte) []byte {
	payload := make([]byte, 1+GroupIdLen)
	payload[0] = ImEvmQualifyOutputIdPrefix
	copy(payload[1:], groupId[:])
	return payload
}

// effecting qualify outputid store, key = prefix + outputid, value = nil
func GetEvmQualifyEffectingOutputIdKey(outputId [OutputIdLen]byte) []byte {
	payload := make([]byte, 1+OutputIdLen)
	payload[0] = ImEvmQualifyEffectingOutputIdPrefix
	copy(payload[1:], outputId[:])
	return payload
}

// store evm qualify effecting output id
func StoreEvmQualifyEffectingOutputId(outputId [OutputIdLen]byte, im *Manager, logger *logger.Logger) error {
	key := GetEvmQualifyEffectingOutputIdKey(outputId)
	err := im.imStore.Set(key, nil)
	if err != nil {
		return err
	}
	return nil
}

// delete evm qualify effecting output id
func DeleteEvmQualifyEffectingOutputId(outputId [OutputIdLen]byte, im *Manager, logger *logger.Logger) error {
	key := GetEvmQualifyEffectingOutputIdKey(outputId)
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

// check exist of evm qualify effecting output id
func EvmQualifyEffectingOutputIdExists(outputId [OutputIdLen]byte, im *Manager, logger *logger.Logger) (bool, error) {
	key := GetEvmQualifyEffectingOutputIdKey(outputId)
	exist, err := im.imStore.Has(key)
	if err != nil {
		return false, err
	}
	return exist, nil
}

// store evm qualify output id
func StoreEvmQualifyOutputId(groupId [GroupIdLen]byte, outputId [OutputIdLen]byte, im *Manager, logger *logger.Logger) error {
	// get current output id, delete effecting
	currentOutputId, err := GetEvmQualifyOutputId(groupId, im, logger)
	if err != nil {
		return err
	}
	if currentOutputId != [OutputIdLen]byte{} {
		err = DeleteEvmQualifyEffectingOutputId(currentOutputId, im, logger)
		if err != nil {
			return err
		}
	}
	key := GetEvmQualifyOutputIdKey(groupId)
	value := outputId[:]
	err = im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	// store effecting
	err = StoreEvmQualifyEffectingOutputId(outputId, im, logger)
	if err != nil {
		return err
	}
	return nil
}

// get evm qualify output id
func GetEvmQualifyOutputId(groupId [GroupIdLen]byte, im *Manager, logger *logger.Logger) ([OutputIdLen]byte, error) {
	key := GetEvmQualifyOutputIdKey(groupId)
	value, err := im.imStore.Get(key)
	if err != nil {
		return [OutputIdLen]byte{}, err
	}
	if value == nil {
		return [OutputIdLen]byte{}, nil
	}
	outputId := [OutputIdLen]byte{}
	copy(outputId[:], value)
	return outputId, nil
}

// filter pairX from LedgerOutput
func (im *Manager) FilterEvmQualifyFromLedgerOutput(inxOutput *inx.LedgerOutput, logger *logger.Logger) (*EvmQualify, error) {
	if inxOutput == nil {
		return nil, nil
	}
	outputIdRaw := inxOutput.OutputId.Id
	outputId := [OutputIdLen]byte{}
	copy(outputId[:], outputIdRaw)
	output, err := inxOutput.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	return im.FilterEvmQualifyFromOutput(outputId, output, logger)
}

// filter evm qualify from output
func (im *Manager) FilterEvmQualifyFromOutput(outputId [OutputIdLen]byte, output iotago.Output, logger *logger.Logger) (*EvmQualify, error) {
	// check if tag is evm qualify
	if output.FeatureSet().TagFeature() == nil ||
		output.FeatureSet().TagFeature().Tag == nil ||
		!bytes.Equal(output.FeatureSet().TagFeature().Tag, evmQualifyTag) {
		return nil, nil
	}
	// try unmarshal evm qualify
	// log evm qualify found
	logger.Infof("found evm qualify output")
	// get metadata
	if output.FeatureSet().MetadataFeature() == nil {
		return nil, fmt.Errorf("metadata not found in evm qualify output")
	}
	qualify, err := UnmarshalEvmQualify(outputId, output.FeatureSet().MetadataFeature().Data, logger)
	if err != nil {
		// log error
		logger.Errorf("failed to unmarshal evm qualify output:%s", err.Error())
		return nil, err
	}
	// log evm qualify, fields by fields in one line
	logger.Infof("found evm qualify output, groupId %s, addressList %v, signature %s", iotago.EncodeHex(qualify.GroupId[:]), qualify.AddressList, iotago.EncodeHex(qualify.Signature))
	return qualify, nil
}

// verify evm qualify signature
func (im *Manager) VerifyEvmQualifySignature(evmQualify *EvmQualify, logger *logger.Logger) error {
	//TODO: verify signature
	return nil
}

// HandleEvmQualifyCreated
func (im *Manager) HandleEvmQualifyCreated(evmQualify *EvmQualify, logger *logger.Logger) error {
	// verify signature
	err := im.VerifyEvmQualifySignature(evmQualify, logger)
	if err != nil {
		return err
	}
	// store evm qualify
	err = im.StoreSingleEvmQualify(evmQualify, logger)
	if err != nil {
		// log error
		logger.Errorf("HandleEvmQualifyCreated store evm qualify error:%s", err.Error())
		return err
	}
	return nil
}

// struct for evm qualify changed event, including groupId, timestamp
type EvmQualifyChangedEvent struct {
	EventCommonFields
	GroupId   [GroupIdLen]byte
	Timestamp uint32
}

// implements InboxItem
func (e *EvmQualifyChangedEvent) GetToken() []byte {
	return e.Token
}

func (e *EvmQualifyChangedEvent) GetEventType() byte {
	return e.EventType
}

func (e *EvmQualifyChangedEvent) SetToken(token []byte) {
	e.Token = token
}

func (e *EvmQualifyChangedEvent) SetEventType(eventType byte) {
	e.EventType = eventType
}

func (e *EvmQualifyChangedEvent) Jsonable() InboxItemJson {
	json := &EvmQualifyChangedEventJson{
		GroupId:   iotago.EncodeHex(e.GroupId[:]),
		Timestamp: e.Timestamp,
	}
	json.SetEventType(e.EventType)
	return json
}

type EvmQualifyChangedEventJson struct {
	EventJsonCommonFields
	GroupId   string `json:"groupId"`
	Timestamp uint32 `json:"timestamp"`
}

// implements InboxItemJson
func (e *EvmQualifyChangedEventJson) SetEventType(eventType byte) {
	e.EventType = eventType
}

// new evm qualify changed event
func NewEvmQualifyChangedEvent(groupId [GroupIdLen]byte, timestamp uint32) *EvmQualifyChangedEvent {
	return &EvmQualifyChangedEvent{
		GroupId:   groupId,
		Timestamp: timestamp,
	}
}

// serialize evm qualify changed event
func Serialize(e *EvmQualifyChangedEvent) []byte {
	// prefix(ImInboxEventTypeEvmQualifyChanged) + group id + timestamp
	var data []byte
	idx := 0
	AppendBytesWithUint16Len(&data, &idx, []byte{ImInboxEventTypeEvmQualifyChanged}, false)
	AppendBytesWithUint16Len(&data, &idx, e.GroupId[:], false)
	AppendBytesWithUint16Len(&data, &idx, Uint32ToBytes(e.Timestamp), false)
	return data
}

// deserialize evm qualify changed event
func UnserializeEvmQualifyChangedEvent(data []byte) (*EvmQualifyChangedEvent, error) {
	idx := 0
	// prefix
	_, err := ReadBytesWithUint16Len(data, &idx, 1)
	if err != nil {
		return nil, err
	}
	groupIdBytes, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	groupId := [GroupIdLen]byte{}
	copy(groupId[:], groupIdBytes)
	timestampBytes, err := ReadBytesWithUint16Len(data, &idx, 4)
	if err != nil {
		return nil, err
	}
	timestamp := BytesToUint32(timestampBytes)
	return &EvmQualifyChangedEvent{
		GroupId:   groupId,
		Timestamp: timestamp,
	}, nil
}

// GetTopicOfEvmQualifyChangedEvent
func GetTopicOfEvmQualifyChangedEvent(e *EvmQualifyChangedEvent) string {
	return iotago.EncodeHex(e.GroupId[:])
}

// GetPayloadOfEvmQualifyChangedEvent
func GetPayloadOfEvmQualifyChangedEvent(e *EvmQualifyChangedEvent) []byte {
	eventBytes := Serialize(e)
	return eventBytes
}

// get inbox of evm qualify changed event
func getInboxOfEvmQualifyChangedEvent(e *EvmQualifyChangedEvent) [][]byte {
	groupQualifications, err := Im.GetAllGroupQualificationsFromGroupId(e.GroupId, Logger)
	if err != nil {
		// log
		Logger.Errorf("getInboxOfEvmQualifyChangedEvent GetAllGroupQualificationsFromGroupId error:%s", err.Error())
		return nil
	}
	var inboxs [][]byte
	for _, groupQualification := range groupQualifications {
		gaddress := groupQualification.Address
		gaddressSha256Hash := Sha256Hash(gaddress)

		inboxs = append(inboxs, gaddressSha256Hash)

	}
	return inboxs
}

// get event type of evm qualify changed event
func getEventTypeOfEvmQualifyChangedEvent(e *EvmQualifyChangedEvent) byte {
	return ImInboxEventTypeEvmQualifyChanged
}

// gen and push evm qualify changed event
func GenAndPushEvmQualifyChangedEvent(e *EvmQualify, im *Manager, logger *logger.Logger) error {
	event := NewEvmQualifyChangedEvent(e.GroupId, CurrentMilestoneTimestamp)
	return PushData(event, GetTopicOfEvmQualifyChangedEvent, getInboxOfEvmQualifyChangedEvent, getEventTypeOfEvmQualifyChangedEvent, GetPayloadOfEvmQualifyChangedEvent, im, logger)
}
