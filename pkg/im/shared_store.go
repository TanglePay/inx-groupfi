package im

import (
	"encoding/binary"
	"errors"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

func (im *Manager) SharedKeyFromGroupId(groupId []byte) []byte {
	index := 0
	key := make([]byte, 1+GroupIdLen)
	key[index] = ImStoreKeyPrefixShared
	index++
	copy(key[index:], groupId)
	return key
}

func (im *Manager) storeSingleShared(shared *Message, logger *logger.Logger) error {
	key := im.SharedKeyFromGroupId(
		shared.GroupId,
	)
	valuePayload := make([]byte, len(shared.OutputId))
	copy(valuePayload[0:], shared.OutputId)
	err := im.imStore.Set(key, valuePayload)
	if err != nil {
		return err
	}
	// StoreSharedForConsolidation
	return im.StoreSharedForConsolidation(shared, logger)

	//TODO remove log
	/*
		keyHex := iotago.EncodeHex(key)
		valueHex := iotago.EncodeHex(valuePayload)
		logger.Infof("store shared with key %s, value %s", keyHex, valueHex)
	*/
}

// store shared for consolidation
func (im *Manager) StoreSharedForConsolidation(shared *Message, logger *logger.Logger) error {
	// key = ImStoreKeyPrefixSharedForConsolidation + senderAddressSha256 + mileStoneIndex + mileStoneTimestamp + metaSha256
	key := im.MessageKeyFromMessage(shared, shared.SenderAddressSha256, ImStoreKeyPrefixSharedForConsolidation)
	// value = timestamp + outputid
	valuePayload := make([]byte, 4+OutputIdLen)
	binary.BigEndian.PutUint32(valuePayload, shared.MileStoneTimestamp)
	copy(valuePayload[4:], shared.OutputId)
	return im.imStore.Set(key, valuePayload)

}

/*
func (im *Manager) ReadInboxForConsolidation(ownerAddress string, thresMileStoneTimestamp uint32, logger *logger.Logger) ([]string, error) {
	ownerAddressSha256 := Sha256Hash(ownerAddress)
	keyPrefix := im.MessageKeyFromGroupId(ownerAddressSha256)
	var outputIds []string
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		// parse message value payload
		message, err := im.ParseMessageValuePayload(value)
		if err != nil {
			// log and continue
			logger.Errorf("ParseMessageValuePayload error %v", err)
			return true
		}
		if message.MileStoneTimestamp < thresMileStoneTimestamp {
			outputIds = append(outputIds, iotago.EncodeHex(message.OutputId))
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return outputIds, nil
}*/

func (im *Manager) ReadSharedForConsolidation(ownerAddress string, thresMileStoneTimestamp uint32, logger *logger.Logger) ([]string, error) {
	ownerAddressSha256 := Sha256Hash(ownerAddress)
	keyPrefix := im.KeyFromSha256hashAndPrefix(ownerAddressSha256, ImStoreKeyPrefixSharedForConsolidation)
	var outputIds []string
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		// parse message value payload
		message, err := im.ParseMessageValuePayload(value)
		if err != nil {
			// log and continue
			logger.Errorf("ParseMessageValuePayload error %v", err)
			return true
		}
		if message.MileStoneTimestamp < thresMileStoneTimestamp {
			outputIds = append(outputIds, iotago.EncodeHex(message.OutputId))
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return outputIds, nil
}
func (im *Manager) storeNewShareds(shareds []*Message, logger *logger.Logger) error {

	for _, shared := range shareds {
		if err := im.storeSingleShared(shared, logger); err != nil {
			return err
		}
	}
	return nil
}

func (im *Manager) ReadSharedFromGroupId(groupId []byte) (*Message, error) {
	keyPrefix := im.SharedKeyFromGroupId(groupId)
	res, err := im.imStore.Get(keyPrefix)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	shared := &Message{
		GroupId:  groupId,
		OutputId: res,
	}
	return shared, nil
}

// delete shared for one group
func (im *Manager) DeleteSharedFromGroupId(groupId []byte) error {
	keyPrefix := im.SharedKeyFromGroupId(groupId)
	return im.imStore.Delete(keyPrefix)
}
