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
	TokenBasicType
	TokenNFTType
	WhaleEligibility
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
	case TokenBasicType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("tokenBasicInitFinished"), extraBytes)
	case TokenNFTType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("tokenNFTInitFinished"), extraBytes)
	case WhaleEligibility:
		return ConcatByteSlices(prefixBytes, Sha256Hash("whaleEligibilityInitFinished"), extraBytes)
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
	case TokenBasicType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("tokenBasicInitOffset"), extraBytes)
	case TokenNFTType:
		return ConcatByteSlices(prefixBytes, Sha256Hash("tokenNFTInitOffset"), extraBytes)
	case WhaleEligibility:
		return ConcatByteSlices(prefixBytes, Sha256Hash("whaleEligibilityInitOffset"), extraBytes)
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

// outputId to output
func (im *Manager) OutputIdToOutput(ctx context.Context, client *nodeclient.Client, outputIdHex string) (iotago.Output, error) {
	outputId, err := iotago.OutputIDFromHex(outputIdHex)
	if err != nil {
		return nil, err
	}
	output, err := client.OutputByID(ctx, outputId)
	if err != nil {
		return nil, err
	}
	return output, nil
}

// query basic output based on tag, with offset, return outputIds and new offset
func (im *Manager) QueryOutputIdsByTag(ctx context.Context, client nodeclient.IndexerClient, tag string, offset *string, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.BasicOutputsQuery{
		Tag: tag,
	}
	return executeQuery(ctx, client, query, offset, logger)
}
func executeQuery(ctx context.Context, client nodeclient.IndexerClient, query nodeclient.IndexerQuery, offset *string, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	offsetStr := "nil"
	if offset != nil {
		offsetStr = *offset
	}
	urlPara, _ := query.URLParas()
	logger.Infof("QueryOutputIds ... offset:%s, urlPara:%s", offsetStr, urlPara)
	if offset != nil {
		query.SetOffset(offset)
	}
	resultSet, err := client.Outputs(ctx, query)
	if err != nil {
		return nil, nil, err
	}

	isNext := resultSet.Next()
	// log isNext
	logger.Infof("QueryOutputIds ... isNext:%v", isNext)

	if resultSet.Error != nil {
		logger.Infof("QueryOutputIds ... got result set,error:%s", resultSet.Error)
		return nil, nil, resultSet.Error
	}
	resp := resultSet.Response
	if resp == nil {
		logger.Infof("QueryOutputIds ... got result set,resp is nil")
		return nil, nil, errors.New("resp is nil")
	}
	// log ledgerIndex
	ledgerIndex := resp.LedgerIndex

	nextOffset := resultSet.Response.Cursor
	outputIds := resultSet.Response.Items
	logger.Infof("QueryOutputIds ... got result set,ledgerIndex:%d, num of outputIds:%d", ledgerIndex, len(outputIds))
	return outputIds, nextOffset, nil
}

// query any outputs, with offset, return outputIds and new offset
func (im *Manager) QueryBasicOutputIds(ctx context.Context, client nodeclient.IndexerClient, offset *string, logger *logger.Logger, pageSize int) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.BasicOutputsQuery{}
	query.PageSize = pageSize
	return executeQuery(ctx, client, query, offset, logger)
}

// query nft output ids
func (im *Manager) QueryNFTOutputIds(ctx context.Context, client nodeclient.IndexerClient, offset *string, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.NFTsQuery{}
	query.PageSize = 10
	return executeQuery(ctx, client, query, offset, logger)
}

// query nfts based on issuer address, with offset, return outputIds and new offset
func (im *Manager) QueryNFTIdsByIssuer(ctx context.Context, client nodeclient.IndexerClient, issuerBech32Address string, offset *string, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.NFTsQuery{
		IssuerBech32: issuerBech32Address,
	}
	return executeQuery(ctx, client, query, offset, logger)
}
