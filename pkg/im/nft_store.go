package im

import (
	"encoding/binary"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/labstack/gommon/log"
	"github.com/pkg/errors"
	"golang.org/x/crypto/blake2b"
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
		addressGroup = NewAddressGroupToken(nft.OwnerAddress, nft.GroupId, nft.TokenId, nft.TokenThres, nft.GroupName)
	} else {
		return errors.New("invalid group qualify type")
	}

	qualification, err := im.GetQualificationFromNFT(nft)
	if err != nil {
		return err
	}
	// store group qualification
	err = im.StoreGroupQualification(qualification, logger)
	if err != nil {
		return err
	}

	err = im.StoreAddressGroup(addressGroup)
	return err
}

// get qualification from nft
func (im *Manager) GetQualificationFromNFT(nft *NFT) (*GroupQualification, error) {
	var groupId32 [GroupIdLen]byte
	copy(groupId32[:], nft.GroupId)
	var nftId32 [Sha256HashLen]byte
	copy(nftId32[:], nft.NFTId)
	qualification := NewGroupQualification(groupId32, string(nft.OwnerAddress), nftId32, nft.GroupName, nft.GroupQualifyType, nft.IpfsLink)
	return qualification, nil
}

// delete single nft
func (im *Manager) DeleteNFT(nft *NFT, logger *logger.Logger) error {
	key := im.NftKeyFromGroupIdAndNftId(
		nft.GroupId,
		nft.NFTId)
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	addressGroup := NewAddressGroup(nft.OwnerAddress, nft.GroupId)
	qualification, err := im.GetQualificationFromNFT(nft)
	if err != nil {
		return err
	}
	// delete group qualification
	err = im.DeleteGroupQualification(qualification, logger)
	if err != nil {
		return err
	}
	err = im.DeleteAddressGroup(addressGroup)
	return err
}

// check if nft exists, input is group id and address
func (im *Manager) NFTExists(groupId []byte, nftId []byte) (bool, error) {
	key := im.NftKeyFromGroupIdAndNftId(groupId, nftId)
	return im.imStore.Has(key)
}
func (im *Manager) StoreNewNFTsDeleteConsumedNfts(createdNfts []*NFT, consumedNfts []*NFT, logger *logger.Logger) error {
	for _, nft := range createdNfts {
		if err := im.storeSingleNFT(nft, logger); err != nil {
			return err
		}
	}
	for _, nft := range consumedNfts {
		if err := im.DeleteNFT(nft, logger); err != nil {
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

// get all nfts from a group
func (im *Manager) ReadNFTsFromGroupId(groupId []byte) ([]*NFT, error) {
	return im.ReadNFTFromPrefix(im.NftKeyPrefixFromGroupId(groupId))
}

// filter out nft outputs by output type and collectionId
func (im *Manager) FilterNftOutput(outputId []byte, output iotago.Output, mileStoneIndex uint32, milestoneTimestamp uint32,
	logger *logger.Logger) ([]*NFT, bool, []*GroupIdAndGroupNamePair) {
	if output.Type() != iotago.OutputNFT {
		return nil, false, nil
	}
	nftOutput, ok := output.(*iotago.NFTOutput)
	if !ok {
		return nil, false, nil
	}

	featureSet, err := nftOutput.ImmutableFeatures.Set()
	if err != nil {
		return nil, false, nil
	}
	issuer := featureSet.IssuerFeature()
	if issuer == nil {
		return nil, false, nil
	}

	if issuer.Address.Type() != iotago.AddressNFT {
		return nil, false, nil
	}
	nftAddress := issuer.Address.(*iotago.NFTAddress)
	collectionId := nftAddress.NFTID().ToHex()

	pairs := ChainNameAndCollectionIdToGroupIdAndGroupNamePairs(HornetChainName, collectionId)
	if len(pairs) == 0 {
		return nil, false, nil
	}

	// ////////////
	var nftIdHex string
	if nftOutput.NFTID.Empty() {
		outputIdHash := blake2b.Sum256(outputId)
		nftIdHex = iotago.EncodeHex(outputIdHash[:])
	} else {
		nftIdHex = nftOutput.NFTID.ToHex()
	}
	nftId, err := iotago.DecodeHex(nftIdHex)
	if err != nil {
		log.Errorf("nftFromINXOutput failed:%s", err)
		return nil, false, nil
	}

	unlockConditionSet := nftOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	var nfts []*NFT
	for _, pair := range pairs {
		groupId := pair.GroupId
		groupName := pair.GroupName

		nft := NewNFT(groupId, ownerAddress, nftId, groupName, "ipfsLink", mileStoneIndex, milestoneTimestamp)
		nfts = append(nfts, nft)
	}
	return nfts, true, pairs
}
