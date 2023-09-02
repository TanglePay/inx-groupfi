package im

// MessagesResponse defines the response of a GET RouteIMMessages REST API call.
type MessagesResponse struct {
	Messages  []*MessageResponse `json:"messages"`
	HeadToken string             `json:"headToken"`
	TailToken string             `json:"tailToken"`
}
type MessageResponse struct {
	OutputId  string `json:"outputId"`
	Timestamp uint32 `json:"timestamp"`
}
type NFTResponse struct {
	PublicKey    string `json:"publicKey"`
	OwnerAddress string `json:"ownerAddress"`
	NFTId        string `json:"nftId"`
}
type SharedResponse struct {
	OutputId string `json:"outputId"`
}

type TokenBalanceResponse struct {
	TokenType    uint16 `json:"TokenType"`
	Balance      string `json:"Balance"`
	TotalBalance string `json:"TotalBalance"`
}

// enum for output type created or consumed
const (
	ImOutputTypeCreated = iota
	ImOutputTypeConsumed
)
