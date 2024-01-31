package im

type TokenStat struct {
	TokenId        []byte
	TokenIdHash    [Sha256HashLen]byte
	InstanceIdHash [Sha256HashLen]byte
	Address        string
	Status         byte
	Amount         string
}

// NewTokenStat creates a new TokenStat.
func (im *Manager) NewTokenStat(rawTokenId []byte, outputId []byte, addressStr string, status byte, amount string) *TokenStat {
	tokenIdHash := Sha256HashBytes(rawTokenId)
	tokenIdHashFixed := [Sha256HashLen]byte{}
	copy(tokenIdHashFixed[:], tokenIdHash)
	instanceIdHash := Sha256HashBytes(outputId)
	instanceIdHashFixed := [Sha256HashLen]byte{}
	copy(instanceIdHashFixed[:], instanceIdHash)
	return &TokenStat{
		TokenId:        rawTokenId,
		TokenIdHash:    tokenIdHashFixed,
		InstanceIdHash: instanceIdHashFixed,
		Address:        addressStr,
		Status:         status,
		Amount:         amount,
	}
}

var SmrTokenId = Sha256Hash("smr")
