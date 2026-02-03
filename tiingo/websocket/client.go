package websocket

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/condrove10/go-tiingo-sdk/tiingo/common"
)

// WebSocket client errors
var (
	ErrAlreadySubscribed = errors.New("already subscribed to ticker")
	ErrNotConnected      = errors.New("websocket not connected")
	ErrAlreadyConnected  = errors.New("already connected")
)

// EndpointType defines the type for WebSocket endpoints.
type EndpointType string

// Constants for Tiingo WebSocket endpoints.
const (
	EndpointTypeIEX    EndpointType = "iex"
	EndpointTypeCrypto EndpointType = "crypto"
	EndpointTypeForex  EndpointType = "fx"
)

// WebsocketClient handles the WebSocket connection and data stream.
type WebsocketClient struct {
	apiKey               string
	endpoint             EndpointType
	config               *WebsocketConfig
	ctx                  context.Context
	mu                   sync.RWMutex
	wg                   sync.WaitGroup
	conn                 *websocket.Conn
	cancel               context.CancelFunc
	subscriptions        map[string]func(message []byte, err error)
	lastMessageTimestamp time.Time
	errorHandler         func(error)
}

// WebsocketConfig holds configuration for the WebSocket client.
type WebsocketConfig struct {
	ThresholdLevel  int
	LivenessTimeout time.Duration
	LivenessCheck   time.Duration
}

// Option is a functional option for configuring the WebsocketClient.
type Option func(*WebsocketClient)

// NewWebsocketClient creates a new client for the WebSocket API.
func NewWebsocketClient(apiKey string, endpointType EndpointType, options ...Option) *WebsocketClient {
	var threshold int
	switch endpointType {
	case EndpointTypeIEX:
		threshold = 6
	case EndpointTypeCrypto:
		threshold = 2
	case EndpointTypeForex:
		threshold = 5
	default:
		threshold = 0
	}

	ctx, cancel := context.WithCancel(context.Background())

	client := &WebsocketClient{
		apiKey:        apiKey,
		endpoint:      endpointType,
		subscriptions: make(map[string]func(message []byte, err error)),
		config: &WebsocketConfig{
			ThresholdLevel:  threshold,
			LivenessTimeout: time.Second * 90,
			LivenessCheck:   time.Second * 15,
		},
		ctx:    ctx,
		cancel: cancel,
	}

	for _, option := range options {
		option(client)
	}

	return client
}

// --- Functional Options ---

// WithThresholdLevel sets the subscription threshold level.
func WithThresholdLevel(level int) Option {
	return func(c *WebsocketClient) { c.config.ThresholdLevel = level }
}

// WithLivenessTimeout sets the duration after which an inactive connection is considered stale.
func WithLivenessTimeout(timeout time.Duration) Option {
	return func(c *WebsocketClient) { c.config.LivenessTimeout = timeout }
}

// WithLivenessCheck sets the interval for checking liveness.
func WithLivenessCheck(interval time.Duration) Option {
	return func(c *WebsocketClient) { c.config.LivenessCheck = interval }
}

// Connect establishes a websocket connection with an error callback for connection-level errors.
func (c *WebsocketClient) Connect(ctx context.Context, onError func(error)) error {
	c.mu.Lock()
	if c.conn != nil {
		c.mu.Unlock()
		return ErrAlreadyConnected
	}

	url := fmt.Sprintf("wss://api.tiingo.com/%s", c.endpoint)
	dialCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	conn, _, err := websocket.Dial(dialCtx, url, nil)
	if err != nil {
		c.mu.Unlock()
		return fmt.Errorf("failed to connect: %w", err)
	}

	c.conn = conn
	c.lastMessageTimestamp = time.Now()
	c.errorHandler = onError
	c.mu.Unlock()

	// Start background routines
	c.startRoutine(c.readLoop)
	c.startRoutine(c.healthcheckLoop)

	return nil
}

