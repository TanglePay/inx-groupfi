package im

import (
	"context"
	"fmt"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

var publicKeyTagRawStr = "GROUPFISELFPUBLICKEY"
var PublicKeyTag = []byte(publicKeyTagRawStr)
var PublicKeyTagStr = iotago.EncodeHex(PublicKeyTag)

func keyFromAddressPublicKey(address string) []byte {
	return ConcatByteSlices([]byte{ImStoreKeyPrefixAddressPublicKey}, Sha256Hash(address))
}
func (im *Manager) StoreOnePublickKey(bech32Address string, publicKey []byte) error {
	key := keyFromAddressPublicKey(bech32Address)
	return im.imStore.Set(key, publicKey)
}

func (im *Manager) ReadOnePublicKey(bech32Address string) ([]byte, error) {
	key := keyFromAddressPublicKey(bech32Address)
	publicKey, err := im.imStore.Get(key)
	if err != nil {
		return nil, err
	}
	return publicKey, nil
}

type IotaNodeInfo struct {
	ID                 int    `json:"id"`
	IsFaucetAvailable  bool   `json:"isFaucetAvailable"`
	ApiUrl             string `json:"apiUrl"`
	ExplorerApiUrl     string `json:"explorerApiUrl"`
	ExplorerApiNetwork string `json:"explorerApiNetwork"`
	NetworkID          string `json:"networkId"`
	InxMqttEndpoint    string `json:"inxMqttEndpoint"`
}

type TransactionItem struct {
	IsSpent  bool   `json:"isSpent"`
	OutputID string `json:"outputId"`
}

type TransactionHistoryResponse struct {
	Items []TransactionItem `json:"items"`
}

var (
	ShimmerMainNet = IotaNodeInfo{
		ID:                 102,
		IsFaucetAvailable:  false,
		ApiUrl:             "https://mainnet.shimmer.node.tanglepay.com",
		ExplorerApiUrl:     "https://explorer-api.shimmer.network/stardust",
		ExplorerApiNetwork: "shimmer",
		NetworkID:          "14364762045254553490",
		InxMqttEndpoint:    "wss://test.api.iotacat.com/api/iotacatmqtt/v1",
	}

	ShimmerTestNet = IotaNodeInfo{
		ID:                 101,
		IsFaucetAvailable:  true,
		ApiUrl:             "https://test.api.iotacat.com",
		ExplorerApiUrl:     "https://explorer-api.shimmer.network/stardust",
		ExplorerApiNetwork: "testnet",
		NetworkID:          "1856588631910923207",
		InxMqttEndpoint:    "wss://test.api.iotacat.com/mqtt",
	}
)

func GetTransactionHistory(ctx context.Context, node IotaNodeInfo, bech32Address string, logger *logger.Logger) (string, error) {
	url := fmt.Sprintf("%s/transactionhistory/%s/%s", node.ExplorerApiUrl, node.ExplorerApiNetwork, bech32Address)
	params := map[string]string{
		"pageSize": "1000",
		"sort":     "newest",
	}

	var resp TransactionHistoryResponse
	if err := PerformGetRequest(ctx, url, params, &resp); err != nil {
		return "", err
	}

	for _, item := range resp.Items {
		if item.IsSpent {
			return item.OutputID, nil
		}
	}

	return "", nil
}

type Unlock struct {
	Type      int `json:"type"`
	Signature struct {
		PublicKey string `json:"publicKey"`
	} `json:"signature"`
}

type TransactionResponse struct {
	Block struct {
		Payload struct {
			Unlocks []Unlock `json:"unlocks"`
		} `json:"payload"`
	} `json:"block"`
}

func GetPublicKeyViaTransactionId(ctx context.Context, client *nodeclient.Client, transactionId string) (string, error) {

	/*
		url := fmt.Sprintf("%s/transaction/%s/%s", node.ExplorerApiUrl, node.ExplorerApiNetwork, transactionId)
		params := map[string]string{} // No additional parameters in the original function
			var resp TransactionResponse
			if err := PerformGetRequest(ctx, url, params, &resp); err != nil {
				return "", err
			}
	*/

	transactionIdBytes, err := iotago.DecodeHex(transactionId)
	if err != nil {
		return "", err
	}
	var txId iotago.TransactionID
	copy(txId[:], transactionIdBytes)
	block, err := client.TransactionIncludedBlock(ctx, txId, nil)
	if err != nil {
		return "", err
	}
	for _, unlock := range block.Payload.(*iotago.Transaction).Unlocks {
		if unlock.Type() == iotago.UnlockSignature {
			sigUnlock := unlock.(*iotago.SignatureUnlock)
			if sigUnlock.Signature.Type() == iotago.SignatureEd25519 {
				ed25519Sig := sigUnlock.Signature.(*iotago.Ed25519Signature)
				return iotago.EncodeHex(ed25519Sig.PublicKey[:]), nil
			}
		}
	}

	return "", nil
}

func (im *Manager) GetAddressPublicKey(ctx context.Context, client *nodeclient.Client, address string, skipStorage bool, logger *logger.Logger) ([]byte, error) {
	// first get from store
	if !skipStorage {
		addressPublicKeyWrapped, err := im.ReadOnePublicKey(address)
		if err != nil {
			if err != kvstore.ErrKeyNotFound {
				return nil, err
			}
		}
		if addressPublicKeyWrapped != nil {
			return addressPublicKeyWrapped, nil
		}
	}

	// if not found, get from http request
	outputIdHex, err := GetTransactionHistory(ctx, CurrentNetwork, address, logger)
	if err != nil {
		return nil, err
	}
	if outputIdHex == "" {
		return nil, nil
	}
	// log output id
	logger.Infof("GetAddressPublicKey, address:%s, TransactionoutputIdHex:%s", address, outputIdHex)
	outputId, err := iotago.OutputIDFromHex(outputIdHex)
	if err != nil {
		return nil, err
	}
	return im.GetAddressPublicKeyFromOutputId(ctx, client, outputId, address, logger)
}

type OutputIdHexAndAddressPair struct {
	OutputIdHex string
	Address     string
}

func (im *Manager) GetAddressPublicKeyFromOutputId(ctx context.Context, client *nodeclient.Client, outputId iotago.OutputID, address string, logger *logger.Logger) ([]byte, error) {
	metaResponse, err := client.OutputMetadataByID(ctx, outputId)
	if err != nil {
		return nil, err
	}
	transactionId := metaResponse.TransactionID
	// log transaction id
	logger.Infof("GetAddressPublicKey, address:%s, transactionId:%s", address, transactionId)
	publicKey, err := GetPublicKeyViaTransactionId(ctx, client, transactionId)
	if err != nil {
		return nil, err
	}
	if publicKey == "" {
		return nil, nil
	}
	publicKeyBytes, err := iotago.DecodeHex(publicKey)
	if err != nil {
		return nil, err
	}
	// store to db
	err = im.StoreOnePublickKey(address, publicKeyBytes)
	if err != nil {
		return nil, err
	}
	return publicKeyBytes, nil

}
