package im

import (
	"bytes"
	"context"
	"encoding/json"
	"math/big"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"golang.org/x/crypto/blake2b"
)

var iotacatTagStr = "GROUPFIV4"
var iotacatTag = []byte(iotacatTagStr)
var iotacatTagHex = iotago.EncodeHex(iotacatTag)

var iotacatsharedTagStr = "GROUPFISHAREDV2"
var iotacatsharedTag = []byte(iotacatsharedTagStr)
var iotacatsharedTagHex = iotago.EncodeHex(iotacatsharedTag)

func nftFromINXLedgerOutput(output *inx.LedgerOutput, log *logger.Logger) []*im.NFT {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil
	}
	return nftFromINXOutput(iotaOutput, output.OutputId.Id, output.MilestoneIndexBooked, output.MilestoneTimestampBooked, log)
}
func nftFromINXOutput(iotaOutput iotago.Output, outputId []byte, milestone uint32, milestoneTimestamp uint32, log *logger.Logger) []*im.NFT {

	// log enter nftFromINXOutput, outputId:%s
	CoreComponent.LogInfof("nftFromINXOutput, outputId:%s", iotago.EncodeHex(outputId))
	if iotaOutput.Type() != iotago.OutputNFT {
		return nil
	}

	nftOutput, is := iotaOutput.(*iotago.NFTOutput)
	if !is {
		return nil
	}
	// log casted to NFTOutput
	CoreComponent.LogInfof("nftFromINXOutput, casted to NFTOutput")
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

	var nftIdHex string
	if nftOutput.NFTID.Empty() {
		outputIdHash := blake2b.Sum256(outputId)
		nftIdHex = iotago.EncodeHex(outputIdHash[:])
	} else {
		nftIdHex = nftOutput.NFTID.ToHex()
	}
	nftId, err := iotago.DecodeHex(nftIdHex)
	if err != nil {
		log.Errorf("nftFromINXOutput failed:%s", err)
		return nil
	}

	// log nftIdHex
	CoreComponent.LogInfof("nftFromINXOutput, nftIdHex:%s", nftIdHex)
	metaMap := make(map[string]interface{})
	err = json.Unmarshal(meta.Data, &metaMap)
	if err != nil {
		log.Errorf("nftFromINXOutput failed:%s", err)
		return nil
	}
	//ipfsLink := metaMap["uri"].(string)
	if issuer.Address.Type() != iotago.AddressNFT {
		return nil
	}
	nftAddress := issuer.Address.(*iotago.NFTAddress)
	collectionId := nftAddress.NFTID().ToHex()
	// log
	CoreComponent.LogInfof("Found NFT output, nftId:%s,collectionId:%s，milestoneIndex:%d,milestoneTimestamp:%d",
		nftIdHex,
		collectionId,
		milestone,
		milestoneTimestamp,
	)

	pairs := im.ChainNameAndCollectionIdToGroupIdAndGroupNamePairs(im.HornetChainName, collectionId)
	if len(pairs) == 0 {
		return nil
	}
	// log pairs, loop then concat groupId:%s,groupName:%s"
	var logStr string
	for _, pair := range pairs {
		logStr += iotago.EncodeHex(pair.GroupId) + "," + pair.GroupName + ";"
	}
	CoreComponent.LogInfof("Found NFT output, nftId:%s,collectionId:%s，milestoneIndex:%d,milestoneTimestamp:%d,groupId,groupName:%s",
		nftIdHex,
		collectionId,
		milestone,
		milestoneTimestamp,
		logStr,
	)

	unlockConditionSet := nftOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(im.HornetChainName))
	var nfts []*im.NFT
	for _, pair := range pairs {
		groupId := pair.GroupId
		groupName := pair.GroupName
		// log groupId, ownerAddress, nftId, milestone, milestoneTimestamp, outputid
		CoreComponent.LogInfof("Found NFT output,groupId:%s,ownerAddress:%s,nftId:%s,milestoneIndex:%d,milestoneTimestamp:%d, outputId:%s",
			iotago.EncodeHex(groupId),
			ownerAddress,
			iotago.EncodeHex(nftId),
			milestone,
			milestoneTimestamp,
			iotago.EncodeHex(outputId),
		)
		nft := im.NewNFT(groupId, ownerAddress, nftId, groupName, "ipfsLink", milestone, milestoneTimestamp)
		nfts = append(nfts, nft)
	}
	return nfts
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
	senderAddressStr := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(im.HornetChainName))
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

