package im

import (
	"bytes"
	"context"
	"math/big"
	"time"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func handleTokenInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	// make drainer
	ItemDrainer := makeTokenInitDrainer(ctx, client, indexerClient)

	// handle token basic init
	handleTokenBasicInit(ctx, client, indexerClient, ItemDrainer)

	// handle nft init
	handleNftTokenInit(ctx, client, indexerClient, ItemDrainer)

	//handle total smr
	handleTotalInit(ctx, client, indexerClient, ItemDrainer)
}
func handleTotalInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, ItemDrainer *im.ItemDrainer) {

	//handle whale eligibility
	isWhaleEligibilityFinished, err := deps.IMManager.IsInitFinished(im.WhaleEligibility, "")
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	//TODO remove
	isWhaleEligibilityFinished = false
	if !isWhaleEligibilityFinished {
		tokenPrefix := deps.IMManager.TokenKeyPrefixForAll()
		var currentTokenId []byte
		var currentTokenIdHash [im.Sha256HashLen]byte
		var currentAddress string
		var currentTotalAddress string
		isPreviousAddressTotalAddress := false
		currentTokenTotal := big.NewInt(0)
		currentAddressTotal := big.NewInt(0)
		// hash set for address
		deps.IMManager.GetImStore().Iterate(tokenPrefix, func(key kvstore.Key, value kvstore.Value) bool {
			tokenStat, err := deps.IMManager.TokenStateFromKeyAndValue(key, value)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... TokenStateFromKeyAndValue failed:%s", err)
				return true
			}

			// if address is different, then current total is address total, if address is total address, then current total is token total
			// when get token total, update it, when get address total, call handle
			if tokenStat != nil && tokenStat.TokenId != nil && tokenStat.Address != "" && tokenStat.Amount != "" {

				var isTokenIdDifferent bool
				var isAddressDifferent bool
				if currentTokenId == nil || bytes.Equal(currentTokenId, tokenStat.TokenId) {
					isTokenIdDifferent = true
					// set currentTokenIdHash, update isCurrentAddressTotalAddress
					currentTokenId = tokenStat.TokenId
					// log currentTokenId
					CoreComponent.LogInfof("start with tokenId:%s", iotago.EncodeHex(currentTokenId))
					currentTokenIdHash = tokenStat.TokenIdHash
				}
				if currentAddress == "" || currentAddress == tokenStat.Address {
					isAddressDifferent = true
					// set currentAddress
					currentAddress = tokenStat.Address
				}
				// if tokenId is different, then first address is total address
				if isTokenIdDifferent {
					currentTotalAddress = currentAddress
					// log currentTotalAddress
					CoreComponent.LogInfof("start with total address:%s", currentTotalAddress)
				}
				// if previous address is total address, and current address is not total address, then current total is token total

				// handle address total
				if isAddressDifferent {
					// case total address
					if isPreviousAddressTotalAddress {
						// log tokenId, currentTotalAddress, currentTokenTotal
						CoreComponent.LogInfof("tokenId:%s,currentTotalAddress:%s,currentTokenTotal:%d", iotago.EncodeHex(currentTokenId), currentTotalAddress, currentAddressTotal)
						GetTokenTotal(currentTokenIdHash).Add(currentAddressTotal)
						currentTokenTotal = currentAddressTotal
					} else {
						//case normal address
						err = handleTokenWhaleEligibilityFromAddressGivenTotalAmount(
							currentTokenId,
							currentTokenIdHash,
							currentAddress,
							currentAddressTotal,
							deps.IMManager,
							CoreComponent.Logger(),
						)
						if err != nil {
							// log error
							CoreComponent.LogWarnf("LedgerInit ... handleTokenWhaleEligibilityFromAddressGivenTotalAmount failed:%s", err)
						}

					}
					// log tokenId, currentAddress, currentAddressTotal
					CoreComponent.LogInfof("tokenId:%s,currentAddress:%s,currentAddressTotal:%d", iotago.EncodeHex(currentTokenId), currentAddress, currentAddressTotal)
					currentAddressTotal = big.NewInt(0)
				}
				// add amount to current total
				amount, ok := new(big.Int).SetString(tokenStat.Amount, 10)
				if !ok {
					// log error
					CoreComponent.LogWarnf("LedgerInit ... SetString failed:%s", err)
				}
				if tokenStat.Status == im.ImTokenStatusCreated {
					currentAddressTotal.Add(currentAddressTotal, amount)
				} else {
					currentAddressTotal.Sub(currentAddressTotal, amount)
				}

				isPreviousAddressTotalAddress = currentTotalAddress == currentAddress

			}
			return true
		})
		err = deps.IMManager.MarkInitFinished(im.WhaleEligibility, "")
		if err != nil {
			CoreComponent.LogPanicf("LedgerInit ... MarkInitFinished failed:%s", err)
		}
	}
}
func handleNftTokenInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, ItemDrainer *im.ItemDrainer) {

	totalNFTOutputProcessed := big.NewInt(0)
	startTimeNFT := time.Now()
	isTokenNFTFinished, err := deps.IMManager.IsInitFinished(im.TokenNFTType, "")
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	for !isTokenNFTFinished {
		select {
		case <-ctx.Done():
			// log ctx cancel then exit
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			ct, isHasMore, err := processInitializationForTokenForNftOutput(ctx, client, indexerClient, ItemDrainer)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForTokenForNftOutput failed:%s", err)
				continue
			}
			totalNFTOutputProcessed.Add(totalNFTOutputProcessed, big.NewInt(int64(ct)))
			timeElapsed := time.Since(startTimeNFT)
			averageProcessedPerSecond := big.NewInt(0)
			averageProcessedPerSecond.Div(totalNFTOutputProcessed, big.NewInt(int64(timeElapsed.Seconds())+1))
			// log totalBasicOutputProcessed and timeElapsed and averageProcessedPerSecond
			CoreComponent.LogInfof("LedgerInit ... totalBasicOutputProcessed:%d,timeElapsed:%s,averageProcessedPerSecond:%d", totalNFTOutputProcessed, timeElapsed, averageProcessedPerSecond)

			if !isHasMore {
				err = deps.IMManager.MarkInitFinished(im.TokenNFTType, "")
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
					continue
				}
				isTokenNFTFinished = true
			}
		}
	}

}
func handleTokenBasicInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, ItemDrainer *im.ItemDrainer) {
	isTokenBasicFinished, err := deps.IMManager.IsInitFinished(im.TokenBasicType, "")
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	for !isTokenBasicFinished {
		select {
		case <-ctx.Done():
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			_, isHasMore, err := processInitializationForTokenForBasicOutput(ctx, client, indexerClient, ItemDrainer)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForTokenForBasicOutput failed:%s", err)
				continue
			}
			if !isHasMore {
				err = deps.IMManager.MarkInitFinished(im.TokenBasicType, "")
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
					continue
				}
				isTokenBasicFinished = true
			}
		}
	}
}
func makeTokenInitDrainer(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) *im.ItemDrainer {
	ItemDrainer := im.NewItemDrainer(ctx, func(outputIdUnwrapped interface{}) {
		outputId := outputIdUnwrapped.(string)
		output, err := deps.IMManager.OutputIdToOutput(ctx, client, outputId)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
			return
		}
		outputIdBytes, _ := iotago.DecodeHex(outputId)
		err = handleTokenFromINXOutput(output, outputIdBytes, ImOutputTypeCreated, false)
		if err != nil {
			// log error then continue
			CoreComponent.LogWarnf("LedgerInit ... handleTokenFromINXOutput failed:%s", err)

		}
		// handle group config
	}, 1000, 100, 2000)
	return ItemDrainer
}
