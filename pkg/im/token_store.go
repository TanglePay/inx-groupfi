package im

import (
	"encoding/binary"
	"math/big"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
	iotago "github.com/iotaledger/iota.go/v3"
)

func (im *Manager) TokenKeyFromToken(token *TokenStat) []byte {
	// key = prefix + type + address + tokenid + status
	key := make([]byte, 1+2+Sha256HashLen+Sha256HashLen+1)
	index := 0
	key[index] = ImStoreKeyPrefixToken
	index++
	// type is 2 bytes
	binary.BigEndian.PutUint16(key[index:], token.tokenType)
	index += 2
	copy(key[index:], Sha256Hash(token.address))
	index += Sha256HashLen
	copy(key[index:], token.tokenId)
	index += Sha256HashLen
	key[index] = token.status
	return key
}
func (im *Manager) TotalTokenKeyFromToken(token *TokenStat) []byte {
	// key = prefix + type + totaladdress + tokenid + status
	totalAddress := im.GetTotalAddressFromType(token.tokenType)
	totalAddressSha256 := Sha256Hash(totalAddress)
	key := make([]byte, 1+2+Sha256HashLen+Sha256HashLen+1)
	index := 0
	key[index] = ImStoreKeyPrefixToken
	index++
	// type is 2 bytes
	binary.BigEndian.PutUint16(key[index:], token.tokenType)
	index += 2
	copy(key[index:], totalAddressSha256)
	index += Sha256HashLen
	copy(key[index:], token.tokenId)
	index += Sha256HashLen
	key[index] = token.status
	return key
}

