package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	iotago "github.com/iotaledger/iota.go/v3"
)

func NodeStatus(ctx context.Context) (confirmedIndex iotago.MilestoneIndex, pruningIndex iotago.MilestoneIndex) {
	status := deps.NodeBridge.NodeStatus()

	return status.GetConfirmedMilestone().GetMilestoneInfo().GetMilestoneIndex(), status.GetTanglePruningIndex()
}

func LedgerUpdates(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex, handler func(index iotago.MilestoneIndex, createdMessage []*im.Message, createdNft []*im.NFT, createdShared []*im.Message) error) error {
	return deps.NodeBridge.ListenToLedgerUpdates(ctx, startIndex, endIndex, func(update *nodebridge.LedgerUpdate) error {
		index := update.MilestoneIndex
		// log
		CoreComponent.LogInfof("LedgerUpdate start:%d, end::%d, milestoneIndex:%d", startIndex, endIndex, index)
		var createdMessage []*im.Message
		var createdNft []*im.NFT
		var createdShared []*im.Message
		for _, output := range update.Created {
			o := messageFromINXLedgerOutput(output)
			if o != nil {
				createdMessage = append(createdMessage, o)
			}
			nft := nftFromINXLedgerOutput(output)
			if nft != nil {
				createdNft = append(createdNft, nft)
			}
			shared := sharedOutputFromINXLedgerOutput(output)
			if shared != nil {
				createdShared = append(createdShared, shared)
			}
		}

		return handler(index, createdMessage, createdNft, createdShared)
	})
}
