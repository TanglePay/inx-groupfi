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
func (im *Manager) PublicMessageKeyFromMessage(message *Message) []byte {
	// key = prefix + groupId + mileStoneIndex + mileStoneTimestamp + metaSha256
	key := []byte{}
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixMessage}, false)
	AppendBytesWithUint16Len(&key, &index, message.GroupId[:], false)
	milestoneIndexReverseBytes := Uint32ToBytes(maxUint32 - message.MileStoneIndex)
	AppendBytesWithUint16Len(&key, &index, milestoneIndexReverseBytes, false)
	milestoneTimestampReverseBytes := Uint32ToBytes(maxUint32 - message.MileStoneTimestamp)
	AppendBytesWithUint16Len(&key, &index, milestoneTimestampReverseBytes, false)
	AppendBytesWithUint16Len(&key, &index, message.MetaSha256[:], false)
	return key
}

// message value from message
func (im *Manager) MessageValueFromMessage(message *Message) []byte {
	// AppendBytesWithUint16Len(bytes *[]byte, idx *int, slice []byte, appendLength bool) {
	// value = mileStoneTimestamp + outputId
	value := []byte{}
	index := 0
	AppendBytesWithUint16Len(&value, &index, Uint32ToBytes(message.MileStoneTimestamp), false)
	AppendBytesWithUint16Len(&value, &index, message.OutputId[:], false)
	return value
}

// parse message from key and value
func (im *Manager) ParseMessagePublicKeyAndValue(key kvstore.Key, value kvstore.Value) (*Message, error) {
	// func ReadBytesWithUint16Len(bytes []byte, idx *int, providedLength ...int) ([]byte, error) {
	// key = prefix + groupId + mileStoneIndex + mileStoneTimestamp + metaSha256
	// value = mileStoneTimestamp + outputId
	index := 0
	_, err := ReadBytesWithUint16Len(key, &index, 1)
	if err != nil {
		return nil, err
	}
	groupId, err := ReadBytesWithUint16Len(key, &index, GroupIdLen)
	if err != nil {
		return nil, err
	}
	milestoneIndexReverseBytes, err := ReadBytesWithUint16Len(key, &index, 4)
	if err != nil {
		return nil, err
	}
	milestoneIndex := maxUint32 - BytesToUint32(milestoneIndexReverseBytes)
	milestoneTimestampReverseBytes, err := ReadBytesWithUint16Len(key, &index, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := maxUint32 - BytesToUint32(milestoneTimestampReverseBytes)
	index = 0
	_, err = ReadBytesWithUint16Len(value, &index, 4)
	if err != nil {
		return nil, err
	}
	outputId, err := ReadBytesWithUint16Len(value, &index, OutputIdLen)
	if err != nil {
		return nil, err
	}
	message := &Message{
		GroupId:            groupId,
		MileStoneIndex:     milestoneIndex,
		MileStoneTimestamp: milestoneTimestamp,
		OutputId:           outputId,
	}
	token := im.PublicMessageTokenFromKey(key)
	message.SetToken(token)
	message.SetEventType(ImInboxEventTypeNewMessage)
	return message, nil
}

// token = mileStoneIndex + mileStoneTimestamp + metaSha256
func (im *Manager) PublicMessageTokenFromKey(key []byte) []byte {
	return key[1+GroupIdLen:]
}

// prefix = prefix + groupId
func (im *Manager) PublicMessageKeyPrefixFromGroupId(groupId []byte) []byte {
	// using AppendBytesWithUint16Len
	key := []byte{}
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixMessage}, false)
	AppendBytesWithUint16Len(&key, &index, groupId, false)
	return key
}

// message key from token for group
func (im *Manager) PublicMessageKeyFromTokenForGroup(token []byte, groupId []byte) []byte {
	// check nil
	if token == nil {
		return nil
	}
	// key = prefix + groupId + mileStoneIndex + mileStoneTimestamp + metaSha256
	key := []byte{}
	index := 0
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixMessage}, false)
	AppendBytesWithUint16Len(&key, &index, groupId, false)
	AppendBytesWithUint16Len(&key, &index, token, false)
	return key
}

