package im

import (
	"errors"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
)

type GroupShared struct {
	GroupId             [GroupIdLen]byte
	OutputId            [OutputIdLen]byte
	SenderBech32Address string
}

func NewGroupShared(groupId []byte, outputId []byte, senderBech32Address string) *GroupShared {
	shared := &GroupShared{
		GroupId:             [GroupIdLen]byte{},
		OutputId:            [OutputIdLen]byte{},
		SenderBech32Address: senderBech32Address,
	}
	copy(shared.GroupId[:], groupId)
	copy(shared.OutputId[:], outputId)
	return shared
}

func (im *Manager) SharedKeyFromGroupId(groupId [GroupIdLen]byte) []byte {
	index := 0
	key := make([]byte, 1+GroupIdLen)
	key[index] = ImStoreKeyPrefixShared
	index++
	copy(key[index:], groupId[:])
	return key
}

func (im *Manager) storeSingleShared(shared *GroupShared, logger *logger.Logger) error {
	if !IsIniting {
		// filter out shareds that address is not in the group's qualification
		isQualify, err := im.GroupQualificationExists(shared.GroupId, shared.SenderBech32Address)
		if err != nil {
			// log error
			logger.Warnf("storeSingleShared ... GroupQualificationExists failed:%s", err)
			return err
		}
		if !isQualify {
			return nil
		}
	}
	key := im.SharedKeyFromGroupId(
		shared.GroupId,
	)
	err := im.imStore.Set(key, shared.OutputId[:])
	if err != nil {
		return err
	}
	return nil
}

func (im *Manager) DeleteSingleShared(shared *GroupShared, logger *logger.Logger) error {
	// key = ImStoreKeyPrefixSharedForConsolidation + senderAddressSha256 + mileStoneIndex + mileStoneTimestamp + metaSha256
	key := im.SharedKeyFromGroupId(
		shared.GroupId,
	)
	return im.imStore.Delete(key)
}

func (im *Manager) storeNewShareds(shareds []*GroupShared, logger *logger.Logger) error {

	for _, shared := range shareds {
		if err := im.storeSingleShared(shared, logger); err != nil {
			return err
		}
	}
	return nil
}

// delete Shareds
func (im *Manager) DeleteConsumedShareds(shareds []*GroupShared, logger *logger.Logger) error {
	for _, shared := range shareds {
		if err := im.DeleteSingleShared(shared, logger); err != nil {
			return err
		}
	}
	return nil
}
func (im *Manager) ReadSharedFromGroupId(groupId [GroupIdLen]byte) (*GroupShared, error) {
	keyPrefix := im.SharedKeyFromGroupId(groupId)
	res, err := im.imStore.Get(keyPrefix)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	var outputId [OutputIdLen]byte
	copy(outputId[:], res)
	shared := &GroupShared{
		GroupId:  groupId,
		OutputId: outputId,
	}
	return shared, nil
}

// delete shared for one group
func (im *Manager) DeleteSharedFromGroupId(groupId [GroupIdLen]byte) error {
	keyPrefix := im.SharedKeyFromGroupId(groupId)
	return im.imStore.Delete(keyPrefix)
}
