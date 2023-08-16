package im

import (
	"context"
	"math/big"
	"net/http"
	"time"

	"go.uber.org/dig"

	"github.com/TanglePay/inx-iotacat/pkg/daemon"
	"github.com/TanglePay/inx-iotacat/pkg/im"
	"github.com/iotaledger/hive.go/core/app"
	"github.com/iotaledger/hive.go/core/app/pkg/shutdown"
	"github.com/iotaledger/hive.go/core/database"
	"github.com/iotaledger/hive.go/core/kvstore"
	hornetdb "github.com/iotaledger/hornet/v2/pkg/database"
	"github.com/iotaledger/inx-app/pkg/httpserver"
	"github.com/iotaledger/inx-app/pkg/nodebridge"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
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
func processInitializationForMessage(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) ([]*im.Message, bool, error) {
	// get init offset
	itemType := im.MessageType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, false, err
	}
	// get outputs and meta data
	messages, nextOffset, err := fetchNextMessage(ctx, client, indexerClient, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextMessage failed:%s", err)
		return nil, false, err
	}
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return nil, false, err
		}
	}
	isHasMore := nextOffset != nil
	return messages, isHasMore, nil
}

// processInitializationForShared
func processInitializationForShared(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) ([]*im.Message, bool, error) {
	// get init offset
	itemType := im.SharedType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, false, err
	}
	// get outputs and meta data
	messages, nextOffset, err := fetchNextShared(ctx, client, indexerClient, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextShared failed:%s", err)
		return nil, false, err
	}
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return nil, false, err
		}
	}
	isHasMore := nextOffset != nil
	return messages, isHasMore, nil
}

