package im

import (
	"context"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

func handleGroupConfigInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	/*
		config := CONFIG_IN_TEXT
		configRawContent := []byte(config)
		deps.IMManager.HandleGroupConfigRawContent(configRawContent, CoreComponent.Logger())
		deps.IMManager.LogConfigStoreGroupIdToGroupConfig(CoreComponent.Logger())
		return
	*/
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
