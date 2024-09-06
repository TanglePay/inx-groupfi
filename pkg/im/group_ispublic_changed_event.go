package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

type GroupIsPublicChangedEvent struct {
	EventCommonFields
	// groupID
	GroupID [GroupIdLen]byte

	IsPublic bool

	// milestone index
	MilestoneIndex uint32
	// milestone timestamp
	MilestoneTimestamp uint32
}

// implements InboxItem
func (g *GroupIsPublicChangedEvent) GetToken() []byte {
	return g.Token
}
func (g *GroupIsPublicChangedEvent) GetEventType() byte {
	return g.EventType
}
func (g *GroupIsPublicChangedEvent) SetToken(token []byte) {
	g.Token = token
}
func (g *GroupIsPublicChangedEvent) SetEventType(eventType byte) {
	g.EventType = eventType
}
func (g *GroupIsPublicChangedEvent) Jsonable() InboxItemJson {
	json := &GroupIsPublicChangedEventJson{
		GroupID:   iotago.EncodeHex(g.GroupID[:]),
		Timestamp: g.MilestoneTimestamp,
		IsPublic:  g.IsPublic,
	}
	json.SetEventType(g.EventType)
	return json
}

type GroupIsPublicChangedEventJson struct {
	EventJsonCommonFields
	GroupID   string `json:"groupId"`
	Timestamp uint32 `json:"timestamp"`
	IsPublic  bool   `json:"isPublic"`
}

// implements InboxItemJson
func (g *GroupIsPublicChangedEventJson) SetEventType(eventType byte) {
	g.EventType = eventType
}

// newGroupIsPublicChangedEvent creates a new GroupIsPublicChangedEvent.
func NewGroupIsPublicChangedEvent(groupID [GroupIdLen]byte, mileStoneIndex uint32, mileStoneTimestamp uint32, isPublic bool) *GroupIsPublicChangedEvent {
	return &GroupIsPublicChangedEvent{
		GroupID:            groupID,
		MilestoneIndex:     mileStoneIndex,
		MilestoneTimestamp: mileStoneTimestamp,
		IsPublic:           isPublic,
	}
}

// get key of GroupIsPublicChangedEvent
func GetTopicOfGroupIsPublicChangedEvent(groupIsPublicChangedEvent *GroupIsPublicChangedEvent) string {
	return iotago.EncodeHex(groupIsPublicChangedEvent.GroupID[:])
}

// get payload of GroupIsPublicChangedEvent
func GetPayloadOfGroupIsPublicChangedEvent(groupIsPublicChangedEvent *GroupIsPublicChangedEvent) []byte {
	eventBytes := SerializeGroupIsPublicChangedEvent(groupIsPublicChangedEvent, Logger)
	return eventBytes
}

// getInbox func(*T) []byte, getEventType func(*T) byte,
func getInboxOfGroupIsPublicChangedEvent(groupIsPublicChangedEvent *GroupIsPublicChangedEvent) [][]byte {
	// get group members
	groupMembers, err := Im.GetGroupMembers(groupIsPublicChangedEvent.GroupID)
	if err != nil {
		// log error
		Logger.Errorf("getInboxOfGroupIsPublicChangedEvent GetGroupMembers error %v", err)
		return nil
	}
	var keys [][]byte
	// loop group members
	for _, groupMember := range groupMembers {
		gaddress := groupMember.Address
		gaddressSha256Hash := Sha256HashAddress(gaddress)
		keys = append(keys, gaddressSha256Hash)
	}

	return keys
}

func getEventTypeOfGroupIsPublicChangedEvent(groupIsPublicChangedEvent *GroupIsPublicChangedEvent) byte {
	return ImInboxEventTypeGroupIsPublicChanged
}

