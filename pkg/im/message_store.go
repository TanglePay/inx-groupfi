package im

import (
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

func messageKeyFromGroupIdMileStoneAndOutputId(groupId []byte, mileStoneIndex uint32, outputId []byte) []byte {

	timeSuffix := maxUint32 - mileStoneIndex
	index := 0
	key := make([]byte, 1+GroupIdLen+4+OutputIdLen) // 4 bytes for uint32
	key[index] = ImStoreKeyPrefixMessage
	index++
	copy(key[index:], groupId)
	index += GroupIdLen
	binary.BigEndian.PutUint32(key[index:], timeSuffix)
	index += 4
	copy(key[index:], outputId)
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
	key := messageKeyFromGroupIdMileStoneAndOutputId(
		message.GroupId,
		message.MileStoneIndex,
		message.OutputId)
	timePayload := make([]byte, 4)
	binary.BigEndian.PutUint32(timePayload, message.MileStoneTimestamp)
	err := im.imStore.Set(key, timePayload)

	keyHex := iotago.EncodeHex(key)
	valueHex := iotago.EncodeHex(timePayload)
	logger.Infof("store message with key %s, value %s", keyHex, valueHex)
	return err
}
func (im *Manager) GetSingleMessage(key []byte, logger *logger.Logger) ([]byte, error) {
	value, err := im.imStore.Get(key)
	if err != nil {
		return nil, err
	}
	keyHex := iotago.EncodeHex(key)
	valueHex := iotago.EncodeHex(value)
	logger.Infof("get message with key %s, value %s", keyHex, valueHex)
	return value, nil
}
func (im *Manager) storeNewMessages(messages []*Message, logger *logger.Logger) error {

	for _, message := range messages {
		if err := im.storeSingleMessage(message, logger); err != nil {
			return err
		}
	}
	im.imStore.Flush()
	return nil
}

func (im *Manager) readMessageAfterToken(groupId []byte, token uint32, size int) ([]*Message, error) {
	keyPrefix := messageKeyPrefixFromGroupIdAndMileStone(groupId, token)
	ct := 0
	var res []*Message
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		res = append(res, &Message{
			OutputId:           key[(1 + GroupIdLen + 4):],
			MileStoneTimestamp: binary.BigEndian.Uint32(value),
		})
		ct++
		return ct >= size
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
