package im

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func DownloadIpfsContent(ipfsUri string) (string, error) {
	// Convert IPFS URI to ipfs.io gateway URL
	gatewayUrl := strings.Replace(ipfsUri, "ipfs://", "https://ipfs.io/ipfs/", 1)

	// Make the HTTP GET request
	resp, err := http.Get(gatewayUrl)
	if err != nil {
		return "", fmt.Errorf("failed to fetch IPFS content: %v", err)
	}
	defer resp.Body.Close()

	// Check if the response is okay
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to fetch IPFS content: %s", resp.Status)
	}

	// Read the content as a string
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read IPFS content: %v", err)
	}

	return string(body), nil
}
