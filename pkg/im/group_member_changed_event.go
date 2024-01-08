package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

type GroupMemberChangedEvent struct {
	EventCommonFields
	// groupID
	GroupID [GroupIdLen]byte

	Address string

	IsNewMember bool

	// milestone index
	MilestoneIndex uint32
	// milestone timestamp
	MilestoneTimestamp uint32
}

// implements InboxItem
func (g *GroupMemberChangedEvent) GetToken() []byte {
	return g.Token
}
func (g *GroupMemberChangedEvent) GetEventType() byte {
	return g.EventType
}
func (g *GroupMemberChangedEvent) SetToken(token []byte) {
	g.Token = token
}
func (g *GroupMemberChangedEvent) SetEventType(eventType byte) {
	g.EventType = eventType
}
func (g *GroupMemberChangedEvent) Jsonable() InboxItemJson {
	json := &GroupMemberChangedEventJson{
		GroupID:           iotago.EncodeHex(g.GroupID[:]),
		Timestamp:         g.MilestoneTimestamp,
		IsNewMember:       g.IsNewMember,
		AddressSha256Hash: iotago.EncodeHex(g.AddressSha256Hash[:]),
	}
	json.SetEventType(g.EventType)
	return json
}

// implements InboxItem
func (g *GroupMemberChangedEvent) ToPushTopic() []byte {
	return g.GroupID[:]
}
func (g *GroupMemberChangedEvent) ToPushPayload() []byte {
	eventBytes := SerializeGroupMemberChangedEvent(g)
	return append([]byte{ImInboxEventTypeGroupMemberChanged}, eventBytes...)
}

type GroupMemberChangedEventJson struct {
	EventJsonCommonFields
	GroupID     string `json:"groupId"`
	Timestamp   uint32 `json:"timestamp"`
	IsNewMember bool   `json:"isNewMember"`
	Address     string `json:"address"`
}

// implements InboxItemJson
func (g *GroupMemberChangedEventJson) SetEventType(eventType byte) {
	g.EventType = eventType
}

// newGroupMemberChangedEvent creates a new GroupMemberChangedEvent.
func NewGroupMemberChangedEvent(groupID [GroupIdLen]byte, mileStoneIndex uint32, mileStoneTimestamp uint32, isNewMember bool, address string) *GroupMemberChangedEvent {
	return &GroupMemberChangedEvent{
		GroupID:            groupID,
		MilestoneIndex:     mileStoneIndex,
		MilestoneTimestamp: mileStoneTimestamp,
		IsNewMember:        isNewMember,
		Address:            address,
	}
}

// inbox
// inbox key from groupmemberchangedevent
func (im *Manager) InboxKeyFromGroupMemberChangedEvent(receiverAddressSha256 []byte, groupMemberChangedEvent *GroupMemberChangedEvent) []byte {
	contentBytes := SerializeGroupMemberChangedEvent(groupMemberChangedEvent)
	return im.InboxKeyFromValues(receiverAddressSha256, groupMemberChangedEvent.MilestoneIndex, groupMemberChangedEvent.MilestoneTimestamp, Sha256HashBytes(contentBytes), ImInboxEventTypeGroupMemberChanged)
}

// serialize group member changed event
func SerializeGroupMemberChangedEvent(groupMemberChangedEvent *GroupMemberChangedEvent) []byte {
	// using func AppendBytesWithUint16Len(bytes *[]byte, idx *int, slice []byte, appendLength bool) {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, groupMemberChangedEvent.GroupID[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(groupMemberChangedEvent.MilestoneIndex), false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(groupMemberChangedEvent.MilestoneTimestamp), false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte{BoolToByte(groupMemberChangedEvent.IsNewMember)}, false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(groupMemberChangedEvent.Address), true)
	return bytes
}

// unserialize group member changed event
func (im *Manager) UnserializeGroupMemberChangedEvent(bytes []byte) (*GroupMemberChangedEvent, error) {
	idx := 0
	groupIDBytes, err := ReadBytesWithUint16Len(bytes, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	groupID := [GroupIdLen]byte{}
	copy(groupID[:], groupIDBytes)
	milestoneIndexBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneIndex := BytesToUint32(milestoneIndexBytes)
	milestoneTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := BytesToUint32(milestoneTimestampBytes)
	isNewMemberBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	isNewMember := BytesToBool(isNewMemberBytes)
	addressBytes, err := ReadBytesWithUint16Len(bytes, &idx)
	if err != nil {
		return nil, err
	}
	address := string(addressBytes)
	return NewGroupMemberChangedEvent(groupID, milestoneIndex, milestoneTimestamp, isNewMember, address), nil
}

// store group member changed event to inbox
func (im *Manager) StoreGroupMemberChangedEventToInbox(receiverAddressSha256 []byte, groupMemberChangedEvent *GroupMemberChangedEvent, logger *logger.Logger) error {
	// serialize group member changed event
	bytes := SerializeGroupMemberChangedEvent(groupMemberChangedEvent)
	// get inbox key
	key := im.InboxKeyFromGroupMemberChangedEvent(receiverAddressSha256, groupMemberChangedEvent)
	// store to inbox
	err := im.imStore.Set(key, bytes)
	if err != nil {
		return err
	}
	return nil
}
