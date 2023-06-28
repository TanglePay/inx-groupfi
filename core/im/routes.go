package im

import (
	"net/http"

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
}