// token key prefix for address
func (im *Manager) TokenKeyPrefixFromAddress(tokenType uint16, addressSha256 []byte) []byte {
	key := make([]byte, 1+2+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixToken
	index++
	// type is 2 bytes
	binary.BigEndian.PutUint16(key[index:], tokenType)
	index += 2
	copy(key[index:], addressSha256)
	return key
}

// prefix for token type
func (im *Manager) TokenKeyPrefixFromTokenType(tokenType uint16) []byte {
	key := make([]byte, 1+2)
	index := 0
	key[index] = ImStoreKeyPrefixToken
	index++
	// type is 2 bytes
	binary.BigEndian.PutUint16(key[index:], tokenType)
	return key
}

// calculate sum of one address
func (im *Manager) CalculateSumOfAddress(tokenType uint16, addressSha256 []byte) (*big.Int, error) {
	prefix := im.TokenKeyPrefixFromAddress(tokenType, addressSha256)
	totalBalance := big.NewInt(0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		amountStr := string(value)
		amount, _ := new(big.Int).SetString(amountStr, 10)
		status := key[len(key)-1]
		if status == ImTokenStatusCreated {
			totalBalance.Add(totalBalance, amount)
		} else if status == ImTokenStatusConsumed {
			totalBalance.Sub(totalBalance, amount)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return totalBalance, nil

}

const (
	SMRTotalAddress  = "totalAddressSMR"
	SOONTotalAddress = "totalAddressSOON"
)

// GetSMRTotalAddress
func (im *Manager) GetSMRTotalAddressStr() string {
	return SMRTotalAddress
}

func (im *Manager) GetTotalAddressFromType(tokenType uint16) string {
	if tokenType == ImTokenTypeSMR {
		return SMRTotalAddress
	}
	return ""
}

func (im *Manager) TokenKeyPrefixFromTokenTypeAndAddress(tokenType uint16, addressSha256 []byte) []byte {
	key := make([]byte, 1+2+Sha256HashLen)
	index := 0
	key[index] = ImStoreKeyPrefixToken
	index++
	// type is 2 bytes
	binary.BigEndian.PutUint16(key[index:], tokenType)
	index += 2
	copy(key[index:], addressSha256)
	return key
}

// (amountStr,addressStr) -> valuepaylod(amountLen,amount,addressLen,address)
func (im *Manager) TokenValuePayloadFromAmountAndAddress(amountStr string, addressStr string) []byte {
	amountBytes := []byte(amountStr)
	amountBytesLen := len(amountBytes)
	addressBytes := []byte(addressStr)
	addressBytesLen := len(addressBytes)
	// value payload = amount len + amount + address len + address
	valuePayload := make([]byte, 4+amountBytesLen+4+addressBytesLen)
	index := 0
	binary.BigEndian.PutUint32(valuePayload[index:], uint32(amountBytesLen))
	index += 4
	copy(valuePayload[index:], amountBytes)
	index += amountBytesLen
	binary.BigEndian.PutUint32(valuePayload[index:], uint32(addressBytesLen))
	index += 4
	copy(valuePayload[index:], addressBytes)
	return valuePayload
}

// valuepaylod(amountLen,amount,addressLen,address) -> (amountStr,addressStr)
func (im *Manager) AmountAndAddressFromTokenValuePayload(valuePayload []byte) (string, string) {
	index := 0
	amountBytesLen := binary.BigEndian.Uint32(valuePayload[index:])
	index += 4
	amountBytes := valuePayload[index : index+int(amountBytesLen)]
	index += int(amountBytesLen)
	addressBytesLen := binary.BigEndian.Uint32(valuePayload[index:])
	index += 4
	addressBytes := valuePayload[index : index+int(addressBytesLen)]
	amountStr := string(amountBytes)
	addressStr := string(addressBytes)
	return amountStr, addressStr
}
func (im *Manager) StoreOneToken(token *TokenStat) error {
	key := im.TokenKeyFromToken(token)
	amountStr := token.amount.Text(10)
	valuePayload := im.TokenValuePayloadFromAmountAndAddress(amountStr, token.address)
	totalKey := im.TotalTokenKeyFromToken(token)
	err := im.imStore.Set(totalKey, valuePayload)
	if err != nil {
		return err
	}
	err = im.imStore.Set(key, valuePayload)
	return err
}
func (im *Manager) GetBalanceOfOneAddress(tokenType uint16, address string) (*big.Int, error) {
	addressSha256 := Sha256Hash(address)
	return im.GetBalanceOfOneAddressSha256(tokenType, addressSha256)
}
func (im *Manager) GetBalanceOfOneAddressSha256(tokenType uint16, addressSha256 []byte) (*big.Int, error) {
	keyPrefix := im.TokenKeyPrefixFromTokenTypeAndAddress(tokenType, addressSha256)
	totalBalance := big.NewInt(0)

	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		amountStr, _ := im.AmountAndAddressFromTokenValuePayload(value)

		amount, _ := new(big.Int).SetString(amountStr, 10)
		status := key[len(key)-1]
		if status == ImTokenStatusCreated {
			totalBalance.Add(totalBalance, amount)
		} else if status == ImTokenStatusConsumed {
			totalBalance.Sub(totalBalance, amount)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return totalBalance, nil
}

// token type to group name
func (im *Manager) GetGroupNameFromTokenType(tokenType uint16) string {
	if tokenType == ImTokenTypeSMR {
		return "smr-whale"
	}
	return "??"
}

// token type to threshold in string format
func (im *Manager) GetThresholdFromTokenType(tokenType uint16) string {
	if tokenType == ImTokenTypeSMR {
		groupId := im.GroupNameToGroupId("smr-whale")
		// return "" if group not found
		if groupId == nil {
			return "0"
		}
		groupIdHex := iotago.EncodeHex(groupId)
		groupConfig := im.GroupIdToGroupConfig(groupIdHex)
		if groupConfig == nil {
			return "0"
		}
		thresTxt := groupConfig.TokenThres
		return thresTxt
	}
	return "??"
}

var NftIdPadding = Sha256Hash("nftIdPadding")

// SetWhaleEligibility(tokenType, address, isEligible)
func (im *Manager) SetWhaleEligibility(tokenType uint16, address string, isEligible bool, logger *logger.Logger) error {
	groupName := im.GetGroupNameFromTokenType(tokenType)
	groupId := im.GroupNameToGroupId(groupName)
	tokenThreshold := im.GetThresholdFromTokenType(tokenType)
	addressSha256 := Sha256Hash(address)
	nft := NewNFTForToken(groupId, address, addressSha256, groupName, tokenType, tokenThreshold)
	if isEligible {
		return im.storeSingleNFT(nft, logger)
	} else {
		return im.DeleteNFT(nft, logger)
	}
}
