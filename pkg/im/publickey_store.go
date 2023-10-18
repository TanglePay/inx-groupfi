package im

import (
	"context"
	"fmt"

	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/iotaledger/iota.go/v3/nodeclient"
)

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

var shimmerMainNet = IotaNodeInfo{
	ID:                 102,
	IsFaucetAvailable:  false,
	ApiUrl:             "https://mainnet.shimmer.node.tanglepay.com",
	ExplorerApiUrl:     "https://explorer-api.shimmer.network/stardust",
	ExplorerApiNetwork: "shimmer",
	NetworkID:          "14364762045254553490",
	InxMqttEndpoint:    "wss://test.api.iotacat.com/api/iotacatmqtt/v1",
}

func GetTransactionHistory(ctx context.Context, node IotaNodeInfo, bech32Address string) (string, error) {
	url := fmt.Sprintf("%s/transactionhistory/%s/%s?pageSize=1000&sort=newest", node.ExplorerApiUrl, node.ExplorerApiNetwork, bech32Address)
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

func GetPublicKeyViaTransactionId(ctx context.Context, node IotaNodeInfo, transactionId string) (string, error) {
	url := fmt.Sprintf("%s/transaction/%s/%s", node.ExplorerApiUrl, node.ExplorerApiNetwork, transactionId)
	params := map[string]string{} // No additional parameters in the original function

	var resp TransactionResponse
	if err := PerformGetRequest(ctx, url, params, &resp); err != nil {
		return "", err
	}

	for _, unlock := range resp.Block.Payload.Unlocks {
		if unlock.Type == 0 {
			return unlock.Signature.PublicKey, nil
		}
	}

	return "", nil
}

func (im *Manager) GetAddressPublicKey(ctx context.Context, client *nodeclient.Client, address string) ([]byte, error) {
	// first get from store
	addressPublicKeyWrapped, err := im.ReadOnePublicKey(address)
	if err != nil {
		if err != kvstore.ErrKeyNotFound {
			return nil, err
		}
	}
	if addressPublicKeyWrapped != nil {
		return addressPublicKeyWrapped, nil
	}
	// if not found, get from http request
	outputIdHex, err := GetTransactionHistory(ctx, shimmerMainNet, address)
	if err != nil {
		return nil, err
	}
	if outputIdHex == "" {
		return nil, nil
	}
	outputId, err := iotago.OutputIDFromHex(outputIdHex)
	if err != nil {
		return nil, err
	}
	metaResponse, err := client.OutputMetadataByID(ctx, outputId)
	if err != nil {
		return nil, err
	}
	transactionId := metaResponse.TransactionID
	publicKey, err := GetPublicKeyViaTransactionId(ctx, shimmerMainNet, transactionId)
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
