package im

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

const IcebergGroup = "iceberg"
const IcebergCollectionConfigIssuerAddress = "smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp"

type GroupIdAndGroupNamePair struct {
	GroupId   []byte
	GroupName string
}

// get dapp groupId from groupId and group meta
// dappGroupId = 'groupfi'+ groupNamespacestriped + keccak256(groupId)
func GetDappGroupId(groupIdHex string, groupMeta *MessageGroupMetaJSON) string {
	var name string
	if groupMeta.QualifyType == "token" {
		name = groupMeta.Symbol
	} else if groupMeta.QualifyType == "nft" {
		name = groupMeta.CollectionName
	}
	// strip white space and tab
	name = strings.ReplaceAll(name, " ", "")
	groupId, err := iotago.DecodeHex(groupIdHex)
	if err != nil {
		return ""
	}
	groupIdShortHash := SHA256HashBytesReturnString(groupId)
	return "groupfi" + name + groupIdShortHash
}
func ChainIdAndCollectionIdToGroupIdAndGroupNamePairs(chainId uint32, contractAddress string, im *Manager) []*GroupIdAndGroupNamePair {
	var res []*GroupIdAndGroupNamePair
	IterateAllGroupIdFromChainIdAndQualifyType(chainId, contractAddress, im, func(groupId [GroupIdLen]byte) bool {
		groupConfigMeta, err := ReadGroupConfigMetaFromGroupId(groupId, im)
		if err != nil {
			// log error then continue
			Logger.Infof("ChainIdAndCollectionIdToGroupIdAndGroupNamePairs ... ReadGroupConfigMetaFromGroupId failed:%s", err)
			return true
		}
		collectionIdInGroupConfig := groupConfigMeta.ContractAddress
		if collectionIdInGroupConfig == contractAddress {
			res = append(res, &GroupIdAndGroupNamePair{
				GroupId:   groupId[:],
				GroupName: groupConfigMeta.GroupName,
			})
		}
		return true
	})
	return res
}

func sortAndSha256Map(m map[string]string) []byte {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	sortedMap := make(map[string]string)
	for _, k := range keys {
		// if value is empty, skip
		if m[k] == "" {
			continue
		}
		sortedMap[k] = m[k]
	}
	b, _ := json.Marshal(sortedMap)
	h := sha256.New()
	h.Write(b)
	return h.Sum(nil)
}

// groupId to group config
func (im *Manager) GroupIdToGroupConfig(groupIdHex string) *MessageGroupMetaJSON {
	groupIdBytes, err := iotago.DecodeHex(groupIdHex)
	if err != nil {
		return nil
	}
	var groupIdFixed [GroupIdLen]byte
	copy(groupIdFixed[:], groupIdBytes)
	groupConfig, err := ReadGroupConfigMetaFromGroupId(groupIdFixed, im)
	if err != nil {
		return nil
	}
	return groupConfig
}

// parse group config nft to renterName name and ipfs link
func (im *Manager) ParseGroupConfigNFT(nftOutput *iotago.NFTOutput) (string, error) {
	// get meta outof nft
	featureSet, err := nftOutput.ImmutableFeatures.Set()
	if err != nil {
		return "", err
	}
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return "", err
	}
	// meta is json string in bytes, parse it to map
	metaMap := make(map[string]interface{})
	err = json.Unmarshal(meta.Data, &metaMap)
	if err != nil {
		return "", err
	}
	// name -> group name, uri -> ipfs link

	ipfsLink := metaMap["uri"].(string)
	return ipfsLink, nil
}

// add extraChains?: {chainId:number,contractAddress:string}[]
type ExtraChain struct {
	ChainId         uint32 `json:"chainId"`
	ContractAddress string `json:"contractAddress"`
}

// customField {key:string,value:string}
type CustomField struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type MessageGroupMetaJSON struct {
	ChainId         uint32        `json:"chainId"`
	SchemaVersion   uint16        `json:"schemaVersion"`
	MessageType     uint8         `json:"messageType"`
	AuthScheme      uint8         `json:"authScheme"`
	QualifyType     string        `json:"qualifyType"`
	ContractAddress string        `json:"contractAddress"`
	GroupName       string        `json:"groupName"`
	TokenThres      string        `json:"tokenThres"`
	TokenDecimals   string        `json:"tokenDecimals"`
	TokenThresValue string        `json:"tokenThresValue"`
	CollectionName  string        `json:"collectionName"`
	Symbol          string        `json:"symbol"`
	ExtraChains     []*ExtraChain `json:"extraChains"`
	Icon            string        `json:"icon"`
	CustomFields    []CustomField `json:"customFields"`
	DappGroupId     string        `json:"dappGroupId"`
}

// struct for MessageGroupMetaJSON plus isPublic
type MessageGroupMetaJSONPlus struct {
	MessageGroupMetaJSON
	IsPublic bool `json:"isPublic"`
}

// checkGroupExists
func (im *Manager) CheckGroupExists(groupIdFixed [GroupIdLen]byte) bool {
	isExist := IsGroupExists(groupIdFixed, im)
	return isExist
}

// get groupId from group config
func GetGroupIdFromGroupConfig(messageGroupMeta *MessageGroupMetaJSON) [GroupIdLen]byte {
	chainId := messageGroupMeta.ChainId
	qualifyType := messageGroupMeta.QualifyType
	contractAddress := messageGroupMeta.ContractAddress
	configFieldsMap := map[string]string{
		"chainId":         fmt.Sprintf("%d", chainId),
		"qualifyType":     qualifyType,
		"contractAddress": contractAddress,
		"tokenThres":      messageGroupMeta.TokenThres,
	}
	groupId := sortAndSha256Map(configFieldsMap)
	var groupIdFixed [GroupIdLen]byte
	copy(groupIdFixed[:], groupId)

	return groupIdFixed
}

