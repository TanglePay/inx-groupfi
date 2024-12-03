package im

import (
	"bytes"
	"fmt"

	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
)

// Constants
var (
	GroupFITagPrefix          = "GROUPFI" // Tag prefix to identify GroupFI outputs
	GroupFITagPrefixBytes     = []byte(GroupFITagPrefix)
	GROUPFICASHTagPrefix      = "GROUPFICASH"
	GROUPFICASHTagPrefixBytes = []byte(GROUPFICASHTagPrefix)
)

// GetGroupFIKey generates the storage key for a GroupFI output.
// Key = prefix + outputIdBytes
func GetGroupFIKey(outputID [OutputIdLen]byte) []byte {
	return append([]byte{ImStoreKeyPrefixGroupFI}, outputID[:]...)
}

// StoreGroupFIOutput stores the GroupFI output in the KV store.
// It marshals the iotago.Output to JSON and stores it under the key prefix + outputId.
func StoreGroupFIOutput(output iotago.Output, outputID [OutputIdLen]byte, milestoneTimestamp uint32, im *Manager) error {
	address, err := GetAddressFromOutput(output, im)
	if err != nil {
		return fmt.Errorf("failed to get address from output: %w", err)
	}
	addressSha256Hash := Sha256HashFixedAddress(address)
	shouldLog := address == "0x0d1d6b852baf39b45790de7a222fd7f51cd0da51"
	if shouldLog {
		Logger.Infof("StoreGroupFIOutput ... address:%s, outputID:%s", address, iotago.EncodeHex(outputID[:]))
	}
	// Marshal the iotago.Output to JSON
	valueBytes, err := output.MarshalJSON()
	if err != nil {
		// log
		Logger.Infof("StoreGroupFIOutput MarshalJSON err %v", err)
		return fmt.Errorf("failed to marshal iotago.Output to JSON: %w", err)
	}
	outputType := output.Type()
	// Generate the storage key
	key := GetGroupFIKey(outputID)
	timestampBytes := Uint32ToBytes(milestoneTimestamp)
	valueBytes = append(valueBytes, timestampBytes...)
	valueBytes = append(valueBytes, byte(outputType))
	// Store in KV store
	err = im.imStore.Set(key, valueBytes)
	if err != nil {
		// log
		Logger.Infof("StoreGroupFIOutput Set err %v", err)
		return fmt.Errorf("failed to store GroupFIOutput in KV store: %w", err)
	}
	isCashOutput := FilterGroupFICashOutput(output, outputID, im)
	if shouldLog {
		Logger.Infof("StoreGroupFIOutput isCashOutput %v", isCashOutput)
	}
	if isCashOutput {

		err = StoreGroupFICashOutput(addressSha256Hash, outputID, im, shouldLog)
		if err != nil {
			return fmt.Errorf("failed to store GroupFICashOutput in KV store: %w", err)
		}
	}

	return nil
}

// GetGroupFIOutput retrieves the GroupFI output from the KV store based on outputID.
// It unmarshals the JSON data back into an iotago.Output.
func GetGroupFIOutput(outputID [OutputIdLen]byte, im *Manager) (iotago.Output, uint32, error) {
	key := GetGroupFIKey(outputID)
	value, err := im.imStore.Get(key)
	// log
	//Logger.Infof("GetGroupFIOutput key %s, value %s, err %v", iotago.EncodeHex(key), iotago.EncodeHex(value), err)
	if err != nil {
		if err == kvstore.ErrKeyNotFound {
			return nil, 0, nil
		}
		return nil, 0, fmt.Errorf("failed to get GroupFIOutput from KV store: %w", err)
	}
	// last byte is output type
	outputType := iotago.OutputType(value[len(value)-1])
	value = value[:len(value)-1]
	timestampBytes := value[len(value)-4:]
	milestoneTimestamp := BytesToUint32(timestampBytes)
	value = value[:len(value)-4]
	var output iotago.Output
	if outputType == iotago.OutputBasic {
		output = &iotago.BasicOutput{}
	} else if outputType == iotago.OutputNFT {
		output = &iotago.NFTOutput{}
	}
	err = output.UnmarshalJSON(value)

	if err != nil {
		return nil, 0, fmt.Errorf("failed to unmarshal iotago.Output from JSON: %w", err)
	}

	return output, milestoneTimestamp, nil
}

// DeleteGroupFIOutput deletes a GroupFI output from the KV store based on outputID.
func DeleteGroupFIOutput(outputID [OutputIdLen]byte, output iotago.Output, im *Manager) error {
	key := GetGroupFIKey(outputID)
	if err := im.imStore.Delete(key); err != nil {
		return fmt.Errorf("failed to delete GroupFIOutput from KV store: %w", err)
	}
	isCashOutput := FilterGroupFICashOutput(output, outputID, im)
	if isCashOutput {
		address, err := GetAddressFromOutput(output, im)
		if err != nil {
			return fmt.Errorf("failed to get address from output: %w", err)
		}
		addressSha256Hash := Sha256HashFixedAddress(address)
		shouldLog := address == "0x0d1d6b852baf39b45790de7a222fd7f51cd0da51"
		err = DeleteGroupFICashOutput(addressSha256Hash, outputID, im, shouldLog)
		if err != nil {
			return fmt.Errorf("failed to store GroupFICashOutput in KV store: %w", err)
		}
	}
	return nil
}

func FilterGroupFIOutput(output iotago.Output, outputID [OutputIdLen]byte, im *Manager) (iotago.Output, bool) {
	tagFeature := output.FeatureSet().TagFeature()
	if tagFeature == nil || !bytes.HasPrefix(tagFeature.Tag, GroupFITagPrefixBytes) {
		return nil, false
	}

	return output, true
}

// FilterGroupFICashOutput checks if the output is a GroupFICash output
func FilterGroupFICashOutput(output iotago.Output, outputID [OutputIdLen]byte, im *Manager) bool {
	tagFeature := output.FeatureSet().TagFeature()
	if tagFeature == nil || !bytes.HasPrefix(tagFeature.Tag, GROUPFICASHTagPrefixBytes) {
		return false
	}

	return true
}
