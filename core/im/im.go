package im

import (
	"strconv"

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
	messages, err := deps.IMManager.ReadMessageUntilPrefix(keyPrefix, size, token)
	if err != nil {
		return nil, err
	}
	// log messages length
	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d,found messages:%d", iotago.EncodeHex(groupId), token, size, len(messages))

	messageResponseArr := make([]*MessageResponse, len(messages))
	var continuationToken string
	for i, message := range messages {
		messageResponseArr[i] = &MessageResponse{
			OutputId:  iotago.EncodeHex(message.OutputId),
			Timestamp: message.MileStoneTimestamp,
		}
		if continuationToken == "" {
			continuationToken = iotago.EncodeHex(message.Token)
		}
	}
	return &MessagesResponse{
		Messages: messageResponseArr,
		Token:    continuationToken,
	}, nil
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
	messages, err := deps.IMManager.ReadMessageFromPrefix(keyPrefix, size, token)
	if err != nil {
		return nil, err
	}
	// log messages length
	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d,found messages:%d", iotago.EncodeHex(groupId), token, size, len(messages))

	messageResponseArr := make([]*MessageResponse, len(messages))
	var continuationToken string
	for i, message := range messages {
		messageResponseArr[i] = &MessageResponse{
			OutputId:  iotago.EncodeHex(message.OutputId),
			Timestamp: message.MileStoneTimestamp,
		}
		continuationToken = iotago.EncodeHex(message.Token)
	}
	return &MessagesResponse{
		Messages: messageResponseArr,
		Token:    continuationToken,
	}, nil
}

// get nfts
func getNFTsFromGroupId(c echo.Context) ([]*NFTResponse, error) {
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