// log ConfigStoreGroupIdToGroupConfig
func (im *Manager) LogConfigStoreGroupIdToGroupConfig(logger *logger.Logger) {
	// loop ConfigStoreGroupIdToGroupConfig
	/*
		for groupIdHex, groupMeta := range ConfigStoreGroupIdToGroupConfig {
			jsonStr, err := json.Marshal(groupMeta)
			if err != nil {
				continue
			}
			logger.Infof("LogConfigStoreGroupIdToGroupConfig ... groupId:%s, groupMeta:%s", groupIdHex, jsonStr)
		}*/
}

// calculate if group is public
func (im *Manager) CalculateIfGroupIsPublic(groupId [GroupIdLen]byte, logger *logger.Logger) error {
	// get votes
	publicCt, privateCt, err := im.CountVotesForGroup(groupId)
	if err != nil {
		return err
	}
	// get member count
	memberCt, err := im.GetGroupMemberAddressesCountFromGroupId(groupId, logger)
	if err != nil {
		return err
	}

	if memberCt > 100 || publicCt > privateCt {
		err = StorePublicGroupId(groupId, im)
		if err != nil {
			return err
		}
	} else if publicCt == privateCt {
		config, err := ReadGroupConfigMetaFromGroupId(groupId, im)
		if err != nil {
			return err
		}
		if config.MessageType == MessageTypePublic {
			err = StorePublicGroupId(groupId, im)
			if err != nil {
				return err
			}
		} else {
			err = DeletePublicGroupId(groupId, im)
			if err != nil {
				return err
			}
		}
	} else {
		err = DeletePublicGroupId(groupId, im)
		if err != nil {
			return err
		}
	}
	return nil
}

// try CalculateIfGroupIsPublic, only actually calculate if not during init
func (im *Manager) TryCalculateIfGroupIsPublic(groupId [GroupIdLen]byte, logger *logger.Logger) error {
	if IsIniting {
		return nil
	}

	return im.CalculateIfGroupIsPublic(groupId, logger)
}

// calculate if group is public for all groups
func (im *Manager) CalculateIfGroupIsPublicForAllGroups(logger *logger.Logger) error {
	//IterateAllGroupIdFromGroupConfigMetaStore
	err := IterateAllGroupIdFromGroupConfigMetaStore(im, func(groupId [GroupIdLen]byte) bool {

		err := im.CalculateIfGroupIsPublic(groupId, logger)
		if err != nil {
			// log error then continue
			logger.Infof("CalculateIfGroupIsPublicForAllGroups ... CalculateIfGroupIsPublic failed:%s", err)
		}
		return true
	})
	return err
}

// get is group public
func (im *Manager) GetIsGroupPublic(groupId [GroupIdLen]byte) bool {
	groupIdHex := iotago.EncodeHex(groupId[:])
	return im.GetIsGroupPublicWithGroupId(groupIdHex)
}

// get is group public with groupId string
func (im *Manager) GetIsGroupPublicWithGroupId(groupIdHex string) bool {
	groupIdBytes, err := iotago.DecodeHex(groupIdHex)
	if err != nil {
		// log error then return false
		Logger.Infof("GetIsGroupPublicWithGroupId ... iotago.DecodeHex failed:%s", err)
		return false
	}
	var groupIdFixed [GroupIdLen]byte
	copy(groupIdFixed[:], groupIdBytes)
	isGroupPublic := CheckIfGroupIdIsPublic(groupIdFixed, im)
	return isGroupPublic
}

// store groupId -> groupConfigMeta
// key = prefix + groupId
func KeyForGroupConfigMeta(groupId [GroupIdLen]byte) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixGroupConfig}, false)
	// groupId
	AppendBytesWithUint16Len(&payload, &idx, groupId[:], false)
	return payload
}

// parse groupId from GroupConfigMeta store key
func ParseGroupIdFromGroupConfigMetaKey(key kvstore.Key) ([GroupIdLen]byte, error) {
	idx := 0
	// prefix
	_, err := ReadBytesWithUint16Len(key, &idx, 1)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	// groupId
	groupIdBytes, err := ReadBytesWithUint16Len(key, &idx, GroupIdLen)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	var groupId [GroupIdLen]byte
	copy(groupId[:], groupIdBytes)
	return groupId, nil
}

// marshal ExtraChain to bytes
func MarshalExtraChain(extraChain *ExtraChain) ([]byte, error) {
	idx := 0
	var payload []byte
	// chainId
	AppendBytesWithUint16Len(&payload, &idx, Uint32ToBytes(extraChain.ChainId), false)
	// contractAddress
	AppendBytesWithUint16Len(&payload, &idx, []byte(extraChain.ContractAddress), true)
	return payload, nil
}

// unmarshal ExtraChain from bytes
func UnmarshalExtraChain(value []byte) (*ExtraChain, error) {
	idx := 0
	// chainId
	chainIdBytes, err := ReadBytesWithUint16Len(value, &idx, 4)
	if err != nil {
		return nil, err
	}
	chainId := BytesToUint32(chainIdBytes)
	// contractAddress
	contractAddressBytes, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	contractAddress := string(contractAddressBytes)
	return &ExtraChain{
		ChainId:         chainId,
		ContractAddress: contractAddress,
	}, nil
}

// marshal groupConfigMeta to bytes
func MarshalGroupConfigMeta(groupConfig *MessageGroupMetaJSON) ([]byte, error) {
	// just marshal as json, then convert string to bytes
	jsonStr, err := json.Marshal(groupConfig)
	if err != nil {
		return nil, err
	}
	return []byte(jsonStr), nil
}

