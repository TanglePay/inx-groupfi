package im

import (
	"strconv"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/labstack/echo/v4"
	"github.com/pkg/errors"
)

func parseTokenQueryParam(c echo.Context) (uint32, error) {
	tokenParams := c.QueryParams()["token"]
	if len(tokenParams) == 0 {
		return 0, echo.ErrBadRequest
	}
	token, err := strconv.ParseUint(tokenParams[0], 10, 32)
	if err != nil {
		return 0, err
	}
	return uint32(token), nil
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

	token, err := parseTokenQueryParam(c)
	if err != nil {
		return nil, err
	}

	groupId, err := parseGroupIdQueryParam(c)
	if err != nil {
		return nil, err
	}

	size, err := parseSizeQueryParam(c)
	if err != nil {
		return nil, err
	}

	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d", iotago.EncodeHex(groupId), token, size)
	messages, err := deps.IMManager.GetMessages(groupId, token, size)
	if err != nil {
		return nil, err
	}
	// log messages length
	CoreComponent.LogInfof("get messages,groupId:%s,token:%d,size:%d,found messages:%d", iotago.EncodeHex(groupId), token, size, len(messages))

	messageResponseArr := make([]*MessageResponse, len(messages))
	var continuationToken uint32
	for i, message := range messages {
		messageResponseArr[i] = &MessageResponse{
			OutputId:  string(message.OutputId),
			Timestamp: message.MileStoneTimestamp,
		}
		continuationToken = message.MileStoneIndex
	}
	return &MessagesResponse{
		Messages: messageResponseArr,
		Token:    continuationToken,
	}, nil
}
