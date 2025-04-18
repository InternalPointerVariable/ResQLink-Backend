package ws

import "encoding/json"

type Message struct {
	Event string          `json:"event"`
	Data  json.RawMessage `json:"data"`
}

// Returns a new Message with the given data on the same Event
func (m *Message) Response(data any) (Message, error) {
	res, err := json.Marshal(data)
	if err != nil {
		return Message{}, err
	}

	msg := Message{
		Event: m.Event,
		Data:  res,
	}

	return msg, nil
}
