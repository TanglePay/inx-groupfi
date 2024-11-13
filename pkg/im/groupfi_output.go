package im

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
)

// Constants
var (
	GroupFITagPrefix      = "GROUPFI" // Tag prefix to identify GroupFI outputs
	GroupFITagPrefixBytes = []byte(GroupFITagPrefix)
)

// GetGroupFIKey generates the storage key for a GroupFI output.
// Key = prefix + outputIdBytes
func GetGroupFIKey(outputID [OutputIdLen]byte) []byte {
	return append([]byte{ImStoreKeyPrefixGroupFI}, outputID[:]...)
}

// StoreGroupFIOutput stores the GroupFI output in the KV store.
// It marshals the iotago.Output to JSON and stores it under the key prefix + outputId.
func StoreGroupFIOutput(output iotago.Output, outputID [OutputIdLen]byte, im *Manager) error {
	// Marshal the iotago.Output to JSON
	valueBytes, err := output.MarshalJSON()
	if err != nil {
		return fmt.Errorf("failed to marshal iotago.Output to JSON: %w", err)
	}
	outputType := output.Type()
	// Generate the storage key
	key := GetGroupFIKey(outputID)

	valueBytes = append(valueBytes, byte(outputType))
	// Store in KV store
	if err := im.imStore.Set(key, valueBytes); err != nil {
		return fmt.Errorf("failed to store GroupFIOutput in KV store: %w", err)
	}

	return nil
}

// GetGroupFIOutput retrieves the GroupFI output from the KV store based on outputID.
// It unmarshals the JSON data back into an iotago.Output.
func GetGroupFIOutput(outputID [OutputIdLen]byte, im *Manager) (iotago.Output, error) {
	key := GetGroupFIKey(outputID)
	value, err := im.imStore.Get(key)
	if err != nil {
		if err == kvstore.ErrKeyNotFound {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to get GroupFIOutput from KV store: %w", err)
	}
	// last byte is output type
	outputType := iotago.OutputType(value[len(value)-1])
	value = value[:len(value)-1]

	var output iotago.Output
	if outputType == iotago.OutputBasic {
		output = &iotago.BasicOutput{}
	} else if outputType == iotago.OutputNFT {
		output = &iotago.NFTOutput{}
	}
	err = output.UnmarshalJSON(value)

	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal iotago.Output from JSON: %w", err)
	}

	return output, nil
}

// DeleteGroupFIOutput deletes a GroupFI output from the KV store based on outputID.
func DeleteGroupFIOutput(outputID [OutputIdLen]byte, im *Manager) error {
	key := GetGroupFIKey(outputID)
	if err := im.imStore.Delete(key); err != nil {
		return fmt.Errorf("failed to delete GroupFIOutput from KV store: %w", err)
	}
	return nil
}

// FilterGroupFIOutput checks if the output is a GroupFI output and stores it.
// Returns the output, the converted EVM address, and a boolean indicating success.
func FilterGroupFIOutput(output iotago.Output, outputID [OutputIdLen]byte, im *Manager) (iotago.Output, bool) {
	tagFeature := output.FeatureSet().TagFeature()
	if tagFeature == nil || !bytes.HasPrefix(tagFeature.Tag, GroupFITagPrefixBytes) {
		return nil, false
	}

	return output, true
}
