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
		GroupID:     iotago.EncodeHex(g.GroupID[:]),
		Timestamp:   g.MilestoneTimestamp,
		IsNewMember: g.IsNewMember,
		Address:     g.Address,
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

// get key of GroupMemberChangedEvent
func GetTopicOfGroupMemberChangedEvent(groupMemberChangedEvent *GroupMemberChangedEvent) string {
	return iotago.EncodeHex(groupMemberChangedEvent.GroupID[:])
}

// get payload of GroupMemberChangedEvent
func GetPayloadOfGroupMemberChangedEvent(groupMemberChangedEvent *GroupMemberChangedEvent) []byte {
	eventBytes := SerializeGroupMemberChangedEvent(groupMemberChangedEvent)
	return append([]byte{ImInboxEventTypeGroupMemberChanged}, eventBytes...)
}

// getInbox func(*T) []byte, getEventType func(*T) byte,
func getInboxOfGroupMemberChangedEvent(groupMemberChangedEvent *GroupMemberChangedEvent) []byte {
	return Sha256Hash(groupMemberChangedEvent.Address)
}

func getEventTypeOfGroupMemberChangedEvent(groupMemberChangedEvent *GroupMemberChangedEvent) byte {
	return ImInboxEventTypeGroupMemberChanged
}

// push event
func GenAndPushGroupMemberChangedEvent(groupMember *GroupMember, isNewMember bool, im *Manager, logger *logger.Logger) error {
	// get group member changed event
	groupMemberChangedEvent := NewGroupMemberChangedEvent(groupMember.GroupId, groupMember.MilestoneIndex, groupMember.Timestamp, isNewMember, groupMember.Address)
	// push event
	return PushData(groupMemberChangedEvent, GetTopicOfGroupMemberChangedEvent, getInboxOfGroupMemberChangedEvent, getEventTypeOfGroupMemberChangedEvent,
		GetPayloadOfGroupMemberChangedEvent, im, logger)
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
func (im *Manager) UnserializeGroupMemberChangedEvent(bytes []byte, logger *logger.Logger) (*GroupMemberChangedEvent, error) {
	idx := 0
	groupIDBytes, err := ReadBytesWithUint16Len(bytes, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	groupID := [GroupIdLen]byte{}
	copy(groupID[:], groupIDBytes)
	// log groupID
	logger.Infof("UnserializeGroupMemberChangedEvent groupID: %s", iotago.EncodeHex(groupID[:]))
	milestoneIndexBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneIndex := BytesToUint32(milestoneIndexBytes)
	// log milestoneIndex
	logger.Infof("UnserializeGroupMemberChangedEvent milestoneIndex: %d", milestoneIndex)
	milestoneTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := BytesToUint32(milestoneTimestampBytes)
	// log milestoneTimestamp
	logger.Infof("UnserializeGroupMemberChangedEvent milestoneTimestamp: %d", milestoneTimestamp)
	isNewMemberBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	isNewMember := BytesToBool(isNewMemberBytes)
	// log isNewMember
	logger.Infof("UnserializeGroupMemberChangedEvent isNewMember: %v", isNewMember)
	addressBytes, err := ReadBytesWithUint16Len(bytes, &idx)
	if err != nil {
		return nil, err
	}
	address := string(addressBytes)
	// log address
	logger.Infof("UnserializeGroupMemberChangedEvent address: %s", address)
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
