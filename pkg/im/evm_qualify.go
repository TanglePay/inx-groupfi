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
	EvmAddressLen = 20
)

var evmQualifyTagRawStr = "GROUPFIQUALIFYV1"
var evmQualifyTag = []byte(evmQualifyTagRawStr)
var EvmQualifyTagStr = iotago.EncodeHex(evmQualifyTag)

type EvmQualify struct {
	GroupId     [GroupIdLen]byte
	AddressList [][EvmAddressLen]byte
	Signature   []byte
}

// new evm qualify
func NewEvmQualify(groupId [GroupIdLen]byte, addressList [][EvmAddressLen]byte, signature []byte) *EvmQualify {
	return &EvmQualify{
		GroupId:     groupId,
		AddressList: addressList,
		Signature:   signature,
	}
}

// unmarsal evm qualify from bytes
func UnmarshalEvmQualify(data []byte) (*EvmQualify, error) {
	// schema(1byte) + signature length  + signature + group id + address list
	idx := 0
	_, err := ReadBytesWithUint16Len(data, &idx, 1)
	if err != nil {
		return nil, err
	}
	signatureBytes, err := ReadBytesWithUint16Len(data, &idx)
	if err != nil {
		return nil, err
	}
	groupId, err := ReadBytesWithUint16Len(data, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	groupIdFixed := [GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	addressList := make([][EvmAddressLen]byte, 0)
	for idx < len(data) {
		if len(data)-idx < EvmAddressLen {
			return nil, fmt.Errorf("invalid evm qualify data")
		}
		address, err := ReadBytesWithUint16Len(data, &idx, EvmAddressLen)
		if err != nil {
			return nil, err
		}
		addressFixed := [EvmAddressLen]byte{}
		copy(addressFixed[:], address)
		addressList = append(addressList, addressFixed)
	}
	return NewEvmQualify(groupIdFixed, addressList, signatureBytes), nil
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
	groupIdHex := iotago.EncodeHex(evmQualify.GroupId[:])
	groupConfig := ConfigStoreGroupIdToGroupConfig[groupIdHex]
	if groupConfig == nil {
		return fmt.Errorf("group config not found for group id %s", groupIdHex)
	}
	// get group qualify type
	groupQualifyType := groupConfig.QualifyType
	// log group qualify type
	logger.Infof("StoreSingleEvmQualify group qualify type %s", groupQualifyType)
	// store if not exist
	for _, addressBytes := range evmQualify.AddressList {
		addressHex := iotago.EncodeHex(addressBytes[:])
		// log address
		logger.Infof("StoreSingleEvmQualify address %s", addressHex)
		exist, err := im.GroupQualificationExists(evmQualify.GroupId, addressHex)
		if err != nil {
			return err
		}
		if exist {
			// log address exist
			logger.Infof("StoreSingleEvmQualify address %s exist", addressHex)
			continue
		}

		hash := [Sha256HashLen]byte{}
		copy(hash[:], Sha256Hash(addressHex))
		var qualifyType int
		if groupQualifyType == "nft" {
			qualifyType = GroupQualifyTypeNft
		} else if groupQualifyType == "token" {
			qualifyType = GroupQualifyTypeToken
		} else {
			return fmt.Errorf("invalid group qualify type %s", groupQualifyType)
		}
		qualification := NewGroupQualification(evmQualify.GroupId, addressHex, hash, "", qualifyType, "")
		// log store group qualification
		logger.Infof("StoreSingleEvmQualify store group qualification %s", addressHex)
		// store group qualification
		err = im.StoreGroupQualification(qualification, logger)
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
		addressPreviouslyQualifiedBytes, err := iotago.DecodeHex(addressPreviouslyQualified)
		if err != nil {
			// log error then continue
			logger.Warnf("failed to decode hex %s", addressPreviouslyQualified)
			continue
		}
		// check if address is in evm qualify
		found := false
		for _, addressBytes := range evmQualify.AddressList {
			if bytes.Equal(addressPreviouslyQualifiedBytes, addressBytes[:]) {
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
		}
	}
	return nil
}

// filter pairX from LedgerOutput
func (im *Manager) FilterEvmQualifyFromLedgerOutput(inxOutput *inx.LedgerOutput, logger *logger.Logger) (*EvmQualify, error) {
	if inxOutput == nil {
		return nil, nil
	}
	output, err := inxOutput.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	return im.FilterEvmQualifyFromOutput(output, logger)
}

// filter evm qualify from output
func (im *Manager) FilterEvmQualifyFromOutput(output iotago.Output, logger *logger.Logger) (*EvmQualify, error) {
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
	qualify, err := UnmarshalEvmQualify(output.FeatureSet().MetadataFeature().Data)
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
