package im

import "math/big"

type TokenStat struct {
	tokenType uint16
	tokenId   []byte
	address   string
	status    byte
	amount    *big.Int
}

// NewTokenStat creates a new TokenStat.
func (im *Manager) NewTokenStat(tokenType uint16, rawTokenId []byte, addressStr string, status byte, amountStr string) *TokenStat {
	tokenId := Sha256HashBytes(rawTokenId)
	if amountStr == "" {
		amountStr = "0"
	}
	amount, _ := new(big.Int).SetString(amountStr, 10)
	return &TokenStat{
		tokenType: tokenType,
		tokenId:   tokenId,
		address:   addressStr,
		status:    status,
		amount:    amount,
	}
}
