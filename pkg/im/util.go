package im

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
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
func Sha256HashBytes(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
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

// uint16 to bytes
func Uint16ToBytes(num uint16) []byte {
	tmp := make([]byte, 2)
	binary.BigEndian.PutUint16(tmp, num)
	return tmp
}

// bytes to uint16
func BytesToUint16(bytes []byte) uint16 {
	return binary.BigEndian.Uint16(bytes)
}
