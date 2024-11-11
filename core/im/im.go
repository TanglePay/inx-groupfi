package im

import (
	"context"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func parseTokenQueryParam(c echo.Context) ([]byte, error) {
	tokenParams := c.QueryParams()["token"]
	if len(tokenParams) == 0 {
		return nil, nil
	}
	token, err := iotago.DecodeHex(tokenParams[0])
	if err != nil {
		return nil, err
	}
	return token, nil
}

// parse chainId
func parseChainIdQueryParam(c echo.Context) (uint32, error) {
	chainIdParams := c.QueryParams()["chainId"]
	if len(chainIdParams) == 0 {
		return 0, nil
	}
	chainId, err := strconv.Atoi(chainIdParams[0])
	if err != nil {
		return 0, err
	}
	return uint32(chainId), nil
}

// parse address from query param
func parseAddressQueryParam(c echo.Context) (string, error) {
	addressParams := c.QueryParams()["address"]
	if len(addressParams) == 0 {
		return "", echo.ErrBadRequest
	}
	address := addressParams[0]
	// to lower case
	if im.IsEvmAddress(address) {
		address = strings.ToLower(address)
	}
	return address, nil
}

// parse addresses from body
func parseAddressesFromBody(c echo.Context) ([]string, error) {
	var addresses []string
	err := c.Bind(&addresses)
	if err != nil {
		return nil, err
	}
	var lowerAddresses []string
	for _, address := range addresses {
		if im.IsEvmAddress(address) {
			address = strings.ToLower(address)
		}
		lowerAddresses = append(lowerAddresses, address)
	}
	return lowerAddresses, nil
}
func parseAddressQueryParamWithNil(c echo.Context) (string, error) {
	addressParams := c.QueryParams()["address"]
	if len(addressParams) == 0 {
		return "", nil
	}
	address := addressParams[0]
	// to lower case
	if im.IsEvmAddress(address) {
		address = strings.ToLower(address)
	}
	return address, nil
}

// parse tokenId from query param
func parseTokenIdQueryParam(c echo.Context) ([]byte, error) {
	tokenIdParams := c.QueryParams()["tokenId"]
	if len(tokenIdParams) == 0 {
		return nil, nil
	}
	tokenId, err := iotago.DecodeHex(tokenIdParams[0])
	if err != nil {
		return nil, err
	}
	return tokenId, nil
}
func parseGroupIdQueryParam(c echo.Context) ([]byte, error) {
	groupIdParams := c.QueryParams()["groupId"]
	if len(groupIdParams) == 0 {
		return nil, echo.ErrBadRequest
	}
	groupId, err := iotago.DecodeHex(groupIdParams[0])
	if err != nil {
		return nil, err
	}
	if len(groupId) != im.GroupIdLen {
		return nil, errors.Errorf("invalid groupId length: %d", len(groupId))
	}
	return groupId, nil
}

// parse outputIds from body
func parseOutputIdsFromBody(c echo.Context) ([]string, error) {
	return parseIdsFromBody(c)
}

// parse ids from body
func parseIdsFromBody(c echo.Context) ([]string, error) {
	var ids []string
	err := c.Bind(&ids)
	if err != nil {
		return nil, err
	}
	return ids, nil
}

// parse given attrName from query param
func parseAttrNameQueryParam(c echo.Context, attrName string) (string, error) {
	// use parseAttrNameQueryParamWithNil
	attr, err := parseAttrNameQueryParamWithNil(c, attrName)
	if err != nil {
		return "", err
	}
	if attr == "" {
		return "", echo.ErrBadRequest
	}
	return attr, nil
}

// parseAttrNameQueryParam with nil
func parseAttrNameQueryParamWithNil(c echo.Context, attrName string) (string, error) {
	attrParams := c.QueryParams()[attrName]
	if len(attrParams) == 0 {
		return "", nil
	}
	attr := attrParams[0]
	return attr, nil
}

// parseAttrNameQueryParam with default
func parseAttrNameQueryParamWithDefault(c echo.Context, attrName string, defaultVal string) (string, error) {
	// use parseAttrNameQueryParamWithNil
	attr, err := parseAttrNameQueryParamWithNil(c, attrName)
	if err != nil {
		return "", err
	}
	if attr == "" {
		return defaultVal, nil
	}
	return attr, nil
}

// parse groupName from query param
func parseGroupNameQueryParam(c echo.Context) (string, error) {
	groupNameParams := c.QueryParams()["groupName"]
	if len(groupNameParams) == 0 {
		return "", echo.ErrBadRequest
	}
	groupName := groupNameParams[0]
	return groupName, nil
}

const defaultSize = 5

func parseSizeQueryParam(c echo.Context) (int, error) {
	sizeParams := c.QueryParams()["size"]
	if len(sizeParams) == 0 {
		return defaultSize, nil
	}
	size, err := strconv.Atoi(sizeParams[0])
	if err != nil {
		return defaultSize, nil
	}
	return size, nil
}

// make inbox items response from inbox items
func makeInboxItemsResponse(items []im.InboxItem) *InboxItemsResponse {
	itemJsonList := make([]im.InboxItemJson, len(items))
	var token string
	for i, item := range items {
		token = iotago.EncodeHex(item.GetToken())
		itemJsonList[i] = item.Jsonable()
	}
	return &InboxItemsResponse{
		Items: itemJsonList,
		Token: token,
	}
}

// make public items response from inbox items
func makePublicItemsResponse(items []im.InboxItem) *PublicItemsResponse {
	itemJsonList := make([]im.InboxItemJson, len(items))
	var startToken string
	var endToken string
	for i, item := range items {
		if i == 0 {
			startToken = iotago.EncodeHex(item.GetToken())
		}
		if i == len(items)-1 {
			endToken = iotago.EncodeHex(item.GetToken())
		}
		itemJsonList[i] = item.Jsonable()
	}
	return &PublicItemsResponse{
		Items:      itemJsonList,
		StartToken: startToken,
		EndToken:   endToken,
	}
}

// make address group details response from address group
func makeAddressGroupDetailsResponse(addressGroup *im.AddressGroup) *AddressGroupDetailsResponse {
	return &AddressGroupDetailsResponse{
		GroupId:          iotago.EncodeHex(addressGroup.GroupId),
		GroupName:        addressGroup.GroupName,
		GroupQualifyType: addressGroup.GroupQualifyType,
		IpfsLink:         addressGroup.NftLink,
		TokenId:          iotago.EncodeHex(addressGroup.TokenId),
		TokenThres:       addressGroup.TokenThres,
	}
}

// get raw nfts from groupId
func getRawNFTsFromGroupId(c echo.Context) ([]*im.NFT, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	return deps.IMManager.GetRawNFTsFromGroupIdImpl(groupId, CoreComponent.Logger())
}

// get raw member in nfts from groupId
func getRawMemberInNFTsFromGroupId(c echo.Context) ([]*im.NFT, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	addresses, err := deps.IMManager.GetGroupMemberAddressesFromGroupId(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	// map addresses to nfts, nft should be created with owner address only
	nfts := make([]*im.NFT, len(addresses))
	for i, address := range addresses {
		nfts[i] = &im.NFT{
			OwnerAddress: []byte(address),
		}
	}
	return nfts, nil
}

// get nfts
func getNFTsFromGroupId(c echo.Context) ([]*im.NFTResponse, error) {
	nfts, err := getRawNFTsFromGroupId(c)
	if err != nil {
		return nil, err
	}
	nftResponseArr := make([]*im.NFTResponse, len(nfts))
	for i, nft := range nfts {
		// nft.OwnerAddress is []bytes{OwnerAddress}
		nftResponseArr[i] = &im.NFTResponse{
			NFTId:        iotago.EncodeHex(nft.NFTId),
			OwnerAddress: string(nft.OwnerAddress),
		}
	}
	return nftResponseArr, nil
}

// getNFTsWithPublicKeyFromGroupId
func getNFTsWithPublicKeyFromGroupId(c echo.Context, drainer *im.ItemDrainer) ([]*im.NFTResponse, error) {
	nfts, err := getRawMemberInNFTsFromGroupId(c)
	if err != nil {
		return nil, err
	}
	return deps.IMManager.FullfillNFTsWithPublickKey(nfts, drainer, CoreComponent.Logger())
}

// get shared from groupId
func getSharedFromGroupId(c echo.Context) (*SharedResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	groupIdFixed := [im.GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	isPublic := deps.IMManager.GetIsGroupPublic(groupIdFixed)
	if isPublic {
		// http code 901
		return nil, echo.NewHTTPError(901, "public group has no shared")
	}
	shared, err := deps.IMManager.ReadSharedFromGroupId(groupIdFixed)
	if err != nil {
		return nil, err
	}
	if shared == nil {
		resp := &SharedResponse{
			OutputId: "",
		}
		return resp, nil
	}
	CoreComponent.LogInfof("get shared from groupId:%s,found shared with outputid:%s", groupId, iotago.EncodeHex(shared.OutputId[:]))
	resp := &SharedResponse{
		OutputId: iotago.EncodeHex(shared.OutputId[:]),
	}
	return resp, nil
}

type SharedResponseV2 struct {
	Code     int    `json:"code"`
	Message  string `json:"message,omitempty"`
	OutputId string `json:"outputId,omitempty"`
}

// IsGroupPublicResponse
type IsGroupPublicResponse struct {
	GroupId  string `json:"groupId"`
	IsPublic bool   `json:"isPublic"`
}

func getSharedFromGroupIdV2(c echo.Context) (*SharedResponseV2, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	groupIdFixed := [im.GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	isPublic := deps.IMManager.GetIsGroupPublic(groupIdFixed)
	if isPublic {
		resp := &SharedResponseV2{
			Code:    901,
			Message: "Public group has no shared",
		}
		return resp, nil
	}
	shared, err := deps.IMManager.ReadSharedFromGroupId(groupIdFixed)
	if err != nil {
		return nil, err
	}
	if shared == nil {
		resp := &SharedResponseV2{
			Code:     0,
			OutputId: "",
		}
		return resp, nil
	}
	CoreComponent.LogInfof("get shared from groupId:%s, found shared with outputid:%s", groupId, iotago.EncodeHex(shared.OutputId[:]))
	resp := &SharedResponseV2{
		Code:     0,
		OutputId: iotago.EncodeHex(shared.OutputId[:]),
	}
	return resp, nil
}

// batchFetchGroupIsPublic
func batchFetchGroupIsPublic(c echo.Context) ([]*IsGroupPublicResponse, error) {
	var groupIds []string
	err := c.Bind(&groupIds)
	if err != nil {
		return nil, err
	}
	// loop groupIds, get isPublic
	var resp []*IsGroupPublicResponse
	for _, groupId := range groupIds {
		groupIdBytes, err := iotago.DecodeHex(groupId)
		if err != nil {
			continue
		}
		var groupIdFixed [im.GroupIdLen]byte
		copy(groupIdFixed[:], groupIdBytes)
		isPublic := deps.IMManager.GetIsGroupPublic(groupIdFixed)
		resp = append(resp, &IsGroupPublicResponse{
			GroupId:  groupId,
			IsPublic: isPublic,
		})
	}
	return resp, nil
}

// delete shared from groupId
func deleteSharedFromGroupId(c echo.Context) error {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return err
	}
	CoreComponent.LogInfof("delete shared from group:%s", groupId)
	groupIdFixed := [im.GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	err = deps.IMManager.DeleteSharedFromGroupId(groupIdFixed)
	if err != nil {
		return err
	}
	return nil
}

// get all groupIds from address
func getGroupIdsFromAddress(c echo.Context) ([]string, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	var groupParam GroupParam
	hasGroupParam := true
	err = c.Bind(&groupParam)
	if err != nil {
		hasGroupParam = false
		CoreComponent.LogWarnf("getGroupIdsFromAddress ... Bind failed:%s", err)
	}
	CoreComponent.LogInfof("get groupIds from address:%s", address)
	isEvmAddress := im.IsEvmAddress(address)
	// if isEvmAddress, return all groupIds
	if isEvmAddress {
		addressSha256 := im.Sha256HashAddress(address)
		var addressSha256Fixed [im.Sha256HashLen]byte
		copy(addressSha256Fixed[:], addressSha256)
		groupIds, err := deps.IMManager.GetGroupIdsFromAddress(addressSha256)
		if err != nil {
			return nil, err
		}
		// filter groupIds, either joined or public
		var groupIdsToSubsribe []string
		for _, groupId := range groupIds {
			var groupIdFixed [im.GroupIdLen]byte
			copy(groupIdFixed[:], groupId)
			groupIdHex := iotago.EncodeHex(groupIdFixed[:])
			isMember, err := deps.IMManager.GroupMemberExistsFromGroupIdAndAddressSha256Hash(groupIdFixed, addressSha256Fixed)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("getGroupIdsFromAddress ... GroupMemberExistsFromGroupIdAndAddressSha256Hash failed:%s", err)
				continue
			}
			if isMember || im.CheckIfGroupIdIsPublic(groupIdFixed, deps.IMManager) {
				groupIdsToSubsribe = append(groupIdsToSubsribe, groupIdHex)
			}
		}
		if hasGroupParam {
			publicGroupIdsFromInlcudes := groupParamToGroupIds(groupParam, true)
			// MergeAndRemoveDupsStringArray
			groupIdsToSubsribe = im.MergeAndRemoveDupsStringArray(groupIdsToSubsribe, publicGroupIdsFromInlcudes)
		}
		return groupIdsToSubsribe, nil
	}
	addressSha256 := im.Sha256HashAddress(address)
	groupIds, err := deps.IMManager.GetGroupIdsFromAddress(addressSha256)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get groupIds from address:%s,found groupIds:%d", address, len(groupIds))
	publicGroupIds := []string{}
	groupIdStrArr := []string{}
	seen := map[string]bool{}
	for _, groupId := range groupIds {
		groupIdHex := iotago.EncodeHex(groupId)
		if _, ok := seen[groupIdHex]; !ok {
			groupIdStrArr = append(groupIdStrArr, groupIdHex)
			seen[groupIdHex] = true
		}
	}
	for _, groupId := range publicGroupIds {
		if _, ok := seen[groupId]; !ok {
			groupIdStrArr = append(groupIdStrArr, groupId)
			seen[groupId] = true
		}
	}
	if hasGroupParam {
		groupIdStrArr = filterGroupIdsFromGroupParam(groupIdStrArr, groupParam)
	}

	return groupIdStrArr, nil
}

// getGroupIdsFromAddressV2
func getGroupIdsFromAddressV2(c echo.Context) ([]string, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	var groupParam GroupParam
	hasGroupParam := true
	err = c.Bind(&groupParam)
	if err != nil {
		hasGroupParam = false
		CoreComponent.LogWarnf("getGroupIdsFromAddress ... Bind failed:%s", err)
	}
	CoreComponent.LogInfof("get groupIds from address:%s", address)
	isEvmAddress := im.IsEvmAddress(address)
	// if isEvmAddress, return all groupIds
	if isEvmAddress {
		groupIds := []string{}
		if hasGroupParam {
			groupIds = filterGroupIdsFromGroupParam(groupIds, groupParam)
		}
		return groupIds, nil
	}
	addressSha256 := im.Sha256HashAddress(address)
	groupIds, err := deps.IMManager.GetGroupIdsFromAddress(addressSha256)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get groupIds from address:%s,found groupIds:%d", address, len(groupIds))
	publicGroupIds := []string{}
	groupIdStrArr := []string{}
	seen := map[string]bool{}
	for _, groupId := range groupIds {
		groupIdHex := iotago.EncodeHex(groupId)
		if _, ok := seen[groupIdHex]; !ok {
			groupIdStrArr = append(groupIdStrArr, groupIdHex)
			seen[groupIdHex] = true
		}
	}
	for _, groupId := range publicGroupIds {
		if _, ok := seen[groupId]; !ok {
			groupIdStrArr = append(groupIdStrArr, groupId)
			seen[groupId] = true
		}
	}
	if hasGroupParam {
		groupIdStrArr = filterGroupIdsFromGroupParam(groupIdStrArr, groupParam)
	}

	return groupIdStrArr, nil
}

type GroupData struct {
	GroupId string `json:"groupId"`
}

type GroupParam struct {
	Includes []GroupData `json:"includes"`
	Excludes []GroupData `json:"excludes"`
}
type DappGroupQuery struct {
	ChainId         int    `json:"chainId"`
	ContractAddress string `json:"contractAddress"`
}
type DappGroupConfig struct {
	GroupName string `json:"groupName"`
	GroupId   string `json:"groupId"`
}

// group param to groupId map
func groupParamToGroupIds(groupParam GroupParam, isPublicOnly bool) []string {
	var groupIds []string
	for _, include := range groupParam.Includes {
		dappGroupId := include.GroupId
		groupId, err := im.ReadGroupIdFromDappGroupId(dappGroupId, deps.IMManager)
		if err != nil {
			continue
		}
		shouldAppend := true
		if isPublicOnly {
			isGroupPublic := im.CheckIfGroupIdIsPublic(groupId, deps.IMManager)
			if !isGroupPublic {
				shouldAppend = false
			}
		}
		if shouldAppend {
			groupIds = append(groupIds, iotago.EncodeHex(groupId[:]))
		}
	}
	return groupIds
}

// filter groupIds from group param
func filterGroupIdsFromGroupParam(groupIds []string, groupParam GroupParam) []string {
	includeGroupNameMap := map[string]bool{}
	if len(groupParam.Includes) > 0 {
		for _, include := range groupParam.Includes {
			key := include.GroupId
			includeGroupNameMap[key] = true
		}
	}
	excludeGroupNameMap := map[string]bool{}
	if len(groupParam.Excludes) > 0 {
		for _, exclude := range groupParam.Excludes {
			key := exclude.GroupId
			excludeGroupNameMap[key] = true
		}
	}
	// map int -> string
	filteredGroupIds := make(map[int]string)
	for idx, groupId := range groupIds {
		groupIdBytes, err := iotago.DecodeHex(groupId)
		if err != nil {
			continue
		}
		var groupIdFixed [im.GroupIdLen]byte
		copy(groupIdFixed[:], groupIdBytes)
		config, err := im.ReadGroupConfigMetaFromGroupId(groupIdFixed, deps.IMManager)
		if err != nil {
			continue
		}
		if config == nil {
			continue
		}
		dappGroupId := im.GetDappGroupId(groupId, config)
		if (len(includeGroupNameMap) > 0) && (!includeGroupNameMap[dappGroupId]) {
			continue
		}
		if (len(excludeGroupNameMap) > 0) && (excludeGroupNameMap[dappGroupId]) {
			continue
		}
		filteredGroupIds[idx] = groupId
	}
	// sort filteredGroupIds by key, return values
	var keys []int
	for k := range filteredGroupIds {
		keys = append(keys, k)
	}
	// sort keys
	sort.Ints(keys)

	var sortedGroupIds []string
	for _, k := range keys {
		sortedGroupIds = append(sortedGroupIds, filteredGroupIds[k])
	}
	return sortedGroupIds
}

// getQualifiedGroupConfigsFromAddress
func getQualifiedGroupConfigsFromAddress(c echo.Context) ([]*im.MessageGroupMetaJSON, error) {
	var groupParam GroupParam
	err := c.Bind(&groupParam)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("getQualifiedGroupConfigsFromAddress ... Bind failed:%s", err)
	}
	CoreComponent.LogInfof("get qualified group configs from address:%s", groupParam)
	var groupConfigs []*im.MessageGroupMetaJSON
	for _, groupData := range groupParam.Includes {
		dappGroupId := groupData.GroupId
		groupId, err := im.ReadGroupIdFromDappGroupId(dappGroupId, deps.IMManager)
		if err != nil {
			continue
		}
		groupConfig, err := im.ReadGroupConfigMetaFromGroupId(groupId, deps.IMManager)
		if err != nil {
			continue
		}
		if groupConfig == nil {
			continue
		}
		groupConfigs = append(groupConfigs, groupConfig)
	}
	return groupConfigs, nil
}

// getPublicGroupConfigs
func getPublicGroupConfigs(c echo.Context) ([]*im.MessageGroupMetaJSON, error) {
	var groupParam GroupParam
	err := c.Bind(&groupParam)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("getPublicGroupConfigs ... Bind failed:%s", err)
		return nil, err
	}
	// if groupParam is empty, return nil
	if len(groupParam.Includes) == 0 && len(groupParam.Excludes) == 0 {
		return nil, nil
	}

	publicGroupIds := groupParamToGroupIds(groupParam, true)
	// loop groupIdHexList
	var groupConfigs []*im.MessageGroupMetaJSON
	for _, groupIdHex := range publicGroupIds {
		config := deps.IMManager.GroupIdToGroupConfig(groupIdHex)
		// if config is not nil, append to groupConfigs
		groupConfigs = append(groupConfigs, config)
	}
	return groupConfigs, nil
}

// getMarkedGroupConfigs
func getMarkedGroupConfigs(c echo.Context) ([]*im.MessageGroupMetaJSON, error) {
	// get address from query param
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	marks, err := deps.IMManager.GetMarksFromAddress(address, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	// loop marks, get groupIdHexList
	var groupIdHexList []string
	for _, mark := range marks {
		groupIdHexList = append(groupIdHexList, iotago.EncodeHex(mark.GroupId[:]))
	}
	// loop groupIdHexList, get groupConfigs
	var groupConfigs []*im.MessageGroupMetaJSON
	for _, groupIdHex := range groupIdHexList {
		config := deps.IMManager.GroupIdToGroupConfig(groupIdHex)
		// if config is not nil, append to groupConfigs
		if config != nil {
			groupConfigs = append(groupConfigs, config)
		}
	}
	return groupConfigs, nil
}

// getForMeGroupConfigs
func getForMeGroupConfigs(c echo.Context) ([]*im.MessageGroupMetaJSONPlus, error) {
	var groupParam GroupParam
	err := c.Bind(&groupParam)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("getForMeGroupConfigs ... Bind failed:%s", err)
		return nil, err
	}
	// check if groupParam is empty
	if len(groupParam.Includes) == 0 && len(groupParam.Excludes) == 0 {
		return nil, nil
	}
	// all groupIds
	groupIdHexList := groupParamToGroupIds(groupParam, false)
	// loop groupIdHexList
	var groupConfigs []*im.MessageGroupMetaJSONPlus
	for _, groupIdHex := range groupIdHexList {
		groupIdBytes, err := iotago.DecodeHex(groupIdHex)
		if err != nil {
			continue
		}
		var groupIdFixed [im.GroupIdLen]byte
		copy(groupIdFixed[:], groupIdBytes)
		config, err := im.ReadGroupConfigMetaFromGroupId(groupIdFixed, deps.IMManager)
		if err != nil {
			continue
		}
		// if config is nil continue
		if config == nil {
			continue
		}
		isPublic := deps.IMManager.GetIsGroupPublicWithGroupId(groupIdHex)

		plusConfig := &im.MessageGroupMetaJSONPlus{
			MessageGroupMetaJSON: *config,
			IsPublic:             isPublic,
		}
		groupConfigs = append(groupConfigs, plusConfig)
	}
	return groupConfigs, nil
}

// getDappQueryGroupConfigs
func getDappQueryGroupConfigs(c echo.Context) ([]*DappGroupConfig, error) {
	var dappGroupQuery DappGroupQuery
	err := c.Bind(&dappGroupQuery)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("getDappQueryGroupConfigs ... Bind failed:%s", err)
		return nil, err
	}
	CoreComponent.LogInfof("get dapp query group configs from chainId:%d,contractAddress:%s", dappGroupQuery.ChainId, dappGroupQuery.ContractAddress)
	// get all groupIds
	groupIds, err := im.ReadAllGroupIdFromChainIdAndContractAddressHash(uint32(dappGroupQuery.ChainId), dappGroupQuery.ContractAddress, deps.IMManager)
	if err != nil {
		return nil, err
	}
	// loop groupIds, get groupConfigs
	var groupConfigs []*DappGroupConfig
	for _, groupId := range groupIds {
		config, err := im.ReadGroupConfigMetaFromGroupId(groupId, deps.IMManager)
		if err != nil {
			continue
		}
		// if config is nil, continue
		if config == nil {
			continue
		}
		groupIdHex := iotago.EncodeHex(groupId[:])
		groupConfigs = append(groupConfigs, &DappGroupConfig{
			GroupName: config.GroupName,
			GroupId:   im.GetDappGroupId(groupIdHex, config),
		})
	}
	return groupConfigs, nil
}

