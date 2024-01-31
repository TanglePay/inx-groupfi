package im

import (
	"math/big"

	"github.com/iotaledger/hive.go/core/kvstore"
	"github.com/iotaledger/hive.go/core/logger"
)

func (im *Manager) TokenKeyFromToken(token *TokenStat) []byte {
	// key = prefix + tokenidHash + addressHash + instanceIdHash + status
	key := make([]byte, 0)
	index := 0
	// append ImStoreKeyPrefixToken
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixToken}, false)
	AppendBytesWithUint16Len(&key, &index, token.TokenIdHash[:], false)
	// append Sha256Hash(token.address)
	AppendBytesWithUint16Len(&key, &index, Sha256Hash(token.Address), false)
	// append instanceId
	AppendBytesWithUint16Len(&key, &index, token.InstanceIdHash[:], false)
	// append status using AppendBytesWithUint16Len
	AppendBytesWithUint16Len(&key, &index, []byte{token.Status}, false)
	return key
}
func (im *Manager) TotalTokenKeyFromToken(token *TokenStat) []byte {
	// key = prefix + tokenid + [Sha256Len]byte with all zero + instanceId + status
	key := make([]byte, 0)
	index := 0
	// append ImStoreKeyPrefixToken
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixToken}, false)
	AppendBytesWithUint16Len(&key, &index, token.TokenIdHash[:], false)
	// append Sha256Hash(token.address)
	allZero := [Sha256HashLen]byte{}
	AppendBytesWithUint16Len(&key, &index, allZero[:], false)
	// append instanceId
	AppendBytesWithUint16Len(&key, &index, token.InstanceIdHash[:], false)
	// append status using AppendBytesWithUint16Len
	AppendBytesWithUint16Len(&key, &index, []byte{token.Status}, false)
	return key
}

// prefix for all
func (im *Manager) TokenKeyPrefixForAll() []byte {
	// key = prefix

	key := make([]byte, 0)
	index := 0
	// append ImStoreKeyPrefixToken
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixToken}, false)
	return key
}

// prefix for token type
func (im *Manager) TokenKeyPrefixFromTokenId(tokenId []byte) []byte {
	// key = prefix + tokenidHash

	key := make([]byte, 0)
	index := 0
	// append ImStoreKeyPrefixToken
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixToken}, false)
	AppendBytesWithUint16Len(&key, &index, Sha256HashBytes(tokenId), false)
	return key
}

