package im

import (
	"context"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"github.com/labstack/gommon/log"
)

type InitContext struct {
	Ctx           context.Context
	Client        *nodeclient.Client
	IndexerClient nodeclient.IndexerClient
	Logger        *logger.Logger
}

type OutputIdsFetcher func(initCtx *InitContext, offset *string) ([]string, *string, error)

type OutputProcessor func(outputId []byte, output iotago.Output, mileStoneIndex uint32, mileStoneTimestamp uint32,
	initCtx *InitContext) error

// handleGenericInit function will
// maintain a mark for is finished,
// iterate all output under certain filer,

func HandleGenericInit(initCtx *InitContext,
	topic string,
	outputIdsFetcher OutputIdsFetcher,
	outputProcessors []OutputProcessor) {

	drainer := im.NewItemDrainer(initCtx.Ctx, func(outputIdUnwrapped interface{}) {
		outputIdHex := outputIdUnwrapped.(string)
		output, milestoneIndex, milestoneTimestamp,
			err := deps.IMManager.OutputIdToOutputAndMilestoneInfo(initCtx.Ctx, initCtx.Client, outputIdHex)
		if err != nil {
			// log error
			initCtx.Logger.Warnf("LedgerInit ... OutputIdToOutput failed:%s", err)
			return
		}
		outputId, err := iotago.DecodeHex(outputIdHex)
		if err != nil {
			// log error
			initCtx.Logger.Warnf("LedgerInit ... DecodeHex failed:%s", err)
			return
		}
		// loop through all output processors
		for _, processor := range outputProcessors {
			err := processor(outputId, output, milestoneIndex, milestoneTimestamp, initCtx)
			if err != nil {
				// log error and continue
				initCtx.Logger.Warnf("LedgerInit ... OutputProcessor failed:%s", err)
				continue
			}
		}
	}, 1000, 100, 1000)
	// check if finished
	isFinished, err := deps.IMManager.IsInitFinished(topic, "")
	if err != nil {
		log.Errorf("failed to ReadInitFinished for %s: %s", topic, err)
		return
	}
	if isFinished {
		return
	}
	// get current offset
	offset, err := deps.IMManager.ReadInitCurrentOffset(topic, "")
	if err != nil {
		log.Errorf("failed to ReadInitCurrentOffset for %s: %s", topic, err)
		return
	}
Loop:
	for {
		// select on ctx done
		select {
		case <-initCtx.Ctx.Done():
			log.Infof("LedgerInit ... ctx.Done()")
			return
		default:
			// get output ids
			outputIds, nextOffset, err := outputIdsFetcher(initCtx, offset)
			if err != nil {
				log.Errorf("failed to fetch output ids for %s: %s", topic, err)
				return
			}
			// convert []string to []interface{}
			outputIdsInterface := make([]interface{}, len(outputIds))
			for i, v := range outputIds {
				outputIdsInterface[i] = v
			}
			// drain
			drainer.Drain(outputIdsInterface)
			// update offset
			if nextOffset != nil {
				err = deps.IMManager.StoreInitCurrentOffset(nextOffset, topic, "")
				if err != nil {
					log.Errorf("failed to StoreInitCurrentOffset for %s: %s", topic, err)
					return
				}
			}
			// check if there is more
			if nextOffset == nil {
				break Loop
			}
			offset = nextOffset
		}
	}

	// mark finished
	err = deps.IMManager.MarkInitFinished(topic, "")
	if err != nil {
		log.Errorf("failed to MarkInitFinished for %s: %s", topic, err)
		return
	}
}

// All nft output fetcher
// deps.IMManager.QueryNFTIds(ctx, indexerClient, initOffset, 1000, logger)
var AllNftOutputIdsFetcher = func(initCtx *InitContext, offset *string) ([]string, *string, error) {
	ids, nextOffset, err := deps.IMManager.QueryNFTIds(initCtx.Ctx, initCtx.IndexerClient, offset, 1000, initCtx.Logger)
	if err != nil {
		return nil, nil, err
	}
	return ids, nextOffset, nil
}

// All basic output fetcher
// outputHexIds, nextOffset, err := deps.IMManager.QueryBasicOutputIds(ctx, indexerClient, initOffset, CoreComponent.Logger(), drainer.FetchSize)
var AllBasicOutputIdsFetcher = func(initCtx *InitContext, offset *string) ([]string, *string, error) {
	ids, nextOffset, err := deps.IMManager.QueryBasicOutputIds(initCtx.Ctx, initCtx.IndexerClient, offset, initCtx.Logger, 1000)
	if err != nil {
		return nil, nil, err
	}
	return ids, nextOffset, nil
}

// basic output with tag fetcher
// outputHexIds, nextOffset, err := deps.IMManager.QueryOutputIdsByTag(ctx, indexerClient, tag, initOffset, CoreComponent.Logger())
var BasicOutputIdsByTagFetcher = func(tag string) OutputIdsFetcher {
	return func(initCtx *InitContext, offset *string) ([]string, *string, error) {
		ids, nextOffset, err := deps.IMManager.QueryOutputIdsByTag(initCtx.Ctx, initCtx.IndexerClient, tag, offset, initCtx.Logger)
		if err != nil {
			return nil, nil, err
		}
		return ids, nextOffset, nil
	}
}
