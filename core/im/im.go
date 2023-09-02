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

// get raw nfts from groupId
func getRawNFTsFromGroupId(c echo.Context) ([]*im.NFT, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}

	keyPrefix := deps.IMManager.NftKeyPrefixFromGroupId(groupId)
	CoreComponent.LogInfof("get nfts from groupid:%s, with prefix:%s", iotago.EncodeHex(groupId), iotago.EncodeHex(keyPrefix))
	nfts, err := deps.IMManager.ReadNFTFromPrefix(keyPrefix)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get nfts from groupId:%s,found nfts:%d", iotago.EncodeHex(groupId), len(nfts))
	return nfts, nil
}

// get nfts
func getNFTsFromGroupId(c echo.Context) ([]*NFTResponse, error) {
	nfts, err := getRawNFTsFromGroupId(c)
	if err != nil {
		return nil, err
	}
	nftResponseArr := make([]*NFTResponse, len(nfts))
	for i, nft := range nfts {
		// nft.OwnerAddress is []bytes{OwnerAddress}
		nftResponseArr[i] = &NFTResponse{
			NFTId:        iotago.EncodeHex(nft.NFTId),
			OwnerAddress: string(nft.OwnerAddress),
		}
	}
	return nftResponseArr, nil
}

// NFTWithRespChan
type NFTWithRespChan struct {
	NFT      *im.NFT
	RespChan chan interface{}
}

// getNFTsWithPublicKeyFromGroupId
func getNFTsWithPublicKeyFromGroupId(c echo.Context, drainer *ItemDrainer) ([]*NFTResponse, error) {
	nfts, err := getRawNFTsFromGroupId(c)
	if err != nil {
		return nil, err
	}
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
	drainer.Drain(nftsInterface)
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
			return nil, errors.Errorf("timeout")
		}
	}
}

// get shared from groupId
func getSharedFromGroupId(c echo.Context) (*SharedResponse, error) {
	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get shared from group:%s", groupId)
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