// Close gracefully disconnects the websocket connection and waits for all routines to finish.
func (c *WebsocketClient) Close() error {
	c.mu.Lock()
	if c.conn == nil {
		c.mu.Unlock()
		return nil
	}

	// Check if already closing
	select {
	case <-c.ctx.Done():
		c.mu.Unlock()
		return nil
	default:
		c.cancel()
	}
	c.mu.Unlock()

	// Wait for all routines to finish
	done := make(chan struct{})
	go func() {
		c.wg.Wait()
		close(done)
	}()

	// Wait for routines with timeout
	select {
	case <-done:
	case <-time.After(5 * time.Second):
	}

	// Close the connection
	c.mu.Lock()
	var closeErr error
	if c.conn != nil {
		closeErr = c.conn.Close(websocket.StatusNormalClosure, "")
		c.conn = nil
	}
	c.errorHandler = nil
	c.mu.Unlock()

	if closeErr != nil {
		return fmt.Errorf("failed to close websocket: %w", closeErr)
	}

	return nil
}

// startRoutine manages routine lifecycle with slot acquisition and release.
func (c *WebsocketClient) startRoutine(fn func()) {
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()

		// Acquire routine slot
		select {
		default:
			fn()
		case <-c.ctx.Done():
			return
		}
	}()
}

// notifyError safely calls the error handler if one is set.
func (c *WebsocketClient) notifyError(err error) {
	c.mu.RLock()
	handler := c.errorHandler
	c.mu.RUnlock()

	if handler != nil {
		handler(err)
	}
}

// updateLastMessageTimestamp updates the last message timestamp.
func (c *WebsocketClient) updateLastMessageTimestamp() {
	c.mu.Lock()
	c.lastMessageTimestamp = time.Now()
	c.mu.Unlock()
}

// getLastMessageTimestamp retrieves the last message timestamp.
func (c *WebsocketClient) getLastMessageTimestamp() time.Time {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.lastMessageTimestamp
}

// getConnection safely retrieves the connection.
func (c *WebsocketClient) getConnection() *websocket.Conn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func (c *WebsocketClient) readLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.notifyError(fmt.Errorf("readLoop panic: %v", r))
		}
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			conn := c.getConnection()
			if conn == nil {
				return
			}

			_, msg, err := conn.Read(c.ctx)
			if err != nil {
				select {
				case <-c.ctx.Done():
					return
				default:
					c.notifyError(fmt.Errorf("error reading message: %w", err))
					go c.Close()
					return
				}
			}

			c.updateLastMessageTimestamp()

			var message common.WebSocketMessage
			if err := json.Unmarshal(msg, &message); err != nil {
				c.notifyError(fmt.Errorf("failed to unmarshal message: %w", err))
				continue
			}

			switch message.MessageType {
			case "I":
				var conf common.SubscriptionConfirmation
				if err := json.Unmarshal(message.Data, &conf); err != nil {
					c.notifyError(fmt.Errorf("failed to unmarshal subscription confirmation: %w", err))
					continue
				}
			case "A":
				var dataPoints []interface{}
				if err := json.Unmarshal(message.Data, &dataPoints); err != nil {
					c.notifyError(fmt.Errorf("failed to unmarshal data points: %w", err))
					continue
				}

				if ticker, ok := c.extractTicker(dataPoints); ok {
					c.mu.RLock()
					handler, exists := c.subscriptions[ticker]
					c.mu.RUnlock()
					if exists {
						handler(msg, nil)
					}
				}
			}
		}
	}
}

func (c *WebsocketClient) healthcheckLoop() {
	defer func() {
		if r := recover(); r != nil {
			c.notifyError(fmt.Errorf("healthcheckLoop panic: %v", r))
		}
	}()

	ticker := time.NewTicker(c.config.LivenessCheck)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			lastHeartbeat := c.getLastMessageTimestamp()
			if time.Since(lastHeartbeat) > c.config.LivenessTimeout {
				c.notifyError(errors.New("connection stale: liveness timeout exceeded"))
				go c.Close()
				return
			}
		}
	}
}

// Subscribe subscribes to a ticker with a given handler.
func (c *WebsocketClient) subscribe(ticker string, handler func(message []byte, err error)) error {
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return ErrNotConnected
	}

	if _, exists := c.subscriptions[ticker]; exists {
		return ErrAlreadySubscribed
	}

	req := common.SubscribeRequest{
		EventName:     "subscribe",
		Authorization: c.apiKey,
		EventData: map[string]interface{}{
			"thresholdLevel": c.config.ThresholdLevel,
			"tickers":        []string{ticker},
		},
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal subscribe request: %w", err)
	}

	writeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.conn.Write(writeCtx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("failed to send subscribe message: %w", err)
	}

	c.subscriptions[ticker] = handler
	return nil
}

