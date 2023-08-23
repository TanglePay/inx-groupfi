package im

import (
	"context"
	"io"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/hive.go/serializer/v2"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
			handleTokenFromINXLedgerOutput(output, ImOutputTypeCreated)
		}
		for _, spent := range update.Consumed {
			output := spent.GetOutput()
			handleTokenFromINXLedgerOutput(output, ImOutputTypeConsumed)
		}
		return handler(index, createdMessage, createdNft, createdShared)
	})
}

func LedgerUpdateBlock(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex, handler func(index iotago.MilestoneIndex, createdMessage []*im.Message, createdNft []*im.NFT, createdShared []*im.Message) error) error {

	stream, err := deps.NodeBridge.Client().ListenToBlocks(ctx, &inx.NoParams{})
	if err != nil {
		return err
	}
	for {
		payload, err := stream.Recv()
		if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
			break
		}
		if ctx.Err() != nil {
			// context got canceled, so stop the updates
			//nolint:nilerr // false positive
			return nil
		}
		if err != nil {
			return err
		}

		block, err := payload.GetBlock().UnwrapBlock(serializer.DeSeriModeNoValidation, nil)
		if err != nil {
			continue
		}
		if block.Payload.PayloadType() != iotago.PayloadTransaction {
			continue
		}
		transaction := block.Payload.(*iotago.Transaction)
		for _, output := range transaction.Essence.Outputs {
			isMessage, groupId, meta := filterOutputForPush(output)
			if isMessage {
				go func() {
					// prefix ImInboxMessageTypeNewMessageP2PV1 to meta
					meta = append([]byte{im.ImInboxMessageTypeNewMessageP2PV1}, meta...)
					deps.IMManager.PushInbox(groupId, meta, CoreComponent.Logger())
				}()
			}
		}
	}
	return nil

}
