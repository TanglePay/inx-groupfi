package im

type Message struct {
	// group id
	GroupId []byte
	// OutputId of the Output that store message body payload
	OutputId []byte

	MileStoneIndex uint32

	MileStoneTimestamp uint32
}

func NewMessage(groupId []byte, outputId []byte, mileStoneIndex uint32, mileStoneTimestamp uint32) *Message {
	return &Message{
		GroupId:            groupId,
		OutputId:           outputId,
		MileStoneIndex:     mileStoneIndex,
		MileStoneTimestamp: mileStoneTimestamp,
	}
}

const GroupIdLen = 10
const OutputIdLen = 34
