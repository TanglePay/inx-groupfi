package im

import (
	"context"
	"fmt"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/core/syncutils"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

const (
	StorePrefixHealth byte = 255
)

var (
	ErrParticipationCorruptedStorage               = errors.New("the groupfi database was not shutdown properly")
	ErrParticipationEventStartedBeforePruningIndex = errors.New("the given groupfi event started before the pruning index of this node")
	ErrParticipationEventBallotCanOverflow         = errors.New("the given groupfi duration in combination with the maximum voting weight can overflow uint64")
	ErrParticipationEventStakingCanOverflow        = errors.New("the given groupfi staking nominator and denominator in combination with the duration can overflow uint64")
	ErrParticipationEventAlreadyExists             = errors.New("the given groupfi event already exists")
)

type ProtocolParametersProvider func() *iotago.ProtocolParameters
type NodeStatusProvider func(ctx context.Context) (confirmedIndex iotago.MilestoneIndex, pruningIndex iotago.MilestoneIndex)
type LedgerUpdatesProvider func(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex, handler func(index iotago.MilestoneIndex, dataFromListenning *DataFromListenning) error) error

// Manager is used to track the outcome of groupfi in the tangle.
type Manager struct {
	// lock used to secure the state of the Manager.
	syncutils.RWMutex

	//nolint:containedctx // false positive
	ctx context.Context

	protocolParametersFunc ProtocolParametersProvider
	nodeStatusFunc         NodeStatusProvider

	ledgerUpdatesFunc LedgerUpdatesProvider

	// holds the Manager options.
	opts *Options

	imStore       kvstore.KVStore
	imStoreHealth *kvstore.StoreHealthTracker
	mqttServer    *MQTTServer
}

// the default options applied to the Manager.
var defaultOptions = []Option{
	WithTagMessage("GROUPFI"),
}

// Options define options for the Manager.
type Options struct {
	// defines the tag payload to track
	tagMessage []byte
}

// applies the given Option.
func (so *Options) apply(opts ...Option) {
	for _, opt := range opts {
		opt(so)
	}
}

// WithTagMessage defines the Manager tag payload to track.
func WithTagMessage(tagMessage string) Option {
	return func(opts *Options) {
		opts.tagMessage = []byte(tagMessage)
	}
}

// Option is a function setting a Manager option.
type Option func(opts *Options)

// NewManager creates a new Manager instance.
func NewManager(
	ctx context.Context,
	imStore kvstore.KVStore,
	protocolParametersProvider ProtocolParametersProvider,
	nodeStatusProvider NodeStatusProvider,
	ledgerUpdatesProvider LedgerUpdatesProvider,
	opts ...Option) (*Manager, error) {

	options := &Options{}
	options.apply(defaultOptions...)
	options.apply(opts...)

	healthTracker, err := kvstore.NewStoreHealthTracker(imStore, []byte{StorePrefixHealth}, DBVersionIM, nil)
	if err != nil {
		return nil, err
	}
	manager := &Manager{
		ctx:                    ctx,
		protocolParametersFunc: protocolParametersProvider,
		nodeStatusFunc:         nodeStatusProvider,
		ledgerUpdatesFunc:      ledgerUpdatesProvider,
		imStore:                imStore,
		imStoreHealth:          healthTracker,
		opts:                   options,
	}

	err = manager.init()
	if err != nil {
		return nil, err
	}

	return manager, nil
}

func (im *Manager) init() error {

	corrupted, err := im.imStoreHealth.IsCorrupted()
	if err != nil {
		return err
	}
	if corrupted {
		return ErrParticipationCorruptedStorage
	}

	correctDatabasesVersion, err := im.imStoreHealth.CheckCorrectStoreVersion()
	if err != nil {
		return err
	}

	if !correctDatabasesVersion {
		databaseVersionUpdated, err := im.imStoreHealth.UpdateStoreVersion()
		if err != nil {
			return err
		}

		if !databaseVersionUpdated {
			//nolint:revive // this error message is shown to the user
			return errors.New("im database version mismatch. The database scheme was updated. Please delete the database folder and start again.")
		}
	}

	// Mark the database as corrupted here and as clean when we shut it down
	return im.imStoreHealth.MarkCorrupted()
}

// CloseDatabase flushes the store and closes the underlying database.
func (im *Manager) CloseDatabase() error {
	var flushAndCloseError error

	if err := im.imStoreHealth.MarkHealthy(); err != nil {
		flushAndCloseError = err
	}

	if err := im.imStore.Flush(); err != nil {
		flushAndCloseError = err
	}
	if err := im.imStore.Close(); err != nil {
		flushAndCloseError = err
	}

	return flushAndCloseError
}

func (im *Manager) LedgerIndex() iotago.MilestoneIndex {
	im.RLock()
	defer im.RUnlock()
	index, err := im.readLedgerIndex()
	if err != nil {
		panic(fmt.Errorf("failed to read ledger index: %w", err))
	}

	return index
}

// get imStore
func (im *Manager) GetImStore() kvstore.KVStore {
	return im.imStore
}

func (im *Manager) ApplyNewLedgerUpdate(index iotago.MilestoneIndex, dataFromListenning *DataFromListenning, logger *logger.Logger, isSkipUpdate bool) error {
	// Lock the state to avoid anyone reading partial results while we apply the state
	im.Lock()
	defer im.Unlock()
	defer im.imStore.Flush()
	if !isSkipUpdate {
		if err := im.storeLedgerIndex(index); err != nil {
			return err
		}
	}
	// unwrap dataFromListenning
	createdMessage := dataFromListenning.CreatedMessage
	createdNft := dataFromListenning.CreatedNft
	createdShared := dataFromListenning.CreatedShared
	consumedMessage := dataFromListenning.ConsumedMessage
	consumedShared := dataFromListenning.ConsumedShared
	consumedNft := dataFromListenning.ConsumedNft
	createdMark := dataFromListenning.CreatedMark
	consumedMark := dataFromListenning.ConsumedMark
	createdMute := dataFromListenning.CreatedMute
	consumedMute := dataFromListenning.ConsumedMute
	createdVote := dataFromListenning.CreatedVote
	consumedVote := dataFromListenning.ConsumedVote
	createdPublicKeyOutputIdHexAndAddressPairs := dataFromListenning.CreatedPublicKeyOutputIdHexAndAddressPairs
	createdDid := dataFromListenning.CreatedDid
	consumedDid := dataFromListenning.ConsumedDid
	if len(createdMessage) > 0 {
		msg := createdMessage[0]
		logger.Infof("store new message: groupId:%s, outputId:%s, milestoneindex:%d, milestonetimestamp:%d", msg.GetGroupIdStr(), msg.GetOutputIdStr(), msg.MileStoneIndex, msg.MileStoneTimestamp)
	}
	if err := im.storeNewMessages(createdMessage, logger, false); err != nil {
		return err
	}
	if err := im.storeNewNFTsDeleteConsumedNfts(createdNft, consumedNft, logger); err != nil {
		return err
	}
	if len(createdShared) > 0 {
		shared := createdShared[0]
		logger.Infof("store new shared: groupId:%s, outputId:%s, milestoneindex:%d, milestonetimestamp:%d", shared.GetGroupIdStr(), shared.GetOutputIdStr(), shared.MileStoneIndex, shared.MileStoneTimestamp)
	}
	if err := im.storeNewShareds(createdShared, logger); err != nil {
		return err
	}
	if err := im.DeleteConsumedShareds(consumedShared, logger); err != nil {
		return err
	}
	if err := im.deleteConsumedMessages(consumedMessage, logger); err != nil {
		return err
	}
	// pair consumed mark with created mark, by address
	outputPairMap := make(map[string]*OutputPair)
	for _, mark := range consumedMark {
		ProcessOutputToOutputPair(outputPairMap, mark, true)
	}
	for _, mark := range createdMark {
		ProcessOutputToOutputPair(outputPairMap, mark, false)
	}
	// for each outputPair, call HandleGroupMarkBasicOutputConsumedAndCreated
	for _, outputPair := range outputPairMap {
		im.HandleGroupMarkBasicOutputConsumedAndCreated(outputPair.ConsumedOutput, outputPair.CreatedOutput, logger)
	}
	if len(consumedMute) > 0 {
		for _, mute := range consumedMute {
			im.HandleUserMuteGroupMemberBasicOutputConsumed(mute)
		}
	}
	if len(createdMute) > 0 {
		for _, mute := range createdMute {
			im.HandleUserMuteGroupMemberBasicOutputCreated(mute, logger)
		}
	}
	if len(consumedVote) > 0 {
		for _, vote := range consumedVote {
			im.HandleUserVoteGroupBasicOutputConsumed(vote, logger)
		}
	}
	if len(createdVote) > 0 {
		for _, vote := range createdVote {
			im.HandleUserVoteGroupBasicOutputCreated(vote, logger)
		}
	}
	if len(createdPublicKeyOutputIdHexAndAddressPairs) > 0 {
		for _, outputIdHexAndAddressPair := range createdPublicKeyOutputIdHexAndAddressPairs {
			outputId, err := iotago.OutputIDFromHex(outputIdHexAndAddressPair.OutputIdHex)
			if err != nil {
				// log then continue
				logger.Infof("ApplyNewLedgerUpdate, iotago.OutputIDFromHex error:%s", err.Error())
				continue
			}
			transactionIdHex := outputId.TransactionID().ToHex()
			im.GetAddressPublicKeyFromTransactionId(ListeningCtx, NodeHTTPAPIClient, transactionIdHex, outputIdHexAndAddressPair.Address, logger)
		}
	}
	if len(consumedDid) > 0 || len(createdDid) > 0 {
		im.HandleDidConsumedAndCreated(consumedDid, createdDid, logger)
	}
	return nil

}

// make mqtt server
func (im *Manager) MakeMqttServer(websocketBindAddress string) (*MQTTServer, error) {
	opts := &MQTTOpts{
		WebsocketBindAddress: websocketBindAddress,
		qos:                  1,
	}

	server, err := NewMQTTServer(opts)
	if err != nil {
		return nil, err
	}
	im.mqttServer = server
	return server, nil
}

type DataFromListenning struct {
	CreatedMessage                             []*Message
	ConsumedMessage                            []*Message
	CreatedNft                                 []*NFT
	CreatedShared                              []*Message
	ConsumedShared                             []*Message
	ConsumedNft                                []*NFT
	CreatedPublicKeyOutputIdHexAndAddressPairs []*OutputIdHexAndAddressPair
	CreatedMark                                []*OutputAndOutputId
	ConsumedMark                               []*OutputAndOutputId
	CreatedMute                                []*iotago.BasicOutput
	ConsumedMute                               []*iotago.BasicOutput
	CreatedVote                                []*iotago.BasicOutput
	ConsumedVote                               []*iotago.BasicOutput
	ConsumedDid                                []*Did
	CreatedDid                                 []*Did
}