// store groupConfigMeta for groupId to groupConfig
func StoreGroupConfigMetaForGroupId(groupId [GroupIdLen]byte, groupConfig *MessageGroupMetaJSON, im *Manager) error {

	key := KeyForGroupConfigMeta(groupId)
	// value = groupConfigMeta
	value, err := MarshalGroupConfigMeta(groupConfig)
	if err != nil {
		return err
	}
	// store
	err = im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return nil
}

// prefix for GroupConfigMeta store
func PrefixForGroupConfigMeta() []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixGroupConfig}, false)
	return payload
}

// UnmarshalGroupConfigMeta
func UnmarshalGroupConfigMeta(value []byte) (*MessageGroupMetaJSON, error) {
	// to string first, then json unmarshal it
	var groupConfig MessageGroupMetaJSON
	err := json.Unmarshal(value, &groupConfig)
	if err != nil {
		return nil, err
	}
	return &groupConfig, nil
}

// read groupConfigMeta from groupId
func ReadGroupConfigMetaFromGroupId(groupId [GroupIdLen]byte, im *Manager) (*MessageGroupMetaJSON, error) {
	// key = prefix + groupId
	key := KeyForGroupConfigMeta(groupId)
	// read
	value, err := im.imStore.Get(key)
	if err != nil {
		return nil, err
	}
	// value to groupConfigMeta
	groupConfig, err := UnmarshalGroupConfigMeta(value)
	if err != nil {
		return nil, err
	}
	return groupConfig, nil
}

// iterate all groupId, PrefixForGroupConfigMeta
func IterateAllGroupIdFromGroupConfigMetaStore(im *Manager, f func(groupId [GroupIdLen]byte) bool) error {
	prefix := PrefixForGroupConfigMeta()
	// iterate
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// key to groupId
		groupId, err := ParseGroupIdFromGroupConfigMetaKey(key)
		if err != nil {
			return true
		}
		return f(groupId)
	})
	if err != nil {
		return err
	}
	return nil
}

// check if group exists, given groupId
func IsGroupExists(groupId [GroupIdLen]byte, im *Manager) bool {
	key := KeyForGroupConfigMeta(groupId)
	isExist, err := im.imStore.Has(key)
	if err != nil {
		return false
	}
	return isExist
}

// delete groupConfigMeta from groupId
func (im *Manager) DeleteGroupConfigMetaFromGroupId(groupId [GroupIdLen]byte) error {
	// key = prefix + groupId
	key := KeyForGroupConfigMeta(groupId)
	// delete
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

// store read and delete for chainId + contract address hash + -> groupId
// key = prefix + chainId + contractAddressHash + groupId
// value is empty
func KeyForChainIdAndContractAddressHashToGroupId(chainId uint32, contractAddress string, groupId [GroupIdLen]byte) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixChainIdAndContractAddressHashToGroupId}, false)
	// chainId
	AppendBytesWithUint16Len(&payload, &idx, Uint32ToBytes(chainId), false)
	// contractAddressHash
	contractAddressHash := Sha256HashAddress(contractAddress)
	AppendBytesWithUint16Len(&payload, &idx, contractAddressHash, false)
	// groupId
	AppendBytesWithUint16Len(&payload, &idx, groupId[:], false)
	return payload
}

// unmarshal chainId + contract address hash + groupId
func UnmarshalChainIdAndContractAddressHashToGroupId(value []byte) ([GroupIdLen]byte, uint32, []byte, error) {
	idx := 0
	// prefix
	_, err := ReadBytesWithUint16Len(value, &idx, 1)
	if err != nil {
		return [GroupIdLen]byte{}, 0, []byte{}, err
	}
	// chainId
	chainIdBytes, err := ReadBytesWithUint16Len(value, &idx, 4)
	if err != nil {
		return [GroupIdLen]byte{}, 0, []byte{}, err
	}
	chainId := BytesToUint32(chainIdBytes)
	// contractAddressHash
	contractAddressHash, err := ReadBytesWithUint16Len(value, &idx, Sha256HashLen)
	if err != nil {
		return [GroupIdLen]byte{}, 0, []byte{}, err
	}
	// groupId
	groupIdBytes, err := ReadBytesWithUint16Len(value, &idx, GroupIdLen)
	if err != nil {
		return [GroupIdLen]byte{}, 0, []byte{}, err
	}
	var groupId [GroupIdLen]byte
	copy(groupId[:], groupIdBytes)
	return groupId, chainId, contractAddressHash, nil
}

// store chainId + contract address hash + groupId -> outputId
func StoreChainIdAndContractAddressHashToGroupId(chainId uint32, contractAddress string, groupId [GroupIdLen]byte, outputId [OutputIdLen]byte, groupConfig *MessageGroupMetaJSON, im *Manager) error {
	// key = prefix + chainId + contractAddressHash + groupId
	key := KeyForChainIdAndContractAddressHashToGroupId(chainId, contractAddress, groupId)
	// value = outputId
	value := outputId[:]
	// store
	err := im.imStore.Set(key, value)
	if err != nil {
		// log error then return
		Logger.Infof("StoreChainIdAndContractAddressHashToGroupId ... imStore.Set failed:%s", err)
		return err
	}
	// log success
	Logger.Infof("StoreChainIdAndContractAddressHashToGroupId ... chainId:%d, contractAddress:%s, groupId:%s, outputId:%s", chainId, contractAddress, iotago.EncodeHex(groupId[:]), iotago.EncodeHex(outputId[:]))
	// extra chains
	if groupConfig.ExtraChains != nil {
		for _, extraChain := range groupConfig.ExtraChains {
			// key = KeyForChainIdAndContractAddressHashToGroupId
			key := KeyForChainIdAndContractAddressHashToGroupId(extraChain.ChainId, extraChain.ContractAddress, groupId)
			// value = outputId
			err := im.imStore.Set(key, value)
			if err != nil {
				// log error then continue
				Logger.Infof("StoreChainIdAndContractAddressHashToGroupId ... extra chains ... imStore.Set failed:%s", err)
				continue
			}
		}
	}
	return nil
}

