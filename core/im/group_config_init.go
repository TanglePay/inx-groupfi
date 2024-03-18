package im

import (
	"context"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func handleGroupConfigInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	config := CONFIG_IN_TEXT
	configRawContent := []byte(config)
	deps.IMManager.HandleGroupConfigRawContent(configRawContent, CoreComponent.Logger())
	deps.IMManager.LogConfigStoreGroupIdToGroupConfig(CoreComponent.Logger())
	return
	issuerBech32Address := im.IcebergCollectionConfigIssuerAddress
	drainer := makeDrainerForGroupConfigNft(ctx, client, indexerClient)
	isNFTInitializationFinished, err := deps.IMManager.IsInitFinished(im.NFTType, issuerBech32Address)
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	for !isNFTInitializationFinished {
		select {
		case <-ctx.Done():
			CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
			return
		default:
			_, isHasMore, err := processInitializationForNftWithIssuer(ctx, client, indexerClient, issuerBech32Address, drainer)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... processInitializationForNftWithIssuer failed:%s", err)
				continue
			}
			if !isHasMore {
				err = deps.IMManager.MarkInitFinished(im.NFTType, issuerBech32Address)
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
					continue
				}
				isNFTInitializationFinished = true
			}
		}
	}
}

// processInitializationForGroupConfigNft
func processInitializationForNftWithIssuer(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, issuerBech32Address string, drainer *im.ItemDrainer) (int, bool, error) {
	// get init offset
	itemType := im.NFTType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, issuerBech32Address)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return 0, false, err
	}
	// get outputs and meta data
	outputHexIds, nextOffset, err := deps.IMManager.QueryNFTIdsByIssuer(ctx, indexerClient, issuerBech32Address, initOffset, drainer.FetchSize, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextGroupConfigNFTs failed:%s", err)
		return 0, false, err
	}
	//drain
	outputHexIdsInterface := make([]interface{}, len(outputHexIds))
	for i, v := range outputHexIds {
		outputHexIdsInterface[i] = v
	}
	drainer.Drain(outputHexIdsInterface)
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, issuerBech32Address)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return 0, false, err
		}
	}
	isHasMore := nextOffset != nil
	return len(outputHexIds), isHasMore, nil
}

// make drainer for group config nft
func makeDrainerForGroupConfigNft(ctx context.Context, nodeHTTPAPIClient *nodeclient.Client, indexerClient nodeclient.IndexerClient) *im.ItemDrainer {
	return im.NewItemDrainer(ctx, func(outputIdUnwrapped interface{}) {
		outputId := outputIdUnwrapped.(string)
		output, err := deps.IMManager.OutputIdToOutput(ctx, nodeHTTPAPIClient, outputId)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
			return
		}
		// check and convert to nft
		isNFT, nftOutput := filterNFTOutput(output)
		if !isNFT {
			return
		}
		// handle group config
		deps.IMManager.HandleGroupNFTOutputCreated(nftOutput, CoreComponent.Logger())
	}, 1000, 100, 2000)

}

// calculate is group public for all group config
func calculateIsGroupPublicForAllGroupConfig(initCtx *InitContext) {
	err := deps.IMManager.CalculateIfGroupIsPublicForAllGroups(initCtx.Logger)
	if err != nil {
		// log error
		initCtx.Logger.Warnf("LedgerInit ... CalculateIfGroupIsPublicForAllGroups failed:%s", err)
	}

}
