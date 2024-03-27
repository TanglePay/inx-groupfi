package im

import (
	"bytes"
	"encoding/json"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	"github.com/iotaledger/hive.go/serializer/v2"
	inx "github.com/iotaledger/inx/go"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/pkg/errors"
)

var pairXTagRawStr = "GROUPFIPAIRXV1"
var pairXTag = []byte(pairXTagRawStr)
var PairXTagStr = iotago.EncodeHex(pairXTag)

// struct for PairX
type PairX struct {
	EvmAddress   string
	PublicKey    string
	PrivateKey   string
	Signature    string
	ProxyAddress string
	Scenery      int // 1 for mm 2 for tp
}

// new PairX
func NewPairX(evmAddress string, publicKey string, privateKey string, signature string, scenery int, proxyAddress string) *PairX {
	return &PairX{
		EvmAddress:   evmAddress,
		PublicKey:    publicKey,
		PrivateKey:   privateKey,
		Signature:    signature,
		Scenery:      scenery,
		ProxyAddress: proxyAddress,
	}
}

// keyForData = prefix + evmAddressSha256Hash
func (im *Manager) PairXKey(pairX *PairX) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixPairXData}, false)
	// evmAddressSha256Hash
	evmAddressSha256Hash := Sha256Hash(pairX.EvmAddress)
	AppendBytesWithUint16Len(&bytes, &idx, evmAddressSha256Hash, false)
	return bytes
}

// valueForData = publicKey + privateKey + evm address
func (im *Manager) PairXValue(pairX *PairX) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(pairX.PublicKey), true)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(pairX.PrivateKey), true)
	AppendBytesWithUint16Len(&bytes, &idx, []byte(pairX.EvmAddress), true)
	return bytes
}

// struct from key and value for Data
func (im *Manager) PairXFromKeyAndValue(key kvstore.Key, value kvstore.Value) *PairX {
	idx := 0
	// publicKey
	publicKey, _ := ReadBytesWithUint16Len(value, &idx)
	// privateKey
	privateKey, _ := ReadBytesWithUint16Len(value, &idx)
	// evm address
	evmAddress, _ := ReadBytesWithUint16Len(value, &idx)
	return NewPairX(string(evmAddress), string(publicKey), string(privateKey), "", 0, "")
}

// keyForPairXEvmAddressSceneryProxyAddress = prefix + evmAddressSha256Hash + scenery
func (im *Manager) PairXEvmAddressSceneryProxyAddressKey(pairX *PairX) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixPairXData}, false)
	// evmAddressSha256Hash
	evmAddressSha256Hash := Sha256Hash(pairX.EvmAddress)
	AppendBytesWithUint16Len(&bytes, &idx, evmAddressSha256Hash, false)
	// scenery
	AppendBytesWithUint16Len(&bytes, &idx, Uint32ToBytes(uint32(pairX.Scenery)), false)
	return bytes
}

// valueForPairXEvmAddressSceneryProxyAddress = proxyAddress
func (im *Manager) PairXEvmAddressSceneryProxyAddressValue(pairX *PairX) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(pairX.ProxyAddress), true)
	return bytes
}

// struct from key and value for PairXEvmAddressSceneryProxyAddress
func (im *Manager) PairXFromKeyAndValueForPairXEvmAddressSceneryProxyAddress(key kvstore.Key, value kvstore.Value) *PairX {
	idx := 0
	// prefix
	_, _ = ReadBytesWithUint16Len(key, &idx, 1)
	// address hash
	_, _ = ReadBytesWithUint16Len(key, &idx, Sha256HashLen)
	// scenery
	sceneryBytes, _ := ReadBytesWithUint16Len(key, &idx, 4)
	scenery := BytesToUint32(sceneryBytes)
	idx = 0
	// proxyAddress
	proxyAddress, _ := ReadBytesWithUint16Len(value, &idx)
	return NewPairX("", "", "", "", int(scenery), string(proxyAddress))
}

// keyForPairXProxyAddressEvmAddress = prefix + proxyAddressSha256Hash
func (im *Manager) PairXProxyAddressEvmAddressKey(pairX *PairX) []byte {
	bytes := make([]byte, 0)
	idx := 0
	// prefix
	AppendBytesWithUint16Len(&bytes, &idx, []byte{ImStoreKeyPrefixPairXProxyAddressEvmAddress}, false)
	// proxyAddressSha256Hash
	proxyAddressSha256Hash := Sha256Hash(pairX.ProxyAddress)
	AppendBytesWithUint16Len(&bytes, &idx, proxyAddressSha256Hash, false)
	return bytes
}

