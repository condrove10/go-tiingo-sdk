# Go Tiingo SDK

A Go client for the [Tiingo API](https://api.tiingo.com), providing access to both REST and WebSocket interfaces for stock, forex, and crypto data.

## Features

-   **REST Client**: A simple, authenticated HTTP client for all Tiingo REST endpoints.
-   **WebSocket Client**: A robust, auto-reconnecting WebSocket client for real-time data feeds (IEX, Forex, Crypto).
-   **Flexible Configuration**: Use functional options to configure client behavior.
-   **Callback-Based**: Asynchronous, non-blocking handling of WebSocket messages using callbacks.
-   **Connection Management**: Automatic reconnection with exponential backoff, and health monitoring via ping/pong checks.
-   **Thread-Safe**: Safe for concurrent use.

## Installation

```sh
go get github.com/condrove10/go-tiingo-sdk
```

## Usage

### REST Client

The `Do` method provides a generic way to interact with any of the Tiingo REST endpoints, but the SDK also provides helper methods for specific endpoints.

#### Example: Fetching End-of-Day Prices

This example fetches daily price data for a ticker.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/condrove10/go-tiingo-sdk/tiingo"
)

func main() {
	apiKey := "YOUR_API_KEY"

	// Create a new REST client
	client, err := tiingo.NewRestClient(context.Background(), apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	prices, err := client.GetEndOfDayPrices(context.Background(), "AAPL", &tiingo.EndOfDayPricesOptions{
		StartDate: "2026-01-01",
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Success: Retrieved %d records\n", len(prices))
}
```

### WebSocket Client

The WebSocket client allows you to subscribe to real-time data feeds.

#### Example: Subscribing to IEX Data

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/condrove10/go-tiingo-sdk/tiingo"
)

func main() {
	apiKey := os.Getenv("TIINGO_API_KEY")
	if apiKey == "" {
		log.Fatal("TIINGO_API_KEY environment variable not set")
	}

	// Create a new client for IEX feed
	client := tiingo.NewWebsocketClient(
		apiKey,
		tiingo.EndpointTypeIEX,
		tiingo.WithThresholdLevel(5),
	)

	// Define error handler for the connection
	onError := func(err error) {
		log.Printf("Connection Error: %v", err)
	}

	// Connect
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := client.Connect(ctx, onError); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Define subscription handler
	onData := func(msg []byte, err error) {
		if err != nil {
			log.Printf("Subscription error: %v", err)
			return
		}
		log.Printf("Received data: %s", string(msg))
	}

	// Subscribe to tickers
	if err := client.Subscribe("AAPL", onData); err != nil {
		log.Printf("Failed to subscribe to AAPL: %v", err)
	}

	// Wait for a signal to gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
```

#### Handling Crypto Data

```go
package main

import (
	"context"
	"log"
	"os"

	"github.com/condrove10/go-tiingo-sdk/tiingo"
)

func main() {
	apiKey := os.Getenv("TIINGO_API_KEY")

	// Create a client for the Crypto feed
	client := tiingo.NewWebsocketClient(
		apiKey,
		tiingo.EndpointTypeCrypto,
		tiingo.WithThresholdLevel(2),
	)

	onError := func(err error) {
		log.Printf("Connection Error: %v", err)
	}
	
	ctx := context.Background()
	if err := client.Connect(ctx, onError); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	onData := func(msg []byte, err error) {
		if err != nil {
			log.Printf("Subscription error: %v", err)
			return
		}
		log.Printf("Received Crypto Data: %s", string(msg))
	}

	if err := client.Subscribe("btcusd", onData); err != nil {
		log.Printf("Failed to subscribe: %v", err)
	}

	// ... wait for shutdown signal
}
```
