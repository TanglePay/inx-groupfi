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
		CoreComponent.LogInfof("LedgerUpdate start:%d, end::%d, milestoneIndex:%d, milestoneTimestamp:%d", startIndex, endIndex, index, im.CurrentMilestoneTimestamp)
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
		var createdEvmQualify []*im.EvmQualify
		var createdGroupConfig []*im.ConfigNftOutputWrapper
		var consumedGroupConfig []*im.ConfigNftOutputWrapper
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

			groupConfig, err := im.FilterLedgerOutputForConfigNftOutputWrapper(output, deps.IMManager)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate FilterOutputForConfigNftOutputWrapper error:%s", err.Error())
			}
			if groupConfig != nil {
				createdGroupConfig = append(createdGroupConfig, groupConfig)
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

			dids, err := deps.IMManager.FilterLedgerOutputForDid(output)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate FilterLedgerOutputForDid error:%s", err.Error())
			}
			if dids != nil {
				consumedDid = append(consumedDid, dids...)
			}

			groupConfig, err := im.FilterLedgerOutputForConfigNftOutputWrapper(output, deps.IMManager)
			if err != nil {
				// log error
				CoreComponent.LogErrorf("LedgerUpdate FilterOutputForConfigNftOutputWrapper error:%s", err.Error())
			}
			if groupConfig != nil {
				consumedGroupConfig = append(consumedGroupConfig, groupConfig)
			}
		}
		if len(createdGroupConfig) > 0 || len(consumedGroupConfig) > 0 {
			deps.IMManager.HandleGroupConfigNFTOutputConsumedOrCreated(consumedGroupConfig, createdGroupConfig, CoreComponent.Logger())
		}
		dataFromListenning := &im.DataFromListenning{
			CreatedMessage: createdMessage,
			CreatedNft:     createdNft,
			CreatedShared:  createdShared,
			CreatedPublicKeyOutputIdHexAndAddressPairs: createdPublicKeyOutputIdHexAndAddressPairs,
			ConsumedMessage:   consumedMessage,
			ConsumedShared:    consumedShared,
			ConsumedNft:       consumedNft,
			CreatedMark:       createdMark,
			ConsumedMark:      consumedMark,
			CreatedVote:       createdVote,
			ConsumedVote:      consumedVote,
			CreatedMute:       createdMute,
			ConsumedMute:      consumedMute,
			CreatedDid:        createdDid,
			ConsumedDid:       consumedDid,
			CreatedPairX:      createdPairX,
			CreatedEvmQualify: createdEvmQualify,
		}
		return handler(index, dataFromListenning)
	})
}

func LedgerUpdateBlock(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex) error {

	stream, err := deps.NodeBridge.Client().ListenToBlocks(ctx, &inx.NoParams{})
	if err != nil {
		// log error
		CoreComponent.LogErrorf("LedgerUpdateBlock ListenToBlocks error:%s", err.Error())
		return err
	}
	for {
		payload, err := stream.Recv()
		if errors.Is(err, io.EOF) || status.Code(err) == codes.Canceled {
			// log error
			CoreComponent.LogErrorf("LedgerUpdateBlock error:%s", err.Error())
			continue
		}
		if ctx.Err() != nil {
			// context got canceled, so stop the updates
			//nolint:nilerr // false positive
			return nil
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
		outputSets, err := transaction.OutputsSet()
		if err != nil {
			// log error
			CoreComponent.LogErrorf("LedgerUpdateBlock OutputsSet error:%s", err.Error())
		} else {
			for outputId, output := range outputSets {
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
					continue
				}
				evmQualify, err := deps.IMManager.FilterEvmQualifyFromOutput(output, CoreComponent.Logger())
				if err != nil {
					// log error
					CoreComponent.LogErrorf("LedgerUpdate FilterEvmQualifyFromLedgerOutput error:%s", err.Error())
				}
				if evmQualify != nil {
					deps.IMManager.HandleEvmQualifyCreated(evmQualify, CoreComponent.Logger())
					continue
				}

				dids, err := deps.IMManager.FilterOutputForDid(output, outputId)
				if err != nil {
					// log error
					CoreComponent.LogErrorf("LedgerUpdate FilterLedgerOutputForDid error:%s", err.Error())
				}
				if dids != nil {
					deps.IMManager.HandleDidConsumedAndCreated(nil, dids, CoreComponent.Logger())
					continue
				}

				mark, is := deps.IMManager.FilterMarkOutput(output, CoreComponent.Logger())

				if is {
					markAndOutputId := &im.OutputAndOutputId{
						Output:   mark,
						OutputId: outputId,
					}
					deps.IMManager.HandleGroupMarkBasicOutputConsumedAndCreated(markAndOutputId, CoreComponent.Logger())
					continue
				}

				mute, is := deps.IMManager.FilterMuteOutput(output, CoreComponent.Logger())
				if is {
					deps.IMManager.HandleUserMuteGroupMemberBasicOutputCreated(mute, CoreComponent.Logger())
					continue
				}

				like, is := deps.IMManager.FilterLikeOutput(output, CoreComponent.Logger())
				if is {
					deps.IMManager.HandleUserLikeGroupMemberBasicOutputCreated(like, CoreComponent.Logger())
				}

				vote, is := deps.IMManager.FilterVoteOutput(output, CoreComponent.Logger())
				if is {
					deps.IMManager.HandleUserVoteGroupBasicOutputCreated(vote, CoreComponent.Logger())
					continue
				}
				pairX, err := deps.IMManager.FilterPairXFromOutput(output, outputId, CoreComponent.Logger())
				if err != nil {
					// log error
					CoreComponent.LogErrorf("LedgerUpdate FilterPairXFromLedgerOutput error:%s", err.Error())
				}
				if pairX != nil {
					deps.IMManager.HandlePairXCreated(pairX, CoreComponent.Logger())
					continue
				}

			}
		}
	}

}
