package im

import iotago "github.com/iotaledger/iota.go/v3"

type Message struct {
	EventCommonFields
	// group id
	GroupId []byte
	// OutputId of the Output that store message body payload
	OutputId []byte

	SenderAddressSha256 []byte

	MetaSha256 []byte

	MileStoneIndex uint32

	MileStoneTimestamp uint32
}
type MessageJson struct {
	EventJsonCommonFields
	GroupId   string `json:"groupId"`
	OutputId  string `json:"outputId"`
	Timestamp uint32 `json:"timestamp"`
}

// implements InboxItemJson
func (m *MessageJson) SetEventType(eventType byte) {
	m.EventType = eventType
}

// implements InboxItem
func (m *Message) GetToken() []byte {
	return m.Token
}
func (m *Message) GetEventType() byte {
	return m.EventType
}
func (m *Message) SetToken(token []byte) {
	m.Token = token
}
func (m *Message) SetEventType(eventType byte) {
	m.EventType = eventType
}
func (m *Message) Jsonable() InboxItemJson {
	json := &MessageJson{
		GroupId:   iotago.EncodeHex(m.GroupId),
		OutputId:  iotago.EncodeHex(m.OutputId),
		Timestamp: m.MileStoneTimestamp,
	}
	json.SetEventType(m.EventType)
	return json
}

func NewMessage(groupId []byte, outputId []byte, mileStoneIndex uint32, mileStoneTimestamp uint32, senderAddressSha256 []byte, metaSha256 []byte) *Message {
	return &Message{
		GroupId:             groupId,
		OutputId:            outputId,
		MileStoneIndex:      mileStoneIndex,
		MileStoneTimestamp:  mileStoneTimestamp,
		SenderAddressSha256: senderAddressSha256,
		MetaSha256:          metaSha256,
	}
}

func (m *Message) GetGroupIdStr() string {
	return iotago.EncodeHex(m.GroupId)
}
func (m *Message) GetOutputIdStr() string {
	return iotago.EncodeHex(m.OutputId)
}

const GroupIdLen = 32
const Sha256HashLen = 32
const OutputIdLen = 34
const NFTIdLen = 32
const TimestampLen = 4
