package websocket

import (
	"encoding/json"
	"fmt"
	"time"
)

// RawMessage from WebSocket
type RawMessage struct {
	MessageType string        `json:"messageType"`
	Service     string        `json:"service"`
	Data        []interface{} `json:"data"`
}

// Crypto types
type CryptoTrade struct {
	UpdateType string
	Ticker     string
	Date       time.Time
	Exchange   string
	LastSize   float64
	LastPrice  float64
}

type CryptoQuote struct {
	UpdateType string
	Ticker     string
	Date       time.Time
	Exchange   string
	BidSize    float64
	BidPrice   float64
	MidPrice   float64
	AskSize    float64
	AskPrice   float64
}

// Forex types
type ForexQuote struct {
	UpdateType string
	Ticker     string
	Date       time.Time
	BidSize    float64
	BidPrice   float64
	MidPrice   float64
	AskSize    float64
	AskPrice   float64
}

// IEX types
type IEXTrade struct {
	UpdateType       string
	Date             time.Time
	Nanoseconds      int64
	Ticker           string
	BidSize          *int32
	BidPrice         *float64
	MidPrice         *float64
	AskPrice         *float64
	AskSize          *int32
	LastPrice        *float64
	LastSize         *int32
	Halted           int32
	AfterHours       int32
	IntermarketSweep int32
	Oddlot           int32
	NMSRule611       int32
}

type IEXQuote struct {
	UpdateType       string
	Date             time.Time
	Nanoseconds      int64
	Ticker           string
	BidSize          *int32
	BidPrice         *float64
	MidPrice         *float64
	AskPrice         *float64
	AskSize          *int32
	LastPrice        *float64
	LastSize         *int32
	Halted           int32
	AfterHours       int32
	IntermarketSweep *int32
	Oddlot           *int32
	NMSRule611       *int32
}

// Helper functions
func toInt32Ptr(v interface{}) *int32 {
	if v == nil {
		return nil
	}
	if f, ok := v.(float64); ok {
		val := int32(f)
		return &val
	}
	return nil
}

func toFloat64Ptr(v interface{}) *float64 {
	if v == nil {
		return nil
	}
	if f, ok := v.(float64); ok {
		return &f
	}
	return nil
}

func toInt32(v interface{}) int32 {
	if f, ok := v.(float64); ok {
		return int32(f)
	}
	return 0
}

// ParseMessage routes to appropriate parser
func ParseMessage(rawMsg []byte) (interface{}, error) {
	var msg RawMessage
	if err := json.Unmarshal(rawMsg, &msg); err != nil {
		return nil, fmt.Errorf("unmarshal error: %w", err)
	}

	if msg.MessageType == "H" {
		return nil, nil // Heartbeat
	}

	if len(msg.Data) == 0 {
		return nil, nil
	}

	updateType, ok := msg.Data[0].(string)
	if !ok {
		return nil, fmt.Errorf("invalid update type")
	}

	switch msg.Service {
	case "crypto_data":
		return parseCryptoMessage(updateType, msg.Data)
	case "fx":
		return parseForexMessage(updateType, msg.Data)
	case "iex":
		return parseIEXMessage(updateType, msg.Data)
	default:
		return nil, fmt.Errorf("unknown service: %s", msg.Service)
	}
}

func parseCryptoMessage(updateType string, data []interface{}) (interface{}, error) {
	switch updateType {
	case "T":
		return ParseCryptoTrade(data)
	case "Q":
		return ParseCryptoQuote(data)
	default:
		return nil, fmt.Errorf("unknown crypto update: %s", updateType)
	}
}

func parseForexMessage(updateType string, data []interface{}) (interface{}, error) {
	if updateType != "Q" {
		return nil, fmt.Errorf("unknown forex update: %s", updateType)
	}
	return ParseForexQuote(data)
}

func parseIEXMessage(updateType string, data []interface{}) (interface{}, error) {
	switch updateType {
	case "T":
		return ParseIEXTrade(data)
	case "Q":
		return ParseIEXQuote(data)
	default:
		return nil, fmt.Errorf("unknown IEX update: %s", updateType)
	}
}

