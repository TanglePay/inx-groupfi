package im

import (
	"io"
	"log"

	shell "github.com/ipfs/go-ipfs-api"
)

var sh *shell.Shell

// init shell with context
func InitIpfsShell() {
	sh = shell.NewShell("localhost:5001")
}
func ReadIpfsFile(ipfsLink string) ([]byte, error) {

	// Fetch data from IPFS
	reader, err := sh.Cat(ipfsLink)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	// Read the content
	content, err := io.ReadAll(reader)
	if err != nil {
		log.Fatalf("Failed to read data: %v", err)
	}
	return content, nil
}