func handlePublicKeyOutputFromINXLedgerOutput(output *inx.LedgerOutput) (*im.OutputIdHexAndAddressPair, error) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	return handlePublicKeyOutputFromINXOutput(iotaOutput, output.OutputId.Id)
}
func handlePublicKeyOutputFromINXOutput(iotaOutput iotago.Output, outputId []byte) (*im.OutputIdHexAndAddressPair, error) {
	//TODO check if output is sent from self

	// Ignore anything other than BasicOutputs
	if iotaOutput.Type() != iotago.OutputBasic {
		return nil, nil
	}
	basicOutput := iotaOutput.(*iotago.BasicOutput)
	// tag should have iotacat and groupId can be retrieved as well
	featureSet := basicOutput.FeatureSet()
	tag := featureSet.TagFeature()
	if tag == nil {
		return nil, nil
	}
	tagPayload := tag.Tag

	if !bytes.Equal(tagPayload, im.PublicKeyTag) {
		return nil, nil
	}

	outputIdHex := iotago.EncodeHex(outputId)
	unlockConditionSet := iotaOutput.UnlockConditionSet()
	address := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(im.HornetChainName))
	CoreComponent.LogInfof("Found PUBLICKEY output,outputId:%s,address:%s",
		outputIdHex,
		address,
	)
	return &im.OutputIdHexAndAddressPair{
		OutputIdHex: outputIdHex,
		Address:     address,
	}, nil

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
	amount := iotaOutput.Amount
	nativeTokens := iotaOutput.NativeTokens
	return handleTokenFromOutputType(amount, nativeTokens, iotaOutput, outputId, outputStatus, isUpdateGlobalAmount)
}
func handleTokenAmount(amount *big.Int, tokenId []byte, iotaOutput iotago.Output, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	tokenIdHash := im.Sha256HashBytes(tokenId)
	tokenIdHashFixed := [im.Sha256HashLen]byte{}
	copy(tokenIdHashFixed[:], tokenIdHash)
	tokenStatus := outputStatusToTokenStatus(outputStatus)
	unlockConditionSet := iotaOutput.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(im.HornetChainName))

	if isUpdateGlobalAmount {
		total := GetTokenTotal(tokenIdHashFixed)
		if tokenStatus == im.ImTokenStatusCreated {
			total.Add(amount)
		} else if tokenStatus == im.ImTokenStatusConsumed {
			total.Sub(amount)
		}
		// handle whale eligibility
		defer handleTokenWhaleEligibilityFromAddressGivenTotalAmount(tokenId, tokenIdHashFixed, ownerAddress, total.Get(), deps.IMManager, CoreComponent.Logger())
	}

	amountText := amount.Text(10)
	tokenStat := deps.IMManager.NewTokenStat(tokenId, outputId, ownerAddress, tokenStatus, amountText)
	return deps.IMManager.StoreOneToken(tokenStat)
}

func handleTokenWhaleEligibilityFromAddressGivenTotalAmount(tokenId []byte, tokenIdHash [im.Sha256HashLen]byte, address string, addressTotalAmount *big.Int, manager *im.Manager, logger *logger.Logger) error {
	// log enter
	//CoreComponent.LogInfof("handleTokenWhaleEligibilityFromAddressGivenTotalAmount,tokenType:%d,address:%s,totalAmount:%s", tokenType, address, totalAmount.Text(10))
	tokenTotalAmount := GetTokenTotal(tokenIdHash).Get()
	// total = total + 1 to prevent divide zero
	tokenTotalAmountFixed := new(big.Int).Add(tokenTotalAmount, big.NewInt(1))
	percentage := new(big.Float).Quo(new(big.Float).SetInt(addressTotalAmount), new(big.Float).SetInt(tokenTotalAmountFixed))
	// loop through all token based group
	if im.ConfigStoreChainNameAndQualifyTypeToGroupId[im.HornetChainName] != nil && im.ConfigStoreChainNameAndQualifyTypeToGroupId[im.HornetChainName]["token"] != nil {
		for _, groupId := range im.ConfigStoreChainNameAndQualifyTypeToGroupId[im.HornetChainName]["token"] {
			groupConfig := im.ConfigStoreGroupIdToGroupConfig[groupId]
			if groupConfig == nil {
				continue
			}
			groupTokenIdStr := groupConfig.TokenId
			groupTokenIdBytes, _ := iotago.DecodeHex(groupTokenIdStr)
			if !bytes.Equal(tokenId, groupTokenIdBytes) {
				continue
			}
			tokenThresStr := groupConfig.TokenThres
			tokenThres, ok := new(big.Float).SetString(tokenThresStr)
			if !ok {
				// log error
				CoreComponent.LogWarnf("handleTokenWhaleEligibilityFromAddressGivenTotalAmount ... SetString failed")
				continue
			}

			isEligible := percentage.Cmp(tokenThres) >= 0
			err := manager.SetWhaleEligibility(tokenId, groupConfig.GroupName, tokenThresStr, address, isEligible, logger)
			if err != nil {
				// log error
				CoreComponent.LogWarnf("handleTokenWhaleEligibilityFromAddressGivenTotalAmount ... SetWhaleEligibility failed:%s", err)
			}
		}
	}
	return nil
}

