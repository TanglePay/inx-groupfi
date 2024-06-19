package im

import (
	"context"

	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

var (
	IsIniting = true
	BootTime  = uint32(0)
	// uint32
	CurrentMilestoneIndex          = uint32(0)
	CurrentMilestoneTimestamp      = uint32(0)
	LastTimeReceiveEventFromHornet = uint32(0)
	HornetChainId                  = uint32(0)
	HornetChainName                = ""
	CurrentNetwork                 = ShimmerMainNet
	NodeHTTPAPIClient              *nodeclient.Client
	NodeIndexerAPIClient           nodeclient.IndexerClient
	ListeningCtx                   context.Context
	CurrentNodeProtocol            *iotago.ProtocolParameters = nil
	Logger                         *logger.Logger             = nil
	Im                             *Manager                   = nil
)
