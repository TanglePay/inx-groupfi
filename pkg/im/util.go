package im

import (
	"context"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
	"github.com/mr-tron/base58"
)

var (
	incrementerOnce sync.Once
	incrementer     *Incrementer
)

type Incrementer struct {
	index   uint32
	counter uint32
	mu      sync.Mutex
}

func GetIncrementer() *Incrementer {
	incrementerOnce.Do(func() {
		incrementer = &Incrementer{}
	})
	return incrementer
}

func (inc *Incrementer) Increment(index uint32) uint32 {
	inc.mu.Lock()
	defer inc.mu.Unlock()

	if index != inc.index {
		inc.index = index
		inc.counter = 0
	}

	inc.counter++
	return inc.counter
}
func Sha256Hash(str string) []byte {
	hasher := sha256.New()
	hasher.Write([]byte(str))
	return hasher.Sum(nil)
}
func Sha256HashFixed(str string) [Sha256HashLen]byte {
	bytes := Sha256Hash(str)
	var fixed [Sha256HashLen]byte
	copy(fixed[:], bytes)
	return fixed
}
func Sha256HashBytes(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

// sha256 hash address, to lower case first, reuse actual sha method
func Sha256HashAddress(address string) []byte {
	return Sha256Hash(strings.ToLower(address))
}
func ConcatByteSlices(slices ...[]byte) []byte {
	var totalLen int
	for _, s := range slices {
		totalLen += len(s)
	}
	result := make([]byte, totalLen)

	var offset int
	for _, s := range slices {
		copy(result[offset:], s)
		offset += len(s)
	}
	return result
}

func PerformGetRequest(ctx context.Context, targetURL string, params map[string]string, result interface{}) error {
	// Parse the base URL
	u, err := url.Parse(targetURL)
	if err != nil {
		return err
	}

	// Add the query parameters
	q := u.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	u.RawQuery = q.Encode()

	// Create a new request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return err
	}

	// Make the HTTP request
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	// Parse the JSON response body
	if err := json.Unmarshal(body, result); err != nil {
		return err
	}
	return nil
}
func GetCurrentEpochTimestamp() uint32 {
	return uint32(time.Now().Unix())
}
func AppendBytesWithUint16Len(bytes *[]byte, idx *int, slice []byte, appendLength bool) {
	length := len(slice)

	if appendLength {
		// Ensure there's enough space in bytes slice for length + actual data
		newSize := *idx + 2 + length
		if cap(*bytes) < newSize {
			*bytes = append(*bytes, make([]byte, 2+length)...)
		} else {
			*bytes = (*bytes)[:newSize]
		}

		// Encode length as uint16 and put in slice
		(*bytes)[*idx] = byte(length >> 8)
		(*bytes)[*idx+1] = byte(length & 0xFF)

		// Copy the slice data
		copy((*bytes)[*idx+2:], slice)

		// Update index
		*idx = newSize
	} else {
		// If not appending the length, just append the slice itself
		*bytes = append(*bytes, slice...)
		*idx += length
	}
}
func MergeAndRemoveDupsStringArray(slice1, slice2 []string) []string {
	// Create a map to track unique elements
	uniqueMap := make(map[string]bool)
	var result []string

	// Add elements from the first slice to the map and result slice
	for _, val := range slice1 {
		if !uniqueMap[val] {
			uniqueMap[val] = true
			result = append(result, val)
		}
	}

	// Add elements from the second slice to the map and result slice
	for _, val := range slice2 {
		if !uniqueMap[val] {
			uniqueMap[val] = true
			result = append(result, val)
		}
	}

	return result
}
func ReadBytesWithUint16Len(bytes []byte, idx *int, providedLength ...int) ([]byte, error) {
	var length int
	if len(providedLength) > 0 {
		// If a length is provided, use it
		length = providedLength[0]
	} else {
		// If no length provided, read length from bytes using the current index
		if *idx+2 > len(bytes) {
			return nil, fmt.Errorf("insufficient bytes for length reading")
		}
		length = int(bytes[*idx])<<8 | int(bytes[*idx+1])
		*idx += 2
	}

	if *idx+length > len(bytes) {
		return nil, fmt.Errorf("insufficient bytes for data reading")
	}

	data := bytes[*idx : *idx+length]
	*idx += length

	return data, nil
}

// const solana address length
const SolanaAddressLength = 32

func UnmarshalSolanaAddress(addressBytes []byte) (string, error) {
	if len(addressBytes) != SolanaAddressLength {
		return "", fmt.Errorf("invalid address length: expected 32 bytes, got %d", len(addressBytes))
	}
	return base58.Encode(addressBytes), nil
}

// uint32 to bytes
func Uint32ToBytes(num uint32) []byte {
	tmp := make([]byte, 4)
	binary.BigEndian.PutUint32(tmp, num)
	return tmp
}

// uint16 to bytes
func Uint16ToBytes(num uint16) []byte {
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, num)
	return tmp
}

// uint8 to bytes
func Uint8ToBytes(num uint8) []byte {
	return []byte{num}
}

// BoolToByte
func BoolToByte(b bool) byte {
	if b {
		return 1
	}
	return 0
}

