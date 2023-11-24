package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func handleSharedInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	// handle shared init
	isSharedInitializationFinished, err := deps.IMManager.IsInitFinished(im.SharedType, "")
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	// loop until isSharedInitializationFinished is true
	for !isSharedInitializationFinished {
		select {
		case <-ctx.Done():
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			messages, isHasMore, err := processInitializationForShared(ctx, client, indexerClient)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForShared failed:%s", err)
				continue
			}
			if len(messages) > 0 {
				DataFromListenning := &im.DataFromListenning{
					CreatedShared: messages,
				}
				err = deps.IMManager.ApplyNewLedgerUpdate(0, DataFromListenning, CoreComponent.Logger(), true)
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... ApplyNewLedgerUpdate failed:%s", err)
					continue
				}
			}
			if !isHasMore {
				err = deps.IMManager.MarkInitFinished(im.SharedType, "")
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
					continue
				}
				isSharedInitializationFinished = true
			}
		}
	}
}

// processInitializationForShared
func processInitializationForShared(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) ([]*im.Message, bool, error) {
	// get init offset
	itemType := im.SharedType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, false, err
	}
	// get outputs and meta data
	messages, nextOffset, err := fetchNextShared(ctx, client, indexerClient, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextShared failed:%s", err)
		return nil, false, err
	}
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return nil, false, err
		}
	}
	isHasMore := nextOffset != nil
	return messages, isHasMore, nil
}
