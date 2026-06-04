package stream

func ParseMessage(msg RawMessage) *StreamMessage {

	get := func(key string) string {
		if v, ok := msg.Values[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
		return ""
	}

	return &StreamMessage{
		ID:            msg.ID,
		AggregateType: get("aggregate_type"),
		AggregateID:   get("aggregate_id"),
		EventType:     get("event_type"),
		Payload:       get("payload"),
	}
}