package im

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

// DownloadUriContent downloads content from a given URI and returns it as a string.
// It performs an HTTP GET request and handles response validation.
func DownloadUriContent(uri string) (string, error) {
	// Make the HTTP GET request
	resp, err := http.Get(uri)
	if err != nil {
		return "", fmt.Errorf("failed to fetch content from URI '%s': %v", uri, err)
	}
	defer resp.Body.Close()

	// Check if the response status is OK
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch content from URI '%s': %s", uri, resp.Status)
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read content from URI '%s': %v", uri, err)
	}

	return string(body), nil
}

// DownloadIpfsContent downloads content from an IPFS URI using a specific gateway and returns it as a string.
// It leverages the DownloadUriContent function for the actual content retrieval.
func DownloadIpfsContent(ipfsUri string) (string, error) {
	// Validate the IPFS URI prefix
	const ipfsPrefix = "ipfs://"
	if !strings.HasPrefix(ipfsUri, ipfsPrefix) {
		return "", fmt.Errorf("invalid IPFS URI: must start with '%s'", ipfsPrefix)
	}

	// Convert IPFS URI to the specified Pinata gateway URL
	gatewayBase := "https://amaranth-payable-cow-395.mypinata.cloud/ipfs/"
	gatewayUrl := strings.Replace(ipfsUri, ipfsPrefix, gatewayBase, 1)

	// Use DownloadUriContent to fetch the content from the gateway URL
	content, err := DownloadUriContent(gatewayUrl)
	if err != nil {
		return "", fmt.Errorf("failed to download IPFS content: %v", err)
	}

	return content, nil
}
