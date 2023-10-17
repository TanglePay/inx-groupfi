package im

import (
	"context"
	"math/big"
	"time"

	"github.com/TanglePay/inx-iotacat/pkg/im"
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
	handleTotalSmrInit(ctx, client, indexerClient, ItemDrainer)
}
func handleTotalSmrInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, ItemDrainer *im.ItemDrainer) {
	smrTotalAddress := deps.IMManager.GetTotalAddressFromType(im.ImTokenTypeSMR)

	smrTotalValue, err := deps.IMManager.GetBalanceOfOneAddress(im.ImTokenTypeSMR, smrTotalAddress)

	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	smrTotal := GetSmrTokenTotal()
	smrTotal.Add(smrTotalValue)

	//handle whale eligibility
	isWhaleEligibilityFinished, err := deps.IMManager.IsInitFinished(im.WhaleEligibility, "")
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	if !isWhaleEligibilityFinished {
		smrTokenPrefix := deps.IMManager.TokenKeyPrefixFromTokenType(im.ImTokenTypeSMR)

		// hash set for address
		addressSet := make(map[string]struct{})
		deps.IMManager.GetImStore().Iterate(smrTokenPrefix, func(key kvstore.Key, value kvstore.Value) bool {
			_, address := deps.IMManager.AmountAndAddressFromTokenValuePayload(value)
			addressSet[address] = struct{}{}
			return true
		})
		// loop addressSet
		for address := range addressSet {
			handleTokenWhaleEligibilityFromAddressGivenTotalAmount(im.ImTokenTypeSMR, address, smrTotal.Get(), deps.IMManager, CoreComponent.Logger())
		}
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

	totalBasicOutputProcessed := big.NewInt(0)
	startTimeBasic := time.Now()
	for !isTokenBasicFinished {
		select {
		case <-ctx.Done():
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			ct, isHasMore, err := processInitializationForTokenForBasicOutput(ctx, client, indexerClient, ItemDrainer)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForTokenForBasicOutput failed:%s", err)
				continue
			}
			// totalBasicOutputProcessed = totalBasicOutputProcessed + ct
			totalBasicOutputProcessed.Add(totalBasicOutputProcessed, big.NewInt(int64(ct)))
			timeElapsed := time.Since(startTimeBasic)
			averageProcessedPerSecond := big.NewInt(0)
			averageProcessedPerSecond.Div(totalBasicOutputProcessed, big.NewInt(int64(timeElapsed.Seconds())+1))
			// log totalBasicOutputProcessed and timeElapsed and averageProcessedPerSecond
			CoreComponent.LogInfof("LedgerInit ... totalBasicOutputProcessed:%d,timeElapsed:%s,averageProcessedPerSecond:%d", totalBasicOutputProcessed, timeElapsed, averageProcessedPerSecond)
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
