package handler

type Broadcaster interface {
	Broadcast(channelID string, eventType string, data interface{})
}

type CombinedBroadcaster struct{}

func (b *CombinedBroadcaster) Broadcast(channelID string, eventType string, data interface{}) {
	SSEPushChannel <- SSEPushRequest{
		ChannelID: channelID,
		SSEMessage: SSEMessage{
			EventType: eventType,
			Data:      data,
		},
	}

	WSPushChannel <- WSPushRequest{
		ChannelID: channelID,
		WSMessage: WSMessage{
			EventType: eventType,
			Data:      data,
		},
	}
}
