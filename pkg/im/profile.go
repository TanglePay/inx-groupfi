package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
)

// struct for Profile
type Profile struct {
	Bech32Address string
	JsonData      string
	OutputId      []byte // store entire outputId
	Timestamp     uint32 // Timestamp is not stored anymore
}

// new profile
func NewProfile(bech32Address string, jsonData string, outputId []byte) *Profile {
	timestamp := GetCurrentEpochTimestamp()

	return &Profile{
		Bech32Address: bech32Address,
		JsonData:      jsonData,
		OutputId:      outputId,
		Timestamp:     timestamp, // Timestamp will be kept for in-memory usage, but not stored in DB
	}
}

// key = prefix + addressHash + outputId
func (im *Manager) ProfileKey(profile *Profile) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixProfile}, false)
	// addressHash
	addressHash := Sha256HashAddress(profile.Bech32Address)
	AppendBytesWithUint16Len(&bytes, &idx, addressHash, false)
	// outputId
	AppendBytesWithUint16Len(&bytes, &idx, profile.OutputId, false) // store entire outputId
	return bytes
}

// value = jsonData + outputId
func (im *Manager) ProfileValue(profile *Profile) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(profile.JsonData), true)
	// Append the outputId directly into the value (optional, if needed for cross-reference)
	AppendBytesWithUint16Len(&bytes, &idx, profile.OutputId, false)
	return bytes
}

// store one profile without generating and pushing an event
func (im *Manager) StoreProfile(profile *Profile) error {
	key := im.ProfileKey(profile)
	value := im.ProfileValue(profile)
	return im.imStore.Set(key, value)
}

// delete one profile without generating and pushing an event
func (im *Manager) DeleteProfile(profile *Profile) error {
	key := im.ProfileKey(profile)
	return im.imStore.Delete(key)
}

// prefix for address hash
func (im *Manager) ProfilePrefixFromAddressHash(addressHash []byte) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix for profile
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixProfile}, false)
	// addressHash
	AppendBytesWithUint16Len(&bytes, &idx, addressHash, false)
	return bytes
}

// get all profiles from address
func (im *Manager) GetProfilesFromAddress(bech32Address string) ([]*Profile, error) {
	addressHash := Sha256HashAddress(bech32Address)
	prefix := im.ProfilePrefixFromAddressHash(addressHash)
	profiles := make([]*Profile, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		profile, err := im.ParseProfileValue(key, value)
		if err != nil {
			return false
		}
		profiles = append(profiles, profile)
		return true
	})
	return profiles, err
}

// parse key and value to Profile (reads only jsonData and outputId)
func (im *Manager) ParseProfileValue(key kvstore.Key, value kvstore.Value) (*Profile, error) {
	idx := 0
	// Read jsonData
	jsonData, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}

	// Extract outputId from key (assuming outputId is part of the key)
	outputId, err := ReadBytesWithUint16Len(key, &idx, OutputIdLen)
	if err != nil {
		return nil, err
	}

	return &Profile{
		JsonData: string(jsonData),
		OutputId: outputId, // Keep outputId stored
	}, nil
}

// filter output for profile
func (im *Manager) FilterOutputForProfile(output iotago.Output, outputId iotago.OutputID) (*Profile, error) {
	// check nil
	if output == nil {
		return nil, nil
	}
	// check if it's a BasicOutput
	basicOutput, ok := output.(*iotago.BasicOutput)
	if !ok {
		return nil, nil
	}
	// get metadata
	if basicOutput.FeatureSet().MetadataFeature() == nil || basicOutput.FeatureSet().MetadataFeature().Data == nil {
		return nil, nil
	}
	// the metadata should be a JSON string
	jsonData := basicOutput.FeatureSet().MetadataFeature().Data
	// get bech32 address from unlock conditions
	if basicOutput.UnlockConditionSet() == nil {
		return nil, nil
	}
	address := basicOutput.UnlockConditionSet().Address()
	if address == nil {
		return nil, nil
	}
	// convert address to Bech32 format
	bech32Address := address.Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	// create profile
	profile := NewProfile(bech32Address, string(jsonData), outputId[:])
	return profile, nil
}
