package bot

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

// DataProvider interface for market data sources
type DataProvider interface {
	GetHistoricalData(symbol string, timeframe Timeframe, count int) ([]Candle, error)
	GetRealTimeData(symbol string, timeframe Timeframe) (<-chan Candle, error)
	Close() error
}

// RealTimeConfig configures real-time data behavior
type RealTimeConfig struct {
	TickInterval    time.Duration // How often to generate price ticks
	CandleInterval  time.Duration // How often to emit completed candles
	EnableDebugLogs bool          // Whether to show debug output
}

// DefaultRealTimeConfigs provides realistic configurations for different timeframes
var DefaultRealTimeConfigs = map[Timeframe]RealTimeConfig{
	FiveMinute: {
		TickInterval:    time.Second * 5, // 5-second price updates
		CandleInterval:  time.Minute * 5, // 5-minute candles
		EnableDebugLogs: false,
	},
	FifteenMinute: {
		TickInterval:    time.Second * 15, // 15-second price updates
		CandleInterval:  time.Minute * 15, // 15-minute candles
		EnableDebugLogs: false,
	},
	FortyFiveMinute: {
		TickInterval:    time.Minute * 1,  // 1-minute price updates
		CandleInterval:  time.Minute * 45, // 45-minute candles
		EnableDebugLogs: false,
	},
	EightHour: {
		TickInterval:    time.Minute * 5, // 5-minute price updates
		CandleInterval:  time.Hour * 8,   // 8-hour candles
		EnableDebugLogs: false,
	},
	Daily: {
		TickInterval:    time.Minute * 15, // 15-minute price updates
		CandleInterval:  time.Hour * 24,   // Daily candles
		EnableDebugLogs: false,
	},
}

// CandleBuilder aggregates ticks into candles
type CandleBuilder struct {
	timeframe     Timeframe
	currentCandle *Candle
	startTime     time.Time
	mutex         sync.RWMutex
}

// NewCandleBuilder creates a new candle builder
func NewCandleBuilder(timeframe Timeframe) *CandleBuilder {
	now := time.Now()
	return &CandleBuilder{
		timeframe: timeframe,
		startTime: now.Truncate(timeframe.Duration()),
	}
}

// AddTick adds a price tick to the current candle
func (cb *CandleBuilder) AddTick(price, volume float64) {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	if cb.currentCandle == nil {
		// Start new candle
		cb.currentCandle = &Candle{
			Timestamp: cb.startTime,
			Open:      price,
			High:      price,
			Low:       price,
			Close:     price,
			Volume:    volume,
		}
	} else {
		// Update current candle
		cb.currentCandle.High = math.Max(cb.currentCandle.High, price)
		cb.currentCandle.Low = math.Min(cb.currentCandle.Low, price)
		cb.currentCandle.Close = price
		cb.currentCandle.Volume += volume
	}
}

// GetCompletedCandle returns completed candle if timeframe elapsed
func (cb *CandleBuilder) GetCompletedCandle() *Candle {
	cb.mutex.Lock()
	defer cb.mutex.Unlock()

	now := time.Now()
	if now.Sub(cb.startTime) >= cb.timeframe.Duration() && cb.currentCandle != nil {
		completed := *cb.currentCandle
		cb.currentCandle = nil
		cb.startTime = now.Truncate(cb.timeframe.Duration())
		return &completed
	}
	return nil
}

// SampleDataProvider generates sample market data for testing
type SampleDataProvider struct {
	symbols        []string
	basePrice      float64
	currentPrice   float64
	volatility     float64
	trendStrength  float64
	running        bool
	stopChan       chan struct{}
	config         RealTimeConfig
	candleBuilders map[Timeframe]*CandleBuilder
	mutex          sync.RWMutex
}

// NewSampleDataProvider creates a new sample data provider
func NewSampleDataProvider(symbols []string, basePrice float64) *SampleDataProvider {
	return &SampleDataProvider{
		symbols:        symbols,
		basePrice:      basePrice,
		currentPrice:   basePrice,
		volatility:     0.02,  // 2% volatility
		trendStrength:  0.001, // 0.1% trend per candle
		stopChan:       make(chan struct{}),
		candleBuilders: make(map[Timeframe]*CandleBuilder),
	}
}

