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
