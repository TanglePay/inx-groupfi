package im

import (
	"context"

	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handle nft init
func handleNFTInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	/*
		issuerBech32AddressList := NFT_ISSUER_LIST
		for _, issuerBech32Address := range issuerBech32AddressList {
			// log start process nft for issuerBech32Address
			CoreComponent.LogInfof("LedgerInit ... start process nft for issuerBech32Address:%s", issuerBech32Address)
			select {
			case <-ctx.Done():
				CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
				return
			default:
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
						nfts, isHasMore, err := processInitializationForNFT(ctx, client, indexerClient, issuerBech32Address)
						// log found len(nfts)
						CoreComponent.LogInfof("LedgerInit ... found len(nfts):%d", len(nfts))
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... processInitializationForNFT failed:%s", err)
							continue
						}
						if len(nfts) > 0 {
							DataFromListenning := &im.DataFromListenning{
								CreatedNft: nfts,
							}
							err = deps.IMManager.ApplyNewLedgerUpdate(0, DataFromListenning, CoreComponent.Logger(), true)
							if err != nil {
								// log error then continue
								CoreComponent.LogWarnf("LedgerInit ... ApplyNewLedgerUpdate failed:%s", err)
								continue
							}
						}
						if !isHasMore {
							err = deps.IMManager.MarkInitFinished(im.NFTType, issuerBech32Address)
							if err != nil {
								// log error then continue
								CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
								continue
							}
							isNFTInitializationFinished = true
							// log finish process nft for issuerBech32Address
							CoreComponent.LogInfof("LedgerInit ... finish process nft for issuerBech32Address:%s", issuerBech32Address)
						}
					}
				}
			}
		}
	*/
	/*
		topic := "nft_init"
		// func(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, initOffset *string, logger *logger.Logger) ([]string, *string, error)
		itemFetcher := func(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, initOffset *string, logger *logger.Logger) ([]string, *string, error) {
			ids, nextOffset, err := deps.IMManager.QueryNFTIds(ctx, indexerClient, initOffset, 1000, logger)
			if err != nil {
				return nil, nil, err
			}
			return ids, nextOffset, nil
		}
		drainer := im.NewItemDrainer(ctx, func(outputIdUnwrapped interface{}) {
			outputIdHex := outputIdUnwrapped.(string)
			output, err := deps.IMManager.OutputIdToOutput(ctx, client, outputIdHex)
			if err != nil {
				// log error
				CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
				return
			}

			outputId, err := iotago.DecodeHex(outputIdHex)
			if err != nil {
				// log error
				CoreComponent.LogWarnf("LedgerInit ... DecodeHex failed:%s", err)
				return
			}

			nfts, is, _ := deps.IMManager.FilterNftOutput(outputId, output, CoreComponent.Logger())
			if !is {
				return
			}

			deps.IMManager.StoreNewNFTsDeleteConsumedNfts(nfts, nil, CoreComponent.Logger())

		}, 100, 10, 200)
		// postEffect func(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) error,
		postEffect :=  func(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) error {
			//
			HandleGenericInit(ctx, client, indexerClient, topic, itemFetcher, nil, drainer, CoreComponent.Logger())
	*/
}