// NewDemoDataProvider creates a sample data provider with debug logs enabled
func NewDemoDataProvider(symbols []string, basePrice float64) *SampleDataProvider {
	provider := NewSampleDataProvider(symbols, basePrice)

	// Enable debug logs for 5-minute timeframe for demo purposes
	provider.EnableDebugLogs(FiveMinute, true)

	return provider
}

// SetRealTimeConfig configures real-time data behavior
func (sdp *SampleDataProvider) SetRealTimeConfig(timeframe Timeframe, config RealTimeConfig) {
	sdp.mutex.Lock()
	defer sdp.mutex.Unlock()
	sdp.config = config
}

// generatePriceTick generates a single price tick
func (sdp *SampleDataProvider) generatePriceTick() float64 {
	// Generate realistic price movement
	changePercent := (rand.Float64() - 0.5) * sdp.volatility * 0.1 // Smaller movements for ticks

	// Add some trend
	trend := sdp.trendStrength * 0.1 * (rand.Float64() - 0.3) // Smaller trend for ticks
	changePercent += trend

	// Calculate new price
	newPrice := sdp.currentPrice * (1 + changePercent)
	return newPrice
}

// generateVolume generates a volume amount for a tick
func (sdp *SampleDataProvider) generateVolume() float64 {
	baseVolume := 100.0 // Smaller base volume for ticks
	volumeMultiplier := 0.5 + rand.Float64()
	return baseVolume * volumeMultiplier
}

// GetHistoricalData generates historical candle data
func (sdp *SampleDataProvider) GetHistoricalData(symbol string, timeframe Timeframe, count int) ([]Candle, error) {
	candles := make([]Candle, count)

	// Start from some time in the past
	startTime := time.Now().Add(-time.Duration(count) * timeframe.Duration())
	price := sdp.basePrice

	for i := 0; i < count; i++ {
		timestamp := startTime.Add(time.Duration(i) * timeframe.Duration())

		// Generate realistic OHLCV data
		candle := sdp.generateCandle(timestamp, price, timeframe)
		candles[i] = candle

		// Update price for next candle
		price = candle.Close
	}

	return candles, nil
}

// GetRealTimeData provides real-time market data simulation with proper candle aggregation
func (sdp *SampleDataProvider) GetRealTimeData(symbol string, timeframe Timeframe) (<-chan Candle, error) {
	candleChan := make(chan Candle, 10)

	// Get configuration for this timeframe
	config, exists := DefaultRealTimeConfigs[timeframe]
	if !exists {
		config = RealTimeConfig{
			TickInterval:    time.Second * 5,
			CandleInterval:  timeframe.Duration(),
			EnableDebugLogs: false,
		}
	}

	// Create candle builder for this timeframe
	sdp.mutex.Lock()
	candleBuilder := NewCandleBuilder(timeframe)
	sdp.candleBuilders[timeframe] = candleBuilder
	sdp.mutex.Unlock()

	go func() {
		defer close(candleChan)

		// Price tick timer
		tickTicker := time.NewTicker(config.TickInterval)
		defer tickTicker.Stop()

		// Candle completion check timer
		candleTicker := time.NewTicker(time.Second * 10) // Check every 10 seconds
		defer candleTicker.Stop()

		sdp.running = true

		if config.EnableDebugLogs {
			fmt.Printf("Starting %s real-time data: ticks every %v, candles every %v\n",
				timeframe.String(), config.TickInterval, config.CandleInterval)
		}

		for {
			select {
			case <-tickTicker.C:
				// Generate price tick
				newPrice := sdp.generatePriceTick()
				volume := sdp.generateVolume()

				// Add tick to candle builder
				candleBuilder.AddTick(newPrice, volume)
				sdp.currentPrice = newPrice

				if config.EnableDebugLogs {
					fmt.Printf("%s tick: $%.2f\n", timeframe.String(), newPrice)
				}

			case <-candleTicker.C:
				// Check for completed candle
				if completedCandle := candleBuilder.GetCompletedCandle(); completedCandle != nil {
					if config.EnableDebugLogs {
						fmt.Printf("%s candle completed: O:%.2f H:%.2f L:%.2f C:%.2f V:%.0f\n",
							timeframe.String(), completedCandle.Open, completedCandle.High,
							completedCandle.Low, completedCandle.Close, completedCandle.Volume)
					}

					select {
					case candleChan <- *completedCandle:
					case <-sdp.stopChan:
						return
					}
				}

			case <-sdp.stopChan:
				return
			}
		}
	}()

	return candleChan, nil
}

