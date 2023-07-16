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
var iotacatTagHex = iotago.EncodeHex(iotacatTag)
var iotacatsharedTagStr = "IOTACATSHARED"
var iotacatsharedTag = []byte(iotacatsharedTagStr)
var iotacatsharedTagHex = iotago.EncodeHex(iotacatsharedTag)

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
	nftId := nftOutput.NFTID
	// log groupId, ownerAddress, nftId, milestone, milestoneTimestamp)
	CoreComponent.LogInfof("Found NFT output,groupId:%s,ownerAddress:%s,nftId:%s,milestoneIndex:%d,milestoneTimestamp:%d",
		iotago.EncodeHex(groupId), ownerAddress, nftId, milestone, milestoneTimestamp)
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
	outputHexIds, offset, err := deps.IMManager.QueryOutputIdsByTag(ctx, indexerClient, iotacatTagHex, offset, log)
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

func fetchNextShared(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, offset *string, log *logger.Logger) ([]*im.Message, *string, error) {
	outputHexIds, offset, err := deps.IMManager.QueryOutputIdsByTag(ctx, indexerClient, iotacatsharedTagHex, offset, log)
	if err != nil {
		return nil, nil, err
	}
	var shareds []*im.Message
	for _, outputHexId := range outputHexIds {
		output, milestoneIndex, milestoneTimestamp, err := deps.IMManager.OutputIdToOutputAndMilestoneInfo(ctx, client, outputHexId)
		if err != nil {
			return nil, nil, err
		}
		outputId, err := iotago.DecodeHex(outputHexId)
		if err != nil {
			return nil, nil, err
		}
		shared := sharedOutputFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp)
		if shared != nil {
			shareds = append(shareds, shared)
		}
	}
	return shareds, offset, nil
}

// fetchNextNFTs fetches next NFTs from indexer
func fetchNextNFTs(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, offset *string, issuerBech32Address string, log *logger.Logger) ([]*im.NFT, *string, error) {
	outputHexIds, offset, err := deps.IMManager.QueryNFTIdsByIssuer(ctx, indexerClient, issuerBech32Address, offset, log)
	if err != nil {
		return nil, nil, err
	}
	var nfts []*im.NFT
	for _, outputHexId := range outputHexIds {
		output, milestoneIndex, milestoneTimestamp, err := deps.IMManager.OutputIdToOutputAndMilestoneInfo(ctx, client, outputHexId)
		if err != nil {
			return nil, nil, err
		}
		outputId, err := iotago.DecodeHex(outputHexId)
		if err != nil {
			return nil, nil, err
		}
		nft := nftFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp)
		if nft != nil {
			nfts = append(nfts, nft)
		}
	}
	return nfts, offset, nil
}