// calculate sum of one address
func (im *Manager) CalculateSumOfAddress(tokenId []byte, addressSha256 []byte) (*big.Int, error) {
	prefix := im.TokenKeyPrefixFromTokenIdAndAddress(tokenId, addressSha256)
	totalBalance := big.NewInt(0)
	err := im.imStore.Iterate(prefix, func(key kvstore.Key, value kvstore.Value) bool {
		tokenStat, err := im.TokenStateFromKeyAndValue(key, value)
		if err != nil {
			// log error then continue
			return true
		}
		amountStr := tokenStat.Amount
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			// log error then continue
			return true
		}
		if tokenStat.Status == ImTokenStatusCreated {
			totalBalance.Add(totalBalance, amount)
		} else if tokenStat.Status == ImTokenStatusConsumed {
			totalBalance.Sub(totalBalance, amount)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return totalBalance, nil

}

func (im *Manager) TokenKeyPrefixFromTokenIdAndAddress(tokenId []byte, addressSha256 []byte) []byte {
	// key = prefix + tokenidHash + address
	key := make([]byte, 0)
	index := 0
	tokenIdHash := Sha256HashBytes(tokenId)
	// append ImStoreKeyPrefixToken
	AppendBytesWithUint16Len(&key, &index, []byte{ImStoreKeyPrefixToken}, false)
	AppendBytesWithUint16Len(&key, &index, tokenIdHash, false)
	// append addressSha256
	AppendBytesWithUint16Len(&key, &index, addressSha256, false)
	return key
}

// (amountStr,addressStr) -> valuepaylod(amountLen,amount,addressLen,address)
func (im *Manager) TokenValuePayloadFromTokenState(state *TokenStat) []byte {
	// value payload = tokenId + amount + address
	valuePayload := make([]byte, 0)
	index := 0
	AppendBytesWithUint16Len(&valuePayload, &index, state.TokenId, true)
	AppendBytesWithUint16Len(&valuePayload, &index, []byte(state.Amount), true)
	AppendBytesWithUint16Len(&valuePayload, &index, []byte(state.Address), true)
	return valuePayload
}

// valuepaylod(amountLen,amount,addressLen,address) -> (amountStr,addressStr)
func (im *Manager) TokenStateFromKeyAndValue(key kvstore.Key, value kvstore.Value) (*TokenStat, error) {
	index := 0
	tokenId, err := ReadBytesWithUint16Len(value, &index)
	if err != nil {
		return nil, err
	}
	amountBytes, err := ReadBytesWithUint16Len(value, &index)
	if err != nil {
		return nil, err
	}
	amount := string(amountBytes)
	addressBytes, err := ReadBytesWithUint16Len(value, &index)
	if err != nil {
		return nil, err
	}
	address := string(addressBytes)

	index = 1
	// key = prefix + tokenidHash + addressHash + instanceIdHash + status
	_, err = ReadBytesWithUint16Len(key, &index, 1)
	if err != nil {
		return nil, err
	}
	_, err = ReadBytesWithUint16Len(key, &index, Sha256HashLen)
	if err != nil {
		return nil, err
	}
	instanceIdHash, err := ReadBytesWithUint16Len(key, &index, Sha256HashLen)
	if err != nil {
		return nil, err
	}
	statusBytes, err := ReadBytesWithUint16Len(key, &index, 1)
	if err != nil {
		return nil, err
	}
	status := statusBytes[0]
	return im.NewTokenStat(tokenId, instanceIdHash, address, status, amount), nil

}
func (im *Manager) StoreOneToken(token *TokenStat) error {
	key := im.TokenKeyFromToken(token)
	valuePayload := im.TokenValuePayloadFromTokenState(token)
	totalKey := im.TotalTokenKeyFromToken(token)
	err := im.imStore.Set(totalKey, valuePayload)
	if err != nil {
		return err
	}
	err = im.imStore.Set(key, valuePayload)
	return err
}
func (im *Manager) GetBalanceOfOneAddress(tokenId []byte, address string) (*big.Int, error) {
	addressSha256 := Sha256Hash(address)
	return im.GetBalanceOfOneAddressSha256(tokenId, addressSha256)
}
func (im *Manager) GetBalanceOfOneAddressSha256(tokenId []byte, addressSha256 []byte) (*big.Int, error) {
	keyPrefix := im.TokenKeyPrefixFromTokenIdAndAddress(tokenId, addressSha256)
	totalBalance := big.NewInt(0)

	err := im.imStore.Iterate(keyPrefix, func(key kvstore.Key, value kvstore.Value) bool {
		tokenStat, err := im.TokenStateFromKeyAndValue(key, value)
		if err != nil {
			// log error then continue
			return true
		}
		amountStr := tokenStat.Amount
		amount, ok := new(big.Int).SetString(amountStr, 10)
		if !ok {
			// log error then continue
			return true
		}
		if tokenStat.Status == ImTokenStatusCreated {
			totalBalance.Add(totalBalance, amount)
		} else if tokenStat.Status == ImTokenStatusConsumed {
			totalBalance.Sub(totalBalance, amount)
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return totalBalance, nil
}

var NftIdPadding = Sha256Hash("nftIdPadding")

// SetWhaleEligibility(tokenType, address, isEligible)
func (im *Manager) SetWhaleEligibility(tokenId []byte, groupName string, tokenThreshold string, address string, isEligible bool, logger *logger.Logger) error {
	groupId := im.GroupNameToGroupId(groupName)
	addressSha256 := Sha256Hash(address)
	nft := NewNFTForToken(groupId, address, addressSha256, groupName, tokenId, tokenThreshold)
	if isEligible {
		return im.storeSingleNFT(nft, logger)
	} else {
		return im.DeleteNFT(nft, logger)
	}
}
