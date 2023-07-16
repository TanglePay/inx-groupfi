package im

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

func (im *Manager) NftKeyFromGroupIdAndNftId(groupId []byte, nftId []byte) []byte {
	index := 0
	key := make([]byte, 1+GroupIdLen+NFTIdLen)
	key[index] = ImStoreKeyPrefixNFT
	index++
	copy(key[index:], groupId)
	index += GroupIdLen
	copy(key[index:], nftId)
	return key
}

// nft keyprefix from collection(subgroup) name
func (im *Manager) NftKeyPrefixFromGroupId(groupId []byte) []byte {
	return im.NftKeyFromGroupIdAndNftId(groupId, []byte{})
}
func (im *Manager) storeSingleNFT(nft *NFT, logger *logger.Logger) error {
	logger.Infof("store new nft: groupId:%s, nftId:%s, milestoneindex:%d, milestonetimestamp:%d", nft.GetGroupIdStr(), nft.GetAddressStr(), nft.MileStoneIndex, nft.MileStoneTimestamp)
	key := im.NftKeyFromGroupIdAndNftId(
		nft.GroupId,
		nft.NFTId)
	valuePayload := make([]byte, 4+4+len(nft.OwnerAddress))
	binary.BigEndian.PutUint32(valuePayload, nft.MileStoneIndex)
	binary.BigEndian.PutUint32(valuePayload[4:], nft.MileStoneTimestamp)
	copy(valuePayload[8:], nft.OwnerAddress)
	err := im.imStore.Set(key, valuePayload)
	keyHex := iotago.EncodeHex(key)
	valueHex := iotago.EncodeHex(valuePayload)
	logger.Infof("store nft with key %s, value %s", keyHex, valueHex)
	return err
}

func (im *Manager) storeNewNFTs(nfts []*NFT, logger *logger.Logger) error {

	for _, nft := range nfts {
		if err := im.storeSingleNFT(nft, logger); err != nil {
			return err
		}
	}
	return nil
}

func (im *Manager) ReadNFTFromPrefix(keyPrefix []byte) ([]*NFT, error) {
	var res []*NFT
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		res = append(res, &NFT{
			GroupId:            key[1 : 1+GroupIdLen],
			NFTId:              key[1+GroupIdLen:],
			OwnerAddress:       value[8:],
			MileStoneIndex:     binary.BigEndian.Uint32(value[:4]),
			MileStoneTimestamp: binary.BigEndian.Uint32(value[4:8]),
		})
		return true
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}
