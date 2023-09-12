package im

import (
	"crypto/sha256"
	"encoding/json"
	"sort"
	"time"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

var (
	PublicKeyDrainer *ItemDrainer
)

const IcebergGroup = "iceberg"
const IcebergCollectionConfigIssuerAddress = "smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp"

func IssuerBech32AddressToGroupIdAndGroupName(address string) ([]byte, string) {
	iceberg := map[string]string{
		"smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp": "iceberg-collection-1",
		"smr1zpz3430fdn4zmheenyjvughsu44ykjzu5st6hg2rp609eevz6czlye60pe7": "iceberg-collection-2",
		"smr1zpqndszdf0p9qy04kq7un5clgzptclqeyv70av5q8thjgxcmk2wfy7pspe5": "iceberg-collection-3",
		"smr1zp46qmajxu0vxc2l73tx0g2j7u579jdzgeplfcnwq6ef0hh3a8zt7wnx9dv": "iceberg-collection-4",
		"smr1zqa6juwmk7lad4rxsddqeprrz60zksdd0k3xa37lelthzsxsal6vjygkl9e": "iceberg-collection-5",
		"smr1zptkmnyuxxvk2qv8exqmyxcytmcf74j3t4apc3hfg4h6n9pnfun5q26j6w4": "iceberg-collection-6",
		"smr1zr8s7kv070hr0zcrjp40fhjgqv9uvzpgx80u7emnp0ncpgchmxpx25paqmf": "iceberg-collection-7",
		"smr1zpvjkgxkzrhyvxy5nh20j6wm0l7grkf5s6l7r2mrhyspvx9khcaysmam589": "iceberg-collection-8",
	}
	groupName := iceberg[address]
	groupId := GroupNameToGroupId(groupName)

	return groupId, groupName
}

func GroupNameToGroupMeta(group string) map[string]string {
	m := map[string]map[string]string{
		"iceberg": {
			"groupName":     "iceberg",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-1": {
			"groupName":     "iceberg-collection-1",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-2": {
			"groupName":     "iceberg-collection-2",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-3": {
			"groupName":     "iceberg-collection-3",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-4": {
			"groupName":     "iceberg-collection-4",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-5": {
			"groupName":     "iceberg-collection-5",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-6": {
			"groupName":     "iceberg-collection-6",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-7": {
			"groupName":     "iceberg-collection-7",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"iceberg-collection-8": {
			"groupName":     "iceberg-collection-8",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
		"smr-whale": {
			"groupName":     "smr-whale",
			"schemaVersion": "1",
			"messageType":   "1",
			"authScheme":    "2",
		},
	}
	return m[group]
}
func sortAndSha256Map(m map[string]string) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedMap := make(map[string]string)
	for _, k := range keys {
		sortedMap[k] = m[k]
	}
	b, _ := json.Marshal(sortedMap)
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}
func GroupNameToGroupId(group string) []byte {
	return sortAndSha256Map(GroupNameToGroupMeta(group))
}

func (im *Manager) GroupNameToGroupId(group string) []byte {
	return sortAndSha256Map(GroupNameToGroupMeta(group))
}

// parse group config nft to renterName name and ipfs link
func (im *Manager) ParseGroupConfigNFT(nftOutput *iotago.NFTOutput) (string, string, error) {
	// get meta outof nft
	featureSet, err := nftOutput.ImmutableFeatures.Set()
	if err != nil {
		return "", "", err
	}
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return "", "", nil
	}
	// meta is json string in bytes, parse it to map
	metaMap := make(map[string]string)
	err = json.Unmarshal(meta.Data, &metaMap)
	if err != nil {
		return "", "", err
	}
	// name -> group name, uri -> ipfs link
	renterName := metaMap["name"]
	ipfsLink := metaMap["uri"]
	return renterName, ipfsLink, nil
}

type MessageGroupMetaJSON struct {
	GroupName     string `json:"groupName"`
	GroupType     int    `json:"groupType"`
	IssuerAddress string `json:"issuerAddress"`
	SchemaVersion int    `json:"schemaVersion"`
	MessageType   int    `json:"messageType"`
	AuthScheme    int    `json:"authScheme"`
}

// handle group config nft created
func (im *Manager) HandleGroupNFTOutputCreated(nftOutput *iotago.NFTOutput, logger *logger.Logger) error {
	// parse name and ipfs link
	renterName, ipfsLink, err := im.ParseGroupConfigNFT(nftOutput)
	if err != nil {
		return err
	}
	// get content from ipfs
	contentRaw, err := ReadIpfsFile(ipfsLink)
	if err != nil {
		return err
	}
	// unmarshal content(json) to MessageGroupMetaJSON[]
	var messageGroupMetaList []MessageGroupMetaJSON
	err = json.Unmarshal(contentRaw, &messageGroupMetaList)
	if err != nil {
		return err
	}
	// store all group config
	for _, messageGroupMeta := range messageGroupMetaList {
		err = im.StoreOneGroupConfig(renterName, messageGroupMeta)
		if err != nil {
			// log error then continue
			logger.Infof("HandleGroupNFTOutputCreated ... StoreOneGroupConfig failed:%s", err)
			continue
		}
	}
	return nil
}
func GroupConfigKeyFromRenterNameAndGroupName(renterName string, groupName string) []byte {
	renterNameSha256 := Sha256Hash(renterName)
	groupNameSha256 := Sha256Hash(groupName)
	return ConcatByteSlices([]byte{ImStoreKeyPrefixGroupConfig}, renterNameSha256, groupNameSha256)
}

// store one group config (group name, MessageGroupMetaJSON)
func (im *Manager) StoreOneGroupConfig(renterName string, messageGroupMeta MessageGroupMetaJSON) error {
	// key = prefix + renterNameSha256 + groupNameSha256
	key := GroupConfigKeyFromRenterNameAndGroupName(renterName, messageGroupMeta.GroupName)
	// value = MessageGroupMetaJSON to bytes
	value, err := json.Marshal(messageGroupMeta)
	if err != nil {
		return err
	}
	// store
	err = im.imStore.Set(key, value)
	return err
}

// read one group config (renterName, groupName)
func (im *Manager) ReadOneGroupConfig(renterName string, groupName string) (*MessageGroupMetaJSON, error) {
	// key = prefix + renterNameSha256 + groupNameSha256
	key := GroupConfigKeyFromRenterNameAndGroupName(renterName, groupName)
	// read
	value, err := im.imStore.Get(key)
	if err != nil {
		return nil, err
	}
	// value to MessageGroupMetaJSON
	var messageGroupMeta MessageGroupMetaJSON
	err = json.Unmarshal(value, &messageGroupMeta)
	if err != nil {
		return nil, err
	}
	return &messageGroupMeta, nil
}
func GroupConfigKeyPrefixFromRenterName(renterName string) []byte {
	renterNameSha256 := Sha256Hash(renterName)
	return ConcatByteSlices([]byte{ImStoreKeyPrefixGroupConfig}, renterNameSha256)
}

// read all group config (renterName)
func (im *Manager) ReadAllGroupConfigForRenter(renterName string) ([]*MessageGroupMetaJSON, error) {
	// key = prefix + renterNameSha256
	keyPrefix := GroupConfigKeyPrefixFromRenterName(renterName)
	// read
	var res []*MessageGroupMetaJSON
	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		// value to MessageGroupMetaJSON
		var messageGroupMeta MessageGroupMetaJSON
		err := json.Unmarshal(value, &messageGroupMeta)
		if err != nil {
			return true
		}
		res = append(res, &messageGroupMeta)
		return true
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// handle group config nft consumed
func (im *Manager) HandleGroupNFTOutputConsumed(nftOutput *iotago.NFTOutput, logger *logger.Logger) error {

	return nil
}

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