// read public message given group id, start token, end token and size
func (im *Manager) ReadPublicItemsFromGroupId(groupId []byte, startToken []byte, endToken []byte, size int, isReverse bool, logger *logger.Logger) ([]InboxItem, error) {
	prefix := im.PublicMessageKeyPrefixFromGroupId(groupId)
	startKeyPoint := im.PublicMessageKeyFromTokenForGroup(startToken, groupId)
	endKeyPoint := im.PublicMessageKeyFromTokenForGroup(endToken, groupId)
	messages := []InboxItem{}
	isStarted := len(startToken) == 0
	var direction = kvstore.IterDirectionForward
	if isReverse {
		direction = kvstore.IterDirectionBackward
	}
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		if !isStarted {
			if startKeyPoint != nil &&
				bytes.Equal(key, startKeyPoint) {
				isStarted = true
			}
			return true
		}
		if endKeyPoint != nil &&
			bytes.Equal(key, endKeyPoint) {
			return false
		}
		message, err := im.ParseMessagePublicKeyAndValue(key, value)
		if err != nil {
			return false
		}
		messages = append(messages, message)
		return len(messages) < size
	}, direction)
	if err != nil {
		return nil, err
	}
	return messages, nil
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
	valuePayload := make([]byte, 4+OutputIdLen)
	binary.BigEndian.PutUint32(valuePayload, message.MileStoneTimestamp)
	copy(valuePayload[4:], message.OutputId)

	go func() {
		var groupId32 [GroupIdLen]byte
		copy(groupId32[:], message.GroupId)
		// TODO cache group member addresses
		addresses, err := im.GetGroupMemberAddressesFromGroupId(groupId32, logger)
		if err != nil {
			logger.Errorf("ReadNFTsFromGroupId error %v", err)
		}
		for _, address := range addresses {
			// log address
			logger.Infof("storeInbox : address %s", string(address))
			//log nft
			err := im.storeInboxMessage([]byte(address), message, valuePayload, logger)
			if err != nil {
				logger.Errorf("storeInbox error %v", err)
			}
		}
	}()
	err := im.StoreMessageForPublicGroup(message, logger)
	return err
}

// store message by groupId only if group is public
func (im *Manager) StoreMessageForPublicGroup(message *Message, logger *logger.Logger) error {
	key := im.PublicMessageKeyFromMessage(message)
	groupIdFixed := [GroupIdLen]byte{}
	copy(groupIdFixed[:], message.GroupId)
	isPublic := im.GetIsGroupPublic(groupIdFixed)
	if isPublic {
		valuePayload := make([]byte, 4+OutputIdLen)
		binary.BigEndian.PutUint32(valuePayload, message.MileStoneTimestamp)
		copy(valuePayload[4:], message.OutputId)
		err := im.imStore.Set(key, valuePayload)
		return err
	}
	return nil
}

// delete message by groupId for public group
func (im *Manager) DeleteMessageForPublicGroup(message *Message, logger *logger.Logger) error {
	key := im.PublicMessageKeyFromMessage(message)
	err := im.imStore.Delete(key)
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
			err := im.DeleteInboxMessage(nft.OwnerAddress, message, logger)
			if err != nil {
				logger.Errorf("storeInbox error %v", err)
			}
		}
	}()
	err := im.DeleteMessageForPublicGroup(message, logger)
	return err
}
func (im *Manager) storeNewMessages(messages []*Message, logger *logger.Logger, isPush bool) error {
	// log is push
	if len(messages) > 0 {
		logger.Infof("storeNewMessages : isPush %v", isPush)
	}
	for _, message := range messages {

		/*
			if isPush {
				groupId := message.GroupId
				// value = one byte type + groupId + outputId
				value := make([]byte, 1+GroupIdLen+OutputIdLen)
				value[0] = ImInboxMessageTypeNewMessage
				copy(value[1:], message.GroupId)
				copy(value[1+GroupIdLen:], message.OutputId)

				go im.PushInbox(groupId, value, logger)
			}
		*/
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
func (imm *Manager) storeInboxMessage(receiverAddress []byte, message *Message, value []byte, logger *logger.Logger) error {
	receiverAddressSha256 := Sha256HashBytes(receiverAddress)
	key := imm.InboxKeyFromMessage(message, receiverAddressSha256)
	// log receiverAddress receiverAddressSha256 key
	logger.Infof("storeInbox : receiverAddress %s, receiverAddressSha256 %s, key %s", string(receiverAddress), iotago.EncodeHex(receiverAddressSha256), iotago.EncodeHex(key))
	err := imm.imStore.Set(key, value)
	return err
}

func (im *Manager) InboxKeyFromMessage(message *Message, receiverAddressSha256 []byte) []byte {
	return im.InboxKeyFromValues(receiverAddressSha256, message.MileStoneIndex, message.MileStoneTimestamp, message.MetaSha256, ImInboxEventTypeNewMessage)
}

// handle inbox delete
func (im *Manager) DeleteInboxMessage(receiverAddress []byte, message *Message, logger *logger.Logger) error {
	receiverAddressSha256 := Sha256HashBytes(receiverAddress)
	key := im.InboxKeyFromMessage(message, receiverAddressSha256)
	err := im.imStore.Delete(key)
	//TODO cleanup any event before this message
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

func (im *Manager) LogAllData(prefix []byte, logger *logger.Logger) error {
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
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
		msg := &Message{
			OutputId:           value[4:],
			MileStoneTimestamp: binary.BigEndian.Uint32(value[:4]),
		}
		msg.SetToken(key)
		res = append(res, msg)
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
		msg := &Message{
			OutputId:           value[4:],
			MileStoneTimestamp: binary.BigEndian.Uint32(value[:4]),
		}
		msg.SetToken(key)
		res = append(res, msg)
		ct++
		return ct < size
	})
	if err != nil {
		return nil, err
	}

	return res, nil
}
