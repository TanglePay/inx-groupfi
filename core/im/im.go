package im

import (
	"strconv"
	"time"

	"github.com/TanglePay/inx-iotacat/pkg/im"
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

// parse address from query param
func parseAddressQueryParam(c echo.Context) (string, error) {
	addressParams := c.QueryParams()["address"]
	if len(addressParams) == 0 {
		return "", echo.ErrBadRequest
	}
	address := addressParams[0]
	return address, nil
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

// parse given attrName from query param
func parseAttrNameQueryParam(c echo.Context, attrName string) (string, error) {
	attrParams := c.QueryParams()[attrName]
	if len(attrParams) == 0 {
		return "", echo.ErrBadRequest
	}
	attr := attrParams[0]
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

func getMesssagesFrom(c echo.Context) (*MessagesResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	token, err := parseTokenQueryParam(c)
	if err != nil {
		return nil, err
	}

	size, err := parseSizeQueryParam(c)
	if err != nil {
		return nil, err
	}

	CoreComponent.LogInfof("get messages from,groupId:%s,token:%d,size:%d", iotago.EncodeHex(groupId), token, size)
	keyPrefix := deps.IMManager.MessageKeyFromGroupId(groupId)
	messages, err := deps.IMManager.ReadMessageFromPrefix(keyPrefix, size, token)
	if err != nil {
		return nil, err
	}
	// log messages length
	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d,found messages:%d", iotago.EncodeHex(groupId), token, size, len(messages))

	messagesResponse := makeMessageResponse(messages)
	return messagesResponse, nil
}
func getMesssagesUntil(c echo.Context) (*MessagesResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	token, err := parseTokenQueryParam(c)
	if err != nil {
		return nil, err
	}

	size, err := parseSizeQueryParam(c)
	if err != nil {
		return nil, err
	}

	CoreComponent.LogInfof("get messages until,groupId:%s,token:%d,size:%d", iotago.EncodeHex(groupId), token, size)
	keyPrefix := deps.IMManager.MessageKeyFromGroupId(groupId)
	messages, err := deps.IMManager.ReadMessageUntilPrefix(keyPrefix, size, token)
	if err != nil {
		return nil, err
	}
	// log messages length
	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d,found messages:%d", iotago.EncodeHex(groupId), token, size, len(messages))
	messagesResponse := makeMessageResponse(messages)
	return messagesResponse, nil
}

// make messsage response from messages
func makeMessageResponse(messages []*im.Message) *MessagesResponse {
	messageResponseArr := make([]*MessageResponse, len(messages))
	var headToken string
	var tailToken string
	for i, message := range messages {
		messageResponseArr[i] = &MessageResponse{
			OutputId:  iotago.EncodeHex(message.OutputId),
			Timestamp: message.MileStoneTimestamp,
		}
		tokenHex := iotago.EncodeHex(message.Token)
		if headToken == "" {
			headToken = tokenHex
		}
		tailToken = tokenHex
	}
	return &MessagesResponse{
		Messages:  messageResponseArr,
		HeadToken: headToken,
		TailToken: tailToken,
	}
}

// make inbox message response from inbox message
func makeInboxMessageResponse(messages []*im.Message) *InboxMessagesResponse {
	messageResponseArr := make([]*MessageResponse, len(messages))
	var token string
	for i, message := range messages {
		token = iotago.EncodeHex(message.Token)
		messageResponseArr[i] = &MessageResponse{
			OutputId:  iotago.EncodeHex(message.OutputId),
			Timestamp: message.MileStoneTimestamp,
			Token:     token,
		}

	}
	return &InboxMessagesResponse{
		Messages: messageResponseArr,
		Token:    token,
	}
}

// make address group details response from address group
func makeAddressGroupDetailsResponse(addressGroup *im.AddressGroup) *AddressGroupDetailsResponse {
	return &AddressGroupDetailsResponse{
		GroupId:          iotago.EncodeHex(addressGroup.GroupId),
		GroupName:        addressGroup.GroupName,
		GroupQualifyType: addressGroup.GroupQualifyType,
		IpfsLink:         addressGroup.NftLink,
		TokenName:        im.GetTokenNameFromType(addressGroup.TokenType),
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
	return deps.IMManager.GetNFTsWithPublicKeyFromGroupIdImpl(nfts, drainer, CoreComponent.Logger())
}

// get shared from groupId
func getSharedFromGroupId(c echo.Context) (*SharedResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get shared from group:%s", groupId)
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
	// log group ct, public ct, private ct
	CoreComponent.LogInfof("get shared from group:%s,group memberCt:%d,public ct:%d,private ct:%d", iotago.EncodeHex(groupId), memberCt, publicCt, privateCt)
	// group is forced to be public if there are more than 100 members, or public votes are more than private votes
	if memberCt > 100 || publicCt > privateCt {
		// throw http error with code 901
		return nil, echo.NewHTTPError(901, "adjusted to be public")
	}
	shared, err := deps.IMManager.ReadSharedFromGroupId(groupId)
	if err != nil {
		return nil, err
	}
	if shared == nil {
		return nil, nil
	}
	CoreComponent.LogInfof("get shared from groupId:%s,found shared with outputid:%s", groupId, iotago.EncodeHex(shared.OutputId))
	resp := &SharedResponse{
		OutputId: iotago.EncodeHex(shared.OutputId),
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
	err = deps.IMManager.DeleteSharedFromGroupId(groupId)
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
	CoreComponent.LogInfof("get groupIds from address:%s", address)
	addressSha256 := im.Sha256Hash(address)
	groupIds, err := deps.IMManager.GetGroupIdsFromAddress(addressSha256)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get groupIds from address:%s,found groupIds:%d", address, len(groupIds))
	groupIdStrArr := make([]string, len(groupIds))
	for i, groupId := range groupIds {
		groupIdStrArr[i] = iotago.EncodeHex(groupId)
	}
	return groupIdStrArr, nil
}

// getAddressGroupDetails
func getAddressGroupDetails(c echo.Context) ([]*AddressGroupDetailsResponse, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get address group details from address:%s", address)
	addressSha256 := im.Sha256Hash(address)
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

const DaysElapsedForConsolidation = 3

// get outputids for consolidation,
func getMessageOutputIdsForConsolidation(c echo.Context) ([]string, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get outputids for consolidation from address:%s", address)
	// calculate timestamp DaysElapsedForConsolidation from now
	thresMileStoneTimestamp := uint32(time.Now().AddDate(0, 0, -DaysElapsedForConsolidation).Unix())
	outputIds, err := deps.IMManager.ReadMessageForConsolidation(address, thresMileStoneTimestamp, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get outputids for consolidation from address:%s,found outputIds:%d", address, len(outputIds))
	return outputIds, nil
}

// get outputids for consolidation, for shared
func getSharedOutputIdsForConsolidation(c echo.Context) ([]string, error) {
	address, err := parseAddressQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get outputids for consolidation from address:%s", address)
	// calculate timestamp DaysElapsedForConsolidation from now
	thresMileStoneTimestamp := uint32(time.Now().AddDate(0, 0, -2*DaysElapsedForConsolidation).Unix())
	outputIds, err := deps.IMManager.ReadSharedForConsolidation(address, thresMileStoneTimestamp, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get outputids for consolidation from address:%s,found outputIds:%d", address, len(outputIds))
	return outputIds, nil
}

// get qualified address for a groupid
func getQualifiedAddressesForGroupId(c echo.Context) ([]string, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get qualified address for groupId:%s", iotago.EncodeHex(groupId))
	nfts, err := deps.IMManager.ReadNFTsFromGroupId(groupId)
	if err != nil {
		return nil, err
	}
	// nfts to addresses
	var addresses []string
	for _, nft := range nfts {
		addresses = append(addresses, string(nft.OwnerAddress))
	}
	CoreComponent.LogInfof("get qualified address for groupId:%s,found addresses:%d", iotago.EncodeHex(groupId), len(addresses))
	return addresses, nil
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

// get all group member addresses from groupId
func getGroupMemberAddressesFromGroupId(c echo.Context) ([]string, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group member addresses from groupId:%s", iotago.EncodeHex(groupId))
	var groupId32 [32]byte
	copy(groupId32[:], groupId)
	addresses, err := deps.IMManager.GetGroupMemberAddressesFromGroupId(groupId32, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get group member addresses from groupId:%s,found addresses:%d", iotago.EncodeHex(groupId), len(addresses))
	return addresses, nil
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
	CoreComponent.LogInfof("get user group reputation from groupId:%s,address:%s,found reputation:%f", iotago.EncodeHex(groupId), address, reputation)
	resp := &GroupUserReputationResponse{
		GroupId:           iotago.EncodeHex(reputation.GroupId[:]),
		AddressSha256Hash: iotago.EncodeHex(reputation.AddrSha256Hash[:]),
		Reputation:        reputation.Reputation,
	}
	return resp, nil
}

// get all groups under renter
func getGroupConfigsForRenter(c echo.Context) ([]*im.MessageGroupMetaJSON, error) {
	renderName, err := parseAttrNameQueryParam(c, "renderName")
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get groups under renter:%s", renderName)
	groupConfigs, err := deps.IMManager.ReadAllGroupConfigForRenter(renderName)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get groups under renter:%s,found groupConfigs:%d", renderName, len(groupConfigs))
	return groupConfigs, nil
}

// get inbox message
func getInboxMessage(c echo.Context) (*InboxMessagesResponse, error) {
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
	CoreComponent.LogInfof("get inbox message from address:%s,token:%d", address, token)
	// get inbox message
	inboxMessage, err := deps.IMManager.ReadInboxMessage(im.Sha256Hash(address), token, size, CoreComponent.Logger())
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get inbox message from address:%s,token:%d,found inboxMessage:%d", address, token, len(inboxMessage))
	// make inbox message response
	inboxMessageResponse := makeInboxMessageResponse(inboxMessage)
	return inboxMessageResponse, nil
}
