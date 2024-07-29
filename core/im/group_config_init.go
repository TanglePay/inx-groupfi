package im

import (
	"github.com/TanglePay/inx-groupfi/pkg/im"
	iotago "github.com/iotaledger/iota.go/v3"
)

func ProcessGroupConfig(initCtx *InitContext) {
	idFetcher := NftOutputIdsByTagFetcher(im.GroupconfigTagStr)
	processors := []OutputProcessor{
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
				groupConfigArr := []*im.ConfigNftOutputWrapper{groupConfig}
				deps.IMManager.HandleGroupConfigNFTOutputConsumedOrCreated(nil, groupConfigArr, initCtx.Logger)
			}
			return nil
		},
	}
	HandleGenericInitParallel(initCtx, "groupconfig", idFetcher, processors)

}

func calculateIsGroupPublicForAllGroupConfig(initCtx *InitContext) {
	// get all group config
	deps.IMManager.CalculateIfGroupIsPublicForAllGroups(initCtx.Logger)

}
