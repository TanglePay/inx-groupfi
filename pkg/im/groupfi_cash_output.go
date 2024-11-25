package im

import "fmt"

// for GroupFI cash output, we just store the output id, under user address hash
// key for GroupFI cash output, prefix + addressSha256Hash + outputID
func GetGroupFICashKey(addressSha256Hash [Sha256HashLen]byte, outputID [OutputIdLen]byte) []byte {
	return ConcatByteSlices([]byte{ImStoreKeyPrefixGroupFICash}, addressSha256Hash[:], outputID[:])
}

// prefix for GroupFI cash output under address
func GetGroupFICashPrefix(addressSha256Hash [Sha256HashLen]byte) []byte {
	return ConcatByteSlices([]byte{ImStoreKeyPrefixGroupFICash}, addressSha256Hash[:])
}

// StoreGroupFICashOutput stores the GroupFI cash output in the KV store.
func StoreGroupFICashOutput(addressSha256Hash [Sha256HashLen]byte, outputID [OutputIdLen]byte, im *Manager) error {
	// Generate the storage key
	key := GetGroupFICashKey(addressSha256Hash, outputID)
	// Store in KV store
	err := im.imStore.Set(key, nil)
	if err != nil {
		return fmt.Errorf("failed to store GroupFICashOutput in KV store: %w", err)
	}
	return nil
}

// delete GroupFI cash output from the KV store.
func DeleteGroupFICashOutput(addressSha256Hash [Sha256HashLen]byte, outputID [OutputIdLen]byte, im *Manager) error {
	// Generate the storage key
	key := GetGroupFICashKey(addressSha256Hash, outputID)
	// Store in KV store
	err := im.imStore.Delete(key)
	if err != nil {
		return fmt.Errorf("failed to delete GroupFICashOutput in KV store: %w", err)
	}
	return nil
}

// get all GroupFI cash output under address
func GetGroupFICashOutputs(addressSha256Hash [Sha256HashLen]byte, im *Manager) ([][OutputIdLen]byte, error) {
	// Generate the storage key
	keyPrefix := GetGroupFICashPrefix(addressSha256Hash)
	// Store in KV store
	outputs := [][OutputIdLen]byte{}
	err := im.imStore.Iterate(keyPrefix, func(key, value []byte) bool {
		outputID := [OutputIdLen]byte{}
		copy(outputID[:], key[len(key)-OutputIdLen:])
		outputs = append(outputs, outputID)
		return true
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get GroupFICashOutputs in KV store: %w", err)
	}
	return outputs, nil
}