// getAddressGroupDetails
func getAddressGroupDetails(c echo.Context) ([]*AddressGroupDetailsResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address group details from address:%s", address)
	addressSha256 := im.Sha256HashAddress(address)
	groupDetails, err := deps.IMManager.GetAddressGroupFromAddress(addressSha256)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address group details from address:%s,found groupIds:%d", address, len(groupDetails))
	var AddressGroupDetailsResponseArr []*AddressGroupDetailsResponse
	for _, groupDetail := range groupDetails {
		AddressGroupDetailsResponseArr = append(AddressGroupDetailsResponseArr, makeAddressGroupDetailsResponse(groupDetail))
	}
	return AddressGroupDetailsResponseArr, nil
}

// get qualified address for a groupid
func getQualifiedAddressesForGroupId(c echo.Context) ([]string, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get qualified address for groupId:%s", iotago.EncodeHex(groupId))
	groupIdFixed := [im.GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	qualifications, err := deps.IMManager.GetAllGroupQualificationsFromGroupId(groupIdFixed, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	// nfts to addresses
	var addresses []string
	for _, qualification := range qualifications {
		addresses = append(addresses, qualification.Address)
	}
	CoreComponent.LogInfof("get qualified address for groupId:%s,found addresses:%d", iotago.EncodeHex(groupId), len(addresses))
	return addresses, nil
}

// isAddressQualifiedGroup
func isAddressQualifiedGroup(c echo.Context) (bool, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return false, err
	}
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return false, err
	}
	CoreComponent.LogInfof("is address qualified group from groupId:%s,address:%s", iotago.EncodeHex(groupId), address)
	groupIdFixed := [im.GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)
	qualified, err := deps.IMManager.GroupQualificationExists(groupIdFixed, address, CoreComponent.Logger())
	if err != nil {
		return false, err
	}
	CoreComponent.LogInfof("is address qualified group from groupId:%s,address:%s,qualified:%t", iotago.EncodeHex(groupId), address, qualified)
	return qualified, nil
}

