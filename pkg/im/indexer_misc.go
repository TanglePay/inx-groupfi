package im

import (
	"context"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/core/marshalutil"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
	"github.com/pkg/errors"
)

type IndexerItemType int

const (
	MessageType IndexerItemType = iota
	NFTType
	SharedType
)

func initFinishedStoreKeyFromType(indexerItemType IndexerItemType) []byte {
	switch indexerItemType {
	case MessageType:
		return []byte("messageInitFinished")
	case NFTType:
		return []byte("nftInitFinished")
	case SharedType:
		return []byte("sharedInitFinished")
	}

	return []byte("unknownInitFinished")
}

// is inited
func (im *Manager) IsInitFinished(indexerItemType IndexerItemType) (bool, error) {
	key := initFinishedStoreKeyFromType(indexerItemType)
	contains, err := im.imStore.Has(key)
	if err != nil {
		msg := "failed to read IsInitFinished status for " + string(indexerItemType)
		return true, errors.New(msg)
	}

	return contains, nil
}

// mark inited
func (im *Manager) MarkInitFinished(indexerItemType IndexerItemType) error {
	key := initFinishedStoreKeyFromType(indexerItemType)
	if err := im.imStore.Set(key, []byte{}); err != nil {
		return errors.New("failed to MarkInitFinished for " + string(indexerItemType))
	}
	return im.imStore.Flush()
}

func initCurrentOffsetKey(indexerItemType IndexerItemType) []byte {
	m := marshalutil.New(12)

	switch indexerItemType {
	case MessageType:
		m.WriteByte(ImStoreKeyPrefixMessage)
	case NFTType:
		m.WriteByte(ImStoreKeyPrefixNFT)
	case SharedType:
		m.WriteByte(ImStoreKeyPrefixShared)
	}

	m.WriteBytes([]byte("offset"))

	return m.Bytes()
}

// store init start index
func (im *Manager) StoreInitCurrentOffset(offset *string, indexerItemType IndexerItemType) error {
	key := initCurrentOffsetKey(indexerItemType)
	m := marshalutil.New(4)
	m.WriteBytes([]byte(*offset))

	return im.imStore.Set(key, m.Bytes())
}

// read init start index
func (im *Manager) ReadInitCurrentOffset(indexerItemType IndexerItemType) (*string, error) {
	key := initCurrentOffsetKey(indexerItemType)
	v, err := im.imStore.Get(key)
	if err != nil {
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	offset := string(v)
	return &offset, nil
}

// TODO add retry
func (im *Manager) OutputIdToOutputAndMilestoneInfo(ctx context.Context, client *nodeclient.Client, outputIdHex string) (iotago.Output, uint32, uint32, error) {
	outputId, err := iotago.OutputIDFromHex(outputIdHex)
	if err != nil {
		return nil, 0, 0, err
	}
	outputResp, err := client.OutputWithMetadataByID(ctx, outputId)
	if err != nil {
		return nil, 0, 0, err
	}
	output, err := outputResp.Output()
	if err != nil {
		return nil, 0, 0, err
	}
	milestoneIndex := outputResp.Metadata.MilestoneIndexBooked
	milestoneTimestamp := outputResp.Metadata.MilestoneTimestampBooked
	return output, milestoneIndex, milestoneTimestamp, nil
}

// query basic output based on tag, with offset, return outputIds and new offset
func (im *Manager) QueryOutputIdsByTag(ctx context.Context, client nodeclient.IndexerClient, tag string, offset *string, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.BasicOutputsQuery{
		Tag: tag,
	}
	offsetStr := "nil"
	if offset != nil {
		offsetStr = *offset
	}
	logger.Infof("QueryOutputIdsByTag ... offset:%s,tag:%s", offsetStr, tag)
	if offset != nil {
		query.SetOffset(offset)
	}
	resultSet, err := client.Outputs(ctx, query)
	if err != nil {
		return nil, nil, err
	}
	logger.Infof("QueryOutputIdsByTag ... got result set")

	nextOffset := resultSet.Response.Cursor
	outputIds := resultSet.Response.Items
	// log lens of outputIds
	logger.Infof("QueryOutputIdsByTag ... got result set,offset:%s,outputIds len:%d", nextOffset, len(outputIds))
	return outputIds, nextOffset, nil
}
