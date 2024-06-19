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
			dids, err := deps.IMManager.FilterOutputForDid(output, outputIDIota)
			if err != nil {
				// log error
				initCtx.Logger.Warnf("LedgerInit ... FilterOutputForDid failed:%s", err)
				return err
			}
			if dids != nil {
				// handle did
				createdDid := dids
				deps.IMManager.HandleDidConsumedAndCreated(nil, createdDid, initCtx.Logger)
			}
			return nil
		},
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
		// handle group config
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			var outputIDIota iotago.OutputID
			copy(outputIDIota[:], outputId)
			groupConfig, err := im.FilterOutputForConfigNftOutputWrapper(output, outputIDIota, deps.IMManager)
			if err != nil {
				return err
			}
			if groupConfig != nil {
				// handle group config
				deps.IMManager.HandleGroupConfigNFTOutputConsumedOrCreated(nil, groupConfig, initCtx.Logger)
			}
			return nil
		},
	}
	HandleGenericInit(initCtx, "allnftfirstpass", idsFetcher, processors)
}
