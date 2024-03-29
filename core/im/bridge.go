package im

import (
	"context"
	"io"

	"github.com/TanglePay/inx-groupfi/pkg/im"
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
		im.LastTimeReceiveEventFromHornet = im.GetCurrentEpochTimestamp()
		// log
		CoreComponent.LogInfof("LedgerUpdate start:%d, end::%d, milestoneIndex:%d", startIndex, endIndex, index)
		var createdMessage []*im.Message
		var createdNft []*im.NFT
		var createdShared []*im.GroupShared
		var consumedMessage []*im.Message
		var consumedShared []*im.GroupShared
		var createdPublicKeyOutputIdHexAndAddressPairs []*im.OutputIdHexAndAddressPair
		var consumedNft []*im.NFT
		var createdMark []*im.OutputAndOutputId
		var consumedMark []*im.OutputAndOutputId
		var createdVote []*iotago.BasicOutput
		var consumedVote []*iotago.BasicOutput
		var createdMute []*iotago.BasicOutput
		var consumedMute []*iotago.BasicOutput
		var consumedDid []*im.Did
		var createdDid []*im.Did
		var createdPairX []*im.PairX
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
			outputIdHexAndAddressPair, err := handlePublicKeyOutputFromINXLedgerOutput(output)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate handlePublicKeyOutputFromINXLedgerOutput error:%s", err.Error())
			}
			if outputIdHexAndAddressPair != nil {
				createdPublicKeyOutputIdHexAndAddressPairs = append(createdPublicKeyOutputIdHexAndAddressPairs, outputIdHexAndAddressPair)
			}
			handleTokenFromINXLedgerOutput(output, ImOutputTypeCreated)

			mark, outputId, is := deps.IMManager.FilterMarkOutputFromLedgerOutput(output, CoreComponent.Logger())
			markAndOutputId := &im.OutputAndOutputId{
				Output:   mark,
				OutputId: outputId,
			}
			if is {
				// log found mark
				CoreComponent.LogInfof("LedgerUpdate just found created mark:%s", iotago.EncodeHex(output.OutputId.Id))
				createdMark = append(createdMark, markAndOutputId)
			}
			mute, is := deps.IMManager.FilterMuteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				createdMute = append(createdMute, mute)
			}
			vote, is := deps.IMManager.FilterVoteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				createdVote = append(createdVote, vote)
			}
			did, err := deps.IMManager.FilterLedgerOutputForDid(output)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate FilterLedgerOutputForDid error:%s", err.Error())
			}
			if did != nil {
				createdDid = append(createdDid, did)
			}
			pairX, err := deps.IMManager.FilterPairXFromLedgerOutput(output)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate FilterPairXFromLedgerOutput error:%s", err.Error())
			}
			if pairX != nil {
				createdPairX = append(createdPairX, pairX)
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

			mark, outputId, is := deps.IMManager.FilterMarkOutputFromLedgerOutput(output, CoreComponent.Logger())
			markAndOutputId := &im.OutputAndOutputId{
				Output:   mark,
				OutputId: outputId,
			}
			if is {
				// log found mark
				CoreComponent.LogInfof("LedgerUpdate just found consumed mark:%s", iotago.EncodeHex(output.OutputId.Id))
				consumedMark = append(consumedMark, markAndOutputId)
			}
			mute, is := deps.IMManager.FilterMuteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				consumedMute = append(consumedMute, mute)
			}
			vote, is := deps.IMManager.FilterVoteOutputFromLedgerOutput(output, CoreComponent.Logger())
			if is {
				consumedVote = append(consumedVote, vote)
			}
			did, err := deps.IMManager.FilterLedgerOutputForDid(output)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate FilterLedgerOutputForDid error:%s", err.Error())
			}
			if did != nil {
				consumedDid = append(consumedDid, did)
			}
		}
		dataFromListenning := &im.DataFromListenning{
			CreatedMessage: createdMessage,
			CreatedNft:     createdNft,
			CreatedShared:  createdShared,
			CreatedPublicKeyOutputIdHexAndAddressPairs: createdPublicKeyOutputIdHexAndAddressPairs,
			ConsumedMessage: consumedMessage,
			ConsumedShared:  consumedShared,
			ConsumedNft:     consumedNft,
			CreatedMark:     createdMark,
			ConsumedMark:    consumedMark,
			CreatedVote:     createdVote,
			ConsumedVote:    consumedVote,
			CreatedMute:     createdMute,
			ConsumedMute:    consumedMute,
			CreatedDid:      createdDid,
			ConsumedDid:     consumedDid,
			CreatedPairX:    createdPairX,
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