// delete chainId + contract address hash + groupId -> outputId
func DeleteChainIdAndContractAddressHashToGroupId(chainId uint32, contractAddress string, groupId [GroupIdLen]byte, groupConfig *MessageGroupMetaJSON, im *Manager) error {
	// key = prefix + chainId + contractAddressHash + groupId
	key := KeyForChainIdAndContractAddressHashToGroupId(chainId, contractAddress, groupId)
	// delete
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	// extra chains
	if groupConfig.ExtraChains != nil {
		for _, extraChain := range groupConfig.ExtraChains {
			// key = KeyForChainIdAndContractAddressHashToGroupId
			key := KeyForChainIdAndContractAddressHashToGroupId(extraChain.ChainId, extraChain.ContractAddress, groupId)
			// delete
			err := im.imStore.Delete(key)
			if err != nil {
				// log error then continue
				Logger.Infof("DeleteChainIdAndContractAddressHashToGroupId ... extra chains ... imStore.Delete failed:%s", err)
				continue
			}
		}
	}
	return nil
}

// prefix for chainId + contract address hash + -> groupId
func PrefixForChainIdAndContractAddressHashToGroupId(chainId uint32, contractAddress string) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixChainIdAndContractAddressHashToGroupId}, false)
	// return in case of default or nil
	// chainId
	if chainId == math.MaxUint32 {
		return payload
	}
	AppendBytesWithUint16Len(&payload, &idx, Uint32ToBytes(chainId), false)
	if contractAddress == "" {
		return payload
	}
	contractAddressHash := Sha256HashAddress(contractAddress)
	AppendBytesWithUint16Len(&payload, &idx, contractAddressHash, false)
	return payload
}

// read all groupId from chainId + contract address hash
func ReadAllGroupIdFromChainIdAndContractAddressHash(chainId uint32, contractAddress string, im *Manager) ([][GroupIdLen]byte, error) {
	prefix := PrefixForChainIdAndContractAddressHashToGroupId(chainId, contractAddress)
	// iterate
	var res [][GroupIdLen]byte
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// value to groupId
		groupId, _, _, err := UnmarshalChainIdAndContractAddressHashToGroupId(key)
		if err != nil {
			return true
		}
		res = append(res, groupId)
		return true
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// delete all groupId from chainId + contract address hash
func DeleteAllGroupIdFromChainIdAndContractAddressHash(chainId uint32, contractAddress string, im *Manager) error {
	prefix := PrefixForChainIdAndContractAddressHashToGroupId(chainId, contractAddress)
	// iterate then delete
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		err := im.imStore.Delete(key)
		if err != nil {
			return true
		}
		return true
	})
	if err != nil {
		return err
	}
	return nil
}

// store read and delete for chainId + qualifyType + -> groupId
// key = prefix + chainId + qualifyTypeHash + groupId
// value is empty
func KeyForChainIdAndQualifyTypeToGroupId(chainId uint32, qualifyType string, groupId [GroupIdLen]byte) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixChainIdAndQualifyTypeToGroupId}, false)
	// chainId
	AppendBytesWithUint16Len(&payload, &idx, Uint32ToBytes(chainId), false)
	// qualifyTypeHash
	qualifyTypeHash := Sha256Hash(qualifyType)
	AppendBytesWithUint16Len(&payload, &idx, qualifyTypeHash, false)
	// groupId
	AppendBytesWithUint16Len(&payload, &idx, groupId[:], false)
	return payload
}

// PrefixForChainIdAndQualifyTypeToGroupId
func PrefixForChainIdAndQualifyTypeToGroupId(chainId uint32, qualifyType string) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixChainIdAndQualifyTypeToGroupId}, false)
	// chainId
	AppendBytesWithUint16Len(&payload, &idx, Uint32ToBytes(chainId), false)
	// qualifyTypeHash
	qualifyTypeHash := Sha256Hash(qualifyType)
	AppendBytesWithUint16Len(&payload, &idx, qualifyTypeHash, false)
	return payload
}

// UnmarshalChainIdAndQualifyTypeToGroupId
func UnmarshalChainIdAndQualifyTypeToGroupId(value []byte) ([GroupIdLen]byte, uint32, string, error) {
	idx := 0
	// chainId
	chainIdBytes, err := ReadBytesWithUint16Len(value, &idx, 4)
	if err != nil {
		return [GroupIdLen]byte{}, 0, "", err
	}
	chainId := BytesToUint32(chainIdBytes)
	// qualifyTypeHash
	qualifyTypeHash, err := ReadBytesWithUint16Len(value, &idx, Sha256HashLen)
	if err != nil {
		return [GroupIdLen]byte{}, 0, "", err
	}
	// groupId
	groupIdBytes, err := ReadBytesWithUint16Len(value, &idx, GroupIdLen)
	if err != nil {
		return [GroupIdLen]byte{}, 0, "", err
	}
	var groupId [GroupIdLen]byte
	copy(groupId[:], groupIdBytes)
	// qualifyType
	qualifyType := string(qualifyTypeHash)
	return groupId, chainId, qualifyType, nil
}

// store chainId + qualifyType + -> groupId
func StoreChainIdAndQualifyTypeToGroupId(chainId uint32, qualifyType string, groupId [GroupIdLen]byte, im *Manager) error {
	// key = prefix + chainId + qualifyTypeHash + groupId
	key := KeyForChainIdAndQualifyTypeToGroupId(chainId, qualifyType, groupId)
	// value is empty
	value := []byte{}
	// store
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return nil
}