func ParseCryptoTrade(data []interface{}) (*CryptoTrade, error) {
	if len(data) != 6 {
		return nil, fmt.Errorf("invalid crypto trade length: %d", len(data))
	}

	date, err := time.Parse(time.RFC3339, data[2].(string))
	if err != nil {
		return nil, err
	}

	return &CryptoTrade{
		UpdateType: data[0].(string),
		Ticker:     data[1].(string),
		Date:       date,
		Exchange:   data[3].(string),
		LastSize:   data[4].(float64),
		LastPrice:  data[5].(float64),
	}, nil
}

func ParseCryptoQuote(data []interface{}) (*CryptoQuote, error) {
	if len(data) != 9 {
		return nil, fmt.Errorf("invalid crypto quote length: %d", len(data))
	}

	date, err := time.Parse(time.RFC3339, data[2].(string))
	if err != nil {
		return nil, err
	}

	return &CryptoQuote{
		UpdateType: data[0].(string),
		Ticker:     data[1].(string),
		Date:       date,
		Exchange:   data[3].(string),
		BidSize:    data[4].(float64),
		BidPrice:   data[5].(float64),
		MidPrice:   data[6].(float64),
		AskSize:    data[7].(float64),
		AskPrice:   data[8].(float64),
	}, nil
}

func ParseForexQuote(data []interface{}) (*ForexQuote, error) {
	if len(data) != 8 {
		return nil, fmt.Errorf("invalid forex quote length: %d", len(data))
	}

	date, err := time.Parse(time.RFC3339, data[2].(string))
	if err != nil {
		return nil, err
	}

	return &ForexQuote{
		UpdateType: data[0].(string),
		Ticker:     data[1].(string),
		Date:       date,
		BidSize:    data[3].(float64),
		BidPrice:   data[4].(float64),
		MidPrice:   data[5].(float64),
		AskSize:    data[6].(float64),
		AskPrice:   data[7].(float64),
	}, nil
}

func ParseIEXTrade(data []interface{}) (*IEXTrade, error) {
	if len(data) != 16 {
		return nil, fmt.Errorf("invalid IEX trade length: %d", len(data))
	}

	date, err := time.Parse(time.RFC3339, data[1].(string))
	if err != nil {
		return nil, err
	}

	return &IEXTrade{
		UpdateType:       data[0].(string),
		Date:             date,
		Nanoseconds:      int64(data[2].(float64)),
		Ticker:           data[3].(string),
		BidSize:          toInt32Ptr(data[4]),
		BidPrice:         toFloat64Ptr(data[5]),
		MidPrice:         toFloat64Ptr(data[6]),
		AskPrice:         toFloat64Ptr(data[7]),
		AskSize:          toInt32Ptr(data[8]),
		LastPrice:        toFloat64Ptr(data[9]),
		LastSize:         toInt32Ptr(data[10]),
		Halted:           toInt32(data[11]),
		AfterHours:       toInt32(data[12]),
		IntermarketSweep: toInt32(data[13]),
		Oddlot:           toInt32(data[14]),
		NMSRule611:       toInt32(data[15]),
	}, nil
}

func ParseIEXQuote(data []interface{}) (*IEXQuote, error) {
	if len(data) != 16 {
		return nil, fmt.Errorf("invalid IEX quote length: %d", len(data))
	}

	date, err := time.Parse(time.RFC3339, data[1].(string))
	if err != nil {
		return nil, err
	}

	return &IEXQuote{
		UpdateType:       data[0].(string),
		Date:             date,
		Nanoseconds:      int64(data[2].(float64)),
		Ticker:           data[3].(string),
		BidSize:          toInt32Ptr(data[4]),
		BidPrice:         toFloat64Ptr(data[5]),
		MidPrice:         toFloat64Ptr(data[6]),
		AskPrice:         toFloat64Ptr(data[7]),
		AskSize:          toInt32Ptr(data[8]),
		LastPrice:        toFloat64Ptr(data[9]),
		LastSize:         toInt32Ptr(data[10]),
		Halted:           toInt32(data[11]),
		AfterHours:       toInt32(data[12]),
		IntermarketSweep: toInt32Ptr(data[13]),
		Oddlot:           toInt32Ptr(data[14]),
		NMSRule611:       toInt32Ptr(data[15]),
	}, nil
}
