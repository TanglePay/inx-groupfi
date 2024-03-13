package im

import (
	"context"
	"net/http"
	"os"
	"strconv"
	"time"

	"go.uber.org/dig"

	"github.com/TanglePay/inx-groupfi/pkg/daemon"
	"github.com/TanglePay/inx-groupfi/pkg/im"
	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/hive.go/core/app/pkg/shutdown"
	"github.com/iotaledger/hive.go/core/database"
	hornetdb "github.com/iotaledger/hornet/v2/pkg/database"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"github.com/joho/godotenv"
	"github.com/pkg/errors"
)

func init() {
	CoreComponent = &app.CoreComponent{
		Component: &app.Component{
			Name:      "IM",
			Params:    params,
			DepsFunc:  func(cDeps dependencies) { deps = cDeps },
			Provide:   provide,
			Configure: configure,
			Run:       run,
		},
	}
}

var (
	CoreComponent *app.CoreComponent
	deps          dependencies
)

type dependencies struct {
	dig.In
	IMManager       *im.Manager
	NodeBridge      *nodebridge.NodeBridge
	ShutdownHandler *shutdown.ShutdownHandler
}

func provide(c *dig.Container) error {

	type imDeps struct {
		dig.In
		NodeBridge *nodebridge.NodeBridge
	}

	return c.Provide(func(deps imDeps) *im.Manager {

		dbEngine, err := database.EngineFromStringAllowed(ParamsIM.Database.Engine)
		if err != nil {
			CoreComponent.LogErrorAndExit(err)
		}
		imStore, err := hornetdb.StoreWithDefaultSettings(ParamsIM.Database.Path, true, dbEngine)
		if err != nil {
			CoreComponent.LogErrorAndExit(err)
		}
		if err != nil {
			CoreComponent.LogErrorAndExit(err)
		}
		im, err := im.NewManager(
			CoreComponent.Daemon().ContextStopped(),
			imStore,
			deps.NodeBridge.ProtocolParameters,
			NodeStatus,
			LedgerUpdates,
		)
		if err != nil {
			CoreComponent.LogErrorAndExit(err)
		}
		CoreComponent.LogInfof("Initialized ImManager at milestone %d", im.LedgerIndex())

		return im
	})
}

func configure() error {
	if err := CoreComponent.App().Daemon().BackgroundWorker("Close Im database", func(ctx context.Context) {
		<-ctx.Done()

		CoreComponent.LogInfo("Syncing Im database to disk ...")
		if err := deps.IMManager.CloseDatabase(); err != nil {
			CoreComponent.LogErrorfAndExit("Syncing Im database to disk ... failed: %s", err)
		}
		CoreComponent.LogInfo("Syncing Im database to disk ... done")
	}, daemon.PriorityCloseIMDatabase); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	return nil
}

// processInitializationForToken
func processInitializationForTokenForBasicOutput(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, drainer *im.ItemDrainer) (int, bool, error) {
	// get init offset
	itemType := im.TokenBasicType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return 0, false, err
	}
	// get outputhexids
	outputHexIds, nextOffset, err := deps.IMManager.QueryBasicOutputIds(ctx, indexerClient, initOffset, CoreComponent.Logger(), drainer.FetchSize)

	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextTokenForBasicOutput failed:%s", err)
		return 0, false, err
	}
	//drain
	// wrap outputHexIds to interface
	outputHexIdsInterface := make([]interface{}, len(outputHexIds))
	for i, v := range outputHexIds {
		outputHexIdsInterface[i] = v
	}
	drainer.Drain(outputHexIdsInterface)

	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return 0, false, err
		}
	}
	isHasMore := nextOffset != nil
	return len(outputHexIds), isHasMore, nil
}

// processInitializationForTokenForNftOutput
func processInitializationForTokenForNftOutput(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, drainer *im.ItemDrainer) (int, bool, error) {
	// get init offset
	itemType := im.TokenNFTType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return 0, false, err
	}
	// get outputhexids
	outputHexIds, nextOffset, err := deps.IMManager.QueryNFTOutputIds(ctx, indexerClient, initOffset, drainer.FetchSize, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextTokenForNftOutput failed:%s", err)
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
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return 0, false, err
		}
	}
	isHasMore := nextOffset != nil
	return len(outputHexIds), isHasMore, nil
}

