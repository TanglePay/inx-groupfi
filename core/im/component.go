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

func run() error {

	var initChan chan iotago.MilestoneIndex
	var isInitSent = false
	// create a background worker that handles the init situation
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerInit", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerInit ... done")
		isInited, err := deps.IMManager.IsInited()
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		if !isInited {
			CoreComponent.LogInfo("LedgerInitFirstTime ... start")
			// make initChan non blocking, contain at most one element
			initChan = make(chan iotago.MilestoneIndex, 1)
			// blocking to wait first milestone to start initing
			firstMilestoneIndex := <-initChan
			CoreComponent.LogInfof("LedgerInitFirstTime ... firstMilestoneIndex:%d", firstMilestoneIndex)
			// store as init end index
			deps.IMManager.StoreInitEndIndex(firstMilestoneIndex)
			// store zero as init start index
			deps.IMManager.StoreInitStartIndex(0)
			// mark inited
			deps.IMManager.MarkInited()
			CoreComponent.LogInfo("LedgerInitFirstTime ... done")
		}
		// get init start index
		startIndex, err := deps.IMManager.ReadInitStartIndex()
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		// get init end index
		endIndex, err := deps.IMManager.ReadInitEndIndex()
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		nodeHTTPAPIClient := nodeclient.New("http://hornet:9092")

		// loop forever when start index - end index > 1
		for endIndex-startIndex > 1 {
			CoreComponent.LogInfof("LedgerInit ... StartIndex:%d, EndIndex:%d", startIndex, endIndex)
			var shouldExit = false
			// retry once
			var tryLefted = 2
			for tryLefted > 0 {
				tryLefted--
				mileStoneResp, err := nodeHTTPAPIClient.MilestoneByIndex(ctx, startIndex)
				if err != nil {
					// log then continue
					CoreComponent.LogWarnf("LedgerInit ... MilestoneByIndex failed:%s", err)
					continue
				}
				mileStoneTimestamp := mileStoneResp.Timestamp
				milestone := mileStoneResp.Index
				resp, err := nodeHTTPAPIClient.MilestoneUTXOChangesByIndex(ctx, startIndex)
				if err != nil {
					// log then continue
					CoreComponent.LogWarnf("LedgerInit ... MilestoneUTXOChangesByIndex failed:%s", err)
					continue
				}
				outputIds := resp.CreatedOutputs
				// get outputs
				// outputResp, err := nodeHTTPAPIClient.OutputByID(ctx context.Context, outputId)
				// output, err := outputResp.Output()
				// map outputIds to outputs via OutputByID
				var outputs []iotago.Output
				var isFailed = false
				for _, outputIdHex := range outputIds {
					outputId, err := iotago.OutputIDFromHex(outputIdHex)
					if err != nil {
						// log then break, continue outer loop
						CoreComponent.LogWarnf("LedgerInit ... OutputIDFromHex failed:%s", err)
						isFailed = true
						break
					}
					output, err := nodeHTTPAPIClient.OutputByID(ctx, outputId)
					if err != nil {
						// log then break, continue outer loop
						CoreComponent.LogWarnf("LedgerInit ... OutputByID failed:%s", err)
						isFailed = true
						break
					}

					if err != nil {
						// log then break, continue outer loop
						CoreComponent.LogWarnf("LedgerInit ... Output failed:%s", err)
						isFailed = true
						break
					}
					outputs = append(outputs, output)
				}
				if isFailed {
					continue
				}

				var createdMessage []*im.Message
				var createdNft []*im.NFT
				var createdShared []*im.Message
				for outputInx, output := range outputs {
					outputIdHex := outputIds[outputInx]
					outputId, err := iotago.DecodeHex(outputIdHex)
					if err != nil {
						// log then break, continue outer loop
						CoreComponent.LogWarnf("LedgerInit ... OutputIDFromHex failed:%s", err)
						break
					}
					o := messageFromINXOutput(output, outputId, milestone, mileStoneTimestamp)
					if o != nil {
						createdMessage = append(createdMessage, o)
					}
					// nft
					nft := nftFromINXOutput(output, outputId, milestone, mileStoneTimestamp)
					if nft != nil {
						createdNft = append(createdNft, nft)
					}
					// shared
					shared := sharedOutputFromINXOutput(output, outputId, milestone, mileStoneTimestamp)
					if shared != nil {
						createdShared = append(createdShared, shared)
					}
				}
				err = deps.IMManager.ApplyNewLedgerUpdate(milestone, createdMessage, createdNft, createdShared, CoreComponent.Logger(), true)
				if err != nil {
					CoreComponent.LogWarnf("Listening to LedgerUpdates failed: %s", err)
					deps.ShutdownHandler.SelfShutdown("disconnected from INX", false)
					shouldExit = true
					break
				} else {
					break
				}
			}
			if shouldExit {
				break
			}
			startIndex++
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
			deps.IMManager.Lock()
			if !isInitSent {
				isInitSent = true
				initChan <- index
			}
			deps.IMManager.Unlock()
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
