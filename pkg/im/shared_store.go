package im

import (
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
	keyHex := iotago.EncodeHex(key)
	valueHex := iotago.EncodeHex(valuePayload)
	logger.Infof("store shared with key %s, value %s", keyHex, valueHex)
	return err
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
