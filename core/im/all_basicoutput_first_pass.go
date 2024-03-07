package im

import (
	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

func ProcessAllBasicOutputFirstPass(initCtx *InitContext) {
	var idsFetcher = AllBasicOutputIdsFetcher
	var processors = []OutputProcessor{
		// handle token attached on basic output
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			err := handleTokenFromINXOutput(output, outputId, ImOutputTypeCreated, false)
			return err
		},
		// handle vote
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter vote output
			basicOutput, is := deps.IMManager.FilterVoteOutput(output, initCtx.Logger)
			if !is {
				return nil
			}
			deps.IMManager.HandleUserVoteGroupBasicOutputCreated(basicOutput, initCtx.Logger)
			return nil
		},
		// handle mute
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter vote output
			basicOutput, is := deps.IMManager.FilterMuteOutput(output, initCtx.Logger)
			if !is {
				return nil
			}
			deps.IMManager.HandleUserMuteGroupMemberBasicOutputCreated(basicOutput, initCtx.Logger)
			return nil
		},
		// handle group shared
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter group shared output
			shared := sharedOutputFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp)
			if shared != nil {
				DataFromListenning := &im.DataFromListenning{
					CreatedShared: []*im.Message{shared},
				}

				err := deps.IMManager.ApplyNewLedgerUpdate(0, DataFromListenning, initCtx.Logger, true)
				if err != nil {
					return err
				}
			}
			return nil
		},
	}
	HandleGenericInit(initCtx, "allbasicoutputfirstpass", idsFetcher, processors)
	// HandleTotalInit after all basic output first pass
	itemDrainer := makeTokenInitDrainer(initCtx.Ctx, initCtx.Client, initCtx.IndexerClient)
	HandleTotalInit(initCtx.Ctx, initCtx.Client, initCtx.IndexerClient, itemDrainer)
}
