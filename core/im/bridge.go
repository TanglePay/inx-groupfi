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
		im.CurrentMilestoneIndex = index
		// log
		CoreComponent.LogInfof("LedgerUpdate start:%d, end::%d, milestoneIndex:%d", startIndex, endIndex, index)
		var createdMessage []*im.Message
		var createdNft []*im.NFT
		var createdShared []*im.Message
		var consumedMessage []*im.Message
		var consumedShared []*im.Message
		var consumedNft []*im.NFT
		var createdMark []*iotago.BasicOutput
		var consumedMark []*iotago.BasicOutput
		var createdVote []*iotago.BasicOutput
		var consumedVote []*iotago.BasicOutput
		var createdMute []*iotago.BasicOutput
		var consumedMute []*iotago.BasicOutput
		for _, output := range update.Created {
			// im.CurrentMilestoneTimestamp = max(im.CurrentMilestoneTimestamp, output.MilestoneTimestampBooked)
			if output.MilestoneTimestampBooked > im.CurrentMilestoneTimestamp {
				im.CurrentMilestoneTimestamp = output.MilestoneTimestampBooked
			}
			o := messageFromINXLedgerOutput(output)
			if o != nil {
				createdMessage = append(createdMessage, o)
			}
			nfts := nftFromINXLedgerOutput(output, CoreComponent.Logger())
			if len(nfts) > 0 {
				createdNft = append(createdNft, nfts...)
			}
			shared := sharedOutputFromINXLedgerOutput(output)
			if shared != nil {
				createdShared = append(createdShared, shared)
			}
			handleTokenFromINXLedgerOutput(output, ImOutputTypeCreated)

			mark, is := deps.IMManager.FilterMarkOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				// log found mark
				CoreComponent.LogInfof("LedgerUpdate just found created mark:%s", iotago.EncodeHex(output.OutputId.Id))
				createdMark = append(createdMark, mark)
			}
			mute, is := deps.IMManager.FilterMuteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				createdMute = append(createdMute, mute)
			}
			vote, is := deps.IMManager.FilterVoteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				createdVote = append(createdVote, vote)
			}
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
			nfts := nftFromINXLedgerOutput(output, CoreComponent.Logger())
			if len(nfts) > 0 {
				consumedNft = append(consumedNft, nfts...)
			}
			handleTokenFromINXLedgerOutput(output, ImOutputTypeConsumed)

			mark, is := deps.IMManager.FilterMarkOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				// log found mark
				CoreComponent.LogInfof("LedgerUpdate just found consumed mark:%s", iotago.EncodeHex(output.OutputId.Id))
				consumedMark = append(consumedMark, mark)
			}
			mute, is := deps.IMManager.FilterMuteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				consumedMute = append(consumedMute, mute)
			}
			vote, is := deps.IMManager.FilterVoteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				consumedVote = append(consumedVote, vote)
			}
		}
		dataFromListenning := &im.DataFromListenning{
			CreatedMessage:  createdMessage,
			CreatedNft:      createdNft,
			CreatedShared:   createdShared,
			ConsumedMessage: consumedMessage,
			ConsumedShared:  consumedShared,
			ConsumedNft:     consumedNft,
			CreatedMark:     createdMark,
			ConsumedMark:    consumedMark,
			CreatedVote:     createdVote,
			ConsumedVote:    consumedVote,
			CreatedMute:     createdMute,
			ConsumedMute:    consumedMute,
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
		// check if block or payload is nil
		if block == nil || block.Payload == nil {
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
					pl := append([]byte{im.ImInboxEventTypeNewMessage}, sender...)
					pl = append(pl, meta...)
					deps.IMManager.PushInbox(groupId, pl, CoreComponent.Logger())
				}()
			}
		}
	}
	return nil

}
