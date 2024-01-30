package im

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

var ConfigStoreGroupIdToGroupConfig = map[string]*MessageGroupMetaJSON{}
var ConfigStoreChainNameAndQualifyTypeToGroupId = map[string]map[string][]string{}
var ConfigStorePublicGroupIds = []string{}

const MessageTypePublic = 2

const IcebergGroup = "iceberg"
const IcebergCollectionConfigIssuerAddress = "smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp"

type GroupIdAndGroupNamePair struct {
	GroupId   []byte
	GroupName string
}

func ChainNameAndCollectionIdToGroupIdAndGroupNamePairs(chainName string, collectionId string) []*GroupIdAndGroupNamePair {
	if (ConfigStoreChainNameAndQualifyTypeToGroupId[chainName] == nil) || (ConfigStoreChainNameAndQualifyTypeToGroupId[chainName]["nft"] == nil) {
		return nil
	}
	var res []*GroupIdAndGroupNamePair
	for _, groupIdHex := range ConfigStoreChainNameAndQualifyTypeToGroupId[chainName]["nft"] {
		if ConfigStoreGroupIdToGroupConfig[groupIdHex].CollectionIds == nil {
			continue
		}
		for _, collectionIdInGroupConfig := range ConfigStoreGroupIdToGroupConfig[groupIdHex].CollectionIds {
			if collectionIdInGroupConfig == collectionId {
				groupId, err := iotago.DecodeHex(groupIdHex)
				if err != nil {
					return nil
				}
				res = append(res, &GroupIdAndGroupNamePair{
					GroupId:   groupId,
					GroupName: ConfigStoreGroupIdToGroupConfig[groupIdHex].GroupName,
				})
				break
			}
		}
	}
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

func (im *Manager) GroupNameToGroupId(group string) []byte {
	// loop ConfigStoreGroupIdToGroupConfig
	for groupIdHex, groupMeta := range ConfigStoreGroupIdToGroupConfig {
		if groupMeta.GroupName == group {
			groupId, err := iotago.DecodeHex(groupIdHex)
			if err != nil {
				return nil
			}
			return groupId
		}
	}
	return nil
}

// groupId to group config
func (im *Manager) GroupIdToGroupConfig(groupIdHex string) *MessageGroupMetaJSON {
	return ConfigStoreGroupIdToGroupConfig[groupIdHex]
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

type MessageGroupMetaJSON struct {
	GroupName     string   `json:"groupName"`
	ChainName     string   `json:"chainName"`
	SchemaVersion int      `json:"schemaVersion"`
	MessageType   int      `json:"messageType"`
	AuthScheme    int      `json:"authScheme"`
	QualifyType   string   `json:"qualifyType"`
	CollectionIds []string `json:"collectionIds"`
	TokenId       string   `json:"tokenId"`
	TokenThres    string   `json:"tokenThres"`
}

// handle group config nft created
func (im *Manager) HandleGroupNFTOutputCreated(nftOutput *iotago.NFTOutput, logger *logger.Logger) error {
	// parse name and ipfs link
	ipfsLink, err := im.ParseGroupConfigNFT(nftOutput)
	if err != nil {
		return err
	}
	// get content from ipfs
	contentRaw, err := ReadIpfsFile(ipfsLink)
	if err != nil {
		return err
	}
	return im.HandleGroupConfigRawContent(contentRaw, logger)
}
func (im *Manager) HandleGroupConfigRawContent(contentRaw []byte, logger *logger.Logger) error {
	// unmarshal content(json) to MessageGroupMetaJSON[]
	var messageGroupMetaList []MessageGroupMetaJSON
	err := json.Unmarshal(contentRaw, &messageGroupMetaList)
	if err != nil {
		// log
		logger.Infof("HandleGroupNFTOutputCreated ... json.Unmarshal failed:%s", err)
		return err
	}
	// store all group config
	for _, messageGroupMeta := range messageGroupMetaList {
		// clone messageGroupMeta
		messageGroupMetaClone := messageGroupMeta
		err = im.StoreOneGroupConfig(&messageGroupMetaClone)
		if err != nil {
			// log error then continue
			logger.Infof("HandleGroupNFTOutputCreated ... StoreOneGroupConfig failed:%s", err)
			continue
		}
	}
	return nil
}

// getAllpublicGroupIds
func (im *Manager) GetAllPublicGroupIds() []string {
	return ConfigStorePublicGroupIds
}

// add groupId to public group ids
func (im *Manager) AddGroupIdToPublicGroupIds(groupIdHex string) {
	// check if already exists
	for _, groupIdHexInPublicGroupIds := range ConfigStorePublicGroupIds {
		if groupIdHexInPublicGroupIds == groupIdHex {
			return
		}
	}
	ConfigStorePublicGroupIds = append(ConfigStorePublicGroupIds, groupIdHex)
}

// checkGroupExists
func (im *Manager) CheckGroupExists(groupIdHex string) bool {
	return ConfigStoreGroupIdToGroupConfig[groupIdHex] != nil
}

// remove groupId from public group ids
func (im *Manager) RemoveGroupIdFromPublicGroupIds(groupIdHex string) {
	// if group config is public, do nothing, check exist first
	if !im.CheckGroupExists(groupIdHex) {
		return
	}
	if ConfigStoreGroupIdToGroupConfig[groupIdHex].MessageType == MessageTypePublic {
		return
	}
	var newPublicGroupIds []string
	for i, groupIdHexInPublicGroupIds := range ConfigStorePublicGroupIds {
		if groupIdHexInPublicGroupIds == groupIdHex {
			continue
		}
		newPublicGroupIds = append(newPublicGroupIds, ConfigStorePublicGroupIds[i])
	}
	ConfigStorePublicGroupIds = newPublicGroupIds
}
func GroupConfigKeyFromRenterNameAndGroupName(renterName string, groupName string) []byte {
	renterNameSha256 := Sha256Hash(renterName)
	groupNameSha256 := Sha256Hash(groupName)
	return ConcatByteSlices([]byte{ImStoreKeyPrefixGroupConfig}, renterNameSha256, groupNameSha256)
}

// store one group config (group name, MessageGroupMetaJSON)
func (im *Manager) StoreOneGroupConfig(messageGroupMeta *MessageGroupMetaJSON) error {
	chainName := messageGroupMeta.ChainName
	qualifyType := messageGroupMeta.QualifyType
	isPublic := messageGroupMeta.MessageType == MessageTypePublic
	// sort then concat collectionIds
	collectionIds := messageGroupMeta.CollectionIds
	sort.Strings(collectionIds)
	collectionIdsStr := ""
	for _, collectionId := range collectionIds {
		collectionIdsStr += collectionId
	}
	configFieldsMap := map[string]string{
		"groupName":     messageGroupMeta.GroupName,
		"chainName":     chainName,
		"schemaVersion": fmt.Sprintf("%d", messageGroupMeta.SchemaVersion),
		"messageType":   fmt.Sprintf("%d", messageGroupMeta.MessageType),
		"authScheme":    fmt.Sprintf("%d", messageGroupMeta.AuthScheme),
		"qualifyType":   qualifyType,
		"tokenId":       messageGroupMeta.TokenId,
		"tokenThres":    messageGroupMeta.TokenThres,
		"collectionIds": collectionIdsStr,
	}
	groupId := sortAndSha256Map(configFieldsMap)
	groupIdHex := iotago.EncodeHex(groupId)
	// store groupId -> group config store
	// ensure ConfigStoreChainNameAndQualifyTypeToGroupId[chainName] exists
	if ConfigStoreChainNameAndQualifyTypeToGroupId[chainName] == nil {
		ConfigStoreChainNameAndQualifyTypeToGroupId[chainName] = map[string][]string{}
	}

	if ConfigStoreChainNameAndQualifyTypeToGroupId[chainName][qualifyType] == nil {
		ConfigStoreChainNameAndQualifyTypeToGroupId[chainName][qualifyType] = []string{}
	}
	// append groupId to ConfigStoreChainNameAndQualifyTypeToGroupId[chainName][qualifyType]
	ConfigStoreChainNameAndQualifyTypeToGroupId[chainName][qualifyType] = append(ConfigStoreChainNameAndQualifyTypeToGroupId[chainName][qualifyType], groupIdHex)
	// set ConfigStoreGroupIdToGroupConfig[groupId] = messageGroupMeta
	ConfigStoreGroupIdToGroupConfig[groupIdHex] = messageGroupMeta
	// if public, append to ConfigStorePublicGroupIds
	if isPublic {
		ConfigStorePublicGroupIds = append(ConfigStorePublicGroupIds, groupIdHex)
	}

	return nil
}

// log ConfigStoreGroupIdToGroupConfig
func (im *Manager) LogConfigStoreGroupIdToGroupConfig(logger *logger.Logger) {
	// loop ConfigStoreGroupIdToGroupConfig
	for groupIdHex, groupMeta := range ConfigStoreGroupIdToGroupConfig {
		jsonStr, err := json.Marshal(groupMeta)
		if err != nil {
			continue
		}
		logger.Infof("LogConfigStoreGroupIdToGroupConfig ... groupId:%s, groupMeta:%s", groupIdHex, jsonStr)
	}
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
