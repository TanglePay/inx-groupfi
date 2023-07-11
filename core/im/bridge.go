package im

import (
	"bytes"
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/hive.go/serializer/v2"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

var iotacatTag = []byte("IOTACAT")

var iotacatsharedTag = []byte("IOTACATSHARED")

func nftFromINXOutput(output *inx.LedgerOutput) *im.NFT {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}
	if iotaOutput.Type() != iotago.OutputNFT {
		return nil
	}
	nftOutput, is := iotaOutput.(*iotago.NFTOutput)
	if !is {
		return nil
	}
	featureSet, err := nftOutput.ImmutableFeatures.Set()
	if err != nil {
		return nil
	}
	issuer := featureSet.IssuerFeature()
	if issuer == nil {
		return nil
	}
	issuerAddress := issuer.Address.Bech32(iotago.PrefixShimmer)
	// log
	CoreComponent.LogInfof("Found NFT output,issuer:%s，milestoneIndex:%d,milestoneTimestamp:%d",
		issuerAddress,
		output.MilestoneIndexBooked,
		output.MilestoneTimestampBooked,
	)

	groupId := im.IssuerBech32AddressToGroupId(issuerAddress)
	if groupId == nil {
		return nil
	}
	unlockConditionSet := nftOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)
	nftId := nftOutput.NFTID.ToHex()
	return im.NewNFT(groupId, ownerAddress, nftId, output.MilestoneIndexBooked, output.MilestoneTimestampBooked)
}

func sharedOutputFromINXOutput(output *inx.LedgerOutput) *im.Message {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}

	// Ignore anything other than BasicOutputs
	if iotaOutput.Type() != iotago.OutputBasic {
		return nil
	}
	// tag should have iotacat and groupId can be retrieved as well
	featureSet := iotaOutput.FeatureSet()
	tag := featureSet.TagFeature()
	meta := featureSet.MetadataFeature()
	if tag == nil || meta == nil || meta.Size() < im.GroupIdLen {
		return nil
	}
	tagPayload := tag.Tag

	metaPayload := meta.Data
	if !bytes.Equal(tagPayload, iotacatsharedTag) {
		return nil
	}
	// groupid is first xxx bytes of meta feature
	groupId := metaPayload[:im.GroupIdLen]
	outputId := output.OutputId.Id
	CoreComponent.LogInfof("Found IOTACATSHARED output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d",
		len(metaPayload),
		len(groupId),
		iotago.EncodeHex(groupId),
		iotago.EncodeHex(outputId),
		output.MilestoneIndexBooked,
		output.MilestoneTimestampBooked,
	)
	return im.NewMessage(groupId, outputId, output.MilestoneIndexBooked, output.MilestoneTimestampBooked)
}

func messageFromINXOutput(output *inx.LedgerOutput) *im.Message {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}

	// Ignore anything other than BasicOutputs
	if iotaOutput.Type() != iotago.OutputBasic {
		return nil
	}
	// tag should have iotacat and groupId can be retrieved as well
	featureSet := iotaOutput.FeatureSet()
	tag := featureSet.TagFeature()
	meta := featureSet.MetadataFeature()
	if tag == nil || meta == nil || meta.Size() < im.GroupIdLen {
		return nil
	}
	tagPayload := tag.Tag

	metaPayload := meta.Data
	if !bytes.Equal(tagPayload, iotacatTag) {
		return nil
	}
	// groupid is first xxx bytes of meta feature
	groupId := metaPayload[:im.GroupIdLen]
	outputId := output.OutputId.Id
	CoreComponent.LogInfof("Found IOTACAT output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d",
		len(metaPayload),
		len(groupId),
		iotago.EncodeHex(groupId),
		iotago.EncodeHex(outputId),
		output.MilestoneIndexBooked,
		output.MilestoneTimestampBooked,
	)
	return im.NewMessage(groupId, outputId, output.MilestoneIndexBooked, output.MilestoneTimestampBooked)
}

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
			o := messageFromINXOutput(output)
			if o != nil {
				createdMessage = append(createdMessage, o)
			}
			nft := nftFromINXOutput(output)
			if nft != nil {
				createdNft = append(createdNft, nft)
			}
			shared := sharedOutputFromINXOutput(output)
			if shared != nil {
				createdShared = append(createdShared, shared)
			}
		}

		return handler(index, createdMessage, createdNft, createdShared)
	})
}
