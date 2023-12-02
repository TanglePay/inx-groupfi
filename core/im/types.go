package im

import "github.com/TanglePay/inx-groupfi/pkg/im"

type InboxItemsResponse struct {
	Items []im.InboxItemJson `json:"items"`
	Token string             `json:"token"`
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

// GroupUserReputationResponse
type GroupUserReputationResponse struct {
	GroupId           string  `json:"groupId"`
	AddressSha256Hash string  `json:"addressSha256Hash"`
	Reputation        float32 `json:"reputation"`
}

// enum for output type created or consumed
const (
	ImOutputTypeCreated = iota
	ImOutputTypeConsumed
)
