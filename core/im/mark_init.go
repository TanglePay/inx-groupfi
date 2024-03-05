package im

import (
	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

// handle mark init
func handleMarkInit(initCtx *InitContext) {
	// handleTypeInit(ctx, client, indexerClient, drainer, im.MarkType, im.MarkTagStr)
	var idsFetcher = BasicOutputIdsByTagFetcher(im.MarkTagStr)
	var processors = []OutputProcessor{
		// handle mark output
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter mark output
			basicOutput, is := deps.IMManager.FilterMarkOutput(output, initCtx.Logger)
			if !is {
				// log error
				initCtx.Logger.Warnf("LedgerInit ... FilterMarkOutput failed")
				return nil
			}
			outputIdIota := iotago.OutputID{}
			copy(outputIdIota[:], outputId)
			outputAndOutputId := &im.OutputAndOutputId{
				Output:   basicOutput,
				OutputId: outputIdIota,
			}
			deps.IMManager.HandleGroupMarkBasicOutputConsumedAndCreated(nil, outputAndOutputId, initCtx.Logger)
			return nil
		},
	}
	HandleGenericInit(initCtx, "markinit", idsFetcher, processors)
}
