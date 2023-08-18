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

func (im *Manager) MessageKeyFromGroupId(groupId []byte) []byte {
	index := 0
	key := make([]byte, 1+GroupIdLen)
	key[index] = ImStoreKeyPrefixMessage
	index++
	copy(key[index:], groupId)
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
	key := im.MessageKeyFromGroupIdMileStone(
		message.GroupId,
		message.MileStoneIndex)
	valuePayload := make([]byte, 4+OutputIdLen)
	binary.BigEndian.PutUint32(valuePayload, message.MileStoneTimestamp)
	copy(valuePayload[4:], message.OutputId)
	err := im.imStore.Set(key, valuePayload)

	keyHex := iotago.EncodeHex(key)
	valueHex := iotago.EncodeHex(valuePayload)
	logger.Infof("store message with key %s, value %s", keyHex, valueHex)
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
			go im.pushInbox(groupId, value, logger)
		}
		if err := im.storeSingleMessage(message, logger); err != nil {
			return err
		}
		message_ := message
		go func() {

			nfts, err := im.ReadNFTsFromGroupId(groupId)
			if err != nil {
				logger.Errorf("ReadNFTsFromGroupId error %v", err)
			}
			for _, nft := range nfts {
				//log nft
				err := im.storeInbox(nft.OwnerAddress, message_, value, logger)
				if err != nil {
					logger.Errorf("storeInbox error %v", err)
				}
			}
		}()

	}
	return nil
}

// handle inbox storage
func (im *Manager) storeInbox(receiverAddress []byte, message *Message, value []byte, logger *logger.Logger) error {
	key := make([]byte, 1+Sha256HashLen+4+4)
	index := 0
	key[index] = ImStoreKeyPrefixInbox
	index++
	addressSha256Hash := Sha256HashBytes(receiverAddress)
	copy(key[index:], addressSha256Hash)
	index += Sha256HashLen
	binary.BigEndian.PutUint32(key[index:], maxUint32-message.MileStoneTimestamp)
	index += 4
	incrementer := GetIncrementer()
	counter := maxUint32 - incrementer.Increment(message.MileStoneIndex)
	binary.BigEndian.PutUint32(key[index:], counter)

	err := im.imStore.Set(key, value)
	return err
}

// push message via mqtt
func (im *Manager) pushInbox(receiverAddress []byte, token []byte, logger *logger.Logger) {
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