// generateCandle creates a realistic candle with OHLCV data
func (sdp *SampleDataProvider) generateCandle(timestamp time.Time, startPrice float64, timeframe Timeframe) Candle {
	// Generate realistic price movement
	changePercent := (rand.Float64() - 0.5) * sdp.volatility

	// Add some trend
	trend := sdp.trendStrength * (rand.Float64() - 0.3) // Slight upward bias
	changePercent += trend

	// Calculate price levels
	open := startPrice
	closePrice := open * (1 + changePercent)

	// Generate high and low
	highChange := rand.Float64() * sdp.volatility * 0.5
	lowChange := -rand.Float64() * sdp.volatility * 0.5

	high := math.Max(open, closePrice) * (1 + highChange)
	low := math.Min(open, closePrice) * (1 + lowChange)

	// Ensure high is highest and low is lowest
	if high < math.Max(open, closePrice) {
		high = math.Max(open, closePrice)
	}
	if low > math.Min(open, closePrice) {
		low = math.Min(open, closePrice)
	}

	// Generate volume (higher volume on bigger moves)
	volumeBase := 10000.0
	volumeMultiplier := 1.0 + math.Abs(changePercent)*10
	volume := volumeBase * volumeMultiplier * (0.5 + rand.Float64())

	return Candle{
		Timestamp: timestamp,
		Open:      open,
		High:      high,
		Low:       low,
		Close:     closePrice,
		Volume:    volume,
	}
}

// EnableDebugLogs enables or disables debug logging for a specific timeframe
func (sdp *SampleDataProvider) EnableDebugLogs(timeframe Timeframe, enabled bool) {
	if config, exists := DefaultRealTimeConfigs[timeframe]; exists {
		config.EnableDebugLogs = enabled
		DefaultRealTimeConfigs[timeframe] = config
	}
}

// Close stops the data provider
func (sdp *SampleDataProvider) Close() error {
	if sdp.running {
		close(sdp.stopChan)
		sdp.running = false

		// Clean up candle builders
		sdp.mutex.Lock()
		sdp.candleBuilders = make(map[Timeframe]*CandleBuilder)
		sdp.mutex.Unlock()
	}
	return nil
}

// APIDataProvider for connecting to real market data APIs
type APIDataProvider struct {
	apiKey    string
	baseURL   string
	symbols   []string
	rateLimit time.Duration
	lastCall  time.Time
}

// NewAPIDataProvider creates a new API data provider
func NewAPIDataProvider(apiKey, baseURL string, symbols []string) *APIDataProvider {
	return &APIDataProvider{
		apiKey:    apiKey,
		baseURL:   baseURL,
		symbols:   symbols,
		rateLimit: time.Second, // 1 second rate limit
	}
}

// GetHistoricalData fetches historical data from API
func (adp *APIDataProvider) GetHistoricalData(symbol string, timeframe Timeframe, count int) ([]Candle, error) {
	// Rate limiting
	if time.Since(adp.lastCall) < adp.rateLimit {
		time.Sleep(adp.rateLimit - time.Since(adp.lastCall))
	}
	adp.lastCall = time.Now()

	// TODO: Implement actual API calls
	// For now, return sample data with realistic prices
	var basePrice float64
	switch symbol {
	case "BTCUSDT":
		basePrice = 50000.0 // Realistic Bitcoin price
	case "ETHUSDT":
		basePrice = 3000.0 // Realistic Ethereum price
	case "BNBUSDT":
		basePrice = 300.0 // Realistic BNB price
	default:
		basePrice = 100.0 // Default for other symbols
	}

	sampleProvider := NewSampleDataProvider([]string{symbol}, basePrice)
	return sampleProvider.GetHistoricalData(symbol, timeframe, count)
}