func handleTokenFromNFTOutput(iotaOutput *iotago.NFTOutput, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	amount := iotaOutput.Amount
	nativeTokens := iotaOutput.NativeTokens
	return handleTokenFromOutputType(amount, nativeTokens, iotaOutput, outputId, outputStatus, isUpdateGlobalAmount)
}
func handleTokenFromOutputType(basicTokenAmount uint64, nativeTokens iotago.NativeTokens, output iotago.Output, outputId []byte, outputStatus int, isUpdateGlobalAmount bool) error {
	// log enter handleTokenFromOutputType
	CoreComponent.LogInfof("handleTokenFromOutputType, basicTokenAmount:%d", basicTokenAmount)

	// loop through all token based group
	if im.ConfigStoreChainNameAndQualifyTypeToGroupId[im.HornetChainName] != nil && im.ConfigStoreChainNameAndQualifyTypeToGroupId[im.HornetChainName]["token"] != nil {
		// log token type is not nul
		CoreComponent.LogInfof("handleTokenFromOutputType, token type is not null")
		for _, groupId := range im.ConfigStoreChainNameAndQualifyTypeToGroupId[im.HornetChainName]["token"] {
			// log enter loop
			CoreComponent.LogInfof("handleTokenFromOutputType, enter loop")
			groupConfig := im.ConfigStoreGroupIdToGroupConfig[groupId]
			if groupConfig == nil {
				continue
			}
			// marshal groupConfig to json str
			jsonStr, err := json.Marshal(groupConfig)
			if err != nil {
				// log error
				CoreComponent.LogWarnf("handleTokenFromOutputType, json.Marshal failed:%s", err)
				continue
			} else {
				// log json str
				CoreComponent.LogInfof("handleTokenFromOutputType, jsonStr:%s", string(jsonStr))
			}
			// log groupConfig json str

			tokenIdStr := iotago.EncodeHex(im.SmrTokenId)
			tokenIdBytes := im.SmrTokenId
			amount := new(big.Int).SetUint64(basicTokenAmount)
			if groupConfig.TokenId != tokenIdStr {
				tokenIdStr = groupConfig.TokenId
				// case no native token, continue
				if nativeTokens == nil {
					continue
				}
				// loop then find based on tokenId
				foundToken := false
				for _, nativeToken := range nativeTokens {
					curTokenId := nativeToken.ID.ToHex()
					if curTokenId == tokenIdStr {
						// log found soon
						CoreComponent.LogInfof("handleTokenFromOutputType,found token soon")
						tokenIdBytes, _ = iotago.DecodeHex(curTokenId)
						amount = nativeToken.Amount
						foundToken = true
						break
					}
				}
				// if not found, continue
				if !foundToken {
					continue
				}
			} else {
				// log handle smr
				CoreComponent.LogInfof("handleTokenFromOutputType,handle smr")
			}
			// tokenThres := groupConfig.TokenThres

			err = handleTokenAmount(amount, tokenIdBytes, output, outputId, outputStatus, isUpdateGlobalAmount)
			if err != nil {
				return err
			}

		}
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
	senderAddressStr := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(im.HornetChainName))
	senderAddressSha256 := im.Sha256Hash(senderAddressStr)
	CoreComponent.LogInfof("Found GROUPFI Message output,payload len:%d,groupId len:%d,groupid:%s,outputId:%s,milestoneIndex:%d,milestoneTimestamp:%d，senderAddress:%s",
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
		nftList := nftFromINXOutput(output, outputId, milestoneIndex, milestoneTimestamp, log)
		if len(nftList) > 0 {
			nfts = append(nfts, nftList...)
		}
	}
	return nfts, offset, nil
}

// struct for outputId and output pair
type OutputIdOutputPair struct {
	OutputId string
	Output   iotago.Output
}
