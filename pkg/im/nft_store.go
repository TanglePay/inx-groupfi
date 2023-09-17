package im

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
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

// given a nft key decompose it into groupId and nftId and just log then
func (im *Manager) NftKeyToGroupIdAndNftId(key []byte, logger *logger.Logger) ([]byte, []byte) {
	index := 0
	groupId := key[index+1 : index+1+GroupIdLen]
	index += 1 + GroupIdLen
	nftId := key[index:]
	logger.Infof("nft key %s, groupId %s, nftId %s", iotago.EncodeHex(key), iotago.EncodeHex(groupId), iotago.EncodeHex(nftId))
	return groupId, nftId
}

// nft keyprefix from collection(subgroup) name
func (im *Manager) NftKeyPrefixFromGroupId(groupId []byte) []byte {
	index := 0
	key := make([]byte, 1+GroupIdLen)
	key[index] = ImStoreKeyPrefixNFT
	index++
	copy(key[index:], groupId)
	return key
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
	if err != nil {
		return err
	}
	var addressGroup *AddressGroup
	if nft.GroupQualifyType == GroupQualifyTypeNft {
		addressGroup = NewAddressGroupNft(nft.OwnerAddress, nft.GroupId, nft.IpfsLink, nft.GroupName)
	} else if nft.GroupQualifyType == GroupQualifyTypeToken {
		addressGroup = NewAddressGroupToken(nft.OwnerAddress, nft.GroupId, nft.TokenType, nft.TokenThres, nft.GroupName)
	} else {
		return errors.New("invalid group qualify type")
	}
	err = im.StoreAddressGroup(addressGroup)
	return err
}

// delete single nft
func (im *Manager) DeleteNFT(nft *NFT) error {
	key := im.NftKeyFromGroupIdAndNftId(
		nft.GroupId,
		nft.NFTId)
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	addressGroup := NewAddressGroup(nft.OwnerAddress, nft.GroupId)
	err = im.DeleteAddressGroup(addressGroup)
	return err
}
func (im *Manager) storeNewNFTsDeleteConsumedNfts(createdNfts []*NFT, consumedNfts []*NFT, logger *logger.Logger) error {
	// hash set store all groupId, groupId is []byte
	groupIdSet := make(map[string]bool)
	defer func() {
		for groupId := range groupIdSet {
			groupIdBytes, err := iotago.DecodeHex(groupId)
			if err != nil {
				logger.Errorf("failed to decode groupId %s", groupId)
				continue
			}
			im.DeleteSharedFromGroupId(groupIdBytes)
			im.CalculateNumberOfGroupMembersWithPublicKey(groupIdBytes, logger)
		}
	}()
	for _, nft := range createdNfts {
		if err := im.storeSingleNFT(nft, logger); err != nil {
			return err
		}
		groupIdSet[iotago.EncodeHex(nft.GroupId)] = true
	}
	for _, nft := range consumedNfts {
		if err := im.DeleteNFT(nft); err != nil {
			return err
		}
		groupIdSet[iotago.EncodeHex(nft.GroupId)] = true
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

// get all nfts from a group
func (im *Manager) ReadNFTsFromGroupId(groupId []byte) ([]*NFT, error) {
	return im.ReadNFTFromPrefix(im.NftKeyPrefixFromGroupId(groupId))
}
