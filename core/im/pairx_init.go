package im

import (
	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

func ProcessPairx(initCtx *InitContext) {
	idFetcher := NftOutputIdsByTagFetcher(im.PairXTagStr)
	processors := []OutputProcessor{
		// handle pairx
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			var outputIDIota iotago.OutputID
			copy(outputIDIota[:], outputId)
			pairX, err := deps.IMManager.FilterPairXFromOutput(output, outputIDIota, initCtx.Logger)
			if err != nil {
				return err
			}
			if pairX != nil {
				// handle pairx
				deps.IMManager.HandlePairXCreated(pairX, initCtx.Logger)
			}
			return nil
		},
	}
	HandleGenericInit(initCtx, "pairx", idFetcher, processors)

}