// bytes to bool
func BytesToBool(bytes []byte) bool {
	return bytes[0] == 1
}

// bytes to uint16
func BytesToUint16(bytes []byte) uint16 {
	return binary.BigEndian.Uint16(bytes)
}

// bytes to uint8
func BytesToUint8(bytes []byte) uint8 {
	return bytes[0]
}

// bytes to uint32
func BytesToUint32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

// Float32ToBytes
func Float32ToBytes(num float32) []byte {
	bits := math.Float32bits(num)
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, bits)
	return bytes
}

// BytesToFloat32
func BytesToFloat32(bytes []byte) float32 {
	bits := binary.BigEndian.Uint32(bytes)
	return math.Float32frombits(bits)
}

type OutputPair struct {
	CreatedOutput  *OutputAndOutputId
	ConsumedOutput *OutputAndOutputId
}
type OutputAndOutputId struct {
	Output   *iotago.BasicOutput
	OutputId iotago.OutputID
}

// process output to OutputPair map
func ProcessOutputToOutputPair(pair map[string]*OutputPair, output *OutputAndOutputId, isConsumed bool) {
	unlockConditionSet := output.Output.UnlockConditionSet()
	ownerAddress := unlockConditionSet.Address().Address.Bech32(iotago.NetworkPrefix(HornetChainName))
	// if address not in map, add new pair
	if _, ok := pair[ownerAddress]; !ok {
		pair[ownerAddress] = &OutputPair{}
	}
	if isConsumed {
		pair[ownerAddress].ConsumedOutput = output
	} else {
		pair[ownerAddress].CreatedOutput = output
	}
}

// is address evm address
func IsEvmAddress(address string) bool {
	// start with 0x and length is 42
	return len(address) == 42 && address[:2] == "0x"
}

// bytes to fixed size bytes, Sha256HashLen
func BytesToFixedSha256HashLenBytes(bytes []byte) [Sha256HashLen]byte {
	var fixed [Sha256HashLen]byte
	copy(fixed[:], bytes)
	return fixed
}
func SHA1Hash(input string) string {
	// use SHA1HashBytes
	hashBytes := SHA1HashBytes([]byte(input))
	return hex.EncodeToString(hashBytes)
}
func SHA1HashBytes(input []byte) []byte {
	hash := sha1.New()
	hash.Write(input)
	hashBytes := hash.Sum(nil)
	return hashBytes
}
func SHA256HashBytesReturnString(input []byte) string {
	hashBytes := Sha256HashBytes(input)
	return hex.EncodeToString(hashBytes)
}
func CalculateDiff[T any](created, existing []*T, getKey func(*T) string) (toCreate, toDelete []*T) {
	existingMap := make(map[string]*T)
	for _, e := range existing {
		key := getKey(e)
		existingMap[key] = e
	}

	createdMap := make(map[string]*T)
	for _, c := range created {
		key := getKey(c)
		createdMap[key] = c
	}

	// Determine what to create (in created but not in existing)
	for k, c := range createdMap {
		if _, exists := existingMap[k]; !exists {
			toCreate = append(toCreate, c)
		}
	}

	// Determine what to delete (in existing but not in created)
	for k, e := range existingMap {
		if _, exists := createdMap[k]; !exists {
			toDelete = append(toDelete, e)
		}
	}

	return toCreate, toDelete
}
func StartOfHour(epochTimestamp uint32) uint32 {
	t := time.Unix(int64(epochTimestamp), 0).UTC()
	startOfHour := time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), 0, 0, 0, time.UTC)
	return uint32(startOfHour.Unix())
}

// push data
func PushData[T any](data *T, getTopic func(*T) string,
	getInbox func(*T) [][]byte, getEventType func(*T) byte,
	getPayload func(*T) []byte, manager *Manager, logger *logger.Logger) error {
	if IsIniting {
		return nil
	}
	topic := getTopic(data)
	payload := getPayload(data)
	err := manager.GetMqttServer().Publish("inbox/"+topic, payload)
	if err != nil {
		return err
	}
	// log topic
	logger.Infof("PushEventData to topic %s", topic)
	eventType := getEventType(data)
	inboxs := getInbox(data)
	// store to ttl store
	for _, inbox := range inboxs {
		err = manager.StoreEventToInbox(inbox, CurrentMilestoneIndex, CurrentMilestoneTimestamp, payload, eventType, logger)
		if err != nil {
			return err
		}
	}

	return nil
}

// get collectionId from nft output
func GetCollectionIdFromNFTOutput(output *iotago.NFTOutput) (string, error) {
	issuer := output.ImmutableFeatureSet().IssuerFeature()
	if issuer == nil {
		return "", fmt.Errorf("issuer not found")
	}
	issuerAddress := issuer.Address
	if issuerAddress == nil {
		return "", fmt.Errorf("issuer address not found")
	}
	if issuerAddress.Type() != iotago.AddressNFT {
		return "", fmt.Errorf("issuer address type is not NFT")
	}
	nftAddress := issuerAddress.(*iotago.NFTAddress)
	collectionId := nftAddress.NFTID().ToHex()

	return collectionId, nil
}