// valueForPairXProxyAddressEvmAddress = evmAddress
func (im *Manager) PairXProxyAddressEvmAddressValue(pairX *PairX) []byte {
	bytes := make([]byte, 0)
	idx := 0
	AppendBytesWithUint16Len(&bytes, &idx, []byte(pairX.EvmAddress), true)
	return bytes
}

// struct from key and value for PairXProxyAddressEvmAddress
func (im *Manager) PairXFromKeyAndValueForPairXProxyAddressEvmAddress(key kvstore.Key, value kvstore.Value) *PairX {
	idx := 0
	// evmAddress
	evmAddress, _ := ReadBytesWithUint16Len(value, &idx)
	return NewPairX(string(evmAddress), "", "", "", 0, "")
}

// store one PairX
func (im *Manager) StorePairX(pairX *PairX) error {
	// data
	keyForData := im.PairXKey(pairX)
	valueForData := im.PairXValue(pairX)
	if err := im.imStore.Set(keyForData, valueForData); err != nil {
		return err
	}
	// scenery proxy address
	keyForPairXEvmAddressSceneryProxyAddress := im.PairXEvmAddressSceneryProxyAddressKey(pairX)
	valueForPairXEvmAddressSceneryProxyAddress := im.PairXEvmAddressSceneryProxyAddressValue(pairX)
	if err := im.imStore.Set(keyForPairXEvmAddressSceneryProxyAddress, valueForPairXEvmAddressSceneryProxyAddress); err != nil {
		return err
	}
	// proxy address evm address
	keyForPairXProxyAddressEvmAddress := im.PairXProxyAddressEvmAddressKey(pairX)
	valueForPairXProxyAddressEvmAddress := im.PairXProxyAddressEvmAddressValue(pairX)
	if err := im.imStore.Set(keyForPairXProxyAddressEvmAddress, valueForPairXProxyAddressEvmAddress); err != nil {
		return err
	}
	return nil
}

// get data from evm address
func (im *Manager) GetPairXFromEvmAddress(evmAddress string) (*PairX, error) {
	key := im.PairXKey(NewPairX(evmAddress, "", "", "", 0, ""))
	value, err := im.imStore.Get(key)
	if err != nil {
		return nil, err
	}
	return im.PairXFromKeyAndValue(key, value), nil
}

// get proxy address from evm address for both mm and tp
func (im *Manager) GetPairXProxyAddressFromEvmAddress(evmAddress string) (string, string, error) {
	// pairX from evm address
	pairXMM := NewPairX(evmAddress, "", "", "", 1, "")

	// mm
	var mmPairX *PairX
	pairXEvmAddressSceneryProxyAddressKey := im.PairXEvmAddressSceneryProxyAddressKey(pairXMM)
	pairXEvmAddressSceneryProxyAddressValue, err := im.imStore.Get(pairXEvmAddressSceneryProxyAddressKey)
	if errors.Is(err, kvstore.ErrKeyNotFound) {

	} else if err != nil {
		return "", "", err
	} else {
		mmPairX = im.PairXFromKeyAndValueForPairXEvmAddressSceneryProxyAddress(pairXEvmAddressSceneryProxyAddressKey, pairXEvmAddressSceneryProxyAddressValue)

	}
	// tp
	var tpPairX *PairX
	pairXTP := NewPairX(evmAddress, "", "", "", 2, "")
	pairXEvmAddressSceneryProxyAddressKey = im.PairXEvmAddressSceneryProxyAddressKey(pairXTP)
	pairXEvmAddressSceneryProxyAddressValue, err = im.imStore.Get(pairXEvmAddressSceneryProxyAddressKey)
	if errors.Is(err, kvstore.ErrKeyNotFound) {

	} else if err != nil {
		return "", "", err
	} else {
		tpPairX = im.PairXFromKeyAndValueForPairXEvmAddressSceneryProxyAddress(pairXEvmAddressSceneryProxyAddressKey, pairXEvmAddressSceneryProxyAddressValue)
	}
	mmProxyAddress := ""
	tpProxyAddress := ""
	if mmPairX != nil {
		mmProxyAddress = mmPairX.ProxyAddress
	}
	if tpPairX != nil {
		tpProxyAddress = tpPairX.ProxyAddress
	}
	return mmProxyAddress, tpProxyAddress, nil
}

