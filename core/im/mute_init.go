package im

import (
	"context"

	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handle mute init
func handleMuteInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	// make drainer
	drainer := makeItemDrainerForMuteInit(ctx, client, indexerClient)
	// handle mute init
	handleTypeInit(ctx, client, indexerClient, drainer, im.MuteType, im.MuteTagStr)
}

func makeItemDrainerForMuteInit(ctx context.Context, nodeHTTPAPIClient *nodeclient.Client, indexerClient nodeclient.IndexerClient) *im.ItemDrainer {
	return im.NewItemDrainer(ctx, func(outputIdUnwrapped interface{}) {
		outputId := outputIdUnwrapped.(string)
		output, err := deps.IMManager.OutputIdToOutput(ctx, nodeHTTPAPIClient, outputId)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
			return
		}
		// filter vote output
		basicOutput, is := deps.IMManager.FilterMuteOutput(output, CoreComponent.Logger())
		if !is {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... FilterVoteOutput failed")
			return
		}
		deps.IMManager.HandleUserMuteGroupMemberBasicOutputCreated(basicOutput, CoreComponent.Logger())
	}, 100, 10, 200)
}
