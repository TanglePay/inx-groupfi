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
		// handle like
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter vote output
			basicOutput, is := deps.IMManager.FilterLikeOutput(output, initCtx.Logger)
			if !is {
				return nil
			}
			deps.IMManager.HandleUserLikeGroupMemberBasicOutputCreated(basicOutput, initCtx.Logger)
			return nil
		},
		// handle group shared
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter group shared output
			shared := sharedOutputFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp)
			if shared != nil {
				DataFromListenning := &im.DataFromListenning{
					CreatedShared: []*im.GroupShared{shared},
				}

				err := deps.IMManager.ApplyNewLedgerUpdate(0, DataFromListenning, initCtx.Logger, true)
				if err != nil {
					return err
				}
			}
			return nil
		},
		// handle evm qualify
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter evm qualify output
			outputIdFixed := [im.OutputIdLen]byte{}
			copy(outputIdFixed[:], outputId)
			evmQualify, err := deps.IMManager.FilterEvmQualifyFromOutput(outputIdFixed, output, initCtx.Logger)
			if err != nil {
				return err
			}
			if evmQualify == nil {
				return nil
			}
			deps.IMManager.HandleEvmQualifyCreated(evmQualify, initCtx.Logger)
			return nil
		},
		// handle profile
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter profile from output
			outputIdFixed := [im.OutputIdLen]byte{}
			copy(outputIdFixed[:], outputId)

			// Call the filter function to extract the profile from the output
			profile, err := deps.IMManager.FilterProfileOutput(output, outputIdFixed, initCtx.Logger)
			if err != nil {
				return err
			}

			// If no profile is found, return without error
			if profile == nil {
				return nil
			}

			// Handle the newly created profile
			err = deps.IMManager.StoreProfile(profile)
			if err != nil {
				initCtx.Logger.Errorf("Error storing profile: %s", err.Error())
				return err
			}

			return nil
		},
		// handle group state sync
		func(outputId []byte, output iotago.Output, milestoneIndex uint32, milestoneTimestamp uint32, initCtx *InitContext) error {
			// filter group state sync output
			var outputIdFixed [im.OutputIdLen]byte
			copy(outputIdFixed[:], outputId)
			groupStateSync, address, is := im.FilterGroupStateSyncOutput(output, outputIdFixed, deps.IMManager)
			if !is {
				return nil
			}
			if groupStateSync == nil {
				return nil
			}
			err := im.StoreGroupStateSync(groupStateSync, address, deps.IMManager)
			return err
		},
		// handle output cache
		func(outputId []byte, output iotago.Output, milestoneIndex, milestoneTimestamp uint32, initCtx *InitContext) error {
			var outputIdFixed [im.OutputIdLen]byte
			copy(outputIdFixed[:], outputId)
			output, is := im.FilterGroupFIOutput(output, outputIdFixed, deps.IMManager)
			if is {
				return im.StoreGroupFIOutput(output, outputIdFixed, deps.IMManager)
			}
			return nil
		},
	}
	HandleGenericInit(initCtx, "allbasicoutputfirstpass", idsFetcher, processors)
	// HandleTotalInit after all basic output first pass
	itemDrainer := makeTokenInitDrainer(initCtx.Ctx, initCtx.Client, initCtx.IndexerClient)
	HandleTotalInit(initCtx.Ctx, initCtx.Client, initCtx.IndexerClient, itemDrainer)
}