func (c *WebsocketClient) Subscribe(ticker string, handler func(message []byte, err error)) error {
	return c.subscribe(ticker, handler)
}

func (c *WebsocketClient) SubscribeCryptoEndpointWithHandlers(ticker string, cryptoQuoteHandler func(msg *CryptoQuote, err error), cryptoTradeHandler func(msg *CryptoTrade, err error)) error {
	if c.endpoint != EndpointTypeCrypto {
		return fmt.Errorf("client not configured for crypto endpoint")
	}

	return c.subscribe(ticker, func(message []byte, err error) {
		if err != nil {
			cryptoQuoteHandler(nil, err)
			cryptoTradeHandler(nil, err)
			return
		}

		res, err := ParseMessage(message)
		if err != nil {
			cryptoQuoteHandler(nil, err)
			cryptoTradeHandler(nil, err)
			return
		}

		switch msg := res.(type) {
		case *CryptoQuote:
			cryptoQuoteHandler(msg, nil)
		case *CryptoTrade:
			cryptoTradeHandler(msg, nil)
		default:
			cryptoQuoteHandler(nil, fmt.Errorf("unknown message type"))
			cryptoTradeHandler(nil, fmt.Errorf("unknown message type"))
		}
	})
}

func (c *WebsocketClient) SubscribeIEXEndpointWithHandlers(ticker string, iexQuoteHandler func(msg *IEXQuote, err error), iexTradeHandler func(msg *IEXTrade, err error)) error {
	if c.endpoint != EndpointTypeIEX {
		return fmt.Errorf("client not configured for IEX endpoint")
	}

	return c.subscribe(ticker, func(message []byte, err error) {
		if err != nil {
			iexQuoteHandler(nil, err)
			iexTradeHandler(nil, err)
			return
		}

		res, err := ParseMessage(message)
		if err != nil {
			iexQuoteHandler(nil, err)
			iexTradeHandler(nil, err)
			return
		}

		switch msg := res.(type) {
		case *IEXQuote:
			iexQuoteHandler(msg, nil)
		case *IEXTrade:
			iexTradeHandler(msg, nil)
		default:
			iexQuoteHandler(nil, fmt.Errorf("unknown message type"))
			iexTradeHandler(nil, fmt.Errorf("unknown message type"))
		}
	})
}

func (c *WebsocketClient) SubscribeForexEndpointWithHandler(ticker string, forexQuoteHandler func(msg *ForexQuote, err error)) error {
	if c.endpoint != EndpointTypeForex {
		return fmt.Errorf("client not configured for forex endpoint")
	}

	return c.subscribe(ticker, func(message []byte, err error) {
		if err != nil {
			forexQuoteHandler(nil, err)
			return
		}

		res, err := ParseMessage(message)
		if err != nil {
			forexQuoteHandler(nil, err)
			return
		}

		switch msg := res.(type) {
		case *ForexQuote:
			forexQuoteHandler(msg, nil)
		default:
			forexQuoteHandler(nil, fmt.Errorf("unknown message type"))
		}
	})
}

// Unsubscribe removes a subscription for a ticker.
func (c *WebsocketClient) Unsubscribe(ticker string) error {
	select {
	case <-c.ctx.Done():
		return c.ctx.Err()
	default:
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn == nil {
		return ErrNotConnected
	}

	if _, exists := c.subscriptions[ticker]; !exists {
		return fmt.Errorf("not subscribed to %s", ticker)
	}

	req := common.SubscribeRequest{
		EventName:     "unsubscribe",
		Authorization: c.apiKey,
		EventData:     map[string]interface{}{"tickers": []string{ticker}},
	}

	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal unsubscribe request: %w", err)
	}

	writeCtx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	if err := c.conn.Write(writeCtx, websocket.MessageText, data); err != nil {
		return fmt.Errorf("failed to send unsubscribe message: %w", err)
	}

	delete(c.subscriptions, ticker)
	return nil
}

func (c *WebsocketClient) extractTicker(data []interface{}) (string, bool) {
	if len(data) < 2 {
		return "", false
	}

	dataType, ok := data[0].(string)
	if !ok {
		return "", false
	}

	var ticker string
	if (dataType == "T" || dataType == "Q") && len(data) > 1 {
		ticker, ok = data[1].(string)
		return ticker, ok
	}

	if dataType == "A" && len(data) > 2 {
		ticker, ok = data[2].(string)
		return ticker, ok
	}

	return "", false
}
