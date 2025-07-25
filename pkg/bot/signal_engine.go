package bot

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"
)

// SignalEngine orchestrates all components of the trading bot
type SignalEngine struct {
	config           Config
	timeframeManager *TimeframeManager
	dataProvider     *DataProviderManager
	signalAggregator *SignalAggregator
	signalChan       chan *TradingSignal
	errorChan        chan error
	stopChan         chan struct{}
	running          bool
	mutex            sync.RWMutex
	lastSignal       *TradingSignal
}

// NewSignalEngine creates a new signal engine
func NewSignalEngine(config Config) *SignalEngine {
	return &SignalEngine{
		config:           config,
		timeframeManager: NewTimeframeManager(config.Symbol),
		dataProvider:     NewDataProviderManager(),
		signalAggregator: NewSignalAggregator(config),
		signalChan:       make(chan *TradingSignal, 100),
		errorChan:        make(chan error, 10),
		stopChan:         make(chan struct{}),
		running:          false,
	}
}

// Start initializes and starts the signal engine
func (se *SignalEngine) Start(ctx context.Context) error {
	se.mutex.Lock()
	defer se.mutex.Unlock()

	if se.running {
		return fmt.Errorf("signal engine is already running")
	}

	// Initialize data provider
	if err := se.initializeDataProvider(); err != nil {
		return fmt.Errorf("failed to initialize data provider: %w", err)
	}

	// Load historical data
	if err := se.loadHistoricalData(); err != nil {
		return fmt.Errorf("failed to load historical data: %w", err)
	}

	// Wait for sufficient data
	if err := se.waitForDataReady(ctx); err != nil {
		return fmt.Errorf("insufficient data: %w", err)
	}

	// Start real-time data feeds
	if err := se.startRealTimeFeeds(); err != nil {
		return fmt.Errorf("failed to start real-time feeds: %w", err)
	}

	// Start signal generation
	se.startSignalGeneration(ctx)

	se.running = true
	log.Printf("Signal engine started for symbol: %s", se.config.Symbol)
	return nil
}

// Stop gracefully shuts down the signal engine
func (se *SignalEngine) Stop() error {
	se.mutex.Lock()
	defer se.mutex.Unlock()

	if !se.running {
		return nil
	}

	close(se.stopChan)
	se.running = false

	// Close data providers
	if err := se.dataProvider.Close(); err != nil {
		return fmt.Errorf("failed to close data provider: %w", err)
	}

	log.Printf("Signal engine stopped for symbol: %s", se.config.Symbol)
	return nil
}

// GetSignalChannel returns the channel for receiving trading signals
func (se *SignalEngine) GetSignalChannel() <-chan *TradingSignal {
	return se.signalChan
}

// GetErrorChannel returns the channel for receiving errors
func (se *SignalEngine) GetErrorChannel() <-chan error {
	return se.errorChan
}

// GetLastSignal returns the most recent trading signal
func (se *SignalEngine) GetLastSignal() *TradingSignal {
	se.mutex.RLock()
	defer se.mutex.RUnlock()
	return se.lastSignal
}

// GetStatus returns the current status of the signal engine
func (se *SignalEngine) GetStatus() SignalEngineStatus {
	se.mutex.RLock()
	defer se.mutex.RUnlock()

	return SignalEngineStatus{
		Running:     se.running,
		Symbol:      se.config.Symbol,
		DataSummary: se.timeframeManager.GetDataSummary(),
		ReadyStatus: se.timeframeManager.GetReadyStatus(),
		LastSignal:  se.lastSignal,
		LastUpdate:  time.Now(),
	}
}

// initializeDataProvider sets up the data provider
func (se *SignalEngine) initializeDataProvider() error {
	// Add sample data provider for testing
	// Use realistic base prices for different symbols
	var basePrice float64
	switch se.config.Symbol {
	case "BTCUSDT":
		basePrice = 50000.0 // Realistic Bitcoin price
	case "ETHUSDT":
		basePrice = 3000.0 // Realistic Ethereum price
	case "BNBUSDT":
		basePrice = 300.0 // Realistic BNB price
	default:
		basePrice = 100.0 // Default for other symbols
	}

	sampleProvider := NewSampleDataProvider([]string{se.config.Symbol}, basePrice)
	se.dataProvider.AddProvider("sample", sampleProvider)

	// Add Binance data provider if configured
	if se.config.DataProvider == "binance" {
		binanceProvider := NewBinanceFuturesDataProvider(se.config.Binance.APIKey, se.config.Binance.SecretKey)
		se.dataProvider.AddProvider("binance", binanceProvider)

		// Set Binance as primary if configured
		log.Printf("Using Binance Futures API for data provider")
		return se.dataProvider.SetPrimary("binance")
	}

	// Default to sample provider
	log.Printf("Using sample data provider for testing")
	return se.dataProvider.SetPrimary("sample")
}