// get all marked addresses from groupId
func getMarkedAddressesFromGroupId(c echo.Context) ([]string, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get marks from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	marks, err := deps.IMManager.GetMarksFromGroupId(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	// marks to addresses
	var addresses []string
	for _, mark := range marks {
		addresses = append(addresses, mark.Address)
	}
	CoreComponent.LogInfof("get marks from groupId:%s,found addresses:%d", iotago.EncodeHex(groupId), len(addresses))
	return addresses, nil
}

// getQualifiedAddressPublicKeyPairsForGroupId
func getQualifiedAddressPublicKeyPairsForGroupId(c echo.Context) ([]*im.NFTResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get qualified address public key pairs for groupId:%s", iotago.EncodeHex(groupId))
	addresses, err := getQualifiedAddressesForGroupId(c)
	if err != nil {
		return nil, err
	}
	// map addresses to nfts, nft should be created with owner address only
	var resp []*im.NFTResponse
	for _, address := range addresses {
		publicKey, err := deps.IMManager.ReadOnePublicKey(address)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("getQualifiedAddressPublicKeyPairsForGroupId ReadOnePublicKey failed:%s", err)
			continue
		}
		var publicKeyHex string
		if publicKey != nil {
			publicKeyHex = iotago.EncodeHex(publicKey)
		}
		resp = append(resp, &im.NFTResponse{
			OwnerAddress: address,
			PublicKey:    publicKeyHex,
		})
	}
	return resp, nil
}

