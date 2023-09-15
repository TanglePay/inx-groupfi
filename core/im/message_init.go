package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func handleMessageInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	// handle messsages init
	isMessageInitializationFinished, err := deps.IMManager.IsInitFinished(im.MessageType, "")
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	// loop until isMessageInitializationFinished is true
	for !isMessageInitializationFinished {
		select {
		case <-ctx.Done():
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			messages, isHasMore, err := processInitializationForMessage(ctx, client, indexerClient)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForMessage failed:%s", err)
				continue
			}
			if len(messages) > 0 {
				DataFromListenning := &im.DataFromListenning{
					CreatedMessage: messages,
				}
				err = deps.IMManager.ApplyNewLedgerUpdate(0, DataFromListenning, CoreComponent.Logger(), true)
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... ApplyNewLedgerUpdate failed:%s", err)
					continue
				}
			}
			if !isHasMore {
				err = deps.IMManager.MarkInitFinished(im.MessageType, "")
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
					continue
				}
				isMessageInitializationFinished = true
			}
		}
	}
}

func processInitializationForMessage(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) ([]*im.Message, bool, error) {
	// get init offset
	itemType := im.MessageType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, false, err
	}
	// get outputs and meta data
	messages, nextOffset, err := fetchNextMessage(ctx, client, indexerClient, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextMessage failed:%s", err)
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
