package im

import iotago "github.com/iotaledger/iota.go/v3"

type Message struct {
	// group id
	GroupId []byte
	// OutputId of the Output that store message body payload
	OutputId []byte

	SenderAddressSha256 []byte

	MetaSha256 []byte

	Token []byte

	MileStoneIndex uint32

	MileStoneTimestamp uint32
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
