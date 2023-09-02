package im

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"sync"
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
