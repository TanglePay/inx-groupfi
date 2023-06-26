package im

import (
	"context"
	"fmt"

	Core "github.com/TanglePay/inx-iotacat/core/im"
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/syncutils"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

const (
	StorePrefixHealth byte = 255
)

var (
	ErrParticipationCorruptedStorage               = errors.New("the participation database was not shutdown properly")
	ErrParticipationEventStartedBeforePruningIndex = errors.New("the given participation event started before the pruning index of this node")
	ErrParticipationEventBallotCanOverflow         = errors.New("the given participation duration in combination with the maximum voting weight can overflow uint64")
	ErrParticipationEventStakingCanOverflow        = errors.New("the given participation staking nominator and denominator in combination with the duration can overflow uint64")
	ErrParticipationEventAlreadyExists             = errors.New("the given participation event already exists")
)

type ProtocolParametersProvider func() *iotago.ProtocolParameters
type NodeStatusProvider func(ctx context.Context) (confirmedIndex iotago.MilestoneIndex, pruningIndex iotago.MilestoneIndex)
type LedgerUpdatesProvider func(ctx context.Context, startIndex iotago.MilestoneIndex, endIndex iotago.MilestoneIndex, handler func(index iotago.MilestoneIndex, created []*Message) error) error

// Manager is used to track the outcome of participation in the tangle.
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
}

// the default options applied to the Manager.
var defaultOptions = []Option{
	WithTagMessage("IOTACAT"),
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

func (im *Manager) ApplyNewLedgerUpdate(index iotago.MilestoneIndex, created []*Message) error {
	// Lock the state to avoid anyone reading partial results while we apply the state
	im.Lock()
	defer im.Unlock()

	if err := im.storeLedgerIndex(index); err != nil {
		return err
	}
	if created != nil && len(created) > 0 {
		msg := created[0]
		Core.CoreComponent.LogInfof("store new message: groupId:%s, outputId:%s, milestoneindex:%d, milestonetimestamp:%d", msg.GetGroupIdStr(), msg.GetOutputIdStr(), msg.MileStoneIndex, msg.MileStoneTimestamp)
	}
	if err := im.storeNewMessages(created); err != nil {
		return err
	}
	return nil
}

func (im *Manager) GetMessages(groupId []byte, token uint32, size int) ([]*Message, error) {
	return im.readMessageAfterToken(groupId, token, size)
}
