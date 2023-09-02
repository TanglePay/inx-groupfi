package im

import (
	"context"
	"net/http"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/labstack/echo/v4"

	"github.com/iotaledger/inx-app/pkg/httpserver"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

var (
	publicKeyDrainer *ItemDrainer
)

const (
	APIRoute     = "groupfi/v1"
	MQTTAPIRoute = "groupfi/mqtt/v1"
	// RouteIMMessages is the route to get a slice of messages belong to the given groupID, get first size of messages, start from token
	RouteIMMessages      = "/messages"
	RouteIMMessagesUntil = "/messages/until"
	// nft
	RouteIMNFTs = "/nfts"
	// nfts that each with public key
	RouteIMNFTsWithPublicKey = "/nftswithpublickey"

	// shared
	RouteIMShared = "/shared"
	// address group ids
	RouteIMAddressGroupIds = "/addressgroupids"

	// consolidation for message
	RouteImConsolidationForMessage = "/consolidation/message"
	// consolidation for shared
	RouteImConsolidationForShared = "/consolidation/shared"

	// group configs for renter
	RouteGroupConfigs = "/groupconfigs"
)

func setupRoutes(e *echo.Echo, ctx context.Context, client *nodeclient.Client) {
	publicKeyDrainer = NewItemDrainer(ctx, func(item interface{}) {
		// unwrap to *NFTWithRespChan
		nftWithRespChan := item.(*NFTWithRespChan)
		// get address from nft
		address := string(nftWithRespChan.NFT.OwnerAddress)
		// get public key from address
		publicKeyBytes, err := deps.IMManager.GetAddressPublicKey(ctx, client, address)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... GetAddressPublicKey failed:%s", err)
			return
		}
		var publicKey string
		if publicKeyBytes != nil {
			publicKey = iotago.EncodeHex(publicKeyBytes)
		} else {
			publicKey = ""
		}

		// make *NFTResponse
		nftResponse := &NFTResponse{
			OwnerAddress: address,
			PublicKey:    publicKey,
		}
		// send to respChan
		nftWithRespChan.RespChan <- nftResponse
	}, 2000, 1000, 1000)
	e.GET(RouteIMMessages, func(c echo.Context) error {
		resp, err := getMesssagesFrom(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	//messages until
	e.GET(RouteIMMessagesUntil, func(c echo.Context) error {
		resp, err := getMesssagesUntil(c)
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
	// nfts with public key
	e.GET(RouteIMNFTsWithPublicKey, func(c echo.Context) error {
		resp, err := getNFTsWithPublicKeyFromGroupId(c, publicKeyDrainer)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})
	//shared
	e.GET(RouteIMShared, func(c echo.Context) error {
		resp, err := getSharedFromGroupId(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// delete shared
	e.GET("/deleteshared", func(c echo.Context) error {
		err := deleteSharedFromGroupId(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, "ok")
	})

	e.GET("/testlist", func(c echo.Context) error {

		err := deps.IMManager.LogAllData(CoreComponent.Logger())
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, "ok")
	})

	// test nft key
	e.GET("/testnftkey", func(c echo.Context) error {
		key, err := parseTokenQueryParam(c)
		if err != nil {
			return err
		}
		groupId, nftId := deps.IMManager.NftKeyToGroupIdAndNftId(key, CoreComponent.Logger())
		return httpserver.JSONResponse(c, http.StatusOK, iotago.EncodeHex(groupId)+iotago.EncodeHex(nftId))
	})

	e.GET("/testGroupName", func(c echo.Context) error {
		groupName, err := parseGroupNameQueryParam(c)
		if err != nil {
			return err
		}
		groupId := deps.IMManager.GroupNameToGroupId(groupName)
		return httpserver.JSONResponse(c, http.StatusOK, iotago.EncodeHex(groupId))
	})
	e.GET("/testtoken", func(c echo.Context) error {
		address, err := parseAddressQueryParam(c)
		if err != nil {
			return err
		}
		balance, err := deps.IMManager.GetBalanceOfOneAddress(im.ImTokenTypeSMR, address)
		if err != nil {
			return err
		}
		totalBalance := GetSmrTokenTotal().Get()
		resp := &TokenBalanceResponse{
			TokenType:    im.ImTokenTypeSMR,
			Balance:      balance.Text(10),
			TotalBalance: totalBalance.Text(10),
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})
	e.GET(RouteIMAddressGroupIds, func(c echo.Context) error {
		groupIds, err := getGroupIdsFromAddress(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, groupIds)
	})

	// consolidation for message
	e.GET(RouteImConsolidationForMessage, func(c echo.Context) error {
		outputIds, err := getMessageOutputIdsForConsolidation(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, outputIds)
	})

	// consolidation for shared
	e.GET(RouteImConsolidationForShared, func(c echo.Context) error {
		outputIds, err := getSharedOutputIdsForConsolidation(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, outputIds)
	})

	// group configs for renter
	e.GET(RouteGroupConfigs, func(c echo.Context) error {
		resp, err := getGroupConfigsForRenter(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})
}