// processInitializationForNFT
func processInitializationForNFT(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, issuerBech32Address string) ([]*im.NFT, bool, error) {
	// get init offset
	itemType := im.NFTType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, issuerBech32Address)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, false, err
	}
	// get outputs and meta data
	nfts, nextOffset, err := fetchNextNFTs(ctx, client, indexerClient, initOffset, issuerBech32Address, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextNFTs failed:%s", err)
		return nil, false, err
	}
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, issuerBech32Address)
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return nil, false, err
		}
	}
	isHasMore := nextOffset != nil
	return nfts, isHasMore, nil
}
func startListeningToLedgerUpdate() {
	// create a background worker that handles the im events
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerUpdates", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerUpdates ... done")

		startIndex := deps.IMManager.LedgerIndex()
		// log start index
		CoreComponent.LogInfof("LedgerUpdates start index:%d", startIndex)
		if startIndex > 0 {
			startIndex++
		}

		if err := LedgerUpdates(ctx, startIndex, 0, func(index iotago.MilestoneIndex, dataFromListenning *im.DataFromListenning) error {
			if err := deps.IMManager.ApplyNewLedgerUpdate(index, dataFromListenning, CoreComponent.Logger(), false); err != nil {
				CoreComponent.LogErrorfAndExit("ApplyNewLedgerUpdate failed: %s", err)

				return err
			}
			// CoreComponent.LogInfof("Applying milestone %d with %d new outputs took %s", index, len(created), time.Since(timeStart).Truncate(time.Millisecond))

			return nil
		}); err != nil {
			CoreComponent.LogWarnf("Listening to LedgerUpdates failed: %s", err)
			deps.ShutdownHandler.SelfShutdown("disconnected from INX", false)
		}

		CoreComponent.LogInfo("Stopping LedgerUpdates ... done")
	}, daemon.PriorityStopIMLedgerConfirmedUpdate); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	// create another background worker that handles the im blocks
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerUpdateBlock", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerUpdateBlock ... done")

		if err := LedgerUpdateBlock(ctx, 0, 0); err != nil {
			CoreComponent.LogWarnf("Listening to LedgerUpdateBlock failed: %s", err)
			deps.ShutdownHandler.SelfShutdown("disconnected from INX", false)
		}

		CoreComponent.LogInfo("Stopping LedgerUpdateBlock ... done")
	}, daemon.PriorityStopIMLedgerBlockUpdate); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
}

