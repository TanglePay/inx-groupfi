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

const erc20ABI = `[{"constant":true,"inputs":[],"name":"totalSupply","outputs":[{"name":"","type":"uint256"}],"payable":false,"stateMutability":"view","type":"function"}]`

var (
	clientCacheInstance *ClientCache
	once2               sync.Once
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
	once2.Do(func() {
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

// GetTotalSupply retrieves the total supply of an ERC-20 token
func GetTotalSupply(client *ethclient.Client, contractAddress string) (*big.Int, error) {
	// The address of the ERC-20 token contract
	address := common.HexToAddress(contractAddress)

	// Parse the contract ABI
	parsedABI, err := abi.JSON(strings.NewReader(erc20ABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	// Create a call message to invoke the totalSupply function
	callMsg := ethereum.CallMsg{
		To:   &address,
		Data: parsedABI.Methods["totalSupply"].ID,
	}

	// Call the contract function
	result, err := client.CallContract(context.Background(), callMsg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to call contract: %w", err)
	}

	// Unpack the result
	totalSupply := new(big.Int)
	err = parsedABI.UnpackIntoInterface(totalSupply, "totalSupply", result)
	if err != nil {
		return nil, fmt.Errorf("failed to unpack result: %w", err)
	}

	return totalSupply, nil
}

// GetSupply retrieves the total supply of an ERC-20 token given the client URL and contract address
func GetSupply(clientURL string, contractAddress string) (*big.Int, error) {
	clientCache := GetClientCache()
	client, err := clientCache.GetClient(clientURL)
	if err != nil {
		return nil, err
	}
	return GetTotalSupply(client, contractAddress)
}
