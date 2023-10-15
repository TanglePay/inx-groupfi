package im

// MessagesResponse defines the response of a GET RouteIMMessages REST API call.
type MessagesResponse struct {
	Messages  []*MessageResponse `json:"messages"`
	HeadToken string             `json:"headToken"`
	TailToken string             `json:"tailToken"`
}
type InboxMessagesResponse struct {
	Messages []*MessageResponse `json:"messages"`
	Token    string             `json:"token"`
}
type MessageResponse struct {
	GroupId   string `json:"groupId"`
	Token     string `json:"token"`
	OutputId  string `json:"outputId"`
	Timestamp uint32 `json:"timestamp"`
}

// AddressGroupDetailsResponse
type AddressGroupDetailsResponse struct {
	GroupId          string `json:"groupId"`
	GroupName        string `json:"groupName"`
	GroupQualifyType int    `json:"groupQualifyType"`
	IpfsLink         string `json:"ipfsLink"`
	TokenName        string `json:"tokenName"`
	TokenThres       string `json:"tokenThres"`
}

type SharedResponse struct {
	OutputId string `json:"outputId"`
}

type TokenBalanceResponse struct {
	TokenType    uint16 `json:"TokenType"`
	Balance      string `json:"Balance"`
	TotalBalance string `json:"TotalBalance"`
}

// VoteResponse
type VoteResponse struct {
	GroupId           string `json:"groupId"`
	AddressSha256Hash string `json:"token"`
	Vote              int    `json:"vote"`
}

// VoteCountResponse
type VoteCountResponse struct {
	GroupId      string `json:"groupId"`
	PublicCount  int    `json:"publicCount"`
	PrivateCount int    `json:"privateCount"`
	MemberCount  int    `json:"memberCount"`
}

// enum for output type created or consumed
const (
	ImOutputTypeCreated = iota
	ImOutputTypeConsumed
)