func run() error {
	im.IsIniting = true
	im.BootTime = im.GetCurrentEpochTimestamp()
	// load .groupfi-env file
	godotenv.Load(".groupfi-env")
	apiUrl := os.Getenv("SHIMMER_API_URL")
	hornetChainName := os.Getenv("HORNET_CHAINNAME")
	networkId := os.Getenv("NETWORK_ID")
	im.HornetChainName = hornetChainName
	networkIdInt, err := strconv.Atoi(networkId)
	if err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	if networkIdInt == im.ShimmerMainNet.ID {
		im.CurrentNetwork = im.ShimmerMainNet
	} else if networkIdInt == im.ShimmerTestNet.ID {
		im.CurrentNetwork = im.ShimmerTestNet
	}
	// log api url
	CoreComponent.LogInfof("apiUrl:%s", apiUrl)

	im.InitIpfsShell()
	nodeHTTPAPIClient := nodeclient.New(apiUrl)
	im.NodeHTTPAPIClient = nodeHTTPAPIClient

	// create a background worker that handles the init situation
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerInit", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerInit ... done")
		resp, err := nodeHTTPAPIClient.Info(ctx)
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		im.CurrentNodeProtocol = &resp.Protocol
		im.ListeningCtx = ctx
		indexerClient, err := nodeHTTPAPIClient.Indexer(ctx)
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		im.NodeIndexerAPIClient = indexerClient
		// handle group config init
		handleGroupConfigInit(ctx, nodeHTTPAPIClient, indexerClient)

		initCtx := &InitContext{
			Ctx:           ctx,
			Client:        nodeHTTPAPIClient,
			IndexerClient: indexerClient,
			Logger:        CoreComponent.Logger(),
		}
		// handle all nft first pass
		ProcessAllNftFirstPass(initCtx)

		// handle all basic output first pass
		ProcessAllBasicOutputFirstPass(initCtx)

		// handle mark init
		handleMarkInit(initCtx)

		// handle message init
		handleMessageInit(initCtx)

		CoreComponent.LogInfo("Finishing LedgerInit ... done")
		im.IsIniting = false
		startListeningToLedgerUpdate()
	}, daemon.PriorityStopIMInit); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	// create a background worker that handles the API
	if err := CoreComponent.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting API ... done")

		CoreComponent.LogInfo("Starting API server ...")

		e := httpserver.NewEcho(CoreComponent.Logger(), nil, ParamsRestAPI.DebugRequestLoggerEnabled)

		setupRoutes(e, ctx, nodeHTTPAPIClient)
		go func() {
			CoreComponent.LogInfof("You can now access the API using: http://%s", ParamsRestAPI.BindAddress)
			if err := e.Start(ParamsRestAPI.BindAddress); err != nil && !errors.Is(err, http.ErrServerClosed) {
				CoreComponent.LogErrorfAndExit("Stopped REST-API server due to an error (%s)", err)
			}
		}()

		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := ParamsRestAPI.BindAddress
		if ParamsRestAPI.AdvertiseAddress != "" {
			advertisedAddress = ParamsRestAPI.AdvertiseAddress
		}

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, APIRoute, advertisedAddress); err != nil {
			CoreComponent.LogErrorfAndExit("Registering INX api route failed: %s", err)
		}

		cancelRegister()

		CoreComponent.LogInfo("Starting API server ... done")
		<-ctx.Done()
		CoreComponent.LogInfo("Stopping API ...")

		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, APIRoute); err != nil {
			CoreComponent.LogWarnf("Unregistering INX api route failed: %s", err)
		}

		shutdownCtx, shutdownCtxCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer shutdownCtxCancel()

		//nolint:contextcheck // false positive
		if err := e.Shutdown(shutdownCtx); err != nil {
			CoreComponent.LogWarn(err)
		}

		CoreComponent.LogInfo("Stopping API ... done")
	}, daemon.PriorityStopIMAPI); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	// create a mqtt server that handles the MQTT connection
	if err := CoreComponent.Daemon().BackgroundWorker("MQTT", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting MQTT server ...")
		server, err := deps.IMManager.MakeMqttServer(ParamsMQTT.Websocket.BindAddress)
		if err != nil {
			CoreComponent.LogErrorfAndExit("Starting MQTT server failed: %s", err)
		}
		go func() {
			err := server.Serve()
			if err != nil {
				CoreComponent.LogErrorfAndExit("Starting MQTT server failed: %s", err)
			}

		}()
		ctxRegister, cancelRegister := context.WithTimeout(ctx, 5*time.Second)

		advertisedAddress := ParamsMQTT.Websocket.BindAddress
		if ParamsMQTT.Websocket.AdvertiseAddress != "" {
			advertisedAddress = ParamsMQTT.Websocket.AdvertiseAddress
		}

		if err := deps.NodeBridge.RegisterAPIRoute(ctxRegister, MQTTAPIRoute, advertisedAddress); err != nil {
			CoreComponent.LogErrorfAndExit("Registering INX mqtt route failed: %s", err)
		}

		cancelRegister()
		CoreComponent.LogInfo("Starting MQTT server ... done")
		//wait ctx done
		<-ctx.Done()
		CoreComponent.LogInfo("Stopping MQTT server ...")
		ctxUnregister, cancelUnregister := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancelUnregister()

		//nolint:contextcheck // false positive
		if err := deps.NodeBridge.UnregisterAPIRoute(ctxUnregister, MQTTAPIRoute); err != nil {
			CoreComponent.LogWarnf("Unregistering INX api route failed: %s", err)
		}
		server.Close()
		CoreComponent.LogInfo("Stopping MQTT server ... done")
	}, daemon.PriorityStopIMMQTT); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}
	return nil
}

func GetManager() *im.Manager {
	return deps.IMManager
}
