package bot

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/websocket"
)

// BinanceFuturesDataProvider implements DataProvider for Binance Futures API
type BinanceFuturesDataProvider struct {
	baseURL    string
	apiKey     string
	secretKey  string
	httpClient *http.Client
	wsConn     *websocket.Conn
	wsURL      string
	running    bool
	stopChan   chan struct{}
}

// BinanceKlineData represents the response from Binance klines endpoint
type BinanceKlineData struct {
	Symbol   string     `json:"symbol"`
	Klines   [][]string `json:"klines"`
	Interval string     `json:"interval"`
}

// BinanceKline represents a single kline from Binance
type BinanceKline []interface{}

// BinanceWSMessage represents WebSocket message structure
type BinanceWSMessage struct {
	Stream string `json:"stream"`
	Data   struct {
		EventType string `json:"e"`
		EventTime int64  `json:"E"`
		Symbol    string `json:"s"`
		Kline     struct {
			OpenTime   int64  `json:"t"`
			CloseTime  int64  `json:"T"`
			Symbol     string `json:"s"`
			Interval   string `json:"i"`
			OpenPrice  string `json:"o"`
			ClosePrice string `json:"c"`
			HighPrice  string `json:"h"`
			LowPrice   string `json:"l"`
			Volume     string `json:"v"`
			IsClosed   bool   `json:"x"`
		} `json:"k"`
	} `json:"data"`
}

// NewBinanceFuturesDataProvider creates a new Binance Futures data provider
func NewBinanceFuturesDataProvider(apiKey, secretKey string) *BinanceFuturesDataProvider {
	return &BinanceFuturesDataProvider{
		baseURL:    "https://fapi.binance.com",
		wsURL:      "wss://fstream.binance.com/ws",
		apiKey:     apiKey,
		secretKey:  secretKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		stopChan:   make(chan struct{}),
	}
}

// GetHistoricalData fetches historical kline data from Binance Futures API
func (b *BinanceFuturesDataProvider) GetHistoricalData(symbol string, timeframe Timeframe, count int) ([]Candle, error) {
	// Convert symbol to Binance format (e.g., BTCUSD -> BTCUSDT)
	binanceSymbol := b.convertSymbol(symbol)

	// Convert timeframe to Binance format
	interval := b.convertTimeframe(timeframe)

	// Build URL
	endpoint := fmt.Sprintf("%s/fapi/v1/klines", b.baseURL)
	params := url.Values{}
	params.Add("symbol", binanceSymbol)
	params.Add("interval", interval)
	params.Add("limit", strconv.Itoa(count))

	// Make HTTP request
	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// API key is not required for public kline data
	// if b.apiKey != "" {
	//     req.Header.Add("X-MBX-APIKEY", b.apiKey)
	// }

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var klines [][]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &klines); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to Candle format
	candles := make([]Candle, len(klines))
	for i, kline := range klines {
		candle, err := b.convertKlineToCandle(kline, binanceSymbol)
		if err != nil {
			return nil, fmt.Errorf("failed to convert kline %d: %w", i, err)
		}
		candles[i] = candle
	}

	return candles, nil
}

// GetRealTimeData provides real-time data via WebSocket
func (b *BinanceFuturesDataProvider) GetRealTimeData(symbol string, timeframe Timeframe) (<-chan Candle, error) {
	candleChan := make(chan Candle, 100)

	// Convert symbol and timeframe to Binance format
	binanceSymbol := b.convertSymbol(symbol)
	interval := b.convertTimeframe(timeframe)

	// WebSocket stream name
	streamName := fmt.Sprintf("%s@kline_%s", strings.ToLower(binanceSymbol), interval)

	go func() {
		defer close(candleChan)

		// Connect to WebSocket
		wsURL := fmt.Sprintf("%s/%s", b.wsURL, streamName)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			fmt.Printf("WebSocket connection failed: %v\n", err)
			return
		}
		defer conn.Close()

		b.wsConn = conn
		b.running = true

		for {
			select {
			case <-b.stopChan:
				return
			default:
				// Read message
				_, message, err := conn.ReadMessage()
				if err != nil {
					fmt.Printf("WebSocket read error: %v\n", err)
					return
				}

				// Parse message
				var wsMsg BinanceWSMessage
				if err := json.Unmarshal(message, &wsMsg); err != nil {
					fmt.Printf("Failed to parse WebSocket message: %v\n", err)
					continue
				}

				// Only process completed klines
				if !wsMsg.Data.Kline.IsClosed {
					continue
				}

				// Convert to Candle
				candle, err := b.convertWSKlineToCandle(wsMsg.Data.Kline, binanceSymbol)
				if err != nil {
					fmt.Printf("Failed to convert WebSocket kline: %v\n", err)
					continue
				}

				select {
				case candleChan <- candle:
				case <-b.stopChan:
					return
				}
			}
		}
	}()

	return candleChan, nil
}

