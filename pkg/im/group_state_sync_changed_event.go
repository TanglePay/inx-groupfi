package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

type GroupStateSyncChangedEvent struct {
	EventCommonFields
	AddressSha256Hash  [Sha256HashLen]byte
	MilestoneTimestamp uint32
}

func NewGroupStateSyncChangedEvent(
	addressSha256Hash [Sha256HashLen]byte,
	milestoneTimestamp uint32) *GroupStateSyncChangedEvent {
	return &GroupStateSyncChangedEvent{
		AddressSha256Hash:  addressSha256Hash,
		MilestoneTimestamp: milestoneTimestamp,
	}
}

// implements InboxItem
func (m *GroupStateSyncChangedEvent) GetToken() []byte {
	return m.Token
}

func (m *GroupStateSyncChangedEvent) GetEventType() byte {
	return m.EventType
}

func (m *GroupStateSyncChangedEvent) SetToken(token []byte) {
	m.Token = token
}

func (m *GroupStateSyncChangedEvent) SetEventType(eventType byte) {
	m.EventType = eventType
}

func (m *GroupStateSyncChangedEvent) Jsonable() InboxItemJson {
	json := &GroupStateSyncChangedEventJson{
		Timestamp: m.MilestoneTimestamp,
	}
	json.SetEventType(m.EventType)
	return json
}

type GroupStateSyncChangedEventJson struct {
	EventJsonCommonFields
	Timestamp uint32 `json:"timestamp"`
}

// implements InboxItemJson
func (m *GroupStateSyncChangedEventJson) SetEventType(eventType byte) {
	m.EventType = eventType
}

// serialize GroupStateSyncChangedEvent to bytes
func SerializeGroupStateSyncChangedEvent(m *GroupStateSyncChangedEvent) []byte {
	bytes := make([]byte, 0)
	idx := 0

	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImInboxEventTypeGroupStateSync}, false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(m.MilestoneTimestamp), false)
	return bytes
}

// unserialize GroupStateSyncChangedEvent from bytes
func UnserializeGroupStateSyncChangedEvent(bytes []byte) (*GroupStateSyncChangedEvent, error) {
	idx := 0
	_, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}

	milestoneTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := BytesToUint32(milestoneTimestampBytes)

	return NewGroupStateSyncChangedEvent(
		[Sha256HashLen]byte{}, // This will be set later
		milestoneTimestamp,
	), nil
}

// implements InboxItem
func (m *GroupStateSyncChangedEvent) ToPushTopic() []byte {
	return m.AddressSha256Hash[:]
}

func GetTopicOfGroupStateSyncChangedEvent(m *GroupStateSyncChangedEvent) string {
	return iotago.EncodeHex(m.AddressSha256Hash[:])
}

func (m *GroupStateSyncChangedEvent) ToPushPayload() []byte {
	eventBytes := SerializeGroupStateSyncChangedEvent(m)
	return append([]byte{ImInboxEventTypeGroupStateSync}, eventBytes...)
}

func GetPayloadOfGroupStateSyncChangedEvent(m *GroupStateSyncChangedEvent) []byte {
	return SerializeGroupStateSyncChangedEvent(m)
}

func getInboxOfGroupStateSyncChangedEvent(m *GroupStateSyncChangedEvent) [][]byte {
	return [][]byte{m.AddressSha256Hash[:]}
}

func getEventTypeOfGroupStateSyncChangedEvent(m *GroupStateSyncChangedEvent) byte {
	return ImInboxEventTypeGroupStateSync
}

// GenAndPushGroupStateSyncChangedEvent generates and pushes a GroupStateSyncChangedEvent
func GenAndPushGroupStateSyncChangedEvent(
	addressSha256Hash [Sha256HashLen]byte,
	milestoneTimestamp uint32,
	im *Manager,
	logger *logger.Logger) error {
	// log gen
	Logger.Infof("GenAndPushGroupStateSyncChangedEvent gen group state sync changed event: %s", iotago.EncodeHex(addressSha256Hash[:]))
	event := NewGroupStateSyncChangedEvent(
		addressSha256Hash,
		milestoneTimestamp,
	)

	return PushData(
		event,
		GetTopicOfGroupStateSyncChangedEvent,
		getInboxOfGroupStateSyncChangedEvent,
		getEventTypeOfGroupStateSyncChangedEvent,
		GetPayloadOfGroupStateSyncChangedEvent,
		im,
		logger,
	)
}
