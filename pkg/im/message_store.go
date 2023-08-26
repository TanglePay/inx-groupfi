package im

import (
	"bytes"
	"encoding/binary"

	"github.com/pkg/errors"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/core/marshalutil"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	DBVersionIM byte = 1
)

var (
	ErrUnknownParticipation                  = errors.New("no participation found")
	ErrEventNotFound                         = errors.New("referenced event does not exist")
	ErrInvalidEvent                          = errors.New("invalid event")
	ErrInvalidPreviouslyTrackedParticipation = errors.New("a previously tracked participation changed and is now invalid")
	ErrInvalidCurrentBallotVoteBalance       = errors.New("current ballot vote balance invalid")
	ErrInvalidCurrentStakedAmount            = errors.New("current staked amount invalid")
	ErrInvalidCurrentRewardsAmount           = errors.New("current rewards amount invalid")
)

// Status

func ledgerIndexKey() []byte {
	m := marshalutil.New(12)
	m.WriteByte(ImStoreKeyPrefixStatus)
	m.WriteBytes([]byte("ledgerIndex"))

	return m.Bytes()
}

func (im *Manager) storeLedgerIndex(index iotago.MilestoneIndex) error {
	m := marshalutil.New(4)
	m.WriteUint32(index)

	return im.imStore.Set(ledgerIndexKey(), m.Bytes())
}

func (im *Manager) readLedgerIndex() (iotago.MilestoneIndex, error) {
	v, err := im.imStore.Get(ledgerIndexKey())
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return 0, nil
		}

		return 0, err
	}
	m := marshalutil.New(v)
	u, err := m.ReadUint32()

	return u, err
}

// Message
const maxUint32 = ^uint32(0)

func (im *Manager) MessageKeyFromGroupIdMileStone(groupId []byte, mileStoneIndex uint32) []byte {
	incrementer := GetIncrementer()
	counter := maxUint32 - incrementer.Increment(mileStoneIndex)
	timeSuffix := maxUint32 - mileStoneIndex
	index := 0
	key := make([]byte, 1+GroupIdLen+4+4) // 4 bytes for uint32
	key[index] = ImStoreKeyPrefixMessage
	index++
	copy(key[index:], groupId)
	index += GroupIdLen
	binary.BigEndian.PutUint32(key[index:], timeSuffix)
	index += 4
	binary.BigEndian.PutUint32(key[index:], counter)
	return key
}

// message key from message
func (im *Manager) MessageKeyFromMessage(message *Message, groupIdOverride []byte, prefix byte) []byte {
	// key = prefix + (groupIdOverride != nil ? groupIdOverride : groupId) + mileStoneIndex + mileStoneTimestamp + metaSha256
	index := 0
	key := make([]byte, 1+Sha256HashLen+4+4+Sha256HashLen) // 4 bytes for uint32
	if prefix == 0 {
		key[index] = ImStoreKeyPrefixMessage
	} else {
		key[index] = prefix
	}
	index++
	groupId := message.GroupId
	if groupIdOverride != nil {
		groupId = groupIdOverride
	}
	copy(key[index:], groupId)
	index += GroupIdLen
	binary.BigEndian.PutUint32(key[index:], message.MileStoneIndex)
	index += 4
	binary.BigEndian.PutUint32(key[index:], message.MileStoneTimestamp)
	index += 4
	copy(key[index:], message.MetaSha256)
	return key
}
func (im *Manager) MessageKeyFromGroupId(groupId []byte) []byte {
	return im.KeyFromSha256hashAndPrefix(groupId, ImStoreKeyPrefixMessage)
}

func (im *Manager) KeyFromSha256hashAndPrefix(sha256hash []byte, prefix byte) []byte {
	index := 0
	key := make([]byte, 1+Sha256HashLen)
	key[index] = prefix
	index++
	copy(key[index:], sha256hash)
	return key
}
func messageKeyPrefixFromGroupIdAndMileStone(groupId []byte, mileStoneIndex uint32) []byte {
	timeSuffix := maxUint32 - mileStoneIndex
	index := 0
	key := make([]byte, 1+GroupIdLen+4) // 4 bytes for uint64
	key[index] = ImStoreKeyPrefixMessage
	index++
	copy(key[index:], groupId)
	index += GroupIdLen
	binary.BigEndian.PutUint32(key[index:], timeSuffix)
	return key
}

func (im *Manager) storeSingleMessage(message *Message, logger *logger.Logger) error {
	key := im.MessageKeyFromMessage(message, nil, 0)
	valuePayload := make([]byte, 4+OutputIdLen)
	binary.BigEndian.PutUint32(valuePayload, message.MileStoneTimestamp)
	copy(valuePayload[4:], message.OutputId)
	err := im.imStore.Set(key, valuePayload)

	//TODO remove log
	/*
		keyHex := iotago.EncodeHex(key)
		valueHex := iotago.EncodeHex(valuePayload)
		logger.Infof("store message with key %s, value %s", keyHex, valueHex)
	*/
	go func() {

		nfts, err := im.ReadNFTsFromGroupId(message.GroupId)
		if err != nil {
			logger.Errorf("ReadNFTsFromGroupId error %v", err)
		}
		for _, nft := range nfts {
			//log nft
			err := im.storeInbox(nft.OwnerAddress, message, valuePayload, logger)
			if err != nil {
				logger.Errorf("storeInbox error %v", err)
			}
		}
	}()
	return err
}

