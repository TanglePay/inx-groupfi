package im

import (
	"context"
	"net/http"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/dig"

	"github.com/TanglePay/inx-iotacat/pkg/daemon"
	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/hive.go/core/app/pkg/shutdown"
	"github.com/iotaledger/hive.go/core/database"
	hornetdb "github.com/iotaledger/hornet/v2/pkg/database"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
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
func processInitializationForMessage(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) ([]*im.Message, error) {
	// get init offset
	itemType := im.MessageType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, err
	}
	// get outputs and meta data
	messages, nextOffset, err := fetchNextMessage(ctx, client, indexerClient, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextMessage failed:%s", err)
		return nil, err
	}
	// update init offset
	err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType)
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
		return nil, err
	}
	return messages, nil
}

func run() error {

	// create a background worker that handles the init situation
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerInit", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerInit ... done")

		// handle messsages init
		isMessageInitializationFinished, err := deps.IMManager.IsInitFinished(im.MessageType)
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}

		nodeHTTPAPIClient := nodeclient.New("https://api.shimmer.network")
		indexerClient, err := nodeHTTPAPIClient.Indexer(ctx)

		// loop until isMessageInitializationFinished is true
		for !isMessageInitializationFinished {
			if !isMessageInitializationFinished {
				messages, err := processInitializationForMessage(ctx, nodeHTTPAPIClient, indexerClient)
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... processInitializationForMessage failed:%s", err)
					continue
				}
				if len(messages) > 0 {
					err = deps.IMManager.ApplyNewLedgerUpdate(0, messages, nil, nil, CoreComponent.Logger(), true)
				} else {
					err = deps.IMManager.MarkInitFinished(im.MessageType)
					if err != nil {
						// log error then continue
						CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
						continue
					}
					isMessageInitializationFinished = true
				}
			}
		}
		CoreComponent.LogInfo("Stopping LedgerInit ... done")
	}, daemon.PriorityStopIM); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	// create a background worker that handles the im events
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerUpdates", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerUpdates ... done")

		startIndex := deps.IMManager.LedgerIndex()
		// log start index
		CoreComponent.LogInfof("LedgerUpdates start index:%d", startIndex)
		if startIndex > 0 {
			startIndex++
		}

		if err := LedgerUpdates(ctx, startIndex, 0, func(index iotago.MilestoneIndex, createdMessage []*im.Message, createdNft []*im.NFT, createdShared []*im.Message) error {
			if err := deps.IMManager.ApplyNewLedgerUpdate(index, createdMessage, createdNft, createdShared, CoreComponent.Logger(), false); err != nil {
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
	}, daemon.PriorityStopIM); err != nil {
		CoreComponent.LogPanicf("failed to start worker: %s", err)
	}

	// create a background worker that handles the API
	if err := CoreComponent.Daemon().BackgroundWorker("API", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting API ... done")

		CoreComponent.LogInfo("Starting API server ...")

		e := httpserver.NewEcho(CoreComponent.Logger(), nil, ParamsRestAPI.DebugRequestLoggerEnabled)

		setupRoutes(e)
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

	return nil
}

func GetManager() *im.Manager {
	return deps.IMManager
}
