package im

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

var iotacatTagStr = "GROUPFIV3"
var iotacatTag = []byte(iotacatTagStr)
var iotacatTagHex = iotago.EncodeHex(iotacatTag)

var iotacatsharedTagStr = "GROUPFISHAREDV2"
var iotacatsharedTag = []byte(iotacatsharedTagStr)
var iotacatsharedTagHex = iotago.EncodeHex(iotacatsharedTag)

func nftFromINXLedgerOutput(output *inx.LedgerOutput, log *logger.Logger) *im.NFT {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}
	return nftFromINXOutput(iotaOutput, output.OutputId.Id, output.MilestoneIndexBooked, output.MilestoneTimestampBooked, log)
}
func nftFromINXOutput(iotaOutput iotago.Output, outputId []byte, milestone uint32, milestoneTimestamp uint32, log *logger.Logger) *im.NFT {

	if iotaOutput.Type() != iotago.OutputNFT {
		return nil
	}

	nftOutput, is := iotaOutput.(*iotago.NFTOutput)
	if !is {
		return nil
	}
	featureSet, err := nftOutput.ImmutableFeatures.Set()
	if err != nil {
		// log error
		log.Errorf("nftFromINXOutput failed:%s", err)
		return nil
	}
	issuer := featureSet.IssuerFeature()
	if issuer == nil {
		return nil
	}
	meta := featureSet.MetadataFeature()
	if meta == nil {
		return nil
	}
	//nftId := nftOutput.NFTID
	nftId := im.Sha256HashBytes(meta.Data)

	metaMap := make(map[string]interface{})
	err = json.Unmarshal(meta.Data, &metaMap)
	if err != nil {
		log.Errorf("nftFromINXOutput failed:%s", err)
		return nil
	}
	ipfsLink := metaMap["uri"].(string)

	issuerAddress := issuer.Address.Bech32(iotago.PrefixShimmer)
	// log
	CoreComponent.LogInfof("Found NFT output,issuer:%s，milestoneIndex:%d,milestoneTimestamp:%d",
		issuerAddress,
		milestone,
		milestoneTimestamp,
	)

	groupId, groupName := im.IssuerBech32AddressToGroupIdAndGroupName(issuerAddress)
	if groupId == nil || groupName == "" {
		return nil
	}
	unlockConditionSet := nftOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)
	// log groupId, ownerAddress, nftId, milestone, milestoneTimestamp, outputid
	CoreComponent.LogInfof("Found NFT output,groupId:%s,ownerAddress:%s,nftId:%s,milestoneIndex:%d,milestoneTimestamp:%d, outputId:%s",
		iotago.EncodeHex(groupId),
		ownerAddress,
		iotago.EncodeHex(nftId),
		milestone,
		milestoneTimestamp,
		iotago.EncodeHex(outputId),
	)

	return im.NewNFT(groupId, ownerAddress, nftId, groupName, ipfsLink, milestone, milestoneTimestamp)
}

// filter nft INXLedgerOutput based on a set of issuer address
func filterNFTINXLedgerOutputViaIssuerAddress(output *inx.LedgerOutput, issuerBech32AddressMap map[string]bool) (isNFT bool, issuerAddress string, nft *iotago.NFTOutput) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return false, "", nil
	}
	return filterNFTOutputViaIssuerAddress(iotaOutput, issuerBech32AddressMap)
}

// filter nft output based on a set of issuer address
func filterNFTOutputViaIssuerAddress(output iotago.Output, issuerBech32AddressMap map[string]bool) (isNFT bool, issuerAddress string, nft *iotago.NFTOutput) {
	// call filterNFTOutput
	isNFT, nftOutput := filterNFTOutput(output)
	if !isNFT {
		return false, "", nil
	}
	featureSet, err := nftOutput.ImmutableFeatures.Set()
	if err != nil {
		return false, "", nil
	}
	issuer := featureSet.IssuerFeature()
	if issuer == nil {
		return false, "", nil
	}
	issuerAddress = issuer.Address.Bech32(iotago.PrefixShimmer)
	_, ok := issuerBech32AddressMap[issuerAddress]
	if !ok {
		return false, "", nil
	}
	return true, issuerAddress, nftOutput
}

