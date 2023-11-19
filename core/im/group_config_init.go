package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func handleGroupConfigInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	config := `[
    {"groupName":"iceberg-1","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x064d0eaefb86a94eb326ff633c22cdf744decca954bb93b1572b449d324ae717"]},
    {"groupName":"iceberg-2","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x451ac5e96cea2ddf399924ce22f0e56a4b485ca417aba1430e9e5ce582d605f2"]},
    {"groupName":"iceberg-3","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x4136c04d4bc25011f5b03dc9d31f4082bc7c19233cfeb2803aef241b1bb29c92"]},
    {"groupName":"iceberg-4","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x6ba06fb2371ec3615ff45667a152f729e2c9a24643f4e26e06b297def1e9c4bf"]},
    {"groupName":"iceberg-5","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x3ba971dbb7bfd6d466835a0c8463169e2b41ad7da26ec7dfcfd77140d0eff4c9"]},
    {"groupName":"iceberg-6","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x576dcc9c3199650187c981b21b045ef09f56515d7a1c46e9456fa994334f2740"]},
    {"groupName":"iceberg-7","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0xcf0f598ff3ee378b03906af4de48030bc6082831dfcf67730be780a317d98265"]},
    {"groupName":"iceberg-8","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x592b20d610ee4618949dd4f969db7ffc81d93486bfe1ab63b9201618b6be3a48"]},
    {"groupName":"smr-whale","schemaVersion":1,"messageType":1,"authScheme":2, "qualifyType":"token", "tokenThres":"0.01","chainName":"smr","collectionIds":[]},
    {"groupName":"iceberg-composite","schemaVersion":1,"messageType":2,"authScheme":2, "qualifyType":"nft", "chainName":"smr","collectionIds":["0x064d0eaefb86a94eb326ff633c22cdf744decca954bb93b1572b449d324ae717", "0x451ac5e96cea2ddf399924ce22f0e56a4b485ca417aba1430e9e5ce582d605f2", "0x4136c04d4bc25011f5b03dc9d31f4082bc7c19233cfeb2803aef241b1bb29c92", "0x6ba06fb2371ec3615ff45667a152f729e2c9a24643f4e26e06b297def1e9c4bf", "0x3ba971dbb7bfd6d466835a0c8463169e2b41ad7da26ec7dfcfd77140d0eff4c9", "0x576dcc9c3199650187c981b21b045ef09f56515d7a1c46e9456fa994334f2740", "0xcf0f598ff3ee378b03906af4de48030bc6082831dfcf67730be780a317d98265", "0x592b20d610ee4618949dd4f969db7ffc81d93486bfe1ab63b9201618b6be3a48"]}
]`
	configRawContent := []byte(config)
	deps.IMManager.HandleGroupConfigRawContent(configRawContent, CoreComponent.Logger())
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
