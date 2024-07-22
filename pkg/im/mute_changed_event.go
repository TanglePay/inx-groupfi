package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

type MuteChangedEvent struct {
	EventCommonFields
	AddressSha256Hash         [Sha256HashLen]byte
	ActionedAddressSha256Hash [Sha256HashLen]byte
	GroupID                   [GroupIdLen]byte
	IsMuted                   bool
	MilestoneTimestamp        uint32
}

func NewMuteChangedEvent(addressSha256Hash [Sha256HashLen]byte,
	actionedAddressSha256Hash [Sha256HashLen]byte,
	groupID [GroupIdLen]byte, isMuted bool, milestoneTimestamp uint32) *MuteChangedEvent {
	return &MuteChangedEvent{
		AddressSha256Hash:         addressSha256Hash,
		ActionedAddressSha256Hash: actionedAddressSha256Hash,
		GroupID:                   groupID,
		IsMuted:                   isMuted,
		MilestoneTimestamp:        milestoneTimestamp,
	}
}

// implements InboxItem
func (m *MuteChangedEvent) GetToken() []byte {
	return m.Token
}

func (m *MuteChangedEvent) GetEventType() byte {
	return m.EventType
}

func (m *MuteChangedEvent) SetToken(token []byte) {
	m.Token = token
}

func (m *MuteChangedEvent) SetEventType(eventType byte) {
	m.EventType = eventType
}

func (m *MuteChangedEvent) Jsonable() InboxItemJson {
	json := &MuteChangedEventJson{
		GroupID:           iotago.EncodeHex(m.GroupID[:]),
		AddressSha256Hash: iotago.EncodeHex(m.ActionedAddressSha256Hash[:]),
		Timestamp:         m.MilestoneTimestamp,
		IsMuted:           m.IsMuted,
	}
	json.SetEventType(m.EventType)
	return json
}

type MuteChangedEventJson struct {
	EventJsonCommonFields
	GroupID           string `json:"groupId"`
	AddressSha256Hash string `json:"addressHash"`
	Timestamp         uint32 `json:"timestamp"`
	IsMuted           bool   `json:"isMuted"`
}

// implements InboxItemJson
func (m *MuteChangedEventJson) SetEventType(eventType byte) {
	m.EventType = eventType
}

// serialize MuteChangedEvent to bytes
func SerializeMuteChangedEvent(m *MuteChangedEvent) []byte {
	bytes := make([]byte, 0)
	idx := 0

	// add prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImInboxEventTypeMuteChanged}, false)

	AppendBytesWithUint16Len(&bytes, &idx, m.GroupID[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, m.ActionedAddressSha256Hash[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(m.MilestoneTimestamp), false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte{BoolToByte(m.IsMuted)}, false)
	return bytes
}

// unserialize MuteChangedEvent from bytes
func UnserializeMuteChangedEvent(bytes []byte) (*MuteChangedEvent, error) {
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
	isMutedBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	isMuted := BytesToBool(isMutedBytes)
	return NewMuteChangedEvent(addressSha256HashFixed, addressSha256HashFixed, groupIDFixed, isMuted, milestoneTimestamp), nil
}

// implements InboxItem
func (m *MuteChangedEvent) ToPushTopic() []byte {
	return m.AddressSha256Hash[:]
}

// getTopic of MuteChangedEvent
func GetTopicOfMuteChangedEvent(m *MuteChangedEvent) string {
	return iotago.EncodeHex(m.AddressSha256Hash[:])
}

func (m *MuteChangedEvent) ToPushPayload() []byte {
	eventBytes := SerializeMuteChangedEvent(m)
	return append([]byte{ImInboxEventTypeMuteChanged}, eventBytes...)
}

// get payload of MuteChangedEvent
func GetPayloadOfMuteChangedEvent(m *MuteChangedEvent) []byte {
	eventBytes := SerializeMuteChangedEvent(m)
	return eventBytes
}

func getInboxOfMuteChangedEvent(m *MuteChangedEvent) [][]byte {
	return [][]byte{m.AddressSha256Hash[:]}
}

func getEventTypeOfMuteChangedEvent(m *MuteChangedEvent) byte {
	return ImInboxEventTypeMuteChanged
}

// gen and push MuteChangedEvent
func GenAndPushMuteChangedEvent(
	receiverAddressSha256Hash [Sha256HashLen]byte,
	actionedAddressSha256Hash [Sha256HashLen]byte,
	groupId [GroupIdLen]byte,
	isMuted bool,
	im *Manager, logger *logger.Logger) error {
	event := NewMuteChangedEvent(receiverAddressSha256Hash, actionedAddressSha256Hash, groupId, isMuted, CurrentMilestoneTimestamp)
	return PushData(event, GetTopicOfMuteChangedEvent, getInboxOfMuteChangedEvent, getEventTypeOfMuteChangedEvent, GetPayloadOfMuteChangedEvent, im, logger)
}