// filterNFTOutput
func filterNFTOutput(output iotago.Output) (isNFT bool, nft *iotago.NFTOutput) {
	if output.Type() != iotago.OutputNFT {
		return false, nil
	}
	nftOutput, is := output.(*iotago.NFTOutput)
	if !is {
		return false, nil
	}
	return true, nftOutput
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
	// groupid is GroupIdLen bytes of second byte of meta feature
	groupId := metaPayload[1 : im.GroupIdLen+1]
	metapayloadSha256 := im.Sha256HashBytes(metaPayload)
	unlockConditionSet := iotaOutput.UnlockConditionSet()
	senderAddressStr := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)
	senderAddressSha256 := im.Sha256Hash(senderAddressStr)
	CoreComponent.LogInfof("Found IOTACATSHARED output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d，senderAddress:%s",
		len(metaPayload),
		len(groupId),
		iotago.EncodeHex(groupId),
		iotago.EncodeHex(outputId),
		milestone,
		milestoneTimestamp,
		senderAddressStr,
	)
	return im.NewMessage(groupId, outputId, milestone, milestoneTimestamp, senderAddressSha256, metapayloadSha256)
}

func handleTokenFromINXLedgerOutput(output *inx.LedgerOutput, outputStatus int) error {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return err
	}
	return handleTokenFromINXOutput(iotaOutput, output.OutputId.Id, outputStatus, true)
}
func handleTokenFromINXOutput(iotaOutput iotago.Output, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	if iotaOutput.Type() == iotago.OutputBasic {
		basicOutput := iotaOutput.(*iotago.BasicOutput)
		err := handleTokenFromBasicOutput(basicOutput, outputId, outputStatus, isUpdateGlobalAmount)
		if err != nil {
			return err
		}
	} else if iotaOutput.Type() == iotago.OutputNFT {
		nftOutput := iotaOutput.(*iotago.NFTOutput)
		err := handleTokenFromNFTOutput(nftOutput, outputId, outputStatus, isUpdateGlobalAmount)
		if err != nil {
			return err
		}
	}

	return nil
}
func outputStatusToTokenStatus(outputStatus int) byte {
	if outputStatus == ImOutputTypeCreated {
		return im.ImTokenStatusCreated
	} else if outputStatus == ImOutputTypeConsumed {
		return im.ImTokenStatusConsumed
	}
	return 0
}

func handleTokenFromBasicOutput(iotaOutput *iotago.BasicOutput, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	// handle smr e.g basic coin
	smrAmount := iotaOutput.Amount
	err := handleSmrAmount(smrAmount, iotaOutput, outputId, outputStatus, isUpdateGlobalAmount)
	if err != nil {
		return err
	}
	return nil
}
func handleSmrAmount(smrAmount uint64, iotaOutput iotago.Output, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	smrAmountBig := new(big.Int).SetUint64(smrAmount)
	tokenStatus := outputStatusToTokenStatus(outputStatus)
	unlockConditionSet := iotaOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)

	if isUpdateGlobalAmount {
		smrTotal := GetSmrTokenTotal()
		if tokenStatus == im.ImTokenStatusCreated {
			smrTotal.Add(smrAmountBig)
		} else if tokenStatus == im.ImTokenStatusConsumed {
			smrTotal.Sub(smrAmountBig)
		}
		// handle whale eligibility
		defer handleTokenWhaleEligibilityFromAddressGivenTotalAmount(im.ImTokenTypeSMR, ownerAddress, smrTotal.Get(), deps.IMManager, CoreComponent.Logger())
	}

	smrAmountText := smrAmountBig.Text(10)
	tokenId := im.Sha256HashBytes(outputId)
	tokenStat := deps.IMManager.NewTokenStat(im.ImTokenTypeSMR, tokenId, ownerAddress, tokenStatus, smrAmountText)
	return deps.IMManager.StoreOneToken(tokenStat)
}
func getThresholdFromTokenType(tokenType uint16) *big.Float {
	if tokenType == im.ImTokenTypeSMR {
		return big.NewFloat(im.ImSMRWhaleThreshold)
	}
	return nil
}
func handleTokenWhaleEligibilityFromAddressGivenTotalAmount(tokenType uint16, address string, totalAmount *big.Int, manager *im.Manager, logger *logger.Logger) error {
	balance, err := manager.GetBalanceOfOneAddress(tokenType, address)
	if err != nil {
		return err
	}
	// total = total + 1 to prevent divide zero
	totalAmount = new(big.Int).Add(totalAmount, big.NewInt(1))
	percentage := new(big.Float).Quo(new(big.Float).SetInt(balance), new(big.Float).SetInt(totalAmount))
	threshold := getThresholdFromTokenType(tokenType)
	if threshold == nil {
		return nil
	}
	isEligible := percentage.Cmp(threshold) >= 0
	return manager.SetWhaleEligibility(tokenType, address, isEligible, logger)
}

