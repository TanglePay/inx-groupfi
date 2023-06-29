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
	messages, err := deps.IMManager.ReadMessageFromPrefix(groupId, size, token)
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

// 0x019f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08ffb943a5fffffffe
// 0x019f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08ffb9428efffffffe
// 0x019f86d081884c7d659a2feaa0c55ad015a3bf4f1b2b0b822cd15d6c15b0f00a08ffb94268fffffffe
