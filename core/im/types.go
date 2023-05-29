package im

// MessagesResponse defines the response of a GET RouteIMMessages REST API call.
type MessagesResponse struct {
	Messages []*MessageResponse `json:"messages"`
	Token    uint32             `json:"token"`
}
type MessageResponse struct {
	OutputId  string `json:"outputId"`
	Timestamp uint32 `json:"timestamp"`
}