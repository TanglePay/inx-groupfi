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
	once                sync.Once
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
	once.Do(func() {
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
	// The address of the ERC-20 token contract
	address := common.HexToAddress(contractAddress)

	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	// Call the totalSupply function
	callMsg := ethereum.CallMsg{
		To:   &address,
		Data: parsedABI.Methods["totalSupply"].ID,
	}
	result, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call totalSupply: %w", err)
	}
	totalSupply := new(big.Int)
	err = parsedABI.UnpackIntoInterface(totalSupply, "totalSupply", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack totalSupply result: %w", err)
	}

	// Call the decimals function
	callMsg = ethereum.CallMsg{
		To:   &address,
		Data: parsedABI.Methods["decimals"].ID,
	}
	result, err = client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call decimals: %w", err)
	}
	var decimals uint8
	err = parsedABI.UnpackIntoInterface(&decimals, "decimals", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack decimals result: %w", err)
	}

	return &TokenInfo{
		TotalSupply: totalSupply,
		Decimals:    decimals,
	}, nil
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
