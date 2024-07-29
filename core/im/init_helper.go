package im

import (
	"bytes"
	"context"
	"sort"
	"sync"

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

type OutputProcessor func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32,
	initCtx *InitContext) error

// OutputWithId struct to hold output and its ID
type OutputWithId struct {
	OutputId           []byte
	Output             iotago.Output
	MilestoneIndex     uint32
	MilestoneTimestamp uint32
}

// handleGenericInit function will maintain a mark for is finished,
// iterate all output under certain filter,
func HandleGenericInit(initCtx *InitContext,
	topic string,
	outputIdsFetcher OutputIdsFetcher,
	outputProcessors []OutputProcessor) {

	outputChan := make(chan *OutputWithId, 200)

	// Object pool for OutputWithId
	outputWithIdPool := sync.Pool{
		New: func() interface{} {
			return &OutputWithId{}
		},
	}

	drainer := im.NewItemDrainer(initCtx.Ctx, func(outputIdUnwrapped interface{}) {
		outputIdHex := outputIdUnwrapped.(string)
		output, milestoneIndex, milestoneTimestamp, err := deps.IMManager.OutputIdToOutputAndMilestoneInfo(initCtx.Ctx, initCtx.Client, outputIdHex)
		if err != nil {
			initCtx.Logger.Warnf("LedgerInit ... OutputIdToOutput failed: %s", err)
			return
		}
		outputId, err := iotago.DecodeHex(outputIdHex)
		if err != nil {
			initCtx.Logger.Warnf("LedgerInit ... DecodeHex failed: %s", err)
			return
		}
		ow := outputWithIdPool.Get().(*OutputWithId)
		ow.OutputId = outputId
		ow.Output = output
		ow.MilestoneIndex = milestoneIndex
		ow.MilestoneTimestamp = milestoneTimestamp
		outputChan <- ow
	}, 200, 100, 1000)

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
		select {
		case <-initCtx.Ctx.Done():
			log.Infof("LedgerInit ... ctx.Done()")
			break Loop
		default:
			outputIds, nextOffset, err := outputIdsFetcher(initCtx, offset)
			if err != nil {
				log.Errorf("failed to fetch output ids for %s: %s", topic, err)
				continue
			}

			outputIdsInterface := make([]interface{}, len(outputIds))
			for i, v := range outputIds {
				outputIdsInterface[i] = v
			}

			initCtx.Logger.Infof("Draining %d outputIds", len(outputIdsInterface))
			drainer.Drain(outputIdsInterface)
			drainer.Wait()
			initCtx.Logger.Info("Finished waiting for drainer")

			// Collect outputs from channel
			var outputs []*OutputWithId
		CollectLoop:
			for {
				select {
				case ow := <-outputChan:
					outputs = append(outputs, ow)
				default:
					break CollectLoop
				}
			}

			initCtx.Logger.Infof("Collected %d outputs", len(outputs))

			// Sort outputs by OutputId
			sort.Slice(outputs, func(i, j int) bool {
				return bytes.Compare(outputs[i].OutputId, outputs[j].OutputId) < 0
			})

			// Process each output in sorted order
			for _, ow := range outputs {
				for _, processor := range outputProcessors {
					if err := processor(ow.OutputId, ow.Output, ow.MilestoneIndex, ow.MilestoneTimestamp, initCtx); err != nil {
						initCtx.Logger.Warnf("LedgerInit ... OutputProcessor failed: %s", err)
						continue
					}
				}
				// Put the used OutputWithId back to the pool
				outputWithIdPool.Put(ow)
			}

			if nextOffset != nil {
				if err := deps.IMManager.StoreInitCurrentOffset(nextOffset, topic, ""); err != nil {
					log.Errorf("failed to StoreInitCurrentOffset for %s: %s", topic, err)
					continue
				}
			}

			if nextOffset == nil {
				break Loop
			}
			offset = nextOffset
		}
	}

	if err := deps.IMManager.MarkInitFinished(topic, ""); err != nil {
		log.Errorf("failed to MarkInitFinished for %s: %s", topic, err)
		return
	}

	log.Infof("LedgerInit ... %s finished", topic)
}

// All nft output fetcher
var AllNftOutputIdsFetcher = func(initCtx *InitContext, offset *string) ([]string, *string, error) {
	ids, nextOffset, err := deps.IMManager.QueryNFTIds(initCtx.Ctx, initCtx.IndexerClient, offset, 1000, initCtx.Logger)
	if err != nil {
		return nil, nil, err
	}
	return ids, nextOffset, nil
}

// All basic output fetcher
var AllBasicOutputIdsFetcher = func(initCtx *InitContext, offset *string) ([]string, *string, error) {
	ids, nextOffset, err := deps.IMManager.QueryBasicOutputIds(initCtx.Ctx, initCtx.IndexerClient, offset, initCtx.Logger, 1000)
	if err != nil {
		return nil, nil, err
	}
	return ids, nextOffset, nil
}

// basic output with tag fetcher
var BasicOutputIdsByTagFetcher = func(tag string) OutputIdsFetcher {
	return func(initCtx *InitContext, offset *string) ([]string, *string, error) {
		ids, nextOffset, err := deps.IMManager.QueryOutputIdsByTag(initCtx.Ctx, initCtx.IndexerClient, tag, offset, initCtx.Logger)
		if err != nil {
			return nil, nil, err
		}
		return ids, nextOffset, nil
	}
}

// nft output with tag fetcher
var NftOutputIdsByTagFetcher = func(tag string) OutputIdsFetcher {
	return func(initCtx *InitContext, offset *string) ([]string, *string, error) {
		ids, nextOffset, err := deps.IMManager.QueryNFTOutputIdsByCollectionId(initCtx.Ctx, initCtx.IndexerClient, tag, offset, initCtx.Logger)
		if err != nil {
			return nil, nil, err
		}
		return ids, nextOffset, nil
	}
}

func HandleGenericInitParallel(initCtx *InitContext,
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
	}, 200, 100, 1000)
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
				continue
			}
			// convert []string to []interface{}
			outputIdsInterface := make([]interface{}, len(outputIds))
			for i, v := range outputIds {
				outputIdsInterface[i] = v
			}
			// drain
			drainer.Drain(outputIdsInterface)
			drainer.Wait() // Wait for all items to be processed
			// update offset
			if nextOffset != nil {
				err = deps.IMManager.StoreInitCurrentOffset(nextOffset, topic, "")
				if err != nil {
					log.Errorf("failed to StoreInitCurrentOffset for %s: %s", topic, err)
					continue
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
	// log topic finished
	log.Infof("LedgerInit ... %s finished", topic)
}
