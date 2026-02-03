package common

import (
	"encoding/json"
)

type Response struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// WebSocketMessage represents the generic wrapper for all incoming messages.
type WebSocketMessage struct {
	MessageType string          `json:"messageType"`
	Service     string          `json:"service"`
	Response    *Response       `json:"response"`
	Data        json.RawMessage `json:"data"`
}

// SubscribeRequest is the payload for subscribing/unsubscribing.
type SubscribeRequest struct {
	EventName     string                 `json:"eventName"`
	Authorization string                 `json:"authorization"`
	EventData     map[string]interface{} `json:"eventData,omitempty"`
}

// SubscriptionConfirmation defines the structure of the subscription confirmation message.
type SubscriptionConfirmation struct {
	SubscriptionID int      `json:"subscriptionId"`
	Tickers        []string `json:"tickers"`
	ThresholdLevel string   `json:"thresholdLevel"`
}