// delete single message
func (im *Manager) deleteSingleMessage(message *Message, logger *logger.Logger) error {
	go func() {

		nfts, err := im.ReadNFTsFromGroupId(message.GroupId)
		if err != nil {
			return
		}
		for _, nft := range nfts {
			//log nft
			err := im.DeleteInbox(nft.OwnerAddress, message, logger)
			if err != nil {
				logger.Errorf("storeInbox error %v", err)
			}
		}
	}()
	key := im.MessageKeyFromMessage(message, nil, 0)
	err := im.imStore.Delete(key)
	return err
}
func (im *Manager) storeNewMessages(messages []*Message, logger *logger.Logger, isPush bool) error {
	// log is push
	logger.Infof("storeNewMessages : isPush %v", isPush)
	for _, message := range messages {

		groupId := message.GroupId
		// value = one byte type + groupId + outputId
		value := make([]byte, 1+GroupIdLen+OutputIdLen)
		value[0] = ImInboxMessageTypeNewMessage
		copy(value[1:], message.GroupId)
		copy(value[1+GroupIdLen:], message.OutputId)
		if isPush {
			go im.PushInbox(groupId, value, logger)
		}
		err := im.storeSingleMessage(message, logger)
		if err != nil {
			return err
		}

	}
	return nil
}

// delete messages
func (im *Manager) deleteConsumedMessages(messages []*Message, logger *logger.Logger) error {
	for _, message := range messages {
		err := im.deleteSingleMessage(message, logger)
		if err != nil {
			// log and continue
			logger.Errorf("deleteSingleMessage error %v", err)
		}
	}
	return nil
}

// handle inbox storage
func (imm *Manager) storeInbox(receiverAddress []byte, message *Message, value []byte, logger *logger.Logger) error {
	receiverAddressSha256 := Sha256HashBytes(receiverAddress)
	key := imm.MessageKeyFromMessage(message, receiverAddressSha256, ImStoreKeyPrefixInbox)
	err := imm.imStore.Set(key, value)
	return err
}

// read inbox, all message with milestonetimestamp < given milestonetimestamp
func (im *Manager) ReadInboxForConsolidation(ownerAddress string, thresMileStoneTimestamp uint32, logger *logger.Logger) ([]string, error) {
	ownerAddressSha256 := Sha256Hash(ownerAddress)
	keyPrefix := im.KeyFromSha256hashAndPrefix(ownerAddressSha256, ImStoreKeyPrefixInbox)
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

// handle inbox delete
func (im *Manager) DeleteInbox(receiverAddress []byte, message *Message, logger *logger.Logger) error {
	receiverAddressSha256 := Sha256HashBytes(receiverAddress)
	key := im.MessageKeyFromMessage(message, receiverAddressSha256, ImStoreKeyPrefixInbox)
	err := im.imStore.Delete(key)
	return err
}

// push message via mqtt
func (im *Manager) PushInbox(receiverAddress []byte, token []byte, logger *logger.Logger) {
	// payload = groupId + outputId

	err := im.mqttServer.Publish("inbox/"+iotago.EncodeHex(receiverAddress), token)
	//log topic only
	logger.Infof("push message to inbox/%s", string(receiverAddress))

	if err != nil {
		logger.Errorf("pushMessage error %v", err)
	}
}

func (im *Manager) LogAllData(logger *logger.Logger) error {
	err := im.imStore.Iterate(kvstore.EmptyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		keyHex := iotago.EncodeHex(key)
		valueHex := iotago.EncodeHex(value)
		logger.Infof("key %s, value %s", keyHex, valueHex)
		return true
	})
	if err != nil {
		return err
	}
	return nil
}
func (im *Manager) ReadMessageFromPrefix(keyPrefix []byte, size int, coninueationToken []byte) ([]*Message, error) {
	ct := 0
	var res []*Message
	skiping := len(coninueationToken) > 0
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {

		if skiping {
			if bytes.Equal(key, coninueationToken) {
				skiping = false
			}
			return true
		}
		res = append(res, &Message{
			OutputId:           value[4:],
			Token:              key,
			MileStoneTimestamp: binary.BigEndian.Uint32(value[:4]),
		})
		ct++
		return ct < size
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}

// parse message value paylod
func (im *Manager) ParseMessageValuePayload(value []byte) (*Message, error) {
	if len(value) != 4+OutputIdLen {
		return nil, errors.New("invalid value length")
	}
	m := &Message{
		MileStoneTimestamp: binary.BigEndian.Uint32(value[:4]),
		OutputId:           value[4:],
	}
	return m, nil
}
func (im *Manager) ReadMessageUntilPrefix(keyPrefix []byte, size int, coninueationToken []byte) ([]*Message, error) {
	ct := 0
	var res []*Message
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		if bytes.Equal(key, coninueationToken) {
			return false
		}
		res = append(res, &Message{
			OutputId:           value[4:],
			Token:              key,
			MileStoneTimestamp: binary.BigEndian.Uint32(value[:4]),
		})
		ct++
		return ct < size
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
