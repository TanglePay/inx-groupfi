package im

import (
	"strconv"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func parseTokenQueryParam(c echo.Context, groupId []byte) ([]byte, int, error) {
	tokenParams := c.QueryParams()["token"]
	if len(tokenParams) == 0 {
		return deps.IMManager.MessageKeyFromGroupId(groupId), 0, nil
	}
	token, err := iotago.DecodeHex(tokenParams[0])
	if err != nil {
		return nil, 0, err
	}
	return token, 1, nil
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
	token, skip, err := parseTokenQueryParam(c, groupId)
	if err != nil {
		return nil, err
	}

	size, err := parseSizeQueryParam(c)
	if err != nil {
		return nil, err
	}

	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d", iotago.EncodeHex(groupId), token, size)
	messages, err := deps.IMManager.ReadMessageFromPrefix(token, size, skip)
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
