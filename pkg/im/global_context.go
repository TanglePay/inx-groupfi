package im

import (
	"context"

	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

var (
	IsIniting = true
	// uint32
	CurrentMilestoneIndex     = uint32(0)
	CurrentMilestoneTimestamp = uint32(0)
	HornetChainName           = ""
	CurrentNetwork            = ShimmerMainNet
	NodeHTTPAPIClient         *nodeclient.Client
	NodeIndexerAPIClient      nodeclient.IndexerClient
	ListeningCtx              context.Context
	CurrentNodeProtocol       *iotago.ProtocolParameters = nil
)