// GetCurrentPrice fetches the real-time current price from Binance ticker API
func (b *BinanceFuturesDataProvider) GetCurrentPrice(symbol string) (float64, error) {
	// Convert symbol to Binance format
	binanceSymbol := b.convertSymbol(symbol)

	// Build URL for price ticker
	endpoint := fmt.Sprintf("%s/fapi/v1/ticker/price", b.baseURL)
	params := url.Values{}
	params.Add("symbol", binanceSymbol)

	// Make HTTP request
	req, err := http.NewRequest("GET", endpoint+"?"+params.Encode(), nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := b.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API error: %s - %s", resp.Status, string(body))
	}

	// Parse response
	var tickerResp struct {
		Symbol string `json:"symbol"`
		Price  string `json:"price"`
		Time   int64  `json:"time"`
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return 0, fmt.Errorf("failed to read response: %w", err)
	}

	if err := json.Unmarshal(body, &tickerResp); err != nil {
		return 0, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert price string to float64
	price, err := strconv.ParseFloat(tickerResp.Price, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse price: %w", err)
	}

	return price, nil
}

// Close closes the data provider connection
func (b *BinanceFuturesDataProvider) Close() error {
	if b.running {
		close(b.stopChan)
		b.running = false

		if b.wsConn != nil {
			b.wsConn.Close()
		}
	}
	return nil
}

// convertSymbol converts internal symbol format to Binance format
func (b *BinanceFuturesDataProvider) convertSymbol(symbol string) string {
	// Convert BTCUSD to BTCUSDT (most common futures pairs use USDT)
	if strings.HasSuffix(symbol, "USD") && !strings.HasSuffix(symbol, "USDT") {
		return strings.TrimSuffix(symbol, "USD") + "USDT"
	}
	return symbol
}

// convertTimeframe converts internal timeframe to Binance interval format
func (b *BinanceFuturesDataProvider) convertTimeframe(timeframe Timeframe) string {
	switch timeframe {
	case FiveMinute:
		return "5m"
	case FifteenMinute:
		return "15m"
	case FortyFiveMinute:
		return "1h" // Binance doesn't have 45m, use 1h as closest
	case EightHour:
		return "8h"
	case Daily:
		return "1d"
	default:
		return "5m"
	}
}

// convertKlineToCandle converts Binance kline data to internal Candle format
func (b *BinanceFuturesDataProvider) convertKlineToCandle(kline []interface{}, symbol string) (Candle, error) {
	if len(kline) < 11 {
		return Candle{}, fmt.Errorf("invalid kline data length: %d", len(kline))
	}

	// Parse timestamp (milliseconds to seconds)
	timestampMs, ok := kline[0].(float64)
	if !ok {
		return Candle{}, fmt.Errorf("invalid timestamp type")
	}
	timestamp := time.Unix(int64(timestampMs)/1000, 0)

	// Parse price data
	open, err := strconv.ParseFloat(kline[1].(string), 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid open price: %w", err)
	}

	high, err := strconv.ParseFloat(kline[2].(string), 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid high price: %w", err)
	}

	low, err := strconv.ParseFloat(kline[3].(string), 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid low price: %w", err)
	}

	close, err := strconv.ParseFloat(kline[4].(string), 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid close price: %w", err)
	}

	volume, err := strconv.ParseFloat(kline[5].(string), 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid volume: %w", err)
	}

	return Candle{
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}, nil
}

// convertWSKlineToCandle converts WebSocket kline data to internal Candle format
func (b *BinanceFuturesDataProvider) convertWSKlineToCandle(kline struct {
	OpenTime   int64  `json:"t"`
	CloseTime  int64  `json:"T"`
	Symbol     string `json:"s"`
	Interval   string `json:"i"`
	OpenPrice  string `json:"o"`
	ClosePrice string `json:"c"`
	HighPrice  string `json:"h"`
	LowPrice   string `json:"l"`
	Volume     string `json:"v"`
	IsClosed   bool   `json:"x"`
}, symbol string) (Candle, error) {

	timestamp := time.Unix(kline.OpenTime/1000, 0)

	open, err := strconv.ParseFloat(kline.OpenPrice, 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid open price: %w", err)
	}

	high, err := strconv.ParseFloat(kline.HighPrice, 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid high price: %w", err)
	}

	low, err := strconv.ParseFloat(kline.LowPrice, 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid low price: %w", err)
	}

	close, err := strconv.ParseFloat(kline.ClosePrice, 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid close price: %w", err)
	}

	volume, err := strconv.ParseFloat(kline.Volume, 64)
	if err != nil {
		return Candle{}, fmt.Errorf("invalid volume: %w", err)
	}

	return Candle{
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     close,
		Volume:    volume,
	}, nil
}