// read all groupId from chainId + qualifyType
func ReadAllGroupIdFromChainIdAndQualifyType(chainId uint32, qualifyType string, im *Manager) ([][GroupIdLen]byte, error) {
	prefix := PrefixForChainIdAndQualifyTypeToGroupId(chainId, qualifyType)
	// iterate
	var res [][GroupIdLen]byte
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// value to groupId
		groupId, _, _, err := UnmarshalChainIdAndQualifyTypeToGroupId(key)
		if err != nil {
			return true
		}
		res = append(res, groupId)
		return true
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// iterate all groupId from chainId + qualifyType
func IterateAllGroupIdFromChainIdAndQualifyType(chainId uint32, qualifyType string, im *Manager, f func(groupId [GroupIdLen]byte) bool) error {
	prefix := PrefixForChainIdAndQualifyTypeToGroupId(chainId, qualifyType)
	// iterate
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		// value to groupId
		groupId, _, _, err := UnmarshalChainIdAndQualifyTypeToGroupId(key)
		if err != nil {
			return true
		}
		return f(groupId)
	})
	if err != nil {
		return err
	}
	return nil
}

// delete chainId + qualifyType + -> groupId
func DeleteGroupIdFromChainIdAndQualifyType(chainId uint32, qualifyType string, groupId [GroupIdLen]byte, im *Manager) error {
	// key = prefix + chainId + qualifyTypeHash + groupId
	key := KeyForChainIdAndQualifyTypeToGroupId(chainId, qualifyType, groupId)
	// delete
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

// store read and delete for dappGroupId + -> groupId
// key = prefix + dappGroupIdHash, value = groupId
func KeyForDappGroupIdToGroupId(dappGroupId string) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixDappGroupIdToGroupId}, false)
	// dappGroupIdHash
	dappGroupIdHash := Sha256Hash(dappGroupId)
	AppendBytesWithUint16Len(&payload, &idx, dappGroupIdHash, false)
	return payload
}

// store dappGroupId + -> groupId
func StoreDappGroupIdToGroupId(dappGroupId string, groupId [GroupIdLen]byte, im *Manager) error {
	// key = prefix + dappGroupIdHash
	key := KeyForDappGroupIdToGroupId(dappGroupId)
	value := groupId[:]
	// store
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return nil
}

// read groupId from dappGroupId
func ReadGroupIdFromDappGroupId(dappGroupId string, im *Manager) ([GroupIdLen]byte, error) {
	// key = prefix + dappGroupIdHash
	key := KeyForDappGroupIdToGroupId(dappGroupId)
	// read
	value, err := im.imStore.Get(key)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	// value to groupId
	var groupId [GroupIdLen]byte
	copy(groupId[:], value)
	return groupId, nil
}

// delete groupId from dappGroupId
func DeleteGroupIdFromDappGroupId(dappGroupId string, im *Manager) error {
	// key = prefix + dappGroupIdHash
	key := KeyForDappGroupIdToGroupId(dappGroupId)
	// delete
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

// store read and delete for chainId + contract address hash + -> outputId
// key = prefix + chainId + contractAddressHash
// value = outputId + groupId
func KeyForChainIdAndContractAddressHashToOutputId(chainId uint32, contractAddress string) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixChainIdAndContractAddressHashToOutputId}, false)
	// chainId
	if chainId == math.MaxUint32 {
		return payload
	}
	AppendBytesWithUint16Len(&payload, &idx, Uint32ToBytes(chainId), false)
	// contractAddressHash
	if contractAddress == "" {
		return payload
	}
	contractAddressHash := Sha256HashAddress(contractAddress)
	AppendBytesWithUint16Len(&payload, &idx, contractAddressHash, false)
	return payload
}

// value = outputId + contractAddress
func ValueForChainIdAndContractAddressHashToOutputId(outputId [OutputIdLen]byte, contractAddress string) []byte {
	idx := 0
	var payload []byte
	// outputId
	AppendBytesWithUint16Len(&payload, &idx, outputId[:], false)
	// contractAddress
	AppendBytesWithUint16Len(&payload, &idx, []byte(contractAddress), true)
	return payload
}

// parse key to chainId
func ParseKeyForChainIdToOutputId(key []byte) (uint32, error) {
	// key = prefix + chainId + contractAddressHash
	idx := 0
	_, err := ReadBytesWithUint16Len(key, &idx, 1)
	if err != nil {
		return 0, err
	}
	// chainId
	chainIdBytes, err := ReadBytesWithUint16Len(key, &idx, 4)
	if err != nil {
		return 0, err
	}
	chainId := BytesToUint32(chainIdBytes)
	return chainId, nil
}

// parse value to outputId and contractAddress
func ParseValueForChainIdAndContractAddressHashToOutputIdAndContractAddress(value []byte) ([OutputIdLen]byte, string, error) {
	idx := 0
	// outputId
	outputIdBytes, err := ReadBytesWithUint16Len(value, &idx, OutputIdLen)
	if err != nil {
		return [OutputIdLen]byte{}, "", err
	}
	var outputId [OutputIdLen]byte
	copy(outputId[:], outputIdBytes)
	// contractAddress
	contractAddressBytes, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return [OutputIdLen]byte{}, "", err
	}
	contractAddress := string(contractAddressBytes)
	return outputId, contractAddress, nil
}

// store chainId + contract address hash + -> outputId + groupId
func StoreChainIdAndContractAddressHashToOutputId(chainId uint32, contractAddress string, outputId [OutputIdLen]byte, groupId [GroupIdLen]byte, im *Manager) error {
	// key = prefix + chainId + contractAddressHash
	key := KeyForChainIdAndContractAddressHashToOutputId(chainId, contractAddress)
	value := ValueForChainIdAndContractAddressHashToOutputId(outputId, contractAddress)
	// store
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return nil
}

