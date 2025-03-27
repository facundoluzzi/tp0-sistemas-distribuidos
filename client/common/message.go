package common

type Message struct {
	ClientID string      `json:"client_id"`
	Type     string      `json:"type"`
	Data     interface{} `json:"data,omitempty"`
}
