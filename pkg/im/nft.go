package im

import iotago "github.com/iotaledger/iota.go/v3"

type NFT struct {
	// group id
	GroupId []byte

	NFTId []byte

	OwnerAddress   []byte
	MileStoneIndex uint32

	MileStoneTimestamp uint32
}

// newNFT creates a new NFT.
func NewNFT(groupId []byte, ownerAddress string, nftId []byte, mileStoneIndex uint32, mileStoneTimestamp uint32) *NFT {
	ownerAddressBytes := []byte(ownerAddress)

	return &NFT{
		GroupId:            groupId,
		OwnerAddress:       ownerAddressBytes,
		NFTId:              nftId,
		MileStoneIndex:     mileStoneIndex,
		MileStoneTimestamp: mileStoneTimestamp,
	}
}

func (n *NFT) GetGroupIdStr() string {
	return iotago.EncodeHex(n.GroupId)
}
func (n *NFT) GetAddressStr() string {
	return iotago.EncodeHex(n.OwnerAddress)
}