// read outputId + contractAddress from chainId + contract address
func ReadOutputIdAndContractAddressFromChainIdAndContractAddress(chainId uint32, contractAddress string, im *Manager) ([OutputIdLen]byte, string, error) {
	// key = prefix + chainId + contractAddressHash
	key := KeyForChainIdAndContractAddressHashToOutputId(chainId, contractAddress)
	// read
	value, err := im.imStore.Get(key)
	if err != nil {
		return [OutputIdLen]byte{}, "", err
	}
	// value to outputId and contractAddress
	outputId, contractAddressFromStore, err := ParseValueForChainIdAndContractAddressHashToOutputIdAndContractAddress(value)
	if err != nil {
		return [OutputIdLen]byte{}, "", err
	}
	return outputId, contractAddressFromStore, nil
}

// delete outputId + contractAddress from chainId + contract address
func DeleteOutputIdAndGroupIdFromChainIdAndContractAddress(chainId uint32, contractAddress string, im *Manager) error {
	// key = prefix + chainId + contractAddressHash
	key := KeyForChainIdAndContractAddressHashToOutputId(chainId, contractAddress)
	// delete
	err := im.imStore.Delete(key)
	if err != nil {
		return err
	}
	return nil
}

// GroupConfigLiteResponse
type GroupConfigNftListResponse struct {
	ChainId         uint32 `json:"chainId"`
	ContractAddress string `json:"contractAddress"`
	OutputId        string `json:"outputId"`
}

// GroupStateSyncResponseItem
type GroupStateSyncResponseItem struct {
	GroupId                            string `json:"groupId"`
	LastTimeReadLatestMessageTimestamp uint32 `json:"lastTimeReadLatestMessageTimestamp"`
}

// GroupStateSyncResponse
type GroupStateSyncResponse struct {
	OutputId string                        `json:"outputId"`
	Items    []*GroupStateSyncResponseItem `json:"items"`
}

type OutputIdCheckResponse struct {
	OutputId    string `json:"outputId"`
	IsEffecting bool   `json:"isEffecting"`
}

// list all outputId + contractAddress from the store, with optional chainId and contractAddress, page and pageSize
func ListOutputIdAndGroupIdFromChainIdAndContractAddress(chainId uint32, contractAddress string, page int, pageSize int, im *Manager) ([]*GroupConfigNftListResponse, error) {
	if page <= 0 || pageSize <= 0 || pageSize > 100 || page >= 1000 {
		return nil, errors.New("page and pageSize must be greater than 0")
	}
	var resp []*GroupConfigNftListResponse
	prefix := KeyForChainIdAndContractAddressHashToOutputId(chainId, contractAddress)
	ct := 0
	skipLefted := (page - 1) * pageSize
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		if skipLefted > 0 {
			skipLefted--
			return true
		}
		chainId, err := ParseKeyForChainIdToOutputId(key)
		if err != nil {
			return true
		}
		// value to outputId and contractAddress
		outputId, contractAddress, err := ParseValueForChainIdAndContractAddressHashToOutputIdAndContractAddress(value)
		if err != nil {
			return true
		}
		resp = append(resp, &GroupConfigNftListResponse{
			ChainId:         chainId,
			ContractAddress: contractAddress,
			OutputId:        iotago.EncodeHex(outputId[:]),
		})
		ct++
		if pageSize > 0 && ct >= pageSize {
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ConfigWithOutputId represents a combination of a group config and the corresponding outputId
type ConfigWithOutputId struct {
	OutputId        string                `json:"outputId"`
	GroupConfigMeta *MessageGroupMetaJSON `json:"config"`
}

// ParseGroupIdFromChainIdAndContractAddressHashKey parses the groupId from the key in the store
func ParseGroupIdFromChainIdAndContractAddressHashKey(key kvstore.Key) ([GroupIdLen]byte, error) {
	idx := 0
	// prefix
	_, err := ReadBytesWithUint16Len(key, &idx, 1)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	// chainId
	_, err = ReadBytesWithUint16Len(key, &idx, 4)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	// contractAddressHash
	_, err = ReadBytesWithUint16Len(key, &idx, Sha256HashLen)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	// groupId
	groupIdBytes, err := ReadBytesWithUint16Len(key, &idx, GroupIdLen)
	if err != nil {
		return [GroupIdLen]byte{}, err
	}
	var groupId [GroupIdLen]byte
	copy(groupId[:], groupIdBytes)
	return groupId, nil
}

// ListConfigWithOutputIdFromChainIdAndContractAddressv2 returns a paginated list of ConfigWithOutputId for a given chainId and contractAddress.
func ListConfigWithOutputIdFromChainIdAndContractAddressv2(chainId uint32, contractAddress string, page int, pageSize int, im *Manager) (int, int, int, []*ConfigWithOutputId, error) {
	if page <= 0 || pageSize <= 0 || pageSize > 100 || page >= 1000 {
		return 0, 0, 0, nil, errors.New("page and pageSize must be greater than 0")
	}

	var result []*ConfigWithOutputId
	prefix := PrefixForChainIdAndContractAddressHashToGroupId(chainId, contractAddress)
	// log method chainId:%d, contractAddress:%s, prefix:%s
	Logger.Infof("ListConfigWithOutputIdFromChainIdAndContractAddressv2 ... chainId:%d, contractAddress:%s, prefix:%s", chainId, contractAddress, prefix)
	total := 0
	skipLefted := (page - 1) * pageSize

	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		total++

		// Skip until we reach the required page
		if total > skipLefted && len(result) < pageSize {
			groupId, err := ParseGroupIdFromChainIdAndContractAddressHashKey(key)
			if err != nil {
				// log error
				Logger.Infof("ListConfigWithOutputIdFromChainIdAndContractAddressv2 ... ParseGroupIdFromChainIdAndContractAddressHashKey failed:%s", err)
				return true
			}

			// value contains outputId
			var outputId [OutputIdLen]byte
			copy(outputId[:], value)

			// Fetch group config metadata from groupId
			groupConfigMeta, err := ReadGroupConfigMetaFromGroupId(groupId, im)
			if err != nil {
				return true
			}

			// Create the ConfigWithOutputId object
			result = append(result, &ConfigWithOutputId{
				OutputId:        iotago.EncodeHex(outputId[:]),
				GroupConfigMeta: groupConfigMeta,
			})
		}
		return true
	})

	if err != nil {
		return 0, 0, 0, nil, err
	}

	return page, pageSize, total, result, nil
}

