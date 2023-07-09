package im

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/iotaledger/inx-app/pkg/httpserver"
)

const (
	APIRoute = "iotacatim/v1"

	// RouteIMMessages is the route to get a slice of messages belong to the given groupID, get first size of messages, start from token
	RouteIMMessages = "/messages"
	// nft
	RouteIMNFTs = "/nfts"
	// shared
	RouteIMShared = "/shared"
)

func setupRoutes(e *echo.Echo) {

	e.GET(RouteIMMessages, func(c echo.Context) error {
		resp, err := getMesssages(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	//nft
	e.GET(RouteIMNFTs, func(c echo.Context) error {
		resp, err := getNFTsFromGroupId(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	//

	e.GET("/testlist2", func(c echo.Context) error {

		err := deps.IMManager.LogAllData(CoreComponent.Logger())
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, "ok")
	})

}
