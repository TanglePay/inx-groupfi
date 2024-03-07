package im

import (
	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

func handleMessageInit(initCtx *InitContext) {
	var idsFetcher = BasicOutputIdsByTagFetcher(iotacatTagHex)
	var processors = []OutputProcessor{
		// handle message output
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter message output
			message := messageFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp)
			if message != nil {
				DataFromListenning := &im.DataFromListenning{
					CreatedMessage: []*im.Message{message},
				}
				err := deps.IMManager.ApplyNewLedgerUpdate(0, DataFromListenning, initCtx.Logger, true)
				return err
			}
			return nil
		},
	}
	HandleGenericInit(initCtx, "messageinit", idsFetcher, processors)
}