func handleTokenFromNFTOutput(iotaOutput *iotago.NFTOutput, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	smrAmount := iotaOutput.Amount
	err := handleSmrAmount(smrAmount, iotaOutput, outputId, outputStatus, isUpdateGlobalAmount)
	if err != nil {
		return err
	}
	return nil
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
	if tag == nil || meta == nil {
		return nil
	}
	tagPayload := tag.Tag

	metaPayload := meta.Data

	if !bytes.Equal(tagPayload, iotacatTag) {
		return nil
	}
	// groupid is GroupIdLen bytes from second byte of meta feature
	groupId := metaPayload[1 : im.GroupIdLen+1]
	metapayloadSha256 := im.Sha256HashBytes(metaPayload)
	unlockConditionSet := iotaOutput.UnlockConditionSet()
	senderAddressStr := unlockConditionSet.Address().Address.Bech32(iotago.PrefixShimmer)
	senderAddressSha256 := im.Sha256Hash(senderAddressStr)
	CoreComponent.LogInfof("Found IOTACAT output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d，senderAddress:%s",
		len(metaPayload),
		len(groupId),
		iotago.EncodeHex(groupId),
		iotago.EncodeHex(outputId),
		milestone,
		milestoneTimestamp,
		senderAddressStr,
	)
	return im.NewMessage(groupId, outputId, milestone, milestoneTimestamp, senderAddressSha256, metapayloadSha256)
}

// filter output for push
func filterOutputForPush(output iotago.Output) (isMessage bool, senderAddressHash []byte, groupId []byte, metafeaturePayload []byte) {
	if output.Type() != iotago.OutputBasic {
		return false, nil, nil, nil
	}
	featureSet := output.FeatureSet()
	tag := featureSet.TagFeature()
	meta := featureSet.MetadataFeature()
	if tag == nil || meta == nil || meta.Size() < im.GroupIdLen {
		return false, nil, nil, nil
	}
	tagPayload := tag.Tag

	if !bytes.Equal(tagPayload, iotacatTag) {
		return false, nil, nil, nil
	}
	metaPayload := meta.Data
	// group id is GroupIdLen bytes of second byte of meta feature
	groupId_ := metaPayload[1 : im.GroupIdLen+1]
	unlockSet := output.UnlockConditionSet()
	senderAddress, _ := unlockSet.Address().Address.Serialize(serializer.DeSeriModeNoValidation, nil)
	return true, senderAddress, groupId_, metaPayload
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
	outputHexIds, offset, err := deps.IMManager.QueryNFTIdsByIssuer(ctx, indexerClient, issuerBech32Address, offset, 100, log)
	// log len(outputHexIds)
	log.Infof("fetchNextNFTs: Found %d NFT outputHexIds", len(outputHexIds))
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
		nft := nftFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp, log)
		if nft != nil {
			nfts = append(nfts, nft)
		}
	}
	return nfts, offset, nil
}

// struct for outputId and output pair
type OutputIdOutputPair struct {
	OutputId string
	Output   iotago.Output
}
