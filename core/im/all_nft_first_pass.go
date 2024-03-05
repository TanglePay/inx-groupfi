package im

import (
	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

func ProcessAllNftFirstPass(initCtx *InitContext) {
	var idsFetcher = AllNftOutputIdsFetcher
	var processors = []OutputProcessor{
		// handle group qualification by nft
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			nfts, is, _ := deps.IMManager.FilterNftOutput(outputId, output, milestoneIndex, milestoneTimestamp,
				initCtx.Logger)
			if !is {
				return nil
			}

			deps.IMManager.StoreNewNFTsDeleteConsumedNfts(nfts, nil, initCtx.Logger)
			return nil
		},
		// handle token attached on nft
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			err := handleTokenFromINXOutput(output, outputId, ImOutputTypeCreated, false)
			return err
		},
		// handle did
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			var outputIDIota iotago.OutputID
			copy(outputIDIota[:], outputId)
			did, err := deps.IMManager.FilterOutputForDid(output, outputIDIota)
			if err != nil {
				// log error
				initCtx.Logger.Warnf("LedgerInit ... FilterOutputForDid failed:%s", err)
				return err
			}
			if did != nil {
				// handle did
				createdDid := []*im.Did{did}
				deps.IMManager.HandleDidConsumedAndCreated(nil, createdDid, initCtx.Logger)
			}
			return nil
		},
	}
	HandleGenericInit(initCtx, "allnftfirstpass", idsFetcher, processors)
}
