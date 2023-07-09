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
func NewNFT(groupId []byte, ownerAddress string, nftIdHex string, mileStoneIndex uint32, mileStoneTimestamp uint32) *NFT {
	ownerAddressBytes := []byte(ownerAddress)
	nftId, err := iotago.DecodeHex(nftIdHex)
	if err != nil {
		return nil
	}
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
