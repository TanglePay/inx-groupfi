package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	iotago "github.com/iotaledger/iota.go/v3"
)

// struct for Profile
type Profile struct {
	Bech32Address      string
	JsonData           string
	OutputIdSha256Hash [Sha256HashLen]byte
	Timestamp          uint32
}

// new profile
func NewProfile(bech32Address string, jsonData string, outputId []byte) *Profile {
	timestamp := GetCurrentEpochTimestamp()
	outputIdSha256Hash := Sha256HashBytes(outputId)
	outputIdSha256HashFixed := [Sha256HashLen]byte{}
	copy(outputIdSha256HashFixed[:], outputIdSha256Hash)

	return &Profile{
		Bech32Address:      bech32Address,
		JsonData:           jsonData,
		OutputIdSha256Hash: outputIdSha256HashFixed,
		Timestamp:          timestamp,
	}
}

// key = prefix + addressHash + outputIdSha256Hash
func (im *Manager) ProfileKey(profile *Profile) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixProfile}, false)
	// addressHash
	addressHash := Sha256HashAddress(profile.Bech32Address)
	AppendBytesWithUint16Len(&bytes, &idx, addressHash, false)
	// outputIdSha256Hash
	AppendBytesWithUint16Len(&bytes, &idx, profile.OutputIdSha256Hash[:], false)
	return bytes
}

// value = jsonData + timestamp
func (im *Manager) ProfileValue(profile *Profile) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(profile.JsonData), true)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(profile.Timestamp), false)
	return bytes
}

// store one profile
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

// parse key and value to Profile
func (im *Manager) ParseProfileValue(key kvstore.Key, value kvstore.Value) (*Profile, error) {
	idx := 0
	jsonData, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	timestampBytes, err := ReadBytesWithUint16Len(value, &idx, 4)
	timestamp := BytesToUint32(timestampBytes)
	return &Profile{
		JsonData:  string(jsonData),
		Timestamp: timestamp,
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
