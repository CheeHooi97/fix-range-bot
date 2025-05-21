package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

type TradeConfig struct {
	Symbol     string
	MinPrice   float64
	MaxPrice   float64
	TakeProfit float64 // In percentage (e.g., 1.5 for 1.5%)
	StopLoss   float64 // In percentage
}

var (
	config         TradeConfig
	entryPrice     float64
	positionOpen   bool
	isLongPosition bool
)

func main() {
	promptConfig()
	startWebSocket()
}

func promptConfig() {
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("Symbol (e.g. btcusdt): ")
	scanner.Scan()
	config.Symbol = strings.ToLower(scanner.Text())

	fmt.Print("Min Price: ")
	config.MinPrice = scanFloat(scanner)

	fmt.Print("Max Price: ")
	config.MaxPrice = scanFloat(scanner)

	fmt.Print("Take Profit (%): ")
	config.TakeProfit = scanFloat(scanner)

	fmt.Print("Stop Loss (%): ")
	config.StopLoss = scanFloat(scanner)
}

func scanFloat(scanner *bufio.Scanner) float64 {
	scanner.Scan()
	value, err := strconv.ParseFloat(scanner.Text(), 64)
	if err != nil {
		log.Fatalf("Invalid number: %v", err)
	}
	return value
}

func startWebSocket() {
	interval := "15m"

	// Connect to Binance WebSocket for kline data
	wsURL := fmt.Sprintf("wss://stream.binance.com:9443/ws/%s@kline_%s", config.Symbol, interval)
	log.Printf("Connecting to %s", wsURL)

	c, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		log.Fatal("WebSocket connection error:", err)
	}
	defer c.Close()

	log.Println("Connected to Binance WebSocket")
	for {
		_, message, err := c.ReadMessage()
		if err != nil {
			log.Println("Read error:", err)
			time.Sleep(time.Second * 3)
			continue
		}
		handleTradeMessage(message)
	}
}

func handleTradeMessage(message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		log.Println("JSON error:", err)
		return
	}

	priceStr := msg["p"].(string)
	price, _ := strconv.ParseFloat(priceStr, 64)
	handlePrice(price)
}

func handlePrice(price float64) {
	if positionOpen {
		change := ((price - entryPrice) / entryPrice) * 100

		if isLongPosition {
			if change >= config.TakeProfit {
				closePosition("Take Profit", price)
			} else if change <= -config.StopLoss {
				closePosition("Stop Loss", price)
			}
		} else {
			if change <= -config.TakeProfit {
				closePosition("Take Profit", price)
			} else if change >= config.StopLoss {
				closePosition("Stop Loss", price)
			}
		}
		return
	}

	margin := 0.1 // $0.10 tolerance
	if math.Abs(price-config.MinPrice) < margin {
		openPosition(price, true) // Long
	} else if math.Abs(price-config.MaxPrice) < margin {
		openPosition(price, false) // Short
	}
}

func openPosition(price float64, long bool) {
	entryPrice = price
	positionOpen = true
	isLongPosition = long
	typeStr := "LONG"
	if !long {
		typeStr = "SHORT"
	}
	log.Printf("Opened %s position at %.2f", typeStr, price)
}

func closePosition(reason string, price float64) {
	log.Printf("Closed position at %.2f due to %s", price, reason)
	positionOpen = false
}