// get evm address from proxy address
func (im *Manager) GetPairXEvmAddressFromProxyAddress(proxyAddress string) (string, error) {
	pairX := NewPairX("", "", "", "", 0, proxyAddress)
	key := im.PairXProxyAddressEvmAddressKey(pairX)
	value, err := im.imStore.Get(key)
	if errors.Is(err, kvstore.ErrKeyNotFound) {
		return "", nil
	} else if err != nil {
		return "", err
	}
	return im.PairXFromKeyAndValueForPairXProxyAddressEvmAddress(key, value).EvmAddress, nil
}

// filter pairX from LedgerOutput
func (im *Manager) FilterPairXFromLedgerOutput(inxOutput *inx.LedgerOutput) (*PairX, error) {
	if inxOutput == nil {
		return nil, nil
	}
	output, err := inxOutput.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	outputID := inxOutput.UnwrapOutputID()
	return im.FilterPairXFromOutput(output, outputID)
}

// filter pairX from output
func (im *Manager) FilterPairXFromOutput(output iotago.Output, outputID iotago.OutputID) (*PairX, error) {
	if output == nil {
		return nil, nil
	}
	nftOutput, ok := output.(*iotago.NFTOutput)
	if !ok {
		return nil, nil
	}
	return im.FilterPairXFromNFTOutput(nftOutput, outputID)
}

// filter pairX from nftOutput
func (im *Manager) FilterPairXFromNFTOutput(output *iotago.NFTOutput, outputID iotago.OutputID) (*PairX, error) {
	if output == nil {
		return nil, nil
	}
	// get tag
	if output.ImmutableFeatureSet().TagFeature() == nil ||
		output.ImmutableFeatureSet().TagFeature().Tag == nil ||
		!bytes.Equal(output.ImmutableFeatureSet().TagFeature().Tag, pairXTag) {
		return nil, nil
	}
	// get metadata
	if output.ImmutableFeatureSet().MetadataFeature() == nil || output.ImmutableFeatureSet().MetadataFeature().Data == nil {
		return nil, nil
	}
	/*
			{
		  "encryptedPrivateKey": "0x7b2276657273696f6e223a227832353531392d7873616c736132302d706f6c7931333035222c226e6f6e6365223a226148496b55726e724a68534c374b445a655057415a707658554c6442495a4344222c22657068656d5075626c69634b6579223a226d4c566e6f2b4943326776417641342f766330516639656638362f65394c5a6377562f34576b56565a6d513d222c2263697068657274657874223a224f477236757734756c6c66465a74712b762b65707a302b4f4c4738714879324e754347445959593078726f6b7464304d763251326646497678527136505041397573353666304e2f4169446a4e6172775768416764486e4d546f312b65796b39536b712b633841635665633d227d",
		  "pairXPublicKey": "0x9b3da30c3aa890958b95e96b65a5e0f77a28cb1211d097ab943ef03d9dab9651",
		  "evmAddress": "0x928100571464c900A2F53689353770455D78a200",
		  "timestamp": 1711449778,
		  "scenery": 1,
		  "signature": "0xccec1e146ff48198566e706d548536c4cc3e6afa3ac351c740fb9f951912b90f1fb064f33682ac12f9e9fad446e3a9dc7ce53dd81c36729fa41cf946f4d1138c1b"
		}*/
	// unmarshal metadata as json, using go library
	metaMap := make(map[string]interface{})
	err := json.Unmarshal(output.ImmutableFeatureSet().MetadataFeature().Data, &metaMap)
	if err != nil {
		return nil, err
	}
	// get each field of pairX, check nil then get from metaMap
	evmAddress, ok := metaMap["evmAddress"].(string)
	if !ok {
		return nil, nil
	}
	publicKey, ok := metaMap["pairXPublicKey"].(string)
	if !ok {
		return nil, nil
	}
	privateKey, ok := metaMap["encryptedPrivateKey"].(string)
	if !ok {
		return nil, nil
	}
	signature, ok := metaMap["signature"].(string)
	if !ok {
		return nil, nil
	}
	scenery, ok := metaMap["scenery"].(float64)
	if !ok {
		return nil, nil
	}
	// get proxy address from unlock condition
	unlockConditionSet := output.UnlockConditionSet()
	if unlockConditionSet == nil {
		return nil, nil
	}
	proxyAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	return NewPairX(evmAddress, publicKey, privateKey, signature, int(scenery), proxyAddress), nil
}

// handle pairX created
func (im *Manager) HandlePairXCreated(pairx *PairX, logger *logger.Logger) {
	//TODO validate signature
	if err := im.StorePairX(pairx); err != nil {
		logger.Warnf("HandlePairXCreated ... StorePairX failed:%s", err)
	}
}
