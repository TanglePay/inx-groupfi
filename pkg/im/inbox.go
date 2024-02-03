package im

import (
	"bytes"
	"encoding/binary"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
)

// inbox event types
const (
	// plain text, new message
	ImInboxEventTypeNewMessage         byte = 1
	ImInboxEventTypeGroupMemberChanged byte = 2
)

type EventCommonFields struct {
	Token     []byte
	EventType byte
}
type EventJsonCommonFields struct {
	EventType byte `json:"type"`
}
type InboxItemJson interface {
	// set event type
	SetEventType(eventType byte)
}

type InboxItem interface {
	// GetToken
	GetToken() []byte
	// GetEventType
	GetEventType() byte

	// set token
	SetToken(token []byte)
	// set event type
	SetEventType(eventType byte)

	// to InboxItemJson
	Jsonable() InboxItemJson
}

// prefix is ImStoreKeyPrefixInbox
// inbox key from message, key = prefix + addressSha256Hash + mileStoneIndex + mileStoneTimestamp + metaSha256 + event type

// inbox key from addressSha256Hash mileStoneIndex mileStoneTimestamp  metaSha256 event type
func (im *Manager) InboxKeyFromValues(addressSha256Hash []byte, mileStoneIndex uint32, mileStoneTimestamp uint32, metaSha256 []byte, eventType byte) []byte {
	index := 0
	key := make([]byte, 1+Sha256HashLen+4+4+Sha256HashLen+1)
	key[index] = ImStoreKeyPrefixInbox
	index++
	copy(key[index:], addressSha256Hash)
	index += Sha256HashLen
	binary.BigEndian.PutUint32(key[index:], mileStoneIndex)
	index += 4
	binary.BigEndian.PutUint32(key[index:], mileStoneTimestamp)
	index += 4
	copy(key[index:], metaSha256)
	index += Sha256HashLen
	key[index] = eventType
	return key
}

// inbox prefix for address hash
func (im *Manager) InboxPrefixFromAddressHash(addressHash []byte) []byte {
	index := 0
	key := make([]byte, 1+Sha256HashLen) // 4 bytes for uint32
	key[index] = ImStoreKeyPrefixInbox
	index++
	copy(key[index:], addressHash)
	index += Sha256HashLen
	return key
}

// read inbox for address, from given token, limit by size
func (im *Manager) ReadInbox(addressSha256Hash []byte, coninueationToken []byte, size int, logger *logger.Logger) ([]InboxItem, error) {
	keyPrefix := im.InboxPrefixFromAddressHash(addressSha256Hash)
	var res []InboxItem
	skiping := len(coninueationToken) > 0
	startPoint := ConcatByteSlices(keyPrefix, coninueationToken)
	ct := 0
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		if skiping {
			if bytes.Equal(key, startPoint) {
				skiping = false
			}
			return true
		}
		// get event type from key
		eventType := key[len(key)-1]
		// token is key remove prefix and groupId
		token := key[(1 + GroupIdLen):]
		var eventItem InboxItem
		if eventType == ImInboxEventTypeNewMessage {
			message, err := im.ParseMessageValuePayload(value)
			if err != nil {
				// log and continue
				logger.Errorf("ParseMessageValuePayload error %v", err)
				return true
			}
			eventItem = message

		}
		if eventType == ImInboxEventTypeGroupMemberChanged {
			groupMemberChangedEvent, err := im.UnserializeGroupMemberChangedEvent(value)
			if err != nil {
				// log and continue
				logger.Errorf("UnserializeGroupMemberChangedEvent error %v", err)
				return true
			}
			eventItem = groupMemberChangedEvent
		}
		eventItem.SetToken(token)
		eventItem.SetEventType(eventType)
		res = append(res, eventItem)
		ct++
		return ct < size
	})
	if err != nil {
		return nil, err
	}
	return res, nil

}
