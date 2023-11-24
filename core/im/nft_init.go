package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handle nft init
func handleNFTInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	issuerBech32AddressList := []string{
		"smr1zqry6r4wlwr2jn4nymlkx0pzehm5fhkv492thya32u45f8fjftn3wkng2mp",
		"smr1zpz3430fdn4zmheenyjvughsu44ykjzu5st6hg2rp609eevz6czlye60pe7",
		"smr1zpqndszdf0p9qy04kq7un5clgzptclqeyv70av5q8thjgxcmk2wfy7pspe5",
		"smr1zp46qmajxu0vxc2l73tx0g2j7u579jdzgeplfcnwq6ef0hh3a8zt7wnx9dv",
		"smr1zqa6juwmk7lad4rxsddqeprrz60zksdd0k3xa37lelthzsxsal6vjygkl9e",
		"smr1zptkmnyuxxvk2qv8exqmyxcytmcf74j3t4apc3hfg4h6n9pnfun5q26j6w4",
		"smr1zr8s7kv070hr0zcrjp40fhjgqv9uvzpgx80u7emnp0ncpgchmxpx25paqmf",
		"smr1zpvjkgxkzrhyvxy5nh20j6wm0l7grkf5s6l7r2mrhyspvx9khcaysmam589",
	}
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
