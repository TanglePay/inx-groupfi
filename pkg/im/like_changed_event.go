package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

type LikeChangedEvent struct {
	EventCommonFields
	AddressSha256Hash         [Sha256HashLen]byte
	ActionedAddressSha256Hash [Sha256HashLen]byte
	GroupID                   [GroupIdLen]byte
	IsLiked                   bool
	MilestoneTimestamp        uint32
}

func NewLikeChangedEvent(addressSha256Hash [Sha256HashLen]byte,
	actionedAddressSha256Hash [Sha256HashLen]byte,
	groupID [GroupIdLen]byte, isLiked bool, milestoneTimestamp uint32) *LikeChangedEvent {
	return &LikeChangedEvent{
		AddressSha256Hash:         addressSha256Hash,
		ActionedAddressSha256Hash: actionedAddressSha256Hash,
		GroupID:                   groupID,
		IsLiked:                   isLiked,
		MilestoneTimestamp:        milestoneTimestamp,
	}
}

// implements InboxItem
func (m *LikeChangedEvent) GetToken() []byte {
	return m.Token
}

func (m *LikeChangedEvent) GetEventType() byte {
	return m.EventType
}

func (m *LikeChangedEvent) SetToken(token []byte) {
	m.Token = token
}

func (m *LikeChangedEvent) SetEventType(eventType byte) {
	m.EventType = eventType
}

func (m *LikeChangedEvent) Jsonable() InboxItemJson {
	json := &LikeChangedEventJson{
		GroupID:           iotago.EncodeHex(m.GroupID[:]),
		AddressSha256Hash: iotago.EncodeHex(m.ActionedAddressSha256Hash[:]),
		Timestamp:         m.MilestoneTimestamp,
		IsLiked:           m.IsLiked,
	}
	json.SetEventType(m.EventType)
	return json
}

type LikeChangedEventJson struct {
	EventJsonCommonFields
	GroupID           string `json:"groupId"`
	AddressSha256Hash string `json:"addressHash"`
	Timestamp         uint32 `json:"timestamp"`
	IsLiked           bool   `json:"isLiked"`
}

// implements InboxItemJson
func (m *LikeChangedEventJson) SetEventType(eventType byte) {
	m.EventType = eventType
}

// serialize LikeChangedEvent to bytes
func SerializeLikeChangedEvent(m *LikeChangedEvent) []byte {
	bytes := make([]byte, 0)
	idx := 0

	// add prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImInboxEventTypeLikeChanged}, false)

	AppendBytesWithUint16Len(&bytes, &idx, m.GroupID[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, m.ActionedAddressSha256Hash[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(m.MilestoneTimestamp), false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte{BoolToByte(m.IsLiked)}, false)
	return bytes
}

// unserialize LikeChangedEvent from bytes
func UnserializeLikeChangedEvent(bytes []byte) (*LikeChangedEvent, error) {
	idx := 0
	_, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	groupID, err := ReadBytesWithUint16Len(bytes, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	var groupIDFixed [GroupIdLen]byte
	copy(groupIDFixed[:], groupID)
	addressSha256Hash, err := ReadBytesWithUint16Len(bytes, &idx, Sha256HashLen)
	if err != nil {
		return nil, err
	}
	var addressSha256HashFixed [Sha256HashLen]byte
	copy(addressSha256HashFixed[:], addressSha256Hash)
	milestoneTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := BytesToUint32(milestoneTimestampBytes)
	isLikedBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	isLiked := BytesToBool(isLikedBytes)
	return NewLikeChangedEvent(addressSha256HashFixed, addressSha256HashFixed, groupIDFixed, isLiked, milestoneTimestamp), nil
}

// implements InboxItem
func (m *LikeChangedEvent) ToPushTopic() []byte {
	return m.AddressSha256Hash[:]
}

// getTopic of LikeChangedEvent
func GetTopicOfLikeChangedEvent(m *LikeChangedEvent) string {
	return iotago.EncodeHex(m.AddressSha256Hash[:])
}

func (m *LikeChangedEvent) ToPushPayload() []byte {
	eventBytes := SerializeLikeChangedEvent(m)
	return append([]byte{ImInboxEventTypeLikeChanged}, eventBytes...)
}

// get payload of LikeChangedEvent
func GetPayloadOfLikeChangedEvent(m *LikeChangedEvent) []byte {
	eventBytes := SerializeLikeChangedEvent(m)
	return eventBytes
}

func getInboxOfLikeChangedEvent(m *LikeChangedEvent) [][]byte {
	return [][]byte{m.AddressSha256Hash[:]}
}

func getEventTypeOfLikeChangedEvent(m *LikeChangedEvent) byte {
	return ImInboxEventTypeLikeChanged
}

// gen and push LikeChangedEvent
func GenAndPushLikeChangedEvent(
	receiverAddressSha256Hash [Sha256HashLen]byte,
	actionedAddressSha256Hash [Sha256HashLen]byte,
	groupId [GroupIdLen]byte, isLiked bool, milestoneTimestamp uint32, im *Manager, logger *logger.Logger) error {
	event := NewLikeChangedEvent(receiverAddressSha256Hash, actionedAddressSha256Hash, groupId, isLiked, milestoneTimestamp)
	return PushData(event, GetTopicOfLikeChangedEvent, getInboxOfLikeChangedEvent, getEventTypeOfLikeChangedEvent, GetPayloadOfLikeChangedEvent, im, logger)
}
