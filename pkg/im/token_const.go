package im

const (
	// token type
	ImTokenTypeSMR  uint16 = 1
	ImTokenTypeSOON uint16 = 2

	// token status
	ImTokenStatusCreated  byte    = 1
	ImTokenStatusConsumed byte    = 2
	ImSMRWhaleThreshold   float64 = 0.01
)
