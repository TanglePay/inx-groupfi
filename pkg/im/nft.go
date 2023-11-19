package im

import iotago "github.com/iotaledger/iota.go/v3"

type NFT struct {
	// group id
	GroupId []byte

	NFTId []byte

	OwnerAddress   []byte
	MileStoneIndex uint32

	MileStoneTimestamp uint32
	GroupName          string
	IpfsLink           string
	GroupQualifyType   int
	TokenType          uint16
	TokenThres         string
}

// newNFT creates a new NFT.
func NewNFT(groupId []byte, ownerAddress string, nftId []byte, groupName string, ipfsLink string, mileStoneIndex uint32, mileStoneTimestamp uint32) *NFT {
	ownerAddressBytes := []byte(ownerAddress)

	return &NFT{
		GroupId:            groupId,
		GroupName:          groupName,
		IpfsLink:           ipfsLink,
		GroupQualifyType:   GroupQualifyTypeNft,
		OwnerAddress:       ownerAddressBytes,
		NFTId:              nftId,
		MileStoneIndex:     mileStoneIndex,
		MileStoneTimestamp: mileStoneTimestamp,
	}
}
func NewNFTForToken(groupId []byte, ownerAddress string, nftId []byte, groupName string, tokenType uint16, tokenThres string) *NFT {
	ownerAddressBytes := []byte(ownerAddress)

	return &NFT{
		GroupId:          groupId,
		GroupName:        groupName,
		GroupQualifyType: GroupQualifyTypeToken,
		OwnerAddress:     ownerAddressBytes,
		NFTId:            nftId,
		TokenType:        tokenType,
		TokenThres:       tokenThres,
	}
}
func (n *NFT) GetGroupIdStr() string {
	return iotago.EncodeHex(n.GroupId)
}

func (n *NFT) GetAddressStr() string {
	return iotago.EncodeHex(n.OwnerAddress)
}
