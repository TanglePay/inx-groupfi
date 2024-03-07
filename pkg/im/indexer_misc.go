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
	MessageType      string = "MessageType"
	NFTType          string = "NFTType"
	SharedType       string = "SharedType"
	TokenBasicType   string = "TokenBasicType"
	TokenNFTType     string = "TokenNFTType"
	WhaleEligibility string = "WhaleEligibility"
	MarkType         string = "MarkType"
	MuteType         string = "MuteType"
	VoteType         string = "VoteType"
)

// get key for init finished
func initFinishedStoreKeyFromTopicAndExtra(topic string, extra string) []byte {
	prefixBytes := []byte{ImStoreKeyPrefixInitStatusFinish}
	topicBytes := Sha256Hash(topic)
	extraBytes := Sha256Hash(extra)
	return ConcatByteSlices(prefixBytes, topicBytes, extraBytes)
}

// get key for init offset
func initOffsetStoreKeyFromTopicAndExtra(topic string, extra string) []byte {
	prefixBytes := []byte{ImStoreKeyPrefixInitStatusOffset}
	topicBytes := Sha256Hash(topic)
	extraBytes := Sha256Hash(extra)
	return ConcatByteSlices(prefixBytes, topicBytes, extraBytes)
}

// is inited
func (im *Manager) IsInitFinished(topic string, extra string) (bool, error) {
	key := initFinishedStoreKeyFromTopicAndExtra(topic, extra)
	contains, err := im.imStore.Has(key)
	if err != nil {
		msg := "failed to read IsInitFinished status for " + topic
		return true, errors.New(msg)
	}

	return contains, nil
}

// mark inited
func (im *Manager) MarkInitFinished(topic string, extra string) error {
	key := initFinishedStoreKeyFromTopicAndExtra(topic, extra)
	if err := im.imStore.Set(key, []byte{}); err != nil {
		return errors.New("failed to MarkInitFinished for " + topic)
	}
	return im.imStore.Flush()
}

// store init start index
func (im *Manager) StoreInitCurrentOffset(offset *string, topic string, extra string) error {
	key := initOffsetStoreKeyFromTopicAndExtra(topic, extra)
	return im.imStore.Set(key, []byte(*offset))
}

// read init start index
func (im *Manager) ReadInitCurrentOffset(topic string, extra string) (*string, error) {
	key := initOffsetStoreKeyFromTopicAndExtra(topic, extra)
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
	query := &nodeclient.BasicOutputsQuery{
		IndexerCursorParas: nodeclient.IndexerCursorParas{
			PageSize: pageSize,
		},
	}
	return executeQuery(ctx, client, query, offset, logger)
}

// query nft output ids
func (im *Manager) QueryNFTOutputIds(ctx context.Context, client nodeclient.IndexerClient, offset *string, pageSize int, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.NFTsQuery{
		IndexerCursorParas: nodeclient.IndexerCursorParas{
			PageSize: 1000,
		},
	}
	return executeQuery(ctx, client, query, offset, logger)
}

// query nfts based on issuer address, with offset, return outputIds and new offset
func (im *Manager) QueryNFTIdsByIssuer(ctx context.Context, client nodeclient.IndexerClient, issuerBech32Address string, offset *string, pageSize int, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.NFTsQuery{
		IssuerBech32: issuerBech32Address,
		IndexerCursorParas: nodeclient.IndexerCursorParas{
			PageSize: pageSize,
		},
	}
	return executeQuery(ctx, client, query, offset, logger)
}

// query nfts with offset, return outputIds and new offset
func (im *Manager) QueryNFTIds(ctx context.Context, client nodeclient.IndexerClient, offset *string, pageSize int, logger *logger.Logger) (iotago.HexOutputIDs, *string, error) {
	query := &nodeclient.NFTsQuery{
		IndexerCursorParas: nodeclient.IndexerCursorParas{
			PageSize: pageSize,
		},
	}
	return executeQuery(ctx, client, query, offset, logger)
}