// loadHistoricalData loads historical market data for all timeframes
func (se *SignalEngine) loadHistoricalData() error {
	log.Printf("Loading historical data for %s...", se.config.Symbol)

	return se.dataProvider.LoadHistoricalDataForAllTimeframes(se.config.Symbol, se.timeframeManager)
}

// waitForDataReady waits until sufficient data is available
func (se *SignalEngine) waitForDataReady(ctx context.Context) error {
	log.Printf("Waiting for sufficient data...")

	timeout := time.NewTimer(30 * time.Second)
	defer timeout.Stop()

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-timeout.C:
			return fmt.Errorf("timeout waiting for data")
		case <-ticker.C:
			if se.timeframeManager.IsReady() {
				log.Printf("Data ready for all timeframes")
				return nil
			}
		}
	}
}

// startRealTimeFeeds starts real-time data feeds
func (se *SignalEngine) startRealTimeFeeds() error {
	log.Printf("Starting real-time data feeds for %s...", se.config.Symbol)

	return se.dataProvider.StartRealTimeDataFeeds(se.config.Symbol, se.timeframeManager)
}

// startSignalGeneration starts the signal generation process
func (se *SignalEngine) startSignalGeneration(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(1 * time.Minute) // Generate signals every minute
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-se.stopChan:
				return
			case <-ticker.C:
				se.generateSignal()
			}
		}
	}()
}

// generateSignal creates a new trading signal
func (se *SignalEngine) generateSignal() {
	// Get multi-timeframe context
	ctx, err := se.timeframeManager.GetMultiTimeframeContext()
	if err != nil {
		se.errorChan <- fmt.Errorf("failed to get multi-timeframe context: %w", err)
		return
	}

	// Generate signal
	signal, err := se.signalAggregator.GenerateSignal(ctx)
	if err != nil {
		se.errorChan <- fmt.Errorf("failed to generate signal: %w", err)
		return
	}

	// Update last signal
	se.mutex.Lock()
	se.lastSignal = signal
	se.mutex.Unlock()

	// Send signal to channel
	select {
	case se.signalChan <- signal:
		log.Printf("Generated signal: %s (%.2f confidence) - %s",
			signal.Signal.String(), signal.Confidence, signal.Reasoning)
	default:
		// Channel is full, skip this signal
		log.Printf("Signal channel full, skipping signal")
	}
}

// SignalEngineStatus represents the current status of the signal engine
type SignalEngineStatus struct {
	Running     bool               `json:"running"`
	Symbol      string             `json:"symbol"`
	DataSummary map[Timeframe]int  `json:"data_summary"`
	ReadyStatus map[Timeframe]bool `json:"ready_status"`
	LastSignal  *TradingSignal     `json:"last_signal"`
	LastUpdate  time.Time          `json:"last_update"`
}