// get all group member addresses from groupId
func getGroupMembersFromGroupId(c echo.Context) ([]*im.NFTResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group member addresses from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	groupmembers, err := deps.IMManager.GetGroupMembers(groupId32)
	if err != nil {
		return nil, err
	}
	// log first address and length
	var firstAddress string
	if len(groupmembers) > 0 {
		firstAddress = groupmembers[0].Address
	} else {
		return nil, nil
	}
	CoreComponent.LogInfof("get group member addresses from groupId:%s,found addresses:%d, first address:%s", iotago.EncodeHex(groupId), len(groupmembers), firstAddress)
	// check if first address evm address
	var isEvmAddress bool
	if len(groupmembers) > 0 {
		isEvmAddress = im.IsEvmAddress(groupmembers[0].Address)
	}
	if isEvmAddress {
		var resp []*im.NFTResponse
		for _, groupmember := range groupmembers {
			publicKey, err := deps.IMManager.ReadOnePublicKey(groupmember.Address)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("getGroupMembersFromGroupId ReadOnePublicKey failed:%s", err)
				continue
			}
			var publicKeyHex string
			if publicKey != nil {
				publicKeyHex = iotago.EncodeHex(publicKey)
			}
			resp = append(resp, &im.NFTResponse{
				OwnerAddress: groupmember.Address,
				PublicKey:    publicKeyHex,
				Timestamp:    groupmember.Timestamp,
			})
		}
		return resp, nil
	} else {
		// map addresses to nfts, nft should be created with owner address only
		nfts := make([]*im.NFT, len(groupmembers))
		var maxTimestamp uint32
		for i, groupmember := range groupmembers {
			nfts[i] = &im.NFT{
				OwnerAddress:       []byte(groupmember.Address),
				MileStoneTimestamp: groupmember.Timestamp,
			}
			if groupmember.Timestamp > maxTimestamp {
				maxTimestamp = groupmember.Timestamp
			}
		}
		CoreComponent.LogInfof("get group member addresses from groupId:%s,found addresses:%d, maxTimestamp:%d", iotago.EncodeHex(groupId), len(nfts), maxTimestamp)
		resp, err := deps.IMManager.FullfillNFTsWithPublickKey(nfts, im.PublicKeyDrainer, CoreComponent.Logger())
		if err != nil {
			return nil, err
		}
		return resp, nil
	}
}

