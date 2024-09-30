package im

import (
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

// struct for profile changed event
type ProfileChangedEvent struct {
	EventCommonFields
	AddressSha256Hash [Sha256HashLen]byte
	Timestamp         uint32
}

// implements InboxItem
func (p *ProfileChangedEvent) GetToken() []byte {
	return p.Token
}

func (p *ProfileChangedEvent) GetEventType() byte {
	return p.EventType
}

func (p *ProfileChangedEvent) SetToken(token []byte) {
	p.Token = token
}

func (p *ProfileChangedEvent) SetEventType(eventType byte) {
	p.EventType = eventType
}

func (p *ProfileChangedEvent) Jsonable() InboxItemJson {
	json := &ProfileChangedEventJson{
		AddressSha256Hash: iotago.EncodeHex(p.AddressSha256Hash[:]),
		Timestamp:         p.Timestamp,
	}
	json.SetEventType(p.EventType)
	return json
}

type ProfileChangedEventJson struct {
	EventJsonCommonFields
	AddressSha256Hash string `json:"address"`
	Timestamp         uint32 `json:"timestamp"`
}

// implements InboxItemJson
func (p *ProfileChangedEventJson) SetEventType(eventType byte) {
	p.EventType = eventType
}

// new profile changed event
func NewProfileChangedEvent(addressSha256Hash []byte) *ProfileChangedEvent {
	timestamp := CurrentMilestoneTimestamp
	addressSha256HashFixed := [Sha256HashLen]byte{}
	copy(addressSha256HashFixed[:], addressSha256Hash)
	return &ProfileChangedEvent{
		AddressSha256Hash: addressSha256HashFixed,
		Timestamp:         timestamp,
	}
}

// serialize profile changed event
func SerializeProfileChangedEvent(profileChangedEvent *ProfileChangedEvent, logger *logger.Logger) []byte {
	var bytes []byte
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImInboxKeyPrefixProfileChangedEvent}, false)
	// addressSha256Hash
	AppendBytesWithUint16Len(&bytes, &idx, profileChangedEvent.AddressSha256Hash[:], false)
	// timestamp
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(profileChangedEvent.Timestamp), false)
	return bytes
}

// unserialize profile changed event
func UnserializeProfileChangedEvent(bytes []byte, logger *logger.Logger) (*ProfileChangedEvent, error) {
	idx := 0
	// prefix
	_, err := ReadBytesWithUint16Len(bytes, &idx, 1)
	if err != nil {
		return nil, err
	}
	// addressSha256Hash
	addressSha256Hash, err := ReadBytesWithUint16Len(bytes, &idx, Sha256HashLen)
	if err != nil {
		return nil, err
	}
	var addressSha256HashFixed [Sha256HashLen]byte
	copy(addressSha256HashFixed[:], addressSha256Hash)
	// timestamp
	timestampBytes, err := ReadBytesWithUint16Len(bytes, &idx, 4)
	if err != nil {
		return nil, err
	}
	timestamp := BytesToUint32(timestampBytes)
	return &ProfileChangedEvent{
		AddressSha256Hash: addressSha256HashFixed,
		Timestamp:         timestamp,
	}, nil
}

// get topic of profile changed event
func GetTopicOfProfileChangedEvent(profileChangedEvent *ProfileChangedEvent) string {
	return iotago.EncodeHex(profileChangedEvent.AddressSha256Hash[:])
}

// get payload of profile changed event
func GetPayloadOfProfileChangedEvent(profileChangedEvent *ProfileChangedEvent) []byte {
	eventBytes := SerializeProfileChangedEvent(profileChangedEvent, nil)
	return append([]byte{ImInboxKeyPrefixProfileChangedEvent}, eventBytes...)
}

// get inbox of profile changed event
func getInboxOfProfileChangedEvent(profileChangedEvent *ProfileChangedEvent) [][]byte {
	return [][]byte{profileChangedEvent.AddressSha256Hash[:]}
}

// get event type of profile changed event
func getEventTypeOfProfileChangedEvent(profileChangedEvent *ProfileChangedEvent) byte {
	return ImInboxKeyPrefixProfileChangedEvent
}

// gen and push profile changed event
func GenAndPushProfileChangedEvent(profile *Profile, im *Manager, logger *logger.Logger) error {
	AddressSha256Hash := Sha256HashAddress(profile.Address)
	event := NewProfileChangedEvent(AddressSha256Hash)
	return PushData(event, GetTopicOfProfileChangedEvent, getInboxOfProfileChangedEvent, getEventTypeOfProfileChangedEvent, GetPayloadOfProfileChangedEvent, im, logger)
}
