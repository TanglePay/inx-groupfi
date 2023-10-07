package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handle mark init
func handleMarkInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	// make drainer
	drainer := makeDrainerForMarkInit(ctx, client, indexerClient)
	// handle mark init
	handleTypeInit(ctx, client, indexerClient, drainer, im.MarkType, im.MarkTagStr)
}

func handleTypeInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, drainer *im.ItemDrainer, itemType im.IndexerItemType, tag string) {

	isInitializationFinished, err := deps.IMManager.IsInitFinished(itemType, tag)
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	for !isInitializationFinished {
		select {
		case <-ctx.Done():
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			_, isHasMore, err := processInitializationForTag(ctx, client, indexerClient, tag, drainer)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForTag failed:%s", err)
				continue
			}
			if !isHasMore {
				err = deps.IMManager.MarkInitFinished(itemType, tag)
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
					continue
				}
				isInitializationFinished = true
			}
		}
	}
}
func processInitializationForTag(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, tag string, drainer *im.ItemDrainer) (int, bool, error) {
	// get init offset
	itemType := im.MarkType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return 0, false, err
	}
	// get outputs and meta data
	outputHexIds, nextOffset, err := deps.IMManager.QueryOutputIdsByTag(ctx, indexerClient, tag, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextGroupConfigNFTs failed:%s", err)
		return 0, false, err
	}
	//drain
	outputHexIdsInterface := make([]interface{}, len(outputHexIds))
	for i, v := range outputHexIds {
		outputHexIdsInterface[i] = v
	}
	drainer.Drain(outputHexIdsInterface)
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return 0, false, err
		}
	}
	isHasMore := nextOffset != nil
	return len(outputHexIds), isHasMore, nil
}

func makeDrainerForMarkInit(ctx context.Context, nodeHTTPAPIClient *nodeclient.Client, indexerClient nodeclient.IndexerClient) *im.ItemDrainer {
	return im.NewItemDrainer(ctx, func(outputIdUnwrapped interface{}) {
		outputId := outputIdUnwrapped.(string)
		output, err := deps.IMManager.OutputIdToOutput(ctx, nodeHTTPAPIClient, outputId)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
			return
		}
		// filter mark output
		basicOutput, is := deps.IMManager.FilterMarkOutput(output)
		if !is {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... FilterMarkOutput failed")
			return
		}
		deps.IMManager.HandleGroupMarkBasicOutputCreated(basicOutput)
	}, 100, 10, 200)
}