// GetRealTimeData provides real-time data from API
func (adp *APIDataProvider) GetRealTimeData(symbol string, timeframe Timeframe) (<-chan Candle, error) {
	// TODO: Implement WebSocket or polling for real-time data
	// For now, return sample data with realistic prices
	var basePrice float64
	switch symbol {
	case "BTCUSDT":
		basePrice = 50000.0 // Realistic Bitcoin price
	case "ETHUSDT":
		basePrice = 3000.0 // Realistic Ethereum price
	case "BNBUSDT":
		basePrice = 300.0 // Realistic BNB price
	default:
		basePrice = 100.0 // Default for other symbols
	}

	sampleProvider := NewSampleDataProvider([]string{symbol}, basePrice)
	return sampleProvider.GetRealTimeData(symbol, timeframe)
}

// Close closes the API connection
func (adp *APIDataProvider) Close() error {
	// TODO: Implement cleanup
	return nil
}

// DataProviderManager manages multiple data providers
type DataProviderManager struct {
	providers map[string]DataProvider
	primary   DataProvider
}

// NewDataProviderManager creates a new data provider manager
func NewDataProviderManager() *DataProviderManager {
	return &DataProviderManager{
		providers: make(map[string]DataProvider),
	}
}

// AddProvider adds a data provider
func (dpm *DataProviderManager) AddProvider(name string, provider DataProvider) {
	dpm.providers[name] = provider
	if dpm.primary == nil {
		dpm.primary = provider
	}
}

// SetPrimary sets the primary data provider
func (dpm *DataProviderManager) SetPrimary(name string) error {
	provider, exists := dpm.providers[name]
	if !exists {
		return fmt.Errorf("provider %s not found", name)
	}
	dpm.primary = provider
	return nil
}

// GetHistoricalData gets historical data from primary provider
func (dpm *DataProviderManager) GetHistoricalData(symbol string, timeframe Timeframe, count int) ([]Candle, error) {
	if dpm.primary == nil {
		return nil, fmt.Errorf("no primary provider set")
	}
	return dpm.primary.GetHistoricalData(symbol, timeframe, count)
}

// GetRealTimeData gets real-time data from primary provider
func (dpm *DataProviderManager) GetRealTimeData(symbol string, timeframe Timeframe) (<-chan Candle, error) {
	if dpm.primary == nil {
		return nil, fmt.Errorf("no primary provider set")
	}
	return dpm.primary.GetRealTimeData(symbol, timeframe)
}

// Close closes all providers
func (dpm *DataProviderManager) Close() error {
	for _, provider := range dpm.providers {
		if err := provider.Close(); err != nil {
			return err
		}
	}
	return nil
}

// LoadHistoricalDataForAllTimeframes loads data for all required timeframes
func (dpm *DataProviderManager) LoadHistoricalDataForAllTimeframes(symbol string, tm *TimeframeManager) error {
	timeframes := []Timeframe{Daily, EightHour, FortyFiveMinute, FifteenMinute, FiveMinute}

	for _, timeframe := range timeframes {
		var count int
		switch timeframe {
		case Daily:
			count = 30
		case EightHour:
			count = 50
		case FortyFiveMinute:
			count = 60
		case FifteenMinute:
			count = 80
		case FiveMinute:
			count = 100
		}

		candles, err := dpm.GetHistoricalData(symbol, timeframe, count)
		if err != nil {
			return fmt.Errorf("failed to load %s data: %w", timeframe.String(), err)
		}

		// Add all candles to timeframe manager
		for _, candle := range candles {
			tm.AddCandle(timeframe, candle)
		}
	}

	return nil
}

// StartRealTimeDataFeeds starts real-time data feeds for all timeframes
func (dpm *DataProviderManager) StartRealTimeDataFeeds(symbol string, tm *TimeframeManager) error {
	timeframes := []Timeframe{Daily, EightHour, FortyFiveMinute, FifteenMinute, FiveMinute}

	for _, timeframe := range timeframes {
		candleChan, err := dpm.GetRealTimeData(symbol, timeframe)
		if err != nil {
			return fmt.Errorf("failed to start %s real-time feed: %w", timeframe.String(), err)
		}

		// Start goroutine to handle incoming candles
		go func(tf Timeframe, ch <-chan Candle) {
			for candle := range ch {
				tm.AddCandle(tf, candle)
			}
		}(timeframe, candleChan)
	}

	return nil
}
