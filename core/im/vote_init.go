package im

import (
	"context"

	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

// handle vote init
func handleVoteInit(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) {
	// make drainer
	drainer := makeItemDrainerForVoteInit(ctx, client, indexerClient)
	// handle vote init
	handleTypeInit(ctx, client, indexerClient, drainer, im.VoteType, im.VoteTagStr)
}

func makeItemDrainerForVoteInit(ctx context.Context, nodeHTTPAPIClient *nodeclient.Client, indexerClient nodeclient.IndexerClient) *im.ItemDrainer {
	return im.NewItemDrainer(ctx, func(outputIdUnwrapped interface{}) {
		outputId := outputIdUnwrapped.(string)
		output, err := deps.IMManager.OutputIdToOutput(ctx, nodeHTTPAPIClient, outputId)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
			return
		}
		// filter vote output
		basicOutput, is := deps.IMManager.FilterVoteOutput(output)
		if !is {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... FilterVoteOutput failed")
			return
		}
		deps.IMManager.HandleUserVoteGroupBasicOutputCreated(basicOutput)
	}, 100, 10, 200)
}
