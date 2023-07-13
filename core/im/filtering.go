package im

import (
	"bytes"
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

var iotacatTagStr = "IOTACAT"
var iotacatTag = []byte(iotacatTagStr)
var iotacatsharedTagStr = "IOTACATSHARED"
var iotacatsharedTag = []byte(iotacatsharedTagStr)

func nftFromINXLedgerOutput(output *inx.LedgerOutput) *im.NFT {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}
	return nftFromINXOutput(iotaOutput, output.OutputId.Id, output.MilestoneIndexBooked, output.MilestoneTimestampBooked)
}
func nftFromINXOutput(iotaOutput iotago.Output, outputId []byte, milestone uint32, milestoneTimestamp uint32) *im.NFT {

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
		milestone,
		milestoneTimestamp,
	)

	groupId := im.IssuerBech32AddressToGroupId(issuerAddress)
	if groupId == nil {
		return nil
	}
	unlockConditionSet := nftOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)
	nftId := nftOutput.NFTID.ToHex()
	return im.NewNFT(groupId, ownerAddress, nftId, milestone, milestoneTimestamp)
}
func sharedOutputFromINXLedgerOutput(output *inx.LedgerOutput) *im.Message {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}
	return sharedOutputFromINXOutput(iotaOutput, output.OutputId.Id, output.MilestoneIndexBooked, output.MilestoneTimestampBooked)
}
func sharedOutputFromINXOutput(iotaOutput iotago.Output, outputId []byte, milestone uint32, milestoneTimestamp uint32) *im.Message {

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
	CoreComponent.LogInfof("Found IOTACATSHARED output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d",
		len(metaPayload),
		len(groupId),
		iotago.EncodeHex(groupId),
		iotago.EncodeHex(outputId),
		milestone,
		milestoneTimestamp,
	)
	return im.NewMessage(groupId, outputId, milestone, milestoneTimestamp)
}
func messageFromINXLedgerOutput(output *inx.LedgerOutput) *im.Message {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}
	return messageFromINXOutput(iotaOutput, output.OutputId.Id, output.MilestoneIndexBooked, output.MilestoneTimestampBooked)
}
func messageFromINXOutput(iotaOutput iotago.Output, outputId []byte, milestone uint32, milestoneTimestamp uint32) *im.Message {

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
	CoreComponent.LogInfof("Found IOTACAT output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d",
		len(metaPayload),
		len(groupId),
		iotago.EncodeHex(groupId),
		iotago.EncodeHex(outputId),
		milestone,
		milestoneTimestamp,
	)
	return im.NewMessage(groupId, outputId, milestone, milestoneTimestamp)
}

func fetchNextMessage(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, offset *string, log *logger.Logger) ([]*im.Message, *string, error) {
	outputHexIds, offset, err := deps.IMManager.QueryOutputIdsByTag(ctx, indexerClient, iotacatTagStr, offset, log)
	if err != nil {
		return nil, nil, err
	}
	var messages []*im.Message
	for _, outputHexId := range outputHexIds {
		output, milestoneIndex, milestoneTimestamp, err := deps.IMManager.OutputIdToOutputAndMilestoneInfo(ctx, client, outputHexId)
		if err != nil {
			return nil, nil, err
		}
		outputId, err := iotago.DecodeHex(outputHexId)
		if err != nil {
			return nil, nil, err
		}
		message := messageFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp)
		if message != nil {
			messages = append(messages, message)
		}
	}
	return messages, offset, nil
}
