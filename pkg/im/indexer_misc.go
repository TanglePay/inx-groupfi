package im

import (
	"context"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
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

func initFinishedStoreKeyFromType(indexerItemType IndexerItemType, extra string) []byte {
	prefixBytes := []byte{ImStoreKeyPrefixInitStatus}
	extraBytes := Sha256Hash(extra)
	switch indexerItemType {
	case MessageType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("messageInitFinished"), extraBytes)
	case NFTType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("nftInitFinished"), extraBytes)
	case SharedType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("sharedInitFinished"), extraBytes)
	}

	return nil
}

// is inited
func (im *Manager) IsInitFinished(indexerItemType IndexerItemType, extra string) (bool, error) {
	key := initFinishedStoreKeyFromType(indexerItemType, extra)
	contains, err := im.imStore.Has(key)
	if err != nil {
		msg := "failed to read IsInitFinished status for " + string(indexerItemType)
		return true, errors.New(msg)
	}

	return contains, nil
}

// mark inited
func (im *Manager) MarkInitFinished(indexerItemType IndexerItemType, extra string) error {
	key := initFinishedStoreKeyFromType(indexerItemType, extra)
	if err := im.imStore.Set(key, []byte{}); err != nil {
		return errors.New("failed to MarkInitFinished for " + string(indexerItemType))
	}
	return im.imStore.Flush()
}

func initCurrentOffsetKey(indexerItemType IndexerItemType, extra string) []byte {
	prefixBytes := []byte{ImStoreKeyPrefixInitStatus}
	extraBytes := Sha256Hash(extra)
	switch indexerItemType {
	case MessageType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("messageInitOffset"), extraBytes)
	case NFTType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("nftInitOffset"), extraBytes)
	case SharedType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("sharedInitOffset"), extraBytes)
	}

	return nil
}

// store init start index
func (im *Manager) StoreInitCurrentOffset(offset *string, indexerItemType IndexerItemType, extra string) error {
	key := initCurrentOffsetKey(indexerItemType, extra)
	return im.imStore.Set(key, []byte(*offset))
}

// read init start index
func (im *Manager) ReadInitCurrentOffset(indexerItemType IndexerItemType, extra string) (*string, error) {
	key := initCurrentOffsetKey(indexerItemType, extra)
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

	isNext := resultSet.Next()
	// log isNext
	logger.Infof("QueryOutputIdsByTag ... isNext:%v", isNext)

	if resultSet.Error != nil {
		logger.Infof("QueryOutputIdsByTag ... got result set,error:%s", resultSet.Error)
		return nil, nil, resultSet.Error
	}
	resp := resultSet.Response
	if resp == nil {
		logger.Infof("QueryOutputIdsByTag ... got result set,resp is nil")
		return nil, nil, errors.New("resp is nil")
	}

	nextOffset := resultSet.Response.Cursor
	outputIds := resultSet.Response.Items

	return outputIds, nextOffset, nil
}

// query nfts based on issuer address, with offset, return outputIds and new offset
func (im *Manager) QueryNFTIdsByIssuer(ctx context.Context, client nodeclient.IndexerClient, issuerBech32Address string, offset *string, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.NFTsQuery{
		IssuerBech32: issuerBech32Address,
	}
	offsetStr := "nil"
	if offset != nil {
		offsetStr = *offset
	}
	logger.Infof("QueryOutputIdsByIssuer ... offset:%s,issuer address:%s", offsetStr, issuerBech32Address)
	if offset != nil {
		query.SetOffset(offset)
	}
	resultSet, err := client.Outputs(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	isNext := resultSet.Next()
	// log isNext
	logger.Infof("QueryOutputIdsByIssuer ... isNext:%v", isNext)
	if resultSet.Error != nil {
		logger.Infof("QueryOutputIdsByIssuer ... got result set,error:%s", resultSet.Error)
		return nil, nil, resultSet.Error
	}
	resp := resultSet.Response
	if resp == nil {
		logger.Infof("QueryOutputIdsByIssuer ... got result set,resp is nil")
		return nil, nil, errors.New("resp is nil")
	}

	nextOffset := resultSet.Response.Cursor
	outputIds := resultSet.Response.Items

	return outputIds, nextOffset, nil
}
