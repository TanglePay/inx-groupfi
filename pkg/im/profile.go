package im

import (
	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

type Profile struct {
	Address   string
	JsonData  string
	OutputId  []byte // store entire outputId
	Timestamp uint32 // Timestamp is not stored anymore
}

// new profile
func NewProfile(address string, jsonData string, outputId []byte) *Profile {
	timestamp := GetCurrentEpochTimestamp()

	return &Profile{
		Address:   address,
		JsonData:  jsonData,
		OutputId:  outputId,
		Timestamp: timestamp, // Timestamp will be kept for in-memory usage, but not stored in DB
	}
}

// key = prefix + addressHash
func (im *Manager) ProfileKey(profile *Profile) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixProfile}, false)
	// addressHash
	addressHash := Sha256HashAddress(profile.Address)
	AppendBytesWithUint16Len(&bytes, &idx, addressHash, false)
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
	err := im.imStore.Set(key, value)
	if err != nil {
		return err
	}
	return GenAndPushProfileChangedEvent(profile, im, Logger)
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

func (im *Manager) GetProfileFromAddress(address string) (*Profile, error) {
	// Create profile key using the address
	profileKey := im.ProfileKey(&Profile{
		Address: address,
	})

	// Fetch the profile value from the store using the profileKey
	value, err := im.imStore.Get(profileKey)
	if err != nil {
		return nil, err
	}

	// Parse the profile from the value
	profile, err := im.ParseProfileValue(profileKey, value)
	if err != nil {
		return nil, err
	}

	// Return the profile
	return profile, nil
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

var profileTagRawStr = "GROUPFIPROFILEV1"
var profileTag = []byte(profileTagRawStr)
var ProfileTagStr = iotago.EncodeHex(profileTag)

// filter out profile output from output
func (im *Manager) FilterProfileOutput(output iotago.Output, outputId iotago.OutputID, logger *logger.Logger) (*Profile, error) {
	output, is := im.FilterOutputByTag(output, profileTag, logger)
	if !is {
		return nil, nil
	}
	profile, err := im.FilterOutputForProfile(output, outputId)
	if err != nil {
		return nil, err
	}
	return profile, nil
}

// filter out profile output from LedgerOutput
func (im *Manager) FilterProfileOutputFromLedgerOutput(output *inx.LedgerOutput, logger *logger.Logger) (*Profile, error) {
	iotaOutput, err := output.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	outputId := output.UnwrapOutputID()
	return im.FilterProfileOutput(iotaOutput, outputId, logger)
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
	// to evm address
	evmAddress := im.ConvertAddressToActualAddress(bech32Address)
	// create profile
	profile := NewProfile(evmAddress, string(jsonData), outputId[:])
	return profile, nil
}
