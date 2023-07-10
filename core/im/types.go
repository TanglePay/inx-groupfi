package im

// MessagesResponse defines the response of a GET RouteIMMessages REST API call.
type MessagesResponse struct {
	Messages []*MessageResponse `json:"messages"`
	Token    string             `json:"token"`
}
type MessageResponse struct {
	OutputId  string `json:"outputId"`
	Timestamp uint32 `json:"timestamp"`
}
type NFTResponse struct {
	OwnerAddress string `json:"ownerAddress"`
	NFTId        string `json:"nftId"`
}
type SharedResponse struct {
	OutputId string `json:"outputId"`
}
