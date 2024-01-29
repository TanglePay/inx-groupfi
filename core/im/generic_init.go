package im

import (
	"context"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handleGenericInit function will
// maintain a mark for is finished,
// iterate all output under certain filer,

func HandleGenericInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, topic string,
	outputIdsFetcher func(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, initOffset *string, logger *logger.Logger) ([]string, *string, error),
	postEffect func(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) error,
	drainer *im.ItemDrainer,
	log *logger.Logger) {
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
		case <-ctx.Done():
			log.Infof("LedgerInit ... ctx.Done()")
			return
		default:
			// get output ids
			outputIds, nextOffset, err := outputIdsFetcher(ctx, client, indexerClient, offset, log)
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
	// post effect
	postEffect(ctx, client, indexerClient)

	// mark finished
	err = deps.IMManager.MarkInitFinished(topic, "")
	if err != nil {
		log.Errorf("failed to MarkInitFinished for %s: %s", topic, err)
		return
	}
}