// store check exist and delete for groupId which is public
// key = prefix + groupId
// value is empty
func KeyForPublicGroupId(groupId [GroupIdLen]byte) []byte {
	idx := 0
	var payload []byte
	// prefix
	AppendBytesWithUint16Len(&payload, &idx, []byte{ImStoreKeyPrefixPublicGroupId}, false)
	// groupId
	AppendBytesWithUint16Len(&payload, &idx, groupId[:], false)
	return payload
}

// store public groupId
func StorePublicGroupId(groupId [GroupIdLen]byte, im *Manager) error {
	// key = prefix + groupId
	key := KeyForPublicGroupId(groupId)
	// is need push
	var isNeedPush bool
	// check if value is exist
	isExist, err := im.imStore.Has(key)
	if err != nil {
		// log error
		Logger.Infof("StorePublicGroupId ... imStore.Has failed:%s", err)
		return err
	}
	// if exist, means groupId is already public, hence no need to push
	if isExist {
		isNeedPush = false
	} else {
		isNeedPush = true
	}

	// value is empty
	value := []byte{}
	// store
	err = im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	// push
	if isNeedPush {
		return GenAndPushGroupIsPublicChangedEvent(groupId, true, im, Logger)
	}
	return nil
}

// check if groupId is public
func CheckIfGroupIdIsPublic(groupId [GroupIdLen]byte, im *Manager) bool {
	// key = prefix + groupId
	key := KeyForPublicGroupId(groupId)
	// check
	isExist, err := im.imStore.Has(key)
	if err != nil {
		return false
	}
	return isExist
}

// delete public groupId
func DeletePublicGroupId(groupId [GroupIdLen]byte, im *Manager) error {
	// key = prefix + groupId
	key := KeyForPublicGroupId(groupId)
	// is need push
	var isNeedPush bool
	// check if value is exist
	isExist, err := im.imStore.Has(key)
	if err != nil {
		// log error
		Logger.Infof("DeletePublicGroupId ... imStore.Has failed:%s", err)
		return err
	}
	// if not exist, means groupId is not public, hence no need to push
	if !isExist {
		isNeedPush = false
	} else {
		isNeedPush = true
	}
	// delete
	err = im.imStore.Delete(key)
	if err != nil {
		return err
	}
	// push
	if isNeedPush {
		return GenAndPushGroupIsPublicChangedEvent(groupId, false, im, Logger)
	}
	return nil
}

type ConfigNftOutputWrapper struct {
	OutputId        [OutputIdLen]byte
	Configs         []*MessageGroupMetaJSON
	ChainId         uint32
	ContractAddress string
}
type ConfigNftOutputMetaJson struct {
	Uri             string `json:"uri"`
	ChainId         uint32 `json:"chainId"`
	ContractAddress string `json:"contractAddress"`
}

// Handle group config nft output created and consumed
func (im *Manager) HandleGroupConfigNFTOutputConsumedOrCreated(consumed []*ConfigNftOutputWrapper, created []*ConfigNftOutputWrapper, logger *logger.Logger) error {
	for _, config := range consumed {
		err := HandleGroupNFTOutputConsumed(config, logger, im)
		if err != nil {
			// log error
			logger.Infof("HandleGroupConfigNFTOutputConsumedOrCreated ... HandleGroupNFTOutputConsumed failed:%s", err)
			return err
		}
	}

	for _, config := range created {
		err := HandleGroupNFTOutputCreated(config, logger, im)
		if err != nil {
			// log error
			logger.Infof("HandleGroupConfigNFTOutputConsumedOrCreated ... HandleGroupNFTOutputCreated failed:%s", err)
			return err
		}
	}

	return nil
}

// extract group config meta list from nft output
func ExtractConfigNftOutputWrapperFromNFTOutput(outputId [OutputIdLen]byte, nftOutput *iotago.NFTOutput, im *Manager) (*ConfigNftOutputWrapper, error) {
	if nftOutput.ImmutableFeatureSet() == nil || nftOutput.ImmutableFeatureSet().MetadataFeature() == nil || nftOutput.ImmutableFeatureSet().MetadataFeature().Data == nil {
		return nil, errors.New("nftOutput.ImmutableFeatureSet().MetadataFeature().Data is nil")
	}
	meta := nftOutput.ImmutableFeatureSet().MetadataFeature().Data
	// unmarshal meta as ConfigNftOutputMetaJson
	var configNftOutputMetaJson ConfigNftOutputMetaJson
	err := json.Unmarshal(meta, &configNftOutputMetaJson)
	if err != nil {
		return nil, err
	}
	// check nil for uri, chainId, contractAddress
	if configNftOutputMetaJson.Uri == "" || configNftOutputMetaJson.ChainId == 0 || configNftOutputMetaJson.ContractAddress == "" {
		return nil, errors.New("configNftOutputMetaJson.Uri, configNftOutputMetaJson.ChainId, configNftOutputMetaJson.ContractAddress is empty")
	}
	configStr, err := DownloadIpfsContent(configNftOutputMetaJson.Uri)
	if err != nil {
		return nil, err
	}
	// log uri and configStr
	Logger.Infof("uri: %s, configStr: %s", configNftOutputMetaJson.Uri, configStr)
	// unmarshal configStr as MessageGroupMetaJSON slice
	var groupConfigMeta []*MessageGroupMetaJSON
	err = json.Unmarshal([]byte(configStr), &groupConfigMeta)
	if err != nil {
		return nil, err
	}
	return &ConfigNftOutputWrapper{
		OutputId:        outputId,
		Configs:         groupConfigMeta,
		ChainId:         configNftOutputMetaJson.ChainId,
		ContractAddress: configNftOutputMetaJson.ContractAddress,
	}, nil
}