// get all group votes from groupId
func getGroupVotes(c echo.Context) ([]*VoteResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group votes from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	votes, err := deps.IMManager.GetAllVotesFromGroupId(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group votes from groupId:%s,found votes:%d", iotago.EncodeHex(groupId), len(votes))
	voteResponseArr := make([]*VoteResponse, len(votes))
	for i, vote := range votes {
		voteResponseArr[i] = &VoteResponse{
			GroupId:           iotago.EncodeHex(vote.GroupId[:]),
			AddressSha256Hash: iotago.EncodeHex(vote.AddressSha256[:]),
			Vote:              int(vote.Vote),
		}
	}
	return voteResponseArr, nil
}

// getAddressVotes
func getAddressVotes(c echo.Context) ([]*VoteResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address votes from address:%s", address)
	addressSha256 := im.Sha256HashFixedAddress(address)
	votes, err := deps.IMManager.GetAllVotesFromAddressSha256Hash(addressSha256, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address votes from address:%s,found votes:%d", address, len(votes))
	voteResponseArr := make([]*VoteResponse, len(votes))
	for i, vote := range votes {
		voteResponseArr[i] = &VoteResponse{
			GroupId: iotago.EncodeHex(vote.GroupId[:]),
			Vote:    int(vote.Vote),
		}
	}
	return voteResponseArr, nil
}

// getAddressMutes
func getAddressMutes(c echo.Context) ([]*MuteResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	addressSha256 := im.Sha256HashFixedAddress(address)
	mutes, err := deps.IMManager.GetAllMuteGroupMembersFromAddress(addressSha256, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address mutes from address:%s,found mutes:%d", address, len(mutes))
	muteResponseArr := make([]*MuteResponse, len(mutes))
	for i, mute := range mutes {
		muteResponseArr[i] = &MuteResponse{
			GroupId:                iotago.EncodeHex(mute.GroupId[:]),
			MutedAddressSha256Hash: iotago.EncodeHex(mute.MutedAddrSha256Hash[:]),
		}
	}
	return muteResponseArr, nil
}

// getAddressLikes
func getAddressLikes(c echo.Context) ([]*LikeResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address likes from address:%s", address)
	addressSha256 := im.Sha256HashFixedAddress(address)
	likes, err := deps.IMManager.GetAllLikeGroupMembersFromAddress(addressSha256, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address likes from address:%s,found likes:%d", address, len(likes))
	likeResponseArr := make([]*LikeResponse, len(likes))
	for i, like := range likes {
		likeResponseArr[i] = &LikeResponse{
			GroupId:                iotago.EncodeHex(like.GroupId[:]),
			LikedAddressSha256Hash: iotago.EncodeHex(like.LikedAddrSha256Hash[:]),
		}
	}
	return likeResponseArr, nil
}

// getGroupVotesCount
func getGroupVotesCount(c echo.Context) (*VoteCountResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group votes count from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	publicCt, privateCt, err := deps.IMManager.CountVotesForGroup(groupId32)
	if err != nil {
		return nil, err
	}
	memberCt, err := deps.IMManager.GetGroupMemberAddressesCountFromGroupId(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	resp := &VoteCountResponse{
		PublicCount:  publicCt,
		PrivateCount: privateCt,
		MemberCount:  memberCt,
		GroupId:      iotago.EncodeHex(groupId),
	}
	return resp, nil
}

// getGroupBlacklist
func getGroupBlacklist(c echo.Context) ([]string, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group blacklist from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	blacklist, err := deps.IMManager.GetAddresseHashsFromGroupBlacklist(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group blacklist from groupId:%s,found blacklist:%d", iotago.EncodeHex(groupId), len(blacklist))
	return blacklist, nil
}

func getAddressMemberGroups(c echo.Context) ([]string, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address member groups from address:%s", address)
	groupIds, err := deps.IMManager.GetMemberGroups(address)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address member groups from address:%s,found groupIds:%d", address, len(groupIds))
	return groupIds, nil
}
func getAddressMarkGroups(c echo.Context) ([]string, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	marks, err := getAddressMarkGroupMarks(address)
	if err != nil {
		return nil, err
	}
	groupIds := make([]string, len(marks))
	for i, mark := range marks {
		groupIds[i] = iotago.EncodeHex(mark.GroupId[:])
	}
	CoreComponent.LogInfof("get address mark groups from address:%s,found groupIds:%d", address, len(groupIds))
	return groupIds, nil
}

// getAddressMarkGroupDetails
func getAddressMarkGroupDetails(c echo.Context) ([]*AddressGroupDetailsResponseLite, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	marks, err := getAddressMarkGroupMarks(address)
	if err != nil {
		return nil, err
	}
	groupDetails := make([]*AddressGroupDetailsResponseLite, len(marks))
	for i, mark := range marks {
		groupDetails[i] = &AddressGroupDetailsResponseLite{
			GroupId:   iotago.EncodeHex(mark.GroupId[:]),
			Timestamp: mark.MilestoneTimestamp,
		}
	}
	return groupDetails, nil
}

// getAddressMarkGroupMarks
func getAddressMarkGroupMarks(address string) ([]*im.Mark, error) {
	CoreComponent.LogInfof("get address mark group marks from address:%s", address)
	marks, err := deps.IMManager.GetMarksFromAddress(address, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	isEvmAddress := im.IsEvmAddress(address)
	var filteredMarks []*im.Mark
	for _, mark := range marks {
		canAppend := true
		if isEvmAddress {
			groupConfig, err := im.ReadGroupConfigMetaFromGroupId(mark.GroupId, deps.IMManager)

			if err != nil || groupConfig == nil || groupConfig.ChainId == im.HornetChainId {
				canAppend = false
			}
		}
		if canAppend {
			filteredMarks = append(filteredMarks, mark)
		}
	}
	return filteredMarks, nil
}

// getGroupUserReputation
func getGroupUserReputation(c echo.Context) ([]*GroupUserReputationResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group user reputation from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	reputations, err := deps.IMManager.GetGroupAllUsersReputation(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group user reputation from groupId:%s,found reputations:%d", iotago.EncodeHex(groupId), len(reputations))
	reputationResponseArr := make([]*GroupUserReputationResponse, len(reputations))
	for i, reputation := range reputations {
		reputationResponseArr[i] = &GroupUserReputationResponse{
			GroupId:           iotago.EncodeHex(reputation.GroupId[:]),
			AddressSha256Hash: iotago.EncodeHex(reputation.AddrSha256Hash[:]),
			Reputation:        reputation.Reputation,
		}
	}
	return reputationResponseArr, nil
}

// get user group reputation
func getUserGroupReputation(c echo.Context) (*GroupUserReputationResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get user group reputation from groupId:%s,address:%s", iotago.EncodeHex(groupId), address)
	var groupId32 [32]byte
	copy(groupId32[:], groupId)

	reputation, err := deps.IMManager.GetUserGroupReputation(groupId32, address, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}

	var score float32
	score = 100
	if reputation != nil {
		score = reputation.Reputation
	}
	addressSha256Hash := im.Sha256HashAddress(address)
	CoreComponent.LogInfof("get user group reputation from groupId:%s,address:%s,score is:%f", iotago.EncodeHex(groupId), address, score)
	resp := &GroupUserReputationResponse{
		GroupId:           iotago.EncodeHex(groupId),
		AddressSha256Hash: iotago.EncodeHex(addressSha256Hash),
		Reputation:        score,
	}
	return resp, nil
}

// get inbox message
func getInboxList(c echo.Context) (*InboxItemsResponse, error) {
	// get address
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	// get continue token
	token, err := parseTokenQueryParam(c)
	if err != nil {
		return nil, err
	}
	// get size, default 10
	size, err := parseSizeQueryParam(c)
	if err != nil {
		return nil, err
	}
	// get inbox message
	inboxItems, err := deps.IMManager.ReadInbox(im.Sha256HashAddress(address), token, size, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	// make inbox message response
	inboxItemsResponse := makeInboxItemsResponse(inboxItems)
	return inboxItemsResponse, nil
}

// getPublicItems
func getPublicItems(c echo.Context) (*PublicItemsResponse, error) {
	startTokenStr, err := parseAttrNameQueryParamWithNil(c, "startToken")
	if err != nil {
		return nil, err
	}
	var startToken []byte
	if startTokenStr != "" {
		startToken, err = iotago.DecodeHex(startTokenStr)
		if err != nil {
			return nil, err
		}
	}
	endTokenStr, err := parseAttrNameQueryParamWithNil(c, "endToken")
	if err != nil {
		return nil, err
	}
	var endToken []byte
	if endTokenStr != "" {
		endToken, err = iotago.DecodeHex(endTokenStr)
		if err != nil {
			return nil, err
		}
	}
	// direction, use parseAttrNameQueryParamWithDefault, default is "head"
	direction, err := parseAttrNameQueryParamWithDefault(c, "direction", "head")
	if err != nil {
		return nil, err
	}
	isReverse := false
	if direction == "tail" {
		isReverse = true
	}
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	size, err := parseSizeQueryParam(c)
	if err != nil {
		return nil, err
	}
	items, err := deps.IMManager.ReadPublicItemsFromGroupId(groupId, startToken, endToken, size, isReverse, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	resp := makePublicItemsResponse(items)
	return resp, nil
}

// getAddressesDids given addresses
func getAddressesDids(addresses []string) ([]*DidAddressResponse, error) {
	respList := make([]*DidAddressResponse, len(addresses))
	for i, address := range addresses {
		dids, err := deps.IMManager.GetDidsFromAddress(address)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("get addresses dids from addresses:%s failed:%s", addresses, err)
			continue
		}
		// find one with earliest timestamp, then append to respList
		var earliestDid *im.Did
		for _, did := range dids {
			if earliestDid == nil {
				earliestDid = did
				continue
			}
			if did.Timestamp < earliestDid.Timestamp {
				earliestDid = did
			}
		}
		if earliestDid == nil {
			// log error then continue
			CoreComponent.LogWarnf("get addresses dids from addresses:%s failed:earliestDid is nil", addresses)
			continue
		}
		respList[i] = &DidAddressResponse{
			Address: address,
			Name:    earliestDid.Name,
			Picture: earliestDid.Picture,
		}
	}
	return respList, nil
}

// getEvmAddressPair, given address
func getEvmAddressPair(address string) (*EvmAddressPairResponse, error) {
	pairX, err := deps.IMManager.GetPairXFromEvmAddress(address)
	if err != nil {
		return nil, err
	}
	if pairX == nil {
		return nil, nil
	}
	mmProxyAddress, tpProxyAddress, err := deps.IMManager.GetPairXProxyAddressFromEvmAddress(address)
	if err != nil {
		return nil, err
	}
	resp := &EvmAddressPairResponse{
		PublicKey:      pairX.PublicKey,
		PrivateKey:     pairX.PrivateKey,
		MMProxyAddress: mmProxyAddress,
		TPProxyAddress: tpProxyAddress,
	}
	return resp, nil
}

// getProfileByEvmAddress, given an EVM address
func getProfileByEvmAddress(c echo.Context) (*ProfileResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	// Retrieve profiles associated with the EVM address
	profile, err := deps.IMManager.GetProfileFromAddress(address)
	if err != nil {
		return nil, err
	}

	// If no profiles are found, return nil
	if profile == nil {
		return nil, nil
	}

	// Construct and return the response with the first profile
	resp := &ProfileResponse{
		Data:     profile.JsonData,
		OutputId: iotago.EncodeHex(profile.OutputId[:]),
	}

	return resp, nil
}

// batchProfileByEvmAddress
func batchProfileByEvmAddress(c echo.Context) ([]*ProfileResponse, error) {
	addresses, err := parseAddressesFromBody(c)
	if err != nil {
		return nil, err
	}
	// Retrieve profiles associated with the EVM addresses
	var profiles []*im.Profile
	for _, address := range addresses {
		profile, err := deps.IMManager.GetProfileFromAddress(address)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("batch profile by evm address from addresses:%s failed:%s", addresses, err)
			continue
		}
		if profile != nil {
			profiles = append(profiles, profile)
		}
	}

	// Construct and return the response with the first profile
	var resp []*ProfileResponse
	for _, profile := range profiles {
		resp = append(resp, &ProfileResponse{
			Address:  profile.Address,
			Data:     profile.JsonData,
			OutputId: iotago.EncodeHex(profile.OutputId[:]),
		})
	}

	return resp, nil
}

// batchSmrAddressToEvmAddress
func batchSmrAddressToEvmAddress(c echo.Context) ([]string, error) {
	addresses, err := parseAddressesFromBody(c)
	if err != nil {
		return nil, err
	}
	evmAddresses := make([]string, len(addresses))
	for i, address := range addresses {
		evmAddress, err := deps.IMManager.GetPairXEvmAddressFromProxyAddress(address)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("batch smr address to evm address from addresses:%s failed:%s", addresses, err)
			continue
		}
		evmAddresses[i] = evmAddress
	}
	return evmAddresses, nil
}

// listGroupConfigs
func listGroupConfigsLitev2(c echo.Context) (map[string]interface{}, error) {
	// get param include chainId uint32, contractAddress string, page int, pageSize int
	// chainId and contract address are optional
	// page and pageSize are optional and default to 1 and 10
	chainIdStr, err := parseAttrNameQueryParamWithNil(c, "chainId")
	if err != nil {
		return nil, err
	}
	var chainId uint32 = math.MaxUint32
	if chainIdStr != "" {
		chainId64, err := strconv.ParseUint(chainIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		chainId = uint32(chainId64)
	}
	contractAddress, err := parseAttrNameQueryParamWithNil(c, "contractAddress")
	if err != nil {
		return nil, err
	}
	pageStr, err := parseAttrNameQueryParamWithDefault(c, "page", "1")
	if err != nil {
		return nil, err
	}
	page, err := strconv.ParseUint(pageStr, 10, 32)
	if err != nil {
		return nil, err
	}
	pageSizeStr, err := parseAttrNameQueryParamWithDefault(c, "pageSize", "10")
	if err != nil {
		return nil, err
	}
	pageSize, err := strconv.ParseUint(pageSizeStr, 10, 32)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("list group configs from chainId:%d,contractAddress:%s,page:%d,pageSize:%d", chainId, contractAddress, page, pageSize)

	// Use the new ListConfigWithOutputIdFromChainIdAndContractAddressv2 function
	currentPage, pageSizeInt, total, resp, err := im.ListConfigWithOutputIdFromChainIdAndContractAddressv2(chainId, contractAddress, int(page), int(pageSize), deps.IMManager)
	if err != nil {
		return nil, err
	}

	CoreComponent.LogInfof("list group configs from chainId:%d,contractAddress:%s,page:%d,pageSize:%d,found groupConfigs:%d", chainId, contractAddress, currentPage, pageSizeInt, len(resp))

	result := map[string]interface{}{
		"currentPage": currentPage,
		"pageSize":    pageSizeInt,
		"total":       total,
		"list":        resp,
	}

	return result, nil
}

func listGroupConfigsLite(c echo.Context) ([]*im.GroupConfigNftListResponse, error) {
	// get param include chainId uint32, contractAddress string, page int, pageSize int
	// chainId and contract address are optional
	// page and pageSize are optional and default to 1 and 10
	chainIdStr, err := parseAttrNameQueryParamWithNil(c, "chainId")
	if err != nil {
		return nil, err
	}
	var chainId uint32
	chainId = math.MaxUint32
	if chainIdStr != "" {
		chainId64, err := strconv.ParseUint(chainIdStr, 10, 32)
		if err != nil {
			return nil, err
		}
		chainId = uint32(chainId64)
	}
	contractAddress, err := parseAttrNameQueryParamWithNil(c, "contractAddress")
	if err != nil {
		return nil, err
	}
	pageStr, err := parseAttrNameQueryParamWithDefault(c, "page", "1")
	if err != nil {
		return nil, err
	}
	page, err := strconv.ParseUint(pageStr, 10, 32)
	if err != nil {
		return nil, err
	}
	pageSizeStr, err := parseAttrNameQueryParamWithDefault(c, "pageSize", "10")
	if err != nil {
		return nil, err
	}
	pageSize, err := strconv.ParseUint(pageSizeStr, 10, 32)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("list group configs from chainId:%d,contractAddress:%s,page:%d,pageSize:%d", chainId, contractAddress, page, pageSize)
	resp, err := im.ListOutputIdAndGroupIdFromChainIdAndContractAddress(chainId, contractAddress, int(page), int(pageSize), deps.IMManager)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("list group configs from chainId:%d,contractAddress:%s,page:%d,pageSize:%d,found groupConfigs:%d", chainId, contractAddress, page, pageSize, len(resp))
	return resp, nil
}

// getGroupConfigUnderNft
func getGroupConfigUnderNft(c echo.Context) ([]*im.MessageGroupMetaJSON, error) {
	// get params including chainId uint32, contractAddress string, all required
	chainIdStr, err := parseAttrNameQueryParam(c, "chainId")
	if err != nil {
		return nil, err
	}
	chainId64, err := strconv.ParseUint(chainIdStr, 10, 32)
	if err != nil {
		return nil, err
	}
	chainId := uint32(chainId64)
	contractAddress, err := parseAttrNameQueryParam(c, "contractAddress")
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group config under nft from chainId:%d,contractAddress:%s", chainId, contractAddress)
	groupIds, err := im.ReadAllGroupIdFromChainIdAndContractAddressHash(chainId, contractAddress, deps.IMManager)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group config under nft from chainId:%d,contractAddress:%s,found groupIds:%d", chainId, contractAddress, len(groupIds))
	var groupConfigs []*im.MessageGroupMetaJSON
	for _, groupId := range groupIds {
		config, err := im.ReadGroupConfigMetaFromGroupId(groupId, deps.IMManager)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("get group config under nft from chainId:%d,contractAddress:%s,groupId:%s failed:%s", chainId, contractAddress, iotago.EncodeHex(groupId[:]), err)
			continue
		}
		groupConfigs = append(groupConfigs, config)
	}
	return groupConfigs, nil
}

// getGroupStateSyncUnderAddress
func getGroupStateSyncUnderAddress(c echo.Context) (*im.GroupStateSyncResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group state sync under address:%s", address)
	groupStateSync, err := im.GetGroupStateSyncFromAddress(address, deps.IMManager)
	if err != nil {
		return nil, err
	}
	var respItems []*im.GroupStateSyncResponseItem
	if groupStateSync == nil {
		respEmpty := &im.GroupStateSyncResponse{
			OutputId: "",
			Items:    respItems,
		}
		return respEmpty, nil
	}
	for _, item := range groupStateSync.Items {
		respItems = append(respItems, &im.GroupStateSyncResponseItem{
			GroupId:                            iotago.EncodeHex(item.GroupId[:]),
			LastTimeReadLatestMessageTimestamp: item.LastTimeReadLatestMessageTimestamp,
		})
	}

	resp := &im.GroupStateSyncResponse{
		OutputId: iotago.EncodeHex(groupStateSync.OutputId[:]),
		Items:    respItems,
	}
	return resp, nil

}

// batchCheckOutputId
func batchCheckOutputId(c echo.Context) ([]*im.OutputIdCheckResponse, error) {
	// get outputIds from body
	outputIds, err := parseOutputIdsFromBody(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("batch check outputId from outputIds:%d", len(outputIds))
	resp := make([]*im.OutputIdCheckResponse, len(outputIds))
	for i, outputId := range outputIds {
		outputIdBytes, err := iotago.DecodeHex(outputId)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("batch check outputId from outputIds:%d failed:%s", len(outputIds), err)
			continue
		}
		outputIdFixed := [im.OutputIdLen]byte{}
		copy(outputIdFixed[:], outputIdBytes)

		checked, err := im.EvmQualifyEffectingOutputIdExists(outputIdFixed, deps.IMManager, CoreComponent.Logger())
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("batch check outputId from outputIds:%d failed:%s", len(outputIds), err)
			continue
		}
		resp[i] = &im.OutputIdCheckResponse{
			OutputId:    outputId,
			IsEffecting: checked,
		}
	}
	return resp, nil
}

// batchOutputIdToOutput
func batchOutputIdToOutput(c echo.Context) ([]*im.OutputIdOutputResponse, error) {
	// get outputIds from body
	outputIds, err := parseOutputIdsFromBody(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("batch outputId to output from outputIds:%d", len(outputIds))
	chanForResp := make(chan interface{})
	defer close(chanForResp)

	var resp []*im.OutputIdOutputResponse
	// map outputIds to OutputIdWithRespChan[]
	var items []interface{}
	for _, outputId := range outputIds {
		req := &im.OutputIdWithRespChan{
			OutputIdHex: outputId,
			RespChan:    chanForResp,
		}
		items = append(items, req)
	}
	im.OutputIdDrainer.Drain(items)
	// get item from chanForResp, also with 5 sec timeout

	// Create a context with a total timeout of 5 seconds
	ctx, cancel := context.WithTimeout(c.Request().Context(), 5*time.Second)
	defer cancel()

Loop:
	for i := 0; i < len(outputIds); i++ {
		select {
		case item, ok := <-chanForResp:
			if !ok {
				CoreComponent.LogWarnf("Channel closed unexpectedly after receiving %d responses", len(resp))
				break Loop
			}

			response, ok := item.(*im.OutputIdOutputResponse)
			if !ok {
				CoreComponent.LogErrorf("Received unexpected type from channel")
				continue // Skip this item or handle the error as needed
			}

			resp = append(resp, response)

		case <-ctx.Done():
			CoreComponent.LogWarnf("Batch processing of outputIds:%d timed out after receiving %d responses", len(outputIds), len(resp))
			break Loop
		}
	}

	return resp, nil
}

// checkGroupIdExists
func checkGroupIdExists(c echo.Context) (*im.GroupIdCheckResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("check groupId exists from groupId:%s", iotago.EncodeHex(groupId))
	groupId32 := [32]byte{}
	copy(groupId32[:], groupId)
	exists := deps.IMManager.CheckGroupExists(groupId32)

	resp := &im.GroupIdCheckResponse{
		GroupIdHex: iotago.EncodeHex(groupId),
		IsExist:    exists,
	}
	return resp, nil
}

// checkGroupIdExists batched version
func checkGroupIdExistsBatch(c echo.Context) ([]*im.GroupIdCheckResponse, error) {
	// get groupIds from body
	groupIds, err := parseIdsFromBody(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("batch check groupId exists from groupIds:%d", len(groupIds))
	resp := make([]*im.GroupIdCheckResponse, len(groupIds))
	for i, groupIdHex := range groupIds {
		groupId, err := iotago.DecodeHex(groupIdHex)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("batch check groupId exists from groupIds:%d failed:%s", len(groupIds), err)
			continue
		}
		groupId32 := [32]byte{}
		copy(groupId32[:], groupId)
		exists := deps.IMManager.CheckGroupExists(groupId32)
		resp[i] = &im.GroupIdCheckResponse{
			GroupIdHex: groupIdHex,
			IsExist:    exists,
		}
	}
	return resp, nil
}

// getGroupMessagesWithCount handles the request to get a list of {groupId, messageCount, timestampOfHour} after an optional start timestamp.
func getGroupMessagesWithCount(c echo.Context) ([]GroupMessageCountWithTimestampResponse, error) {
	// Parse optional start timestampOfHour
	startTimestampOfHour, err := parseOptionalTimestampOfHourParam(c, "startTimestampOfHour")
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid startTimestampOfHour")
	}

	// If startTimestampOfHour is not provided, use the earliest possible timestamp
	if startTimestampOfHour == 0 {
		startTimestampOfHour = 0
	}
	startTimestampOfHour = im.StartOfHour(startTimestampOfHour)
	// Get the list of messages with their respective groupIds, timestamps, and message counts using im.GetGroupMessagesAfterTimestamp
	groupMessages, err := im.GetGroupMessagesAfterTimestamp(startTimestampOfHour, deps.IMManager)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get group messages")
	}

	// Prepare the response
	response := make([]GroupMessageCountWithTimestampResponse, len(groupMessages))
	for i, gm := range groupMessages {
		response[i] = GroupMessageCountWithTimestampResponse{
			GroupId:         iotago.EncodeHex(gm.GroupId[:]),
			MessageCount:    gm.MessageCount,
			TimestampOfHour: gm.TimestampOfHour,
		}
	}

	return response, nil
}

// parseOptionalTimestampOfHourParam parses an optional timestampOfHour from query parameters.
func parseOptionalTimestampOfHourParam(c echo.Context, paramName string) (uint32, error) {
	timestampParams := c.QueryParams()[paramName]
	if len(timestampParams) == 0 {
		return 0, nil
	}
	timestamp, err := strconv.ParseUint(timestampParams[0], 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(timestamp), nil
}

// GroupMessageCountWithTimestampResponse represents the response structure for the group message count with timestamp request.
type GroupMessageCountWithTimestampResponse struct {
	GroupId         string `json:"groupId"`
	MessageCount    uint32 `json:"messageCount"`
	TimestampOfHour uint32 `json:"timestampOfHour"`
}

// getMessageCountWithOptionalRange handles the request to get the message count for a specific groupId within an optional time range.
func getMessageCountWithOptionalRange(c echo.Context) (*GroupMessageCountResponse, error) {
	// Parse groupId from query params
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}

	// Convert groupId to fixed-length array
	groupIdFixed := [im.GroupIdLen]byte{}
	copy(groupIdFixed[:], groupId)

	// Parse optional start and end timestamps
	startTimestamp, err := parseOptionalTimestampParam(c, "startTimestamp")
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid startTimestamp")
	}

	endTimestamp, err := parseOptionalTimestampParam(c, "endTimestamp")
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusBadRequest, "Invalid endTimestamp")
	}

	// Get the message count for the groupId within the specified range using GetMessageCountForGroupInRange
	totalCount, err := im.GetMessageCountForGroupInRange(groupIdFixed, startTimestamp, endTimestamp, deps.IMManager)
	if err != nil {
		return nil, echo.NewHTTPError(http.StatusInternalServerError, "Failed to get message count")
	}

	// Prepare and return the response
	resp := &GroupMessageCountResponse{
		GroupId:      iotago.EncodeHex(groupId),
		MessageCount: totalCount,
	}
	return resp, nil
}

// GroupMessageCountResponse represents the response structure for the group message count request.
type GroupMessageCountResponse struct {
	GroupId      string `json:"groupId"`
	MessageCount uint32 `json:"messageCount"`
}

// parseOptionalTimestampParam parses an optional timestamp from query parameters.
func parseOptionalTimestampParam(c echo.Context, paramName string) (uint32, error) {
	timestampParams := c.QueryParams()[paramName]
	if len(timestampParams) == 0 {
		return 0, nil
	}
	timestamp, err := strconv.ParseUint(timestampParams[0], 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(timestamp), nil
}
