package im

import (
	"context"
	"fmt"
	"math/big"
	"strings"
	"sync"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
)

const erc20ABI = `[
    {"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"},
    {"constant":true,"inputs":[],"name":"decimals","outputs":[{"name":"","type":"uint8"}],"payable":false,"stateMutability":"view","type":"function"}
]`

var (
	clientCacheInstance *ClientCache
	onceEvmTokenMeta    sync.Once
)

// ClientCache stores Ethereum clients mapped by their endpoint URL
type ClientCache struct {
	clients map[string]*ethclient.Client
	mu      sync.Mutex
}

// NewClientCache creates a new ClientCache
func NewClientCache() *ClientCache {
	return &ClientCache{
		clients: make(map[string]*ethclient.Client),
	}
}

// GetClientCache initializes and returns the singleton instance of ClientCache
func GetClientCache() *ClientCache {
	onceEvmTokenMeta.Do(func() {
		clientCacheInstance = NewClientCache()
	})
	return clientCacheInstance
}

// GetClient retrieves or creates an Ethereum client for the given endpoint
func (c *ClientCache) GetClient(endpoint string) (*ethclient.Client, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if client, exists := c.clients[endpoint]; exists {
		return client, nil
	}

	client, err := ethclient.Dial(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to the Ethereum client: %w", err)
	}

	c.clients[endpoint] = client
	return client, nil
}

// TokenInfo holds the total supply and decimals of an ERC-20 token
type TokenInfo struct {
	TotalSupply *big.Int `json:"totalSupply"`
	Decimals    uint8    `json:"decimals"`
}

// GetTotalSupplyAndDecimals retrieves the total supply and decimals of an ERC-20 token
func GetTotalSupplyAndDecimals(client *ethclient.Client, contractAddress string) (*TokenInfo, error) {
	address := common.HexToAddress(contractAddress)
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	// Call the totalSupply function
	totalSupply, err := callBigIntFunction(client, parsedABI, address, "totalSupply")
	if err != nil {
		return nil, fmt.Errorf("failed to get totalSupply: %w", err)
	}

	// Call the decimals function
	decimals, err := callUint8Function(client, parsedABI, address, "decimals")
	if err != nil {
		return nil, fmt.Errorf("failed to get decimals: %w", err)
	}

	return &TokenInfo{
		TotalSupply: totalSupply,
		Decimals:    decimals,
	}, nil
}

func callBigIntFunction(client *ethclient.Client, parsedABI abi.ABI, address common.Address, methodName string) (*big.Int, error) {
	// Prepare the call data
	data, err := parsedABI.Pack(methodName)
	if err != nil {
		return nil, fmt.Errorf("failed to pack method %s: %w", methodName, err)
	}

	// Call the contract
	callMsg := ethereum.CallMsg{
		To:   &address,
		Data: data,
	}
	result, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call %s: %w", methodName, err)
	}

	// Unpack the result
	var output = new(big.Int)
	err = parsedABI.UnpackIntoInterface(output, methodName, result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack %s result: %w", methodName, err)
	}
	return output, nil
}

func callUint8Function(client *ethclient.Client, parsedABI abi.ABI, address common.Address, methodName string) (uint8, error) {
	// Prepare the call data
	data, err := parsedABI.Pack(methodName)
	if err != nil {
		return 0, fmt.Errorf("failed to pack method %s: %w", methodName, err)
	}

	// Call the contract
	callMsg := ethereum.CallMsg{
		To:   &address,
		Data: data,
	}
	result, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to call %s: %w", methodName, err)
	}

	// Unpack the result
	var output uint8
	err = parsedABI.UnpackIntoInterface(&output, methodName, result)
	if err != nil {
		return 0, fmt.Errorf("failed to unpack %s result: %w", methodName, err)
	}
	return output, nil
}

// GetSupplyAndDecimals retrieves the total supply and decimals of an ERC-20 token given the client URL and contract address
func GetSupplyAndDecimals(clientURL string, contractAddress string) (*TokenInfo, error) {
	clientCache := GetClientCache()
	client, err := clientCache.GetClient(clientURL)
	if err != nil {
		return nil, err
	}
	return GetTotalSupplyAndDecimals(client, contractAddress)
}