var groupconfigTagRawStr = "GROUPFIGROUPCONFIGV1"
var groupconfigTag = []byte(groupconfigTagRawStr)
var GroupconfigTagStr = iotago.EncodeHex(groupconfigTag)

// filter output for config nft output wrapper
// output iotago.Output, outputId iotago.OutputID
func FilterOutputForConfigNftOutputWrapper(output iotago.Output, outputId iotago.OutputID, im *Manager) (*ConfigNftOutputWrapper, error) {
	if output == nil {
		return nil, nil
	}
	// check nft output
	nftOutput, ok := output.(*iotago.NFTOutput)
	if !ok {
		return nil, nil
	}
	featureSet := nftOutput.FeatureSet()
	tag := featureSet.TagFeature()
	if tag == nil {
		return nil, nil
	}
	// check tag
	if !bytes.Equal(tag.Tag[:], groupconfigTag) {
		return nil, nil
	}

	// ExtractConfigNftOutputWrapperFromNFTOutput
	configNftOutputWrapper, err := ExtractConfigNftOutputWrapperFromNFTOutput(outputId, nftOutput, im)
	if err != nil {
		return nil, err
	}
	return configNftOutputWrapper, nil
}

// filter ledger output for config nft output wrapper
func FilterLedgerOutputForConfigNftOutputWrapper(inxOutput *inx.LedgerOutput, im *Manager) (*ConfigNftOutputWrapper, error) {
	// check nil
	if inxOutput == nil {
		return nil, nil
	}
	// get output
	output, err := inxOutput.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	outputId := inxOutput.UnwrapOutputID()
	// filter output
	config, err := FilterOutputForConfigNftOutputWrapper(output, outputId, im)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// HandleGroupNFTOutputConsumed
func HandleGroupNFTOutputConsumed(configWrapper *ConfigNftOutputWrapper, logger *logger.Logger, im *Manager) error {
	// get outputId and contract address
	//outputId := configWrapper.OutputId

	configs := configWrapper.Configs
	// get groupId from outputId and contract address
	for _, config := range configs {
		// delete public groupId
		groupId := GetGroupIdFromGroupConfig(config)
		err := DeletePublicGroupId(groupId, im)
		if err != nil {
			return err
		}
		// delete chainId + qualifyType + -> groupId
		err = DeleteGroupIdFromChainIdAndQualifyType(config.ChainId, config.QualifyType, groupId, im)
		if err != nil {
			return err
		}
		// delete chainId + contract address hash + groupId -> outputId
		err = DeleteChainIdAndContractAddressHashToGroupId(configWrapper.ChainId, configWrapper.ContractAddress, groupId, config, im)
		if err != nil {
			return err
		}
		// delete groupId from group config meta
		err = im.DeleteGroupConfigMetaFromGroupId(groupId)
		if err != nil {
			return err
		}
		dappGroupId := GetDappGroupId(iotago.EncodeHex(groupId[:]), config)
		// delete groupId from dappGroupId
		err = DeleteGroupIdFromDappGroupId(dappGroupId, im)
		if err != nil {
			return err
		}

	}
	return nil
}

// HandleGroupNFTOutputCreated
func HandleGroupNFTOutputCreated(configWrapper *ConfigNftOutputWrapper, logger *logger.Logger, im *Manager) error {
	// get outputId and contract address
	outputId := configWrapper.OutputId
	contractAddress := configWrapper.ContractAddress
	chainId := configWrapper.ChainId
	// store chainId + qualifyType + -> groupId
	for _, config := range configWrapper.Configs {
		groupId := GetGroupIdFromGroupConfig(config)
		groupIdHex := iotago.EncodeHex(groupId[:])
		// log groupIdHex, chainId, contractAddress, config.QualifyType
		logger.Infof("groupIdHex: %s, chainId: %d, contractAddress: %s, qualifyType: %s", groupIdHex, chainId, contractAddress, config.QualifyType)
		err := StoreChainIdAndQualifyTypeToGroupId(chainId, config.QualifyType, groupId, im)
		if err != nil {
			return err
		}

		// store chainId + contract address hash + -> groupId
		err = StoreChainIdAndContractAddressHashToGroupId(chainId, contractAddress, groupId, outputId, config, im)
		if err != nil {
			return err
		}
		// store groupId from dappGroupId
		dappGroupId := GetDappGroupId(groupIdHex, config)
		config.DappGroupId = dappGroupId
		err = StoreDappGroupIdToGroupId(dappGroupId, groupId, im)
		if err != nil {
			return err
		}
		// store group config meta
		err = StoreGroupConfigMetaForGroupId(groupId, config, im)
		if err != nil {
			return err
		}
		// CalculateIfGroupIsPublic
		err = im.CalculateIfGroupIsPublic(groupId, logger)
		if err != nil {
			return err
		}
	}
	return nil
}