// push event
func GenAndPushGroupIsPublicChangedEvent(groupID [GroupIdLen]byte, isPublic bool, im *Manager, logger *logger.Logger) error {
	// get group is public changed event
	groupIsPublicChangedEvent := NewGroupIsPublicChangedEvent(groupID, CurrentMilestoneIndex, CurrentMilestoneTimestamp, isPublic)
	// push event
	return PushData(groupIsPublicChangedEvent, GetTopicOfGroupIsPublicChangedEvent, getInboxOfGroupIsPublicChangedEvent, getEventTypeOfGroupIsPublicChangedEvent,
		GetPayloadOfGroupIsPublicChangedEvent, im, logger)
}

// inbox
// inbox key from groupispublicchangedevent
func (im *Manager) InboxKeyFromGroupIsPublicChangedEvent(receiverAddressSha256 []byte, groupIsPublicChangedEvent *GroupIsPublicChangedEvent,
	contentHash []byte) []byte {
	return im.InboxKeyFromValues(receiverAddressSha256, groupIsPublicChangedEvent.MilestoneIndex, groupIsPublicChangedEvent.MilestoneTimestamp, contentHash, ImInboxEventTypeGroupIsPublicChanged)
}

// serialize group is public changed event
func SerializeGroupIsPublicChangedEvent(groupIsPublicChangedEvent *GroupIsPublicChangedEvent, logger *logger.Logger) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// add prefix ImInboxEventTypeGroupIsPublicChanged
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImInboxEventTypeGroupIsPublicChanged}, false)
	AppendBytesWithUint16Len(&bytes, &idx, groupIsPublicChangedEvent.GroupID[:], false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(groupIsPublicChangedEvent.MilestoneIndex), false)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(groupIsPublicChangedEvent.MilestoneTimestamp), false)
	AppendBytesWithUint16Len(&bytes, &idx, []byte{BoolToByte(groupIsPublicChangedEvent.IsPublic)}, false)
	return bytes
}

// unserialize group is public changed event
func (im *Manager) UnserializeGroupIsPublicChangedEvent(bytes []byte, logger *logger.Logger) (*GroupIsPublicChangedEvent, error) {
	// log bytes
	logger.Infof("UnserializeGroupIsPublicChangedEvent bytes: %s", iotago.EncodeHex(bytes))
	idx := 0
	// prefix
	_, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	groupIDBytes, err := ReadBytesWithUint16Len(bytes, &idx, GroupIdLen)
	if err != nil {
		return nil, err
	}
	groupID := [GroupIdLen]byte{}
	copy(groupID[:], groupIDBytes)
	// log groupID
	logger.Infof("UnserializeGroupIsPublicChangedEvent groupID: %s", iotago.EncodeHex(groupID[:]))
	milestoneIndexBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneIndex := BytesToUint32(milestoneIndexBytes)
	// log milestoneIndex
	logger.Infof("UnserializeGroupIsPublicChangedEvent milestoneIndex: %d", milestoneIndex)
	milestoneTimestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	milestoneTimestamp := BytesToUint32(milestoneTimestampBytes)
	// log milestoneTimestamp
	logger.Infof("UnserializeGroupIsPublicChangedEvent milestoneTimestamp: %d", milestoneTimestamp)
	isPublicBytes, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	isPublic := BytesToBool(isPublicBytes)
	// log isPublic
	logger.Infof("UnserializeGroupIsPublicChangedEvent isPublic: %v", isPublic)
	return NewGroupIsPublicChangedEvent(groupID, milestoneIndex, milestoneTimestamp, isPublic), nil
}

// store group is public changed event to inbox
func (im *Manager) StoreGroupIsPublicChangedEventToInbox(receiverAddressSha256 []byte, groupIsPublicChangedEvent *GroupIsPublicChangedEvent, logger *logger.Logger) error {
	// serialize group is public changed event
	bytes := SerializeGroupIsPublicChangedEvent(groupIsPublicChangedEvent, logger)
	hashOfBytes := Sha256HashBytes(bytes)
	// get inbox key
	key := im.InboxKeyFromGroupIsPublicChangedEvent(receiverAddressSha256, groupIsPublicChangedEvent, hashOfBytes)
	// store to inbox
	err := im.imStore.Set(key, bytes)
	if err != nil {
		return err
	}
	return nil
}
