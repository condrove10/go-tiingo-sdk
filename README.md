[![Go Reference](https://pkg.go.dev/badge/github.com/condrove10/go-tiingo-sdk.svg)](https://pkg.go.dev/github.com/condrove10/go-tiingo-sdk)
[![Apache 2.0 License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](https://opensource.org/licenses/Apache-2.0)

# Go Tiingo SDK

This is the official Go SDK for the [Tiingo API](https://api.tiingo.com). It provides a convenient and easy-to-use interface for accessing both REST and WebSocket APIs for stock, forex, and crypto data.

## Features

-   **REST Client**: A simple, authenticated HTTP client for all Tiingo REST endpoints.
-   **WebSocket Client**: A robust, auto-reconnecting WebSocket client for real-time data feeds (IEX, Forex, Crypto).
-   **Flexible Configuration**: Use functional options to configure client behavior.
-   **Callback-Based**: Asynchronous, non-blocking handling of WebSocket messages using callbacks.
-   **Connection Management**: Automatic reconnection with exponential backoff, and health monitoring.
-   **Thread-Safe**: Safe for concurrent use.

## Installation

To use the SDK in your project, you can use `go get`:

```sh
go get github.com/condrove10/go-tiingo-sdk
```

## Usage

First, you need to get your API key from the [Tiingo website](https://api.tiingo.com).

### REST Client

The REST client provides access to all of Tiingo's REST API endpoints.

#### Example: Fetching End-of-Day Prices

This example demonstrates how to fetch daily price data for a specific stock ticker.

```go
package main

import (
	"context"
	"fmt"
	"log"

	"github.com/condrove10/go-tiingo-sdk"
)

func main() {
	apiKey := "YOUR_API_KEY"

	// Create a new REST client
	client, err := tiingo.NewRestClient(context.Background(), apiKey)
	if err != nil {
		log.Fatalf("Failed to create client: %v", err)
	}

	// Fetch end-of-day prices for Apple Inc.
	prices, err := client.GetEndOfDayPrices(context.Background(), "AAPL", &tiingo.EndOfDayPricesOptions{
		StartDate: "2023-01-01",
	})
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Printf("Success: Retrieved %d records for AAPL\n", len(prices))
}
```

### WebSocket Client

The WebSocket client allows you to subscribe to real-time data feeds for stocks (IEX), forex, and crypto.

#### Example: Subscribing to IEX Data

This example shows how to subscribe to real-time IEX data for a stock ticker.

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/condrove10/go-tiingo-sdk"
)

func main() {
	apiKey := os.Getenv("TIINGO_API_KEY")
	if apiKey == "" {
		log.Fatal("TIINGO_API_KEY environment variable not set")
	}

	// Create a new client for the IEX feed
	client := tiingo.NewWebsocketClient(
		apiKey,
		tiingo.EndpointTypeIEX,
		tiingo.WithThresholdLevel(5),
	)

	// Define an error handler for the connection
	onError := func(err error) {
		log.Printf("Connection Error: %v", err)
	}

	// Connect to the WebSocket
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
	if err := client.Connect(ctx, onError); err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer client.Close()

	// Define a subscription handler for the data
	onData := func(msg []byte, err error) {
		if err != nil {
			log.Printf("Subscription error: %v", err)
			return
		}
		log.Printf("Received data: %s", string(msg))
	}

	// Subscribe to the AAPL ticker
	if err := client.Subscribe("AAPL", onData); err != nil {
		log.Printf("Failed to subscribe to AAPL: %v", err)
	}

	// Wait for a shutdown signal to gracefully close the connection
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
```

#### Example: Handling Crypto Data

This example demonstrates how to subscribe to real-time crypto data.

```go
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/condrove10/go-tiingo-sdk"
)

func main() {
	apiKey := os.Getenv("TIINGO_API_KEY")
	if apiKey == "" {
		log.Fatal("TIINGO_API_KEY environment variable not set")
	}

	// Create a client for the Crypto feed
	client := tiingo.NewWebsocketClient(
		apiKey,
		tiingo.EndpointTypeCrypto,
		tiingo.WithThresholdLevel(2),
	)

	onError := func(err error) {
		log.Printf("Connection Error: %v", err)
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	
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

	// Wait for a shutdown signal to gracefully close the connection
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down...")
}
```

## Contributing

Contributions are welcome! If you find any issues or have suggestions for improvements, please open an issue or create a pull request.

## License

This project is licensed under the Apache 2.0 License. See the [LICENSE](LICENSE) file for more details.
