package im

import (
	"time"

	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

var (
	PublicKeyDrainer *ItemDrainer
)

// get key for group publickey count, prefix + groupId
func GroupPublicKeyCountKeyFromGroupId(groupId []byte) []byte {
	return ConcatByteSlices([]byte{ImStoreKeyPrefixGroupPublicKeyCount}, groupId)
}

// store group publickey count
func (im *Manager) StoreGroupPublicKeyCount(groupId []byte, count int) error {
	key := GroupPublicKeyCountKeyFromGroupId(groupId)
	// cast to uint16 then to bytes
	value := Uint16ToBytes(uint16(count))
	return im.imStore.Set(key, value)
}

// get group publickey count
func (im *Manager) GetGroupPublicKeyCount(groupId []byte) (int, error) {
	key := GroupPublicKeyCountKeyFromGroupId(groupId)
	value, err := im.imStore.Get(key)
	if err != nil {
		return 0, err
	}
	// cast bytes to uint16
	count := BytesToUint16(value)
	return int(count), nil
}

// calculate number of group members with public key then store
func (imm *Manager) CalculateNumberOfGroupMembersWithPublicKey(groupId []byte, logger *logger.Logger) error {
	nfts, err := imm.GetRawNFTsFromGroupIdImpl(groupId, logger)
	if err != nil {
		return err
	}
	resp, err := imm.GetNFTsWithPublicKeyFromGroupIdImpl(nfts, PublicKeyDrainer, logger)
	if err != nil {
		return err
	}
	ct := 0
	for _, nft := range resp {
		if nft.PublicKey != "" {
			ct++
		}
	}
	// log
	logger.Infof("CalculateNumberOfGroupMembersWithPublicKey ... groupId:%s, count:%d", iotago.EncodeHex(groupId), ct)
	err = imm.StoreGroupPublicKeyCount(groupId, ct)
	if err != nil {
		return err
	}
	return nil
}

func (im *Manager) GetNFTsWithPublicKeyFromGroupIdImpl(nfts []*NFT, drainer *ItemDrainer, logger *logger.Logger) ([]*NFTResponse, error) {
	// make respChan as chan[NFTResponse]
	respChan := make(chan interface{})
	// wrap nfts to {nft *im.NFT, respChan chan interface{}} and drain
	nftsInterface := make([]interface{}, len(nfts))
	for i, nft := range nfts {
		nftsInterface[i] = &NFTWithRespChan{
			NFT:      nft,
			RespChan: respChan,
		}
	}
	if drainer != nil {
		drainer.Drain(nftsInterface)
	} else {
		// log
		logger.Warnf("GetNFTsWithPublicKeyFromGroupIdImpl drainer is nil")
	}
	// make nftResponseArr, start empty, then fill it with respChan, plus timeout, then return
	nftResponseArr := make([]*NFTResponse, 0)
	timeout := time.After(10 * time.Second)
	for {
		select {
		case resp := <-respChan:
			nftResponseArr = append(nftResponseArr, resp.(*NFTResponse))
			if len(nftResponseArr) == len(nfts) {
				return nftResponseArr, nil
			}
		case <-timeout:
			// log
			logger.Warnf("getNFTsWithPublicKeyFromGroupId timeout")
			return nftResponseArr, nil
		}
	}
}
func (im *Manager) GetRawNFTsFromGroupIdImpl(groupId []byte, logger *logger.Logger) ([]*NFT, error) {
	keyPrefix := im.NftKeyPrefixFromGroupId(groupId)
	logger.Infof("get nfts from groupid:%s, with prefix:%s", iotago.EncodeHex(groupId), iotago.EncodeHex(keyPrefix))
	nfts, err := im.ReadNFTFromPrefix(keyPrefix)
	if err != nil {
		return nil, err
	}
	logger.Infof("get nfts from groupId:%s,found nfts:%d", iotago.EncodeHex(groupId), len(nfts))
	return nfts, nil
}

type NFTResponse struct {
	PublicKey    string `json:"publicKey"`
	OwnerAddress string `json:"ownerAddress"`
	NFTId        string `json:"nftId"`
}

// NFTWithRespChan
type NFTWithRespChan struct {
	NFT      *NFT
	RespChan chan interface{}
}
