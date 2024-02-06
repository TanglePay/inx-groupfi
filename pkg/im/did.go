package im

import (
	"encoding/json"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
)

// struct for Did
type Did struct {
	Bech32Address      string
	Name               string
	Picture            string
	OutputIdSha256Hash [Sha256HashLen]byte
	Timestamp          uint32
}

// new did
func NewDid(bech32Address string, name string, picture string, outputId []byte) *Did {
	timestamp := GetCurrentEpochTimestamp()
	outputIdSha256Hash := Sha256HashBytes(outputId)
	outputIdSha256HashFixed := [Sha256HashLen]byte{}
	copy(outputIdSha256HashFixed[:], outputIdSha256Hash)

	return &Did{
		Bech32Address:      bech32Address,
		Name:               name,
		Picture:            picture,
		OutputIdSha256Hash: outputIdSha256HashFixed,
		Timestamp:          timestamp,
	}
}

// key = prefix + addressHash + outputIdSha256Hash
func (im *Manager) DidKey(did *Did) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixDid}, false)
	// addressHash
	addresHash := Sha256Hash(did.Bech32Address)
	AppendBytesWithUint16Len(&bytes, &idx, addresHash, false)
	// outputIdSha256Hash
	AppendBytesWithUint16Len(&bytes, &idx, did.OutputIdSha256Hash[:], false)
	return bytes
}

// prefix for address hash
func (im *Manager) DidPrefixFromAddressHash(addressHash []byte) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixDid}, false)
	// addressHash
	AppendBytesWithUint16Len(&bytes, &idx, addressHash, false)
	return bytes
}

// value = name + picture + timestamp
func (im *Manager) DidValue(did *Did) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(did.Name), true)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(did.Picture), true)
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(did.Timestamp), false)
	return bytes
}

// parse key and value to Did
func (im *Manager) ParseDidValue(key kvstore.Key, value kvstore.Value) (*Did, error) {
	idx := 0
	name, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	picture, err := ReadBytesWithUint16Len(value, &idx)
	if err != nil {
		return nil, err
	}
	timestampBytes, err := ReadBytesWithUint16Len(value, &idx, 4)
	timestamp := BytesToUint32(timestampBytes)
	return &Did{
		Name:      string(name),
		Picture:   string(picture),
		Timestamp: timestamp,
	}, nil
}

// store one did
func (im *Manager) StoreDid(did *Did) error {
	key := im.DidKey(did)
	value := im.DidValue(did)
	return im.imStore.Set(key, value)
}

// delete one did
func (im *Manager) DeleteDid(did *Did) error {
	key := im.DidKey(did)
	return im.imStore.Delete(key)
}

// get all dids from address
func (im *Manager) GetDidsFromAddress(bech32Address string) ([]*Did, error) {
	addressHash := Sha256Hash(bech32Address)
	prefix := im.DidPrefixFromAddressHash(addressHash)
	dids := make([]*Did, 0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		did, err := im.ParseDidValue(key, value)
		if err != nil {
			return false
		}
		dids = append(dids, did)
		return true
	})
	return dids, err
}

// filter ledger output for did
func (im *Manager) FilterLedgerOutputForDid(inxOutput *inx.LedgerOutput) (*Did, error) {
	// check nil
	if inxOutput == nil {
		return nil, nil
	}
	// get output
	output, err := inxOutput.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	outputId := inxOutput.UnwrapOutputID()
	// filter output for did
	return im.FilterOutputForDid(output, outputId)
}

// filter output for did
func (im *Manager) FilterOutputForDid(output iotago.Output, outputId iotago.OutputID) (*Did, error) {
	// check nil
	if output == nil {
		return nil, nil
	}
	// check nft output
	nftOutput, ok := output.(*iotago.NFTOutput)
	if !ok {
		return nil, nil
	}
	// filter nft output for did
	return im.FilterNftOutputForDid(nftOutput, outputId)
}

// filter nft output for did
// nft with meta['property'] == 'groupfi-name' will be considered as did, it should have picture and name in metadata
func (im *Manager) FilterNftOutputForDid(output *iotago.NFTOutput, outputId iotago.OutputID) (*Did, error) {
	// check nil
	if output == nil {
		return nil, nil
	}
	// get metadata
	if output.ImmutableFeatureSet().MetadataFeature() == nil || output.ImmutableFeatureSet().MetadataFeature().Data == nil {
		return nil, nil
	}
	// unmarshal metadata as json, using go library
	metaMap := make(map[string]interface{})
	err := json.Unmarshal(output.ImmutableFeatureSet().MetadataFeature().Data, &metaMap)
	if err != nil {
		return nil, err
	}
	var nameStr string
	if metaMap["name"] != nil {
		nameStr = metaMap["name"].(string)
	}
	var pictureStr string
	if metaMap["picture"] != nil {
		pictureStr = metaMap["picture"].(string)
	}
	// check nil
	if nameStr == "" || pictureStr == "" {
		return nil, nil
	}
	// check if property is groupfi-name
	propertyStr := metaMap["property"].(string)
	if propertyStr != "groupfi-name" {
		return nil, nil
	}
	// get bech32 address from unlock conditions
	if output.UnlockConditionSet() == nil {
		return nil, nil
	}
	address := output.UnlockConditionSet().Address()
	if address == nil {
		return nil, nil
	}
	bech32Address := address.Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	// create did
	return NewDid(bech32Address, nameStr, pictureStr, outputId[:]), nil
}

// handle consumed and created did
func (im *Manager) HandleDidConsumedAndCreated(consumedDids []*Did, createdDids []*Did, logger *logger.Logger) {
	// delete consumed dids
	for _, did := range consumedDids {
		err := im.DeleteDid(did)
		if err != nil {
			// log error
			logger.Warnf("HandleDidConsumedAndCreated, DeleteDid error:%s", err.Error())
		}
	}
	// store created dids
	for _, did := range createdDids {
		err := im.StoreDid(did)
		if err != nil {
			// log error
			logger.Warnf("HandleDidConsumedAndCreated, StoreDid error:%s", err.Error())
		}
	}

}