// processInitializationForToken
func processInitializationForTokenForBasicOutput(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient, drainer *OutputIdDrainer) (int, bool, error) {
	// get init offset
	itemType := im.TokenBasicType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return 0, false, err
	}
	// get outputhexids
	outputHexIds, nextOffset, err := deps.IMManager.QueryBasicOutputIds(ctx, indexerClient, initOffset, CoreComponent.Logger(), drainer.fetchSize)

	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextTokenForBasicOutput failed:%s", err)
		return 0, false, err
	}
	//drain
	drainer.Drain(outputHexIds)

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
func processInitializationForTokenForNftOutput(ctx context.Context, client *nodeclient.Client, indexerClient nodeclient.IndexerClient) (map[string]iotago.Output, bool, error) {
	// get init offset
	itemType := im.TokenNFTType
	initOffset, err := deps.IMManager.ReadInitCurrentOffset(itemType, "")
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... ReadInitOffset failed:%s", err)
		return nil, false, err
	}
	// get outputs and meta data
	outputsMap, nextOffset, err := fetchNextOutputsForNFTType(ctx, client, indexerClient, initOffset, CoreComponent.Logger())
	if err != nil {
		// log error
		CoreComponent.LogWarnf("LedgerInit ... fetchNextTokenForNftOutput failed:%s", err)
		return nil, false, err
	}
	// update init offset
	if nextOffset != nil {
		err = deps.IMManager.StoreInitCurrentOffset(nextOffset, itemType, "")
		if err != nil {
			// log error
			CoreComponent.LogWarnf("LedgerInit ... StoreInitCurrentOffset failed:%s", err)
			return nil, false, err
		}
	}
	isHasMore := nextOffset != nil
	return outputsMap, isHasMore, nil
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

}
func run() error {

	// create a background worker that handles the init situation
	if err := CoreComponent.Daemon().BackgroundWorker("LedgerInit", func(ctx context.Context) {
		CoreComponent.LogInfo("Starting LedgerInit ... done")

		// handle messsages init
		isMessageInitializationFinished, err := deps.IMManager.IsInitFinished(im.MessageType, "")
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}

		// shared init
		isSharedInitializationFinished, err := deps.IMManager.IsInitFinished(im.SharedType, "")
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		nodeHTTPAPIClient := nodeclient.New("https://api.shimmer.network")
		indexerClient, err := nodeHTTPAPIClient.Indexer(ctx)
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		// loop until isMessageInitializationFinished is true
		for !isMessageInitializationFinished || !isSharedInitializationFinished {
			select {
			case <-ctx.Done():
				CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
				return
			default:
				if !isMessageInitializationFinished {
					messages, isHasMore, err := processInitializationForMessage(ctx, nodeHTTPAPIClient, indexerClient)
					if err != nil {
						// log error then continue
						CoreComponent.LogWarnf("LedgerInit ... processInitializationForMessage failed:%s", err)
						continue
					}
					if len(messages) > 0 {
						err = deps.IMManager.ApplyNewLedgerUpdate(0, messages, nil, nil, CoreComponent.Logger(), true)
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... ApplyNewLedgerUpdate failed:%s", err)
							continue
						}
					}
					if !isHasMore {
						err = deps.IMManager.MarkInitFinished(im.MessageType, "")
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
							continue
						}
						isMessageInitializationFinished = true
					}
				}
				if !isSharedInitializationFinished {
					messages, isHasMore, err := processInitializationForShared(ctx, nodeHTTPAPIClient, indexerClient)
					if err != nil {
						// log error then continue
						CoreComponent.LogWarnf("LedgerInit ... processInitializationForShared failed:%s", err)
						continue
					}
					if len(messages) > 0 {
						err = deps.IMManager.ApplyNewLedgerUpdate(0, nil, nil, messages, CoreComponent.Logger(), true)
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... ApplyNewLedgerUpdate failed:%s", err)
							continue
						}
					}
					if !isHasMore {
						err = deps.IMManager.MarkInitFinished(im.SharedType, "")
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
							continue
						}
						isSharedInitializationFinished = true
					}
				}
			}
		}
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
						nfts, isHasMore, err := processInitializationForNFT(ctx, nodeHTTPAPIClient, indexerClient, issuerBech32Address)
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... processInitializationForNFT failed:%s", err)
							continue
						}
						if len(nfts) > 0 {
							err = deps.IMManager.ApplyNewLedgerUpdate(0, nil, nfts, nil, CoreComponent.Logger(), true)
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
						}
					}
				}
			}
		}

		isTokenBasicFinished, err := deps.IMManager.IsInitFinished(im.TokenBasicType, "")
		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		TokenBasicDrainer := NewOutputIdDrainer(ctx, func(outputId string) {
			// log outputId
			CoreComponent.LogInfof("LedgerInit ... TokenBasicDrainer outputId:%s", outputId)
			output, err := deps.IMManager.OutputIdToOutput(ctx, nodeHTTPAPIClient, outputId)
			if err != nil {
				// log error
				CoreComponent.LogWarnf("LedgerInit ... OutputIdToOutput failed:%s", err)
				return
			}
			outputIdBytes, _ := iotago.DecodeHex(outputId)
			err = handleTokenFromINXOutput(output, outputIdBytes, ImOutputTypeCreated, false)
			if err != nil {
				// log error then continue
				CoreComponent.LogWarnf("LedgerInit ... handleTokenFromINXOutput failed:%s", err)

			}
		}, 1000, 100, 2000)
		totalBasicOutputProcessed := big.NewInt(0)
		startTime := time.Now()
		for !isTokenBasicFinished {
			select {
			case <-ctx.Done():
				CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
				return
			default:
				ct, isHasMore, err := processInitializationForTokenForBasicOutput(ctx, nodeHTTPAPIClient, indexerClient, TokenBasicDrainer)
				if err != nil {
					// log error then continue
					CoreComponent.LogWarnf("LedgerInit ... processInitializationForTokenForBasicOutput failed:%s", err)
					continue
				}
				// totalBasicOutputProcessed = totalBasicOutputProcessed + ct
				totalBasicOutputProcessed.Add(totalBasicOutputProcessed, big.NewInt(int64(ct)))
				timeElapsed := time.Since(startTime)
				averageProcessedPerSecond := big.NewInt(0)
				averageProcessedPerSecond.Div(totalBasicOutputProcessed, big.NewInt(int64(timeElapsed.Seconds())+1))
				// log totalBasicOutputProcessed and timeElapsed and averageProcessedPerSecond
				CoreComponent.LogInfof("LedgerInit ... totalBasicOutputProcessed:%d,timeElapsed:%s,averageProcessedPerSecond:%d", totalBasicOutputProcessed, timeElapsed, averageProcessedPerSecond)
				if !isHasMore {
					err = deps.IMManager.MarkInitFinished(im.TokenBasicType, "")
					if err != nil {
						// log error then continue
						CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
						continue
					}
					isTokenBasicFinished = true
				}
			}
		}

		/*
			isTokenNFTFinished, err := deps.IMManager.IsInitFinished(im.TokenNFTType, "")
			if err != nil {
				CoreComponent.LogPanicf("failed to start worker: %s", err)
			}
			for !isTokenNFTFinished {
				select {
				case <-ctx.Done():
					// log ctx cancel then exit
					CoreComponent.LogInfo("LedgerInit ... ctx.Done()")
					return
				default:
					outputsMap, isHasMore, err := processInitializationForTokenForNftOutput(ctx, nodeHTTPAPIClient, indexerClient)
					if err != nil {
						// log error then continue
						CoreComponent.LogWarnf("LedgerInit ... processInitializationForTokenForNftOutput failed:%s", err)
						continue
					}
					if len(outputsMap) > 0 {
						// loop outputs
						for outputId, output := range outputsMap {
							outputIdBytes, _ := iotago.DecodeHex(outputId)
							err = handleTokenFromINXOutput(output, outputIdBytes, ImOutputTypeCreated, false)
							if err != nil {
								// log error then continue
								CoreComponent.LogWarnf("LedgerInit ... handleTokenFromINXOutput failed:%s", err)
								continue
							}
						}
					}
					if !isHasMore {
						err = deps.IMManager.MarkInitFinished(im.TokenNFTType, "")
						if err != nil {
							// log error then continue
							CoreComponent.LogWarnf("LedgerInit ... MarkInitFinished failed:%s", err)
							continue
						}
						isTokenNFTFinished = true
					}
				}
			}
		*/
		//TODO handle total smr
		CoreComponent.LogInfo("Finishing LedgerInit ... done")

		smrTotalAddress := deps.IMManager.GetTotalAddressFromType(im.ImTokenTypeSMR)

		smrTotalValue, err := deps.IMManager.GetBalanceOfOneAddress(im.ImTokenTypeSMR, smrTotalAddress)

		if err != nil {
			CoreComponent.LogPanicf("failed to start worker: %s", err)
		}
		smrTotal := GetSmrTokenTotal()
		smrTotal.Add(smrTotalValue)

		smrTokenPrefix := deps.IMManager.TokenKeyPrefixFromTokenType(im.ImTokenTypeSMR)

		// hash set for address
		addressSet := make(map[string]struct{})
		deps.IMManager.GetImStore().Iterate(smrTokenPrefix, func(key kvstore.Key, value kvstore.Value) bool {
			_, address := deps.IMManager.AmountAndAddressFromTokenValuePayload(value)
			addressSet[address] = struct{}{}
			return true
		})
		// loop addressSet
		for address := range addressSet {
			handleTokenWhaleEligibilityFromAddressGivenTotalAmount(im.ImTokenTypeSMR, address, smrTotal.Get(), deps.IMManager, CoreComponent.Logger())
		}
		//
		startListeningToLedgerUpdate()
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
