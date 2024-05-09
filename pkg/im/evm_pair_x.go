package im

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/ethereum/go-ethereum/crypto"
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
	Scenery      int // 1 for tp 2 for mm
	Timestamp    int
}

const TPScenery = 1
const MMScenery = 2

// new PairX
func NewPairX(evmAddress string, publicKey string, privateKey string, signature string, scenery int, proxyAddress string, timestamp int) *PairX {
	return &PairX{
		EvmAddress:   evmAddress,
		PublicKey:    publicKey,
		PrivateKey:   privateKey,
		Signature:    signature,
		Scenery:      scenery,
		ProxyAddress: proxyAddress,
		Timestamp:    timestamp,
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
	return NewPairX(string(evmAddress), string(publicKey), string(privateKey), "", 0, "", 0)
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
	return NewPairX("", "", "", "", int(scenery), string(proxyAddress), 0)
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
	return NewPairX(string(evmAddress), "", "", "", 0, "", 0)
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
	// store evm address publickey
	publicKeyBytes, err := iotago.DecodeHex(pairX.PublicKey)
	if err != nil {
		return err
	}
	im.StoreOnePublickKey(pairX.EvmAddress, publicKeyBytes)
	return nil
}

// get data from evm address
func (im *Manager) GetPairXFromEvmAddress(evmAddress string) (*PairX, error) {
	key := im.PairXKey(NewPairX(evmAddress, "", "", "", 0, "", 0))
	value, err := im.imStore.Get(key)
	if err != nil {
		// return nil for key not found
		if errors.Is(err, kvstore.ErrKeyNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return im.PairXFromKeyAndValue(key, value), nil
}

// get proxy address from evm address for both mm and tp
func (im *Manager) GetPairXProxyAddressFromEvmAddress(evmAddress string) (string, string, error) {
	// pairX from evm address
	pairXMM := NewPairX(evmAddress, "", "", "", MMScenery, "", 0)

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
	pairXTP := NewPairX(evmAddress, "", "", "", TPScenery, "", 0)
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
	pairX := NewPairX("", "", "", "", 0, proxyAddress, 0)
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
func (im *Manager) FilterPairXFromLedgerOutput(inxOutput *inx.LedgerOutput, logger *logger.Logger) (*PairX, error) {
	if inxOutput == nil {
		return nil, nil
	}
	output, err := inxOutput.UnwrapOutput(serializer.DeSeriModeNoValidation, nil)
	if err != nil {
		return nil, err
	}
	outputID := inxOutput.UnwrapOutputID()
	return im.FilterPairXFromOutput(output, outputID, logger)
}

// filter pairX from output
func (im *Manager) FilterPairXFromOutput(output iotago.Output, outputID iotago.OutputID, logger *logger.Logger) (*PairX, error) {
	if output == nil {
		return nil, nil
	}
	nftOutput, ok := output.(*iotago.NFTOutput)
	if !ok {
		return nil, nil
	}
	return im.FilterPairXFromNFTOutput(nftOutput, outputID, logger)
}

type PairXData struct {
	EncryptedPrivateKey string `json:"encryptedPrivateKey"`
	PairXPublicKey      string `json:"pairXPublicKey"`
	EvmAddress          string `json:"evmAddress"`
	Timestamp           int64  `json:"timestamp"`
	Scenery             int    `json:"scenery"`
	Signature           string `json:"signature"`
}

// filter pairX from nftOutput
func (im *Manager) FilterPairXFromNFTOutput(output *iotago.NFTOutput, outputID iotago.OutputID, logger *logger.Logger) (*PairX, error) {
	if output == nil {
		return nil, nil
	}
	// get tag
	if output.FeatureSet().TagFeature() == nil ||
		output.FeatureSet().TagFeature().Tag == nil ||
		!bytes.Equal(output.FeatureSet().TagFeature().Tag, pairXTag) {
		return nil, nil
	}
	// log tag match
	logger.Infof("FilterPairXFromNFTOutput ... tag match:%s", PairXTagStr)
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
	var data PairXData
	// log unmarshal metadata
	logger.Infof("FilterPairXFromNFTOutput ... metadata:%s", string(output.ImmutableFeatureSet().MetadataFeature().Data))
	err := json.Unmarshal(output.ImmutableFeatureSet().MetadataFeature().Data, &data)
	if err != nil {
		return nil, err
	}
	// get each field of pairX, check nil then get from metaMap
	evmAddress := data.EvmAddress
	if evmAddress == "" {
		// log evm address nil
		logger.Infof("FilterPairXFromNFTOutput ... evmAddress nil")
		return nil, nil
	}
	// to lower
	evmAddress = strings.ToLower(evmAddress)
	pairXPublicKey := data.PairXPublicKey
	if pairXPublicKey == "" {
		// log public key nil
		logger.Infof("FilterPairXFromNFTOutput ... pairXPublicKey nil")
		return nil, nil
	}
	encryptedPrivateKey := data.EncryptedPrivateKey
	if encryptedPrivateKey == "" {
		// log encrypted private key nil
		logger.Infof("FilterPairXFromNFTOutput ... encryptedPrivateKey nil")
		return nil, nil
	}
	signature := data.Signature
	if signature == "" {
		// log signature nil
		logger.Infof("FilterPairXFromNFTOutput ... signature nil")
		return nil, nil
	}
	scenery := data.Scenery
	timestamp := int(data.Timestamp)
	// log each field
	logger.Infof("FilterPairXFromNFTOutput ... evmAddress:%s, pairXPublicKey:%s, encryptedPrivateKey:%s, signature:%s, scenery:%d, timestamp:%d",
		evmAddress, pairXPublicKey, encryptedPrivateKey, signature, scenery, timestamp)

	// get proxy address from unlock condition
	unlockConditionSet := output.UnlockConditionSet()
	if unlockConditionSet == nil {
		return nil, nil
	}
	proxyAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	// log proxy address
	logger.Infof("FilterPairXFromNFTOutput ... proxyAddress:%s", proxyAddress)
	pairX := NewPairX(evmAddress, pairXPublicKey, encryptedPrivateKey, signature, scenery, proxyAddress, timestamp)

	return pairX, nil
}

// verify signature
func (im *Manager) VerifyPairXSignature(pairX *PairX) bool {
	message := fmt.Sprintf("%s%s%s%d%d",
		pairX.PrivateKey,
		pairX.EvmAddress,
		pairX.PublicKey,
		pairX.Scenery,
		pairX.Timestamp,
	)

	// Recover the public key from the signature
	sigHex := pairX.Signature

	// Convert message to hash
	messageHash := crypto.Keccak256Hash([]byte(message))

	signature, err := iotago.DecodeHex(sigHex)
	if err != nil {
		log.Fatalf("Invalid signature hex: %v", err)
	}

	// Extract the Ethereum account address from the signature
	sigPublicKey, err := crypto.Ecrecover(messageHash.Bytes(), signature)
	if err != nil {
		log.Fatalf("Ecrecover failed: %v", err)
	}

	// Generate the public key using the recovered public key
	publicKey, err := crypto.UnmarshalPubkey(sigPublicKey)
	if err != nil {
		log.Fatalf("UnmarshalPubkey failed: %v", err)
	}

	// Get the original signer's address from the public key
	signerAddress := crypto.PubkeyToAddress(*publicKey)

	// Print the signer address
	fmt.Println("Signer Address:", signerAddress.Hex())

	// Check if the recovered address matches the provided EvmAddress
	return strings.ToLower(signerAddress.Hex()) == strings.ToLower(pairX.EvmAddress)

}

// handle pairX created
func (im *Manager) HandlePairXCreated(pairx *PairX, logger *logger.Logger) {
	// log pairX creation
	logger.Infof("HandlePairXCreated ... pairX:%+v", pairx)
	//TODO validate signature
	if err := im.StorePairX(pairx); err != nil {
		logger.Warnf("HandlePairXCreated ... StorePairX failed:%s", err)
	}
}
// convert address to actual address if a mapping exists
func (im *Manager) ConvertAddressToActualAddress(address string) string {
	// get pairX from evm address
	evmAddress, err := im.GetPairXEvmAddressFromProxyAddress(address)
	if err != nil {
		return ""
	}
	if evmAddress == "" {
		return address
	}
	// to lower
	evmAddress = strings.ToLower(evmAddress)
	return evmAddress
}
