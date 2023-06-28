package im

import (
	"net/http"
	"strconv"

	"github.com/labstack/echo/v4"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	iotago "github.com/iotaledger/iota.go/v3"
)

const (
	APIRoute = "iotacatim/v1"

	// RouteIMMessages is the route to get a slice of messages belong to the given groupID, get first size of messages, start from token
	RouteIMMessages = "/messages"
)

func setupRoutes(e *echo.Echo) {

	e.GET(RouteIMMessages, func(c echo.Context) error {
		resp, err := getMesssages(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})
	e.GET("/test1", func(c echo.Context) error {
		keyHex := c.QueryParams()["key"]
		if len(keyHex) == 0 {
			return echo.ErrBadRequest
		}
		keyBytes, err := iotago.DecodeHex(keyHex[0])
		if err != nil {
			return err
		}
		resp, err := deps.IMManager.GetSingleMessage(keyBytes, CoreComponent.Logger())
		valueHex := iotago.EncodeHex(resp)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, valueHex)
	})
	e.GET("/test2", func(c echo.Context) error {
		groupIDHex := c.QueryParams()["groupId"]
		if len(groupIDHex) == 0 {
			return echo.ErrBadRequest
		}
		groupID, err := iotago.DecodeHex(groupIDHex[0])
		if err != nil {
			return err
		}
		milestoneIndexStr := c.QueryParams()["milestoneIndex"]
		if len(milestoneIndexStr) == 0 {
			return echo.ErrBadRequest
		}
		milestoneIndex, err := strconv.ParseUint(milestoneIndexStr[0], 10, 32)

		if err != nil {
			return err
		}
		outputIdHex := c.QueryParams()["outputId"]
		if len(outputIdHex) == 0 {
			return echo.ErrBadRequest
		}
		outputId, err := iotago.DecodeHex(outputIdHex[0])
		if err != nil {
			return err
		}
		resp, err := deps.IMManager.GetSingleMessage2(groupID, uint32(milestoneIndex), outputId, CoreComponent.Logger())
		valueHex := iotago.EncodeHex(resp)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, valueHex)
	})
	e.GET("/testlist1", func(c echo.Context) error {
		keyHex := c.QueryParams()["key"]
		if len(keyHex) == 0 {
			return echo.ErrBadRequest
		}
		keyBytes, err := iotago.DecodeHex(keyHex[0])
		if err != nil {
			return err
		}
		resp, err := deps.IMManager.GetMessageList1(keyBytes, CoreComponent.Logger())
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, len(resp))
	})
	e.GET("/testlist2", func(c echo.Context) error {
		groupIDHex := c.QueryParams()["groupId"]
		if len(groupIDHex) == 0 {
			return echo.ErrBadRequest
		}
		groupID, err := iotago.DecodeHex(groupIDHex[0])
		if err != nil {
			return err
		}
		milestoneIndexStr := c.QueryParams()["milestoneIndex"]
		if len(milestoneIndexStr) == 0 {
			return echo.ErrBadRequest
		}
		milestoneIndex, err := strconv.ParseUint(milestoneIndexStr[0], 10, 32)

		if err != nil {
			return err
		}
		outputIdHex := c.QueryParams()["outputId"]
		if len(outputIdHex) == 0 {
			return echo.ErrBadRequest
		}
		outputId, err := iotago.DecodeHex(outputIdHex[0])
		if err != nil {
			return err
		}
		resp, err := deps.IMManager.GetMessageList2(groupID, uint32(milestoneIndex), outputId, CoreComponent.Logger())
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, len(resp))
	})
}
