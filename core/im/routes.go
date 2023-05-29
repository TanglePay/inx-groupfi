package im

import (
	"net/http"

	"github.com/labstack/echo/v4"

	"github.com/iotaledger/inx-app/pkg/httpserver"
)

const (
	APIRoute = "im/v1"

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

}
