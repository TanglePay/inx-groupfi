package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handle nft init
func handleNFTInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
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
}
