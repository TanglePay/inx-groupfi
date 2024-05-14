package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

type MarkChangedEvent struct {
	EventCommonFields
	// groupID
	AddressSha256Hash  [Sha256HashLen]byte
	GroupID            [GroupIdLen]byte
	IsNewMark          bool
	MilestoneTimestamp uint32
}

func NewMarkChangedEvent(addressSha256Hash [Sha256HashLen]byte, groupID [GroupIdLen]byte, IsNewMark bool, milestoneTimestamp uint32) *MarkChangedEvent {
	return &MarkChangedEvent{
		AddressSha256Hash:  addressSha256Hash,
		GroupID:            groupID,
		IsNewMark:          IsNewMark,
		MilestoneTimestamp: milestoneTimestamp,
	}
}

// implements InboxItem
func (m *MarkChangedEvent) GetToken() []byte {
	return m.Token
}
func (m *MarkChangedEvent) GetEventType() byte {
	return m.EventType
}
func (m *MarkChangedEvent) SetToken(token []byte) {
	m.Token = token
}
func (m *MarkChangedEvent) SetEventType(eventType byte) {
	m.EventType = eventType
}
func (m *MarkChangedEvent) Jsonable() InboxItemJson {
	json := &MarkChangedEventJson{
		GroupID:   iotago.EncodeHex(m.GroupID[:]),
		Timestamp: m.MilestoneTimestamp,
		IsNewMark: m.IsNewMark,
	}
	json.SetEventType(m.EventType)
	return json
}

type MarkChangedEventJson struct {
	EventJsonCommonFields
	GroupID   string `json:"groupId"`
	Timestamp uint32 `json:"timestamp"`
	IsNewMark bool   `json:"isNewMark"`
}

// implements InboxItemJson
func (m *MarkChangedEventJson) SetEventType(eventType byte) {
	m.EventType = eventType
}

// serialize MarkChangedEvent to bytes
func SerializeMarkChangedEvent(m *MarkChangedEvent) []byte {
	bytes := make([]byte, 0)
	idx := 0

	// add prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImInboxEventTypeMarkChanged}, false)

	AppendBytesWithUint16Len(&bytes, &idx, m.GroupID[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(m.MilestoneTimestamp), false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte{BoolToByte(m.IsNewMark)}, false)
	return bytes
}

// unserialize MarkChangedEvent from bytes
func (im *Manager) UnserializeMarkChangedEvent(bytes []byte) (*MarkChangedEvent, error) {
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
	milestoneTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := BytesToUint32(milestoneTimestampBytes)
	isNewMarkBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	isNewMark := BytesToBool(isNewMarkBytes)
	return NewMarkChangedEvent(groupIDFixed, groupIDFixed, isNewMark, milestoneTimestamp), nil
}

// implements InboxItem
func (m *MarkChangedEvent) ToPushTopic() []byte {
	return m.AddressSha256Hash[:]
}

// getTopic of MarkChangedEvent
func GetTopicOfMarkChangedEvent(m *MarkChangedEvent) string {
	return iotago.EncodeHex(m.AddressSha256Hash[:])
}
func (m *MarkChangedEvent) ToPushPayload() []byte {
	eventBytes := SerializeMarkChangedEvent(m)
	return append([]byte{ImInboxEventTypeMarkChanged}, eventBytes...)
}

// get payload of MarkChangedEvent
func GetPayloadOfMarkChangedEvent(m *MarkChangedEvent) []byte {
	eventBytes := SerializeMarkChangedEvent(m)
	return eventBytes
}

func getInboxOfMarkChangedEvent(m *MarkChangedEvent) [][]byte {
	return [][]byte{m.AddressSha256Hash[:]}
}

func getEventTypeOfMarkChangedEvent(m *MarkChangedEvent) byte {
	return ImInboxEventTypeMarkChanged
}

// gen and push MarkChangedEvent
func GenAndPushMarkChangedEvent(mark *Mark, isNewMark bool, im *Manager, logger *logger.Logger) error {
	event := NewMarkChangedEvent(Sha256HashFixed(mark.Address), mark.GroupId, isNewMark, CurrentMilestoneTimestamp)
	return PushData(event, GetTopicOfMarkChangedEvent, getInboxOfMarkChangedEvent, getEventTypeOfMarkChangedEvent, GetPayloadOfMarkChangedEvent, im, logger)
}
