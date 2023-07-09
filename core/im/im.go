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

const defaultSize = 100

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

func getMesssages(c echo.Context) (*MessagesResponse, error) {
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

	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d", iotago.EncodeHex(groupId), token, size)
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
	CoreComponent.LogInfof("get nfts from group:%s", groupId)
	keyPrefix := deps.IMManager.NftKeyPrefixFromGroupId(groupId)
	nfts, err := deps.IMManager.ReadNFTFromPrefix(keyPrefix)
	if err != nil {
		return nil, err
	}
	CoreComponent.LogInfof("get nfts from groupId:%s,found nfts:%d", groupId, len(nfts))
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

//
