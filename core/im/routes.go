package im

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/hive.go/core/kvstore"
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

	// address qualified group configs
	RouteIMAddressQualifiedGroupConfigs = "/addressqualifiedgroupconfigs"

	// consolidation for message
	RouteImConsolidationForMessage = "/consolidation/message"
	// consolidation for shared
	RouteImConsolidationForShared = "/consolidation/shared"

	// health check
	RouteHealthCheck = "/healthcheck"

	// group configs for renter
	RouteGroupConfigs = "/groupconfigs"

	// inbox items
	RouteInboxItems = "/inboxitems"

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

	// get address mark groups
	RouteAddressMarkGroups = "/addressmarkgroups"

	// get group user reputation
	RouteGroupUserReputation = "/groupuserreputation"

	// get user group reputation
	RouteUserGroupReputation = "/usergroupreputation"

	// test repuation
	RouteTestReputation = "/testreputation"

	// get address balance
	RouteAddressBalance = "/addressbalance"
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
		timestamp := nftWithRespChan.NFT.MileStoneTimestamp

		nftResponse := &im.NFTResponse{
			OwnerAddress: address,
			PublicKey:    "",
			Timestamp:    timestamp,
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
		address, err := parseAddressQueryParamWithNil(c)
		if err != nil {
			return err
		}
		tokenId, err := parseTokenQueryParam(c)
		if err != nil {
			return err
		}
		if tokenId == nil {
			tokenId = im.SmrTokenId
		}
		tokenIdFixed := [im.Sha256HashLen]byte{}
		copy(tokenIdFixed[:], im.Sha256HashBytes(tokenId))
		balance, err := deps.IMManager.GetBalanceOfOneAddress(tokenId, address)
		if err != nil {
			return err
		}
		totalBalance := GetTokenTotal(tokenIdFixed).Get()
		resp := &TokenBalanceResponse{
			Balance:      balance.Text(10),
			TotalBalance: totalBalance.Text(10),
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})
	// get address balance
	e.GET(RouteAddressBalance, func(c echo.Context) error {
		address, err := parseAddressQueryParam(c)
		if err != nil {
			return err
		}
		tokenId, err := parseTokenQueryParam(c)
		if err != nil {
			return err
		}
		if tokenId == nil {
			tokenId = im.SmrTokenId
		}
		balance, err := deps.IMManager.GetBalanceOfOneAddress(tokenId, address)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, balance.Text(10))
	})
	// log address tokenstat
	e.GET("/logaddresstokenstat", func(c echo.Context) error {
		address, err := parseAddressQueryParamWithNil(c)
		if err != nil {
			return err
		}
		tokenId, err := parseTokenQueryParam(c)
		if err != nil {
			return err
		}
		if tokenId == nil {
			tokenId = im.SmrTokenId
		}
		allZero := [im.Sha256HashLen]byte{}
		addressSha256 := []byte(allZero[:])
		if address != "" {
			addressSha256 = im.Sha256Hash(address)
		}
		keyPrefix := deps.IMManager.TokenKeyPrefixFromTokenIdAndAddress(tokenId, addressSha256)
		deps.IMManager.GetImStore().Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
			tokenStat, err := deps.IMManager.TokenStateFromKeyAndValue(key, value)
			if err != nil {
				// log error then continue
				CoreComponent.Logger().Errorf("TokenStateFromKeyAndValue failed:%s", err)
				return true
			}
			// log tokenStat, amount, address, status
			amountStr := tokenStat.Amount
			amount, ok := new(big.Int).SetString(amountStr, 10)
			if !ok {
				// log error then continue
				CoreComponent.Logger().Errorf("SetString failed:%s", err)
				return true
			}
			tokenIdHex := iotago.EncodeHex(tokenStat.TokenId)
			CoreComponent.Logger().Infof("tokenId:%s, address:%s, amount:%s, status:%d", tokenIdHex, address, amount.Text(10), tokenStat.Status)
			return true
		})
		return httpserver.JSONResponse(c, http.StatusOK, "ok")
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

	//addressqualifiedgroupconfigs
	// switch to using post
	e.POST(RouteIMAddressQualifiedGroupConfigs, func(c echo.Context) error {
		groupConfigs, err := getQualifiedGroupConfigsFromAddress(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, groupConfigs)
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

	// inbox items
	e.GET(RouteInboxItems, func(c echo.Context) error {
		resp, err := getInboxList(c)
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
		resp, err := getGroupMembersFromGroupId(c)
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

	// RouteHealthCheck
	e.GET(RouteHealthCheck, func(c echo.Context) error {
		bootTime := im.BootTime
		isIniting := im.IsIniting
		lastTimeReceiveEventFromHornet := im.LastTimeReceiveEventFromHornet
		currentTime := im.GetCurrentEpochTimestamp()
		// if isIniting is true, then if timeelapsed since boot time is more than 1 hour, then return 503 service unavailable
		// if isIniting is false, then if timeelapsed since last time receive event from hornet is more than 5 minutes, then return 503 service unavailable
		// otherwise return 200 ok
		if isIniting {
			timeElapsed := currentTime - bootTime
			if timeElapsed > 5*60 {
				return c.String(http.StatusServiceUnavailable, "503 service unavailable")
			}
		} else {
			timeElapsed := currentTime - lastTimeReceiveEventFromHornet
			if timeElapsed > 5*60 {
				return c.String(http.StatusServiceUnavailable, "503 service unavailable")
			}
		}
		return httpserver.JSONResponse(c, http.StatusOK, "ok")
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

	// get address mark groups
	e.GET(RouteAddressMarkGroups, func(c echo.Context) error {
		groupIds, err := getAddressMarkGroups(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, groupIds)
	})

	// get public key of one address
	e.GET("/getaddresspublickey", func(c echo.Context) error {
		address, err := parseAddressQueryParam(c)
		if err != nil {
			return err
		}
		CoreComponent.LogInfof("get address public key from address:%s", address)
		publicKeyBytes, err := deps.IMManager.GetAddressPublicKey(ctx, client, address, false, CoreComponent.Logger())
		if err != nil {
			return err
		}
		publicKey := iotago.EncodeHex(publicKeyBytes)
		return httpserver.JSONResponse(c, http.StatusOK, publicKey)
	})

	// delete public key of one address
	e.GET("/deleteaddresspublickey", func(c echo.Context) error {
		address, err := parseAddressQueryParam(c)
		if err != nil {
			return err
		}
		CoreComponent.LogInfof("delete address public key from address:%s", address)
		err = deps.IMManager.DeleteOnePublicKey(address)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, "ok")
	})
	// get group user reputation
	e.GET(RouteGroupUserReputation, func(c echo.Context) error {
		resp, err := getGroupUserReputation(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// get user group reputation
	e.GET(RouteUserGroupReputation, func(c echo.Context) error {
		resp, err := getUserGroupReputation(c)
		if err != nil {
			return err
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

	// test reputation
	e.GET(RouteTestReputation, func(c echo.Context) error {
		address, err := parseAddressQueryParam(c)
		if err != nil {
			return err
		}
		// get group id
		groupId, err := parseGroupIdQueryParam(c)
		if err != nil {
			return err
		}
		addressSha256Hash := im.Sha256Hash(address)
		var groupId32 [32]byte
		copy(groupId32[:], groupId)
		var addressSha256Hash32 [32]byte
		copy(addressSha256Hash32[:], addressSha256Hash)
		mutedTimes, err := deps.IMManager.CountMutedTimes(groupId32, addressSha256Hash32)
		if err != nil {
			return err
		}
		addresses, err := deps.IMManager.GetGroupMemberAddressesFromGroupId(groupId32, nil)
		if err != nil {
			return err
		}
		groupMemberCount := len(addresses)
		// log address and group id and muted times and group member count
		CoreComponent.Logger().Infof("address:%s, groupId:%s, mutedTimes:%d, groupMemberCount:%d", address, iotago.EncodeHex(groupId[:]), mutedTimes, groupMemberCount)
		resp := &TestReputationResponse{
			MutedCount:       mutedTimes,
			GroupMemberCount: groupMemberCount,
		}
		return httpserver.JSONResponse(c, http.StatusOK, resp)
	})

}
