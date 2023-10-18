package im

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"github.com/labstack/echo/v4"
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
	// address group details
	RouteIMAddressGroupDetails = "/addressgroupdetails"
	// consolidation for message
	RouteImConsolidationForMessage = "/consolidation/message"
	// consolidation for shared
	RouteImConsolidationForShared = "/consolidation/shared"

	// group configs for renter
	RouteGroupConfigs = "/groupconfigs"

	// inbox message
	RouteInboxMessage = "/inboxmessage"

	// group qualified addresses
	RouteGroupQualifiedAddresses = "/groupqualifiedaddresses"

	// group marked addresses
	RouteGroupMarkedAddresses = "/groupmarkedaddresses"

	// group member addresses
	RouteGroupMemberAddresses = "/groupmemberaddresses"

	// get group votes
	RouteGroupVotes = "/groupvotes"

	// get group votes count
	RouteGroupVotesCount = "/groupvotescount"

	// get group blacklist
	RouteGroupBlacklist = "/groupblacklist"

	// get address member groups
	RouteAddressMemberGroups = "/addressmembergroups"
)

func AddCORS(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		c.Response().Header().Set("Access-Control-Allow-Origin", "*")
		return next(c)
	}
}
func setupRoutes(e *echo.Echo, ctx context.Context, client *nodeclient.Client) {
	im.PublicKeyDrainer = im.NewItemDrainer(ctx, func(item interface{}) {

		// unwrap to *NFTWithRespChan
		nftWithRespChan := item.(*im.NFTWithRespChan)
		// get address from nft
		address := string(nftWithRespChan.NFT.OwnerAddress)

		nftResponse := &im.NFTResponse{
			OwnerAddress: address,
			PublicKey:    "",
		}
		// get public key from address
		publicKeyBytes, err := deps.IMManager.GetAddressPublicKey(ctx, client, address, false, CoreComponent.Logger())
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... GetAddressPublicKey failed:%s", err)
		}
		var publicKey string
		if publicKeyBytes != nil {
			publicKey = iotago.EncodeHex(publicKeyBytes)
		} else {
			publicKey = ""
		}

		// make *NFTResponse
		nftResponse.PublicKey = publicKey
		// send to respChan
		nftWithRespChan.RespChan <- nftResponse
	}, 2000, 1000, 1000)
	//e.Use(AddCORS)
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
		resp, err := getNFTsWithPublicKeyFromGroupId(c, im.PublicKeyDrainer)
		// filter out nfts with empty public key
		filteredNFTs := make([]*im.NFTResponse, 0)
		for _, nft := range resp {
			if nft.PublicKey != "" {
				filteredNFTs = append(filteredNFTs, nft)
			}
		}
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, filteredNFTs)
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

	e.GET("/testinboxlist", func(c echo.Context) error {
		//prefix = []byte{im.ImStoreKeyPrefixInbox}
		prefix := []byte{im.ImStoreKeyPrefixInbox}
		err := deps.IMManager.LogAllData(prefix, CoreComponent.Logger())
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
	// testmeta
	e.GET("/testnftmeta", func(c echo.Context) error {
		outputIdHex, err := parseAttrNameQueryParam(c, "outputId")
		if err != nil {
			return err
		}
		nodeHttpClient := nodeclient.New("https://api.shimmer.network")
		outputId, err := iotago.OutputIDFromHex(outputIdHex)
		if err != nil {
			return err
		}
		output, err := nodeHttpClient.OutputByID(context.Background(), outputId)
		if err != nil {
			return err
		}
		nftOutput, is := output.(*iotago.NFTOutput)
		if !is {
			// create a new error
			return errors.New("output is not nft")
		}
		featureSet, err := nftOutput.ImmutableFeatures.Set()
		if err != nil {
			return err
		}
		meta := featureSet.MetadataFeature()
		if meta == nil {
			return errors.New("meta is nil")
		}
		// meta is json string in bytes, parse it to map
		metaMap := make(map[string]interface{})
		CoreComponent.Logger().Infof("meta data:%s", string(meta.Data))
		err = json.Unmarshal(meta.Data, &metaMap)
		if err != nil {
			return err
		}
		uri := metaMap["uri"].(string)
		return httpserver.JSONResponse(c, http.StatusOK, uri)
	})

	e.GET(RouteIMAddressGroupIds, func(c echo.Context) error {
		groupIds, err := getGroupIdsFromAddress(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, groupIds)
	})

	//RouteIMAddressGroupDetails
	e.GET(RouteIMAddressGroupDetails, func(c echo.Context) error {
		resp, err := getAddressGroupDetails(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
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

	// inbox message
	e.GET(RouteInboxMessage, func(c echo.Context) error {
		resp, err := getInboxMessage(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// group qualified addresses
	e.GET(RouteGroupQualifiedAddresses, func(c echo.Context) error {
		resp, err := getQualifiedAddressesForGroupId(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// group marked addresses
	e.GET(RouteGroupMarkedAddresses, func(c echo.Context) error {
		resp, err := getMarkedAddressesFromGroupId(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// group member addresses
	e.GET(RouteGroupMemberAddresses, func(c echo.Context) error {
		resp, err := getGroupMemberAddressesFromGroupId(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// get group votes
	e.GET(RouteGroupVotes, func(c echo.Context) error {
		resp, err := getGroupVotes(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// get group votes count
	e.GET(RouteGroupVotesCount, func(c echo.Context) error {
		resp, err := getGroupVotesCount(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// get group blacklist
	e.GET(RouteGroupBlacklist, func(c echo.Context) error {
		resp, err := getGroupBlacklist(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// get address member groups
	e.GET(RouteAddressMemberGroups, func(c echo.Context) error {
		resp, err := getAddressMemberGroups(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// get public key of one address
	e.GET("/getaddresspublickey", func(c echo.Context) error {
		address, err := parseAddressQueryParam(c)
		if err != nil {
			return err
		}
		CoreComponent.LogInfof("get address public key from address:%s", address)
		publicKeyBytes, err := deps.IMManager.GetAddressPublicKey(ctx, client, address, true, CoreComponent.Logger())
		if err != nil {
			return err
		}
		publicKey := iotago.EncodeHex(publicKeyBytes)
		return httpserver.JSONResponse(c, http.StatusOK, publicKey)
	})
}