// TradingBot is the main trading bot that uses the signal engine
type TradingBot struct {
	config        Config
	signalEngine  *SignalEngine
	tradeExecutor *TradeExecutor // Pine Script ATR strategy trading engine
	ctx           context.Context
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// NewTradingBot creates a new trading bot
func NewTradingBot(config Config) *TradingBot {
	ctx, cancel := context.WithCancel(context.Background())

	// Create trade executor with initial balance (default: $10,000 for testing)
	initialBalance := 10000.0 // $10,000 demo balance
	tradeExecutor := NewTradeExecutor(config, initialBalance)

	return &TradingBot{
		config:        config,
		signalEngine:  NewSignalEngine(config),
		tradeExecutor: tradeExecutor,
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Start starts the trading bot
func (tb *TradingBot) Start() error {
	log.Printf("Starting trading bot for symbol: %s", tb.config.Symbol)

	// Start signal engine
	if err := tb.signalEngine.Start(tb.ctx); err != nil {
		return fmt.Errorf("failed to start signal engine: %w", err)
	}

	// Start signal handler
	tb.wg.Add(1)
	go tb.handleSignals()

	// Start error handler
	tb.wg.Add(1)
	go tb.handleErrors()

	return nil
}

// Stop stops the trading bot
func (tb *TradingBot) Stop() error {
	log.Printf("Stopping trading bot...")

	// Cancel context
	tb.cancel()

	// Stop signal engine
	if err := tb.signalEngine.Stop(); err != nil {
		return fmt.Errorf("failed to stop signal engine: %w", err)
	}

	// Wait for goroutines to finish
	tb.wg.Wait()

	log.Printf("Trading bot stopped")
	return nil
}

// GetStatus returns the current status
func (tb *TradingBot) GetStatus() SignalEngineStatus {
	return tb.signalEngine.GetStatus()
}

// GetLastSignal returns the most recent trading signal
func (tb *TradingBot) GetLastSignal() *TradingSignal {
	return tb.signalEngine.GetLastSignal()
}

// GetCurrentPrice returns the real-time current market price
func (tb *TradingBot) GetCurrentPrice() (float64, error) {
	if tb.signalEngine == nil {
		return 0, fmt.Errorf("signal engine not initialized")
	}

	// Try to get real-time price from Binance provider
	if tb.config.DataProvider == "binance" && tb.signalEngine.dataProvider.primary != nil {
		if binanceProvider, ok := tb.signalEngine.dataProvider.primary.(*BinanceFuturesDataProvider); ok {
			if price, err := binanceProvider.GetCurrentPrice(tb.config.Symbol); err == nil {
				return price, nil
			}
		}
	}

	// Fallback to latest candle data if real-time price unavailable
	if tb.signalEngine.timeframeManager == nil {
		return 0, fmt.Errorf("timeframe manager not initialized")
	}
	return tb.signalEngine.timeframeManager.GetCurrentPrice()
}

// EnsureDataAvailable ensures all required timeframes have sufficient data, fetching on-demand if needed
func (tb *TradingBot) EnsureDataAvailable() error {
	if tb.signalEngine == nil {
		return fmt.Errorf("signal engine not initialized")
	}

	// Check if data is already available
	if tb.signalEngine.timeframeManager.IsReady() {
		return nil // Data already available
	}

	// Initialize data provider if not already done
	if tb.signalEngine.dataProvider.primary == nil {
		if err := tb.signalEngine.initializeDataProvider(); err != nil {
			return fmt.Errorf("failed to initialize data provider: %w", err)
		}
	}

	// Load historical data for all timeframes
	log.Printf("Fetching historical data on-demand for %s...", tb.config.Symbol)
	if err := tb.signalEngine.dataProvider.LoadHistoricalDataForAllTimeframes(tb.config.Symbol, tb.signalEngine.timeframeManager); err != nil {
		return fmt.Errorf("failed to load historical data: %w", err)
	}

	return nil
}

// 🔥 NEW: ForceFreshDataUpdate ensures fresh Binance data for each prediction call
func (tb *TradingBot) ForceFreshDataUpdate() error {
	if tb.signalEngine == nil {
		return fmt.Errorf("signal engine not initialized")
	}

	// Initialize data provider if not already done
	if tb.signalEngine.dataProvider.primary == nil {
		if err := tb.signalEngine.initializeDataProvider(); err != nil {
			return fmt.Errorf("failed to initialize data provider: %w", err)
		}
	}

	// FORCE fresh data fetch from Binance (bypass cache)
	log.Printf("🔄 FORCING fresh Binance data update for %s...", tb.config.Symbol)
	if err := tb.signalEngine.dataProvider.LoadHistoricalDataForAllTimeframes(tb.config.Symbol, tb.signalEngine.timeframeManager); err != nil {
		return fmt.Errorf("failed to fetch fresh Binance data: %w", err)
	}

	// Validate we have sufficient data after update
	if !tb.signalEngine.timeframeManager.IsReady() {
		return fmt.Errorf("insufficient data after fresh fetch")
	}

	log.Printf("✅ Fresh Binance data loaded successfully")
	return nil
}

// GenerateImmediatePrediction generates a trading signal immediately using available or freshly fetched data
func (tb *TradingBot) GenerateImmediatePrediction() (*TradingSignal, error) {
	if tb.signalEngine == nil {
		return nil, fmt.Errorf("signal engine not initialized")
	}

	// 🔥 FORCE fresh Binance data update for each prediction call
	if err := tb.ForceFreshDataUpdate(); err != nil {
		return nil, fmt.Errorf("failed to fetch fresh Binance data: %w", err)
	}

	// Generate signal SYNCHRONOUSLY for API (not the async version used by real-time engine)
	ctx, err := tb.signalEngine.timeframeManager.GetMultiTimeframeContext()
	if err != nil {
		return nil, fmt.Errorf("failed to get multi-timeframe context: %w", err)
	}

	// Generate fresh signal directly using signal aggregator with fresh data
	signal, err := tb.signalEngine.signalAggregator.GenerateSignal(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to generate fresh signal: %w", err)
	}

	log.Printf("🎯 Generated fresh prediction with latest Binance data - Signal: %s, Confidence: %.1f%%",
		signal.Signal.String(), signal.Confidence*100)

	return signal, nil
}

// handleSignals processes incoming trading signals
func (tb *TradingBot) handleSignals() {
	defer tb.wg.Done()

	for {
		select {
		case <-tb.ctx.Done():
			return
		case signal := <-tb.signalEngine.GetSignalChannel():
			tb.processSignal(signal)
		}
	}
}

// handleErrors processes errors from the signal engine
func (tb *TradingBot) handleErrors() {
	defer tb.wg.Done()

	for {
		select {
		case <-tb.ctx.Done():
			return
		case err := <-tb.signalEngine.GetErrorChannel():
			log.Printf("Signal engine error: %v", err)
		}
	}
}

// processSignal handles a trading signal and executes trades
func (tb *TradingBot) processSignal(signal *TradingSignal) {
	// Log the signal
	log.Printf("📊 SIGNAL: %s %s", signal.Symbol, signal.Signal.String())
	log.Printf("   Confidence: %.2f%%", signal.Confidence*100)
	log.Printf("   Reasoning: %s", signal.Reasoning)

	if signal.TargetPrice > 0 {
		log.Printf("   Target: %.2f", signal.TargetPrice)
	}
	if signal.StopLoss > 0 {
		log.Printf("   Stop Loss: %.2f", signal.StopLoss)
	}

	// Print individual indicator signals
	log.Printf("   Indicators:")
	for _, indSig := range signal.IndicatorSignals {
		log.Printf("     %s: %s (%.2f)", indSig.Name, indSig.Signal.String(), indSig.Strength)
	}

	// Get current price for trade execution
	currentPrice, err := tb.GetCurrentPrice()
	if err != nil {
		log.Printf("❌ Failed to get current price: %v", err)
		return
	}

	// Get ATR trailing stop value
	var atrTrailStop float64 = 0

	// Get ATR indicator from timeframe manager to get trailing stop value
	if tb.signalEngine.signalAggregator != nil {
		// Look for ATR indicator in the signal
		for _, indSig := range signal.IndicatorSignals {
			if indSig.Name == "ATR_5m" {
				atrTrailStop = indSig.Value // Use ATR indicator value as trailing stop
				break
			}
		}
	}

	// If no ATR trailing stop found, calculate basic stop based on current price
	if atrTrailStop == 0 {
		// Default stop loss calculation: current price ± (ATR multiplier × estimated volatility)
		estimatedVolatility := currentPrice * 0.02 // 2% estimated volatility
		if signal.Signal == Buy {
			atrTrailStop = currentPrice - (tb.config.ATR.Multiplier * estimatedVolatility)
		} else if signal.Signal == Sell {
			atrTrailStop = currentPrice + (tb.config.ATR.Multiplier * estimatedVolatility)
		}
	}

	// Execute trade via Pine Script ATR strategy
	if err := tb.tradeExecutor.ExecuteSignal(signal, currentPrice, atrTrailStop); err != nil {
		log.Printf("❌ Trade execution failed: %v", err)
	}

	// Log current trading status
	position := tb.tradeExecutor.GetCurrentPosition()
	if position != nil {
		log.Printf("📍 Current Position: %s %.6f @ $%.2f (PnL: $%.2f)",
			position.Side, position.Quantity, position.EntryPrice, position.PnL)
		log.Printf("🛡️  ATR Trailing Stop: $%.2f", position.ATRTrailStop)
	} else {
		log.Printf("📍 No open position")
	}
}

// GetTradingStatus returns current trading status
func (tb *TradingBot) GetTradingStatus() interface{} {
	if tb.tradeExecutor == nil {
		return map[string]interface{}{
			"enabled": false,
			"error":   "Trade executor not initialized",
		}
	}
	return tb.tradeExecutor.GetStatus()
}

// GetCurrentTradingPosition returns current trading position
func (tb *TradingBot) GetCurrentTradingPosition() *Position {
	if tb.tradeExecutor == nil {
		return nil
	}
	return tb.tradeExecutor.GetCurrentPosition()
}

// GetTradeHistory returns recent trade history
func (tb *TradingBot) GetTradeHistory(limit int) []*Trade {
	if tb.tradeExecutor == nil {
		return []*Trade{}
	}
	return tb.tradeExecutor.GetTradeHistory(limit)
}

// EnableTrading enables trade execution
func (tb *TradingBot) EnableTrading() {
	if tb.tradeExecutor != nil {
		tb.tradeExecutor.Enable()
	}
}

// DisableTrading disables trade execution
func (tb *TradingBot) DisableTrading() {
	if tb.tradeExecutor != nil {
		tb.tradeExecutor.Disable()
	}
}

// ForceClosePosition manually closes current position
func (tb *TradingBot) ForceClosePosition() error {
	if tb.tradeExecutor == nil {
		return fmt.Errorf("trade executor not initialized")
	}

	// Get current price
	currentPrice, err := tb.GetCurrentPrice()
	if err != nil {
		return fmt.Errorf("failed to get current price: %w", err)
	}

	return tb.tradeExecutor.ForceClosePosition(currentPrice)
}
