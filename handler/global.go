package handler

type SSEPushRequest struct {
	SSEMessage SSEMessage
	ChannelID  string
}

type SSEMessage struct {
	EventType string
	Data      interface{}
}

var SSEPushChannel = make(chan SSEPushRequest, 100)

type WSPushRequest struct {
	ChannelID string
	WSMessage WSMessage
}

var WSPushChannel = make(chan WSPushRequest, 100)
