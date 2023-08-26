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

func LedgerUpdates(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex, handler func(index iotago.MilestoneIndex, dataFromListenning *im.DataFromListenning) error) error {
	return deps.NodeBridge.ListenToLedgerUpdates(ctx, startIndex, endIndex, func(update *nodebridge.LedgerUpdate) error {
		index := update.MilestoneIndex
		// log
		CoreComponent.LogInfof("LedgerUpdate start:%d, end::%d, milestoneIndex:%d", startIndex, endIndex, index)
		var createdMessage []*im.Message
		var createdNft []*im.NFT
		var createdShared []*im.Message
		var consumedMessage []*im.Message
		var consumedShared []*im.Message
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
			o := messageFromINXLedgerOutput(output)
			if o != nil {
				// found consumed message
				CoreComponent.LogInfof("LedgerUpdate just found consumed message:%s", o.GetOutputIdStr())
				consumedMessage = append(consumedMessage, o)
			}
			shared := sharedOutputFromINXLedgerOutput(output)
			if shared != nil {
				consumedShared = append(consumedShared, shared)
			}
			handleTokenFromINXLedgerOutput(output, ImOutputTypeConsumed)
		}
		dataFromListenning := &im.DataFromListenning{
			CreatedMessage:  createdMessage,
			CreatedNft:      createdNft,
			CreatedShared:   createdShared,
			ConsumedMessage: consumedMessage,
			ConsumedShared:  consumedShared,
		}
		return handler(index, dataFromListenning)
	})
}

func LedgerUpdateBlock(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex) error {

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
			isMessage, sender, groupId, meta := filterOutputForPush(output)
			if isMessage {
				// log sender length
				CoreComponent.LogInfof("LedgerUpdateBlock before push sender len:%d", len(sender))
				go func() {
					// prefix ImInboxMessageTypeNewMessageP2PV1 + sender + meta
					pl := append([]byte{im.ImInboxMessageTypeNewMessageP2PV1}, sender...)
					pl = append(pl, meta...)
					deps.IMManager.PushInbox(groupId, pl, CoreComponent.Logger())
				}()
			}
		}
	}
	return nil

}
