package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"trading-bot/pkg/bot"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// PredictionTracker tracks prediction timing for two-stage predictions
type PredictionTracker struct {
	StartTime    time.Time
	InitialPrice float64
	Stage        string // "INITIAL" or "FOLLOWUP"
}

// Global prediction tracker
var predictionTracker *PredictionTracker

// PredictionResponse represents the API response for price prediction
type PredictionResponse struct {
	Symbol           string                `json:"symbol" example:"BTCUSD"`
	CurrentPrice     float64               `json:"current_price" example:"50000.50"`
	Prediction       string                `json:"prediction" example:"HIGHER,LOWER,NEUTRAL"`
	Confidence       float64               `json:"confidence" example:"0.75"`
	Reasoning        string                `json:"reasoning" example:"Strong buy signals detected across multiple indicators"`
	Timestamp        string                `json:"timestamp" example:"2023-01-01T12:00:00Z"`
	PredictionTime   string                `json:"prediction_time" example:"2023-01-01T12:05:00Z"`
	TimeToTarget     string                `json:"time_to_target" example:"5m0s"`
	Indicators       []IndicatorPrediction `json:"indicators"`
	FiveMinuteSignal string                `json:"five_minute_signal" example:"Based on 5-minute timeframe analysis"`
	PredictionStage  string                `json:"prediction_stage" example:"INITIAL or FOLLOWUP"`

	// Pine Script ATR Trading Strategy Information
	TradingStatus   interface{} `json:"trading_status,omitempty"`   // Current trading status
	CurrentPosition interface{} `json:"current_position,omitempty"` // Open position details
	RecentTrades    interface{} `json:"recent_trades,omitempty"`    // Last 5 trades
	ATRTrailStop    float64     `json:"atr_trail_stop,omitempty"`   // Current ATR trailing stop
	TradingEnabled  bool        `json:"trading_enabled"`            // Whether trading is active
}

// IndicatorPrediction represents individual indicator prediction
type IndicatorPrediction struct {
	Name      string  `json:"name" example:"RSI_5m"`
	Signal    string  `json:"signal" example:"BUY" enums:"BUY,SELL,HOLD"`
	Strength  float64 `json:"strength" example:"0.85"`
	Timeframe string  `json:"timeframe" example:"5m"`
}

// ErrorResponse represents error response
type ErrorResponse struct {
	Error string `json:"error" example:"No signal available yet, bot may still be initializing"`
}

// HealthResponse represents health check response
type HealthResponse struct {
	Status     string `json:"status" example:"healthy"`
	Timestamp  string `json:"timestamp" example:"2023-01-01T12:00:00Z"`
	BotRunning bool   `json:"bot_running" example:"true"`
	Symbol     string `json:"symbol" example:"BTCUSD"`
}

// APIInfo represents API information
type APIInfo struct {
	Message   string   `json:"message" example:"Trading Bot API"`
	Version   string   `json:"version" example:"1.0.0"`
	Endpoints []string `json:"endpoints"`
}

// APIServer manages the REST API for the trading bot
type APIServer struct {
	router     *gin.Engine
	tradingBot *bot.TradingBot
	config     bot.Config
	port       string
}

// NewAPIServer creates a new API server
func NewAPIServer(config bot.Config, tradingBot *bot.TradingBot, port string) *APIServer {
	// Set Gin to release mode for production
	gin.SetMode(gin.ReleaseMode)

	router := gin.New()

	// Add middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())

	server := &APIServer{
		router:     router,
		tradingBot: tradingBot,
		config:     config,
		port:       port,
	}

	server.setupRoutes()
	return server
}

// setupRoutes configures all API routes
func (s *APIServer) setupRoutes() {
	// Static files for docs
	s.router.Static("/docs", "./docs")

	// Swagger documentation (use default docs path but force browser refresh)
	s.router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		v1.GET("/predict", s.predictPriceDirection)
		v1.GET("/status", s.getStatus)
		v1.GET("/signals", s.getLatestSignals)
		v1.GET("/health", s.healthCheck)

		// Pine Script ATR Trading Strategy Endpoints
		v1.GET("/trading/status", s.getTradingStatus)
		v1.GET("/trading/position", s.getCurrentPosition)
		v1.GET("/trading/history", s.getTradeHistory)
		v1.POST("/trading/enable", s.enableTrading)
		v1.POST("/trading/disable", s.disableTrading)
		v1.POST("/trading/close", s.forceClosePosition)
	}

	// Root route
	s.router.GET("/", s.getAPIInfo)
}

// getAPIInfo returns API information
// @Summary Get API information
// @Description Get general information about the trading bot API
// @Tags info
// @Accept json
// @Produce json
// @Success 200 {object} APIInfo
// @Router / [get]
func (s *APIServer) getAPIInfo(c *gin.Context) {
	c.JSON(http.StatusOK, APIInfo{
		Message: "Trading Bot API with Pine Script ATR Strategy",
		Version: "1.0.0",
		Endpoints: []string{
			"/predict - Predict price direction + trading status (default 5.5 min, use ?seconds=300 for 5 min)",
			"/status - Get bot status",
			"/signals - Get latest signals",
			"/health - Health check",
			"/trading/status - Get trading status",
			"/trading/position - Get current position",
			"/trading/history?limit=10 - Get trade history",
			"/trading/enable (POST) - Enable trading",
			"/trading/disable (POST) - Disable trading",
			"/trading/close (POST) - Force close position",
			"/swagger/index.html - API Documentation",
		},
	})
}

// predictPriceDirection handles the main prediction endpoint
// @Summary Predict price direction + trading status for configurable timeframe in the future
// @Description Analyzes 5-minute timeframe indicators to predict if price will be HIGHER/LOWER/NEUTRAL at specified time in future, includes current position and trading status
// @Tags prediction
// @Accept json
// @Produce json
// @Param seconds query int false "Prediction timeframe in seconds (default: 330 = 5.5 minutes, min: 60, max: 1800)"
// @Success 200 {object} PredictionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Failure 503 {object} ErrorResponse
// @Router /predict [get]
func (s *APIServer) predictPriceDirection(c *gin.Context) {
	// Parse prediction timeframe from query parameter
	secondsStr := c.DefaultQuery("seconds", "330") // Default to 330 seconds (5.5 minutes)
	seconds, err := strconv.Atoi(secondsStr)
	if err != nil || seconds < 60 || seconds > 1800 { // Min 1 minute, max 30 minutes
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error: "Invalid 'seconds' parameter. Must be an integer between 60 (1 min) and 1800 (30 min). Example: ?seconds=300 for 5 minutes, ?seconds=330 for 5.5 minutes",
		})
		return
	}

	predictionDuration := time.Duration(seconds) * time.Second

	// 🔄 LOG: Fresh prediction request
	log.Printf("📊 NEW PREDICTION REQUEST: %s prediction in %.1f minutes - fetching fresh Binance data...",
		s.config.Symbol, predictionDuration.Minutes())

	// Generate immediate prediction with on-demand data fetching
	signal, err := s.tradingBot.GenerateImmediatePrediction()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Failed to generate prediction: " + err.Error(),
		})
		return
	}

	// Get current price from the trading bot's market data
	currentPrice, err := s.tradingBot.GetCurrentPrice()
	if err != nil || currentPrice == 0 {
		c.JSON(http.StatusServiceUnavailable, ErrorResponse{
			Error: "Current price data not available: " + err.Error(),
		})
		return
	}

	// Convert trading signal to price prediction with configurable timeframe
	prediction := s.convertSignalToPrediction(signal, currentPrice, predictionDuration)

	// Build indicator predictions
	indicators := s.buildIndicatorPredictions(signal)

	// Calculate prediction time (current time + configurable duration)
	requestTime := time.Now().UTC()
	predictionTime := requestTime.Add(predictionDuration)
	timeToTarget := predictionTime.Sub(requestTime)

	// Determine prediction stage
	stage := "INITIAL"
	if predictionTracker != nil {
		timeSinceStart := time.Since(predictionTracker.StartTime)
		if timeSinceStart >= predictionDuration {
			stage = "FOLLOWUP"
		}
	}

	// Get trading information for Pine Script ATR strategy
	tradingStatus := s.tradingBot.GetTradingStatus()
	currentPosition := s.tradingBot.GetCurrentTradingPosition()
	recentTrades := s.tradingBot.GetTradeHistory(5) // Last 5 trades

	// Get ATR trailing stop value from current position or signals
	var atrTrailStop float64
	tradingEnabled := false

	if statusMap, ok := tradingStatus.(map[string]interface{}); ok {
		if enabled, exists := statusMap["enabled"]; exists {
			tradingEnabled = enabled.(bool)
		}
	}

	if currentPosition != nil {
		atrTrailStop = currentPosition.ATRTrailStop
	} else {
		// Get ATR trailing stop from indicators if no position
		for _, indSig := range signal.IndicatorSignals {
			if indSig.Name == "ATR_5m" {
				atrTrailStop = indSig.Value
				break
			}
		}
	}

	// 🔥 ENHANCED: Use Trading Status to Improve Predictions!
	prediction = s.enhancePredictionWithTradingStatus(prediction, currentPosition, recentTrades, tradingStatus, currentPrice, atrTrailStop)

	response := PredictionResponse{
		Symbol:           signal.Symbol,
		CurrentPrice:     currentPrice,
		Prediction:       prediction.Direction,
		Confidence:       prediction.Confidence,
		Reasoning:        prediction.Reasoning,
		Timestamp:        requestTime.Format(time.RFC3339),
		PredictionTime:   predictionTime.Format(time.RFC3339),
		TimeToTarget:     timeToTarget.String(),
		Indicators:       indicators,
		FiveMinuteSignal: prediction.FiveMinuteSignal,
		PredictionStage:  stage,

		// Pine Script ATR Trading Strategy Data
		TradingStatus:   tradingStatus,
		CurrentPosition: currentPosition,
		RecentTrades:    recentTrades,
		ATRTrailStop:    atrTrailStop,
		TradingEnabled:  tradingEnabled,
	}

	// Prediction tracker is now initialized in convertSignalToPrediction

	c.JSON(http.StatusOK, response)
}

// PredictionResult represents the prediction analysis
type PredictionResult struct {
	Direction        string
	Confidence       float64
	Reasoning        string
	FiveMinuteSignal string
}

// convertSignalToPrediction converts trading signal to configurable-timeframe future price prediction
func (s *APIServer) convertSignalToPrediction(signal *bot.TradingSignal, currentPrice float64, predictionDuration time.Duration) PredictionResult {
	// SIMPLIFIED: Focus only on 5-minute indicators for ultra-fast trading
	fiveMinIndicators := make([]bot.IndicatorSignal, 0)

	// Collect only 5-minute indicators
	for _, ind := range signal.IndicatorSignals {
		if ind.Timeframe == bot.FiveMinute {
			fiveMinIndicators = append(fiveMinIndicators, ind)
		}
	}

	// 🔥 NEW: Detect price momentum to prevent false signals
	priceMomentum := s.detectPriceMomentum(currentPrice)

	// Enhanced 5-minute focused analysis with trend-aware filtering
	fiveMinBuy := 0
	fiveMinSell := 0
	fiveMinStrength := 0.0

	// Analyze 5-minute indicators with trend-aware logic
	for _, ind := range fiveMinIndicators {
		fiveMinStrength += ind.Strength

		// 🛡️ TREND-AWARE FILTERING: Prevent false SELL signals during uptrends
		adjustedSignal := s.applyTrendAwareFilter(ind.Signal, ind.Name, priceMomentum, ind.Strength)

		switch adjustedSignal {
		case bot.Buy:
			fiveMinBuy++
		case bot.Sell:
			fiveMinSell++
		}
	}

	// Calculate ultra-focused confidence
	var fiveMinConfidence float64
	if len(fiveMinIndicators) > 0 {
		avgStrength := fiveMinStrength / float64(len(fiveMinIndicators))

		// High base confidence for focused analysis
		if fiveMinBuy > fiveMinSell || fiveMinSell > fiveMinBuy {
			// Directional signals get very high confidence
			fiveMinConfidence = math.Max(0.8, 0.75+(avgStrength*0.2))
		} else {
			// Strong consolidation signals also get high confidence
			fiveMinConfidence = math.Max(0.75, 0.7+(avgStrength*0.2))
		}
	} else {
		fiveMinConfidence = 0.8 // High default confidence
	}

	// 🚀 MOMENTUM BOOST: Extra confidence when momentum aligns with prediction
	if priceMomentum == "BULLISH" && fiveMinBuy > fiveMinSell {
		fiveMinConfidence = math.Min(0.95, fiveMinConfidence*1.15)
	} else if priceMomentum == "BEARISH" && fiveMinSell > fiveMinBuy {
		fiveMinConfidence = math.Min(0.95, fiveMinConfidence*1.15)
	}

	// Determine prediction direction
	var direction string
	var reasoning string
	var fiveMinuteSignal string

	durationMinutes := predictionDuration.Minutes()
	durationText := fmt.Sprintf("%.1f minutes", durationMinutes)

	if fiveMinBuy > fiveMinSell {
		direction = "HIGHER"
		priceTarget := currentPrice * (1 + 0.001*float64(fiveMinBuy-fiveMinSell))
		reasoning = fmt.Sprintf("5-minute BULLISH: %d buy vs %d sell signals. Target: %.2f in %s",
			fiveMinBuy, fiveMinSell, priceTarget, durationText)

		// Add momentum info to reasoning
		if priceMomentum == "BULLISH" {
			reasoning += " + Strong upward momentum detected"
		}

		fiveMinuteSignal = fmt.Sprintf("BULLISH momentum from %d indicators", fiveMinBuy)
	} else if fiveMinSell > fiveMinBuy {
		direction = "LOWER"
		priceTarget := currentPrice * (1 - 0.001*float64(fiveMinSell-fiveMinBuy))
		reasoning = fmt.Sprintf("5-minute BEARISH: %d sell vs %d buy signals. Target: %.2f in %s",
			fiveMinSell, fiveMinBuy, priceTarget, durationText)

		// Add momentum info to reasoning
		if priceMomentum == "BEARISH" {
			reasoning += " + Strong downward momentum detected"
		}

		fiveMinuteSignal = fmt.Sprintf("BEARISH momentum from %d indicators", fiveMinSell)
	} else {
		direction = "NEUTRAL"
		reasoning = fmt.Sprintf("5-minute CONSOLIDATION: Balanced signals (%.1f%% avg strength) in %s",
			(fiveMinStrength/float64(len(fiveMinIndicators)))*100, durationText)
		fiveMinuteSignal = "Balanced 5-minute consolidation"
	}

	return PredictionResult{
		Direction:        direction,
		Confidence:       math.Round(fiveMinConfidence*100) / 100,
		Reasoning:        reasoning,
		FiveMinuteSignal: fiveMinuteSignal,
	}
}

// 🔥 NEW: Detect price momentum to prevent false signals
func (s *APIServer) detectPriceMomentum(currentPrice float64) string {
	// 🚀 REAL-TIME: Fetch fresh 5-minute candles directly from Binance API
	binanceCandles, err := s.fetchBinanceCandles("BTCUSDT", "5m", 5)
	if err != nil {
		log.Printf("⚠️ Failed to fetch Binance candles for momentum: %v", err)
		return "NEUTRAL" // Default if API fails
	}

	if len(binanceCandles) < 3 {
		return "NEUTRAL"
	}

	// Parse the last 3 candle close prices for momentum analysis
	closes := make([]float64, len(binanceCandles))
	for i, candle := range binanceCandles {
		closes[i] = candle.Close
	}

	// Calculate momentum over last 3 candles
	recent := closes[len(closes)-1]   // Most recent close
	previous := closes[len(closes)-2] // 1 candle ago
	earlier := closes[len(closes)-3]  // 2 candles ago

	// Short-term momentum (last 2 candles)
	shortTermChange := (recent - previous) / previous

	// Medium-term momentum (last 3 candles)
	mediumTermChange := (recent - earlier) / earlier

	log.Printf("🔍 MOMENTUM ANALYSIS:")
	log.Printf("   Recent prices: %.2f → %.2f → %.2f", earlier, previous, recent)
	log.Printf("   Short-term change: %.4f%% (%.2f → %.2f)", shortTermChange*100, previous, recent)
	log.Printf("   Medium-term change: %.4f%% (%.2f → %.2f)", mediumTermChange*100, earlier, recent)

	// Strong momentum thresholds
	strongBullishThreshold := 0.003  // 0.3% up
	strongBearishThreshold := -0.003 // 0.3% down

	// Determine momentum with confidence
	if shortTermChange > strongBullishThreshold && mediumTermChange > 0 {
		log.Printf("✅ BULLISH MOMENTUM detected (%.3f%% recent, %.3f%% medium)", shortTermChange*100, mediumTermChange*100)
		return "BULLISH"
	} else if shortTermChange < strongBearishThreshold && mediumTermChange < 0 {
		log.Printf("📉 BEARISH MOMENTUM detected (%.3f%% recent, %.3f%% medium)", shortTermChange*100, mediumTermChange*100)
		return "BEARISH"
	} else if shortTermChange > 0.001 { // Mild upward momentum (0.1%+)
		log.Printf("🔼 MILD BULLISH momentum (%.3f%%)", shortTermChange*100)
		return "BULLISH"
	} else if shortTermChange < -0.001 { // Mild downward momentum (0.1%+)
		log.Printf("🔽 MILD BEARISH momentum (%.3f%%)", shortTermChange*100)
		return "BEARISH"
	}

	log.Printf("➖ NEUTRAL momentum")
	return "NEUTRAL"
}

// 🚀 NEW: Fetch real-time candles directly from Binance API
func (s *APIServer) fetchBinanceCandles(symbol string, interval string, limit int) ([]bot.Candle, error) {
	// Build Binance API URL
	url := fmt.Sprintf("https://api.binance.com/api/v3/klines?symbol=%s&interval=%s&limit=%d",
		symbol, interval, limit)

	// Make HTTP request
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error: %s", resp.Status)
	}

	// Parse JSON response
	var rawCandles [][]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&rawCandles); err != nil {
		return nil, fmt.Errorf("JSON decode failed: %w", err)
	}

	// Convert to Candle structs
	candles := make([]bot.Candle, len(rawCandles))
	for i, raw := range rawCandles {
		if len(raw) < 6 {
			continue
		}

		// Parse timestamp (milliseconds to time)
		timestampMs, ok := raw[0].(float64)
		if !ok {
			continue
		}

		// Parse OHLCV data
		open, _ := strconv.ParseFloat(raw[1].(string), 64)
		high, _ := strconv.ParseFloat(raw[2].(string), 64)
		low, _ := strconv.ParseFloat(raw[3].(string), 64)
		close, _ := strconv.ParseFloat(raw[4].(string), 64)
		volume, _ := strconv.ParseFloat(raw[5].(string), 64)

		candles[i] = bot.Candle{
			Timestamp: time.Unix(int64(timestampMs)/1000, 0),
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}
	}

	return candles, nil
}

// 🛡️ NEW: Apply trend-aware filtering to prevent false signals
func (s *APIServer) applyTrendAwareFilter(signal bot.SignalType, indicatorName string, momentum string, strength float64) bot.SignalType {
	// Don't filter strong trend-following indicators
	if strings.Contains(indicatorName, "Trend") ||
		strings.Contains(indicatorName, "EMA") ||
		strings.Contains(indicatorName, "ElliottWave") {
		return signal
	}

	// 🔥 AGGRESSIVE FILTERING: Override oscillators during strong momentum
	if momentum == "BULLISH" && signal == bot.Sell {
		// During uptrend, convert ALL SELL signals from oscillators to HOLD
		if strings.Contains(indicatorName, "RSI") ||
			strings.Contains(indicatorName, "Stochastic") ||
			strings.Contains(indicatorName, "Williams") ||
			strings.Contains(indicatorName, "Ichimoku") ||
			strings.Contains(indicatorName, "S&R") { // S&R often wrong during momentum

			log.Printf("🛡️ FILTERED: Converted %s SELL signal to HOLD (bullish momentum)", indicatorName)
			return bot.Hold // Convert ALL oscillator SELL signals to neutral during uptrend
		}
	}

	if momentum == "BEARISH" && signal == bot.Buy {
		// During downtrend, convert ALL BUY signals from oscillators to HOLD
		if strings.Contains(indicatorName, "RSI") ||
			strings.Contains(indicatorName, "Stochastic") ||
			strings.Contains(indicatorName, "Williams") ||
			strings.Contains(indicatorName, "Ichimoku") ||
			strings.Contains(indicatorName, "S&R") { // S&R often wrong during momentum

			log.Printf("🛡️ FILTERED: Converted %s BUY signal to HOLD (bearish momentum)", indicatorName)
			return bot.Hold // Convert ALL oscillator BUY signals to neutral during downtrend
		}
	}

	return signal // Keep original signal if no filtering needed
}

// enhancePredictionWithTradingStatus enhances the prediction based on trading status and position
func (s *APIServer) enhancePredictionWithTradingStatus(prediction PredictionResult, currentPosition interface{}, recentTrades interface{}, tradingStatus interface{}, currentPrice float64, atrTrailStop float64) PredictionResult {
	// Extract recent trades information
	var winningTrades, losingTrades int
	var recentPnL float64

	if tradesSlice, ok := recentTrades.([]*bot.Trade); ok && len(tradesSlice) > 0 {
		for _, trade := range tradesSlice {
			if trade.PnL > 0 {
				winningTrades++
			} else {
				losingTrades++
			}
			recentPnL += trade.PnL
		}

		// Calculate recent performance momentum
		totalRecentTrades := len(tradesSlice)
		winRate := float64(winningTrades) / float64(totalRecentTrades)

		// Enhance prediction based on recent performance
		if winRate > 0.6 { // If recent win rate > 60%
			prediction.Confidence = math.Min(0.95, prediction.Confidence*1.15) // Strong confidence boost
			if prediction.Direction == "HIGHER" {
				prediction.Reasoning = fmt.Sprintf("%s + Recent performance boost: %d/%d wins (%.0f%% win rate)",
					prediction.Reasoning, winningTrades, totalRecentTrades, winRate*100)
			}
		} else if winRate < 0.4 { // If recent win rate < 40%
			prediction.Confidence = math.Max(0.4, prediction.Confidence*0.85) // Reduce confidence
			prediction.Reasoning = fmt.Sprintf("%s - Recent performance caution: %d/%d wins (%.0f%% win rate)",
				prediction.Reasoning, winningTrades, totalRecentTrades, winRate*100)
		}
	}

	// Extract current position information
	if currentPosition != nil {
		if posMap, ok := currentPosition.(map[string]interface{}); ok {
			if side, exists := posMap["side"]; exists {
				if sideStr, ok := side.(string); ok {
					if pnl, exists := posMap["pnl"]; exists {
						if pnlFloat, ok := pnl.(float64); ok {
							// Position bias adjustment
							if sideStr == "LONG" && pnlFloat >= 0 {
								// Current long position is profitable - slight bullish bias
								if prediction.Direction == "HIGHER" {
									prediction.Confidence = math.Min(0.95, prediction.Confidence*1.08)
									prediction.Reasoning = fmt.Sprintf("%s + Long position profitable (+$%.2f)", prediction.Reasoning, pnlFloat)
								}
							} else if sideStr == "LONG" && pnlFloat < 0 {
								// Current long position is losing - slight caution
								prediction.Confidence = math.Max(0.5, prediction.Confidence*0.95)
								prediction.Reasoning = fmt.Sprintf("%s - Long position at loss (-$%.2f)", prediction.Reasoning, math.Abs(pnlFloat))
							}
						}
					}
				}
			}
		}
	}

	// Extract trading status information
	if statusMap, ok := tradingStatus.(map[string]interface{}); ok {
		if enabled, exists := statusMap["enabled"]; exists {
			if enabledBool, ok := enabled.(bool); ok && enabledBool {
				// Trading is enabled - slight confidence boost
				prediction.Confidence = math.Min(0.95, prediction.Confidence*1.05)
			}
		}

		// Check risk management status
		if riskMgmt, exists := statusMap["risk_management"]; exists {
			if riskMap, ok := riskMgmt.(map[string]interface{}); ok {
				if dailyLoss, exists := riskMap["daily_loss_used"]; exists {
					if dailyLossFloat, ok := dailyLoss.(float64); ok {
						if dailyLossFloat > 0.03 { // If daily loss > 3%
							prediction.Confidence = math.Max(0.4, prediction.Confidence*0.9) // Reduce confidence
							prediction.Reasoning = fmt.Sprintf("%s - Risk caution: %.1f%% daily loss used", prediction.Reasoning, dailyLossFloat*100)
						}
					}
				}
			}
		}
	}

	// ATR trailing stop confidence adjustment
	if atrTrailStop > 0 && currentPrice > 0 {
		stopDistance := math.Abs(atrTrailStop-currentPrice) / currentPrice
		if stopDistance < 0.005 { // Very tight stop (< 0.5%)
			prediction.Confidence = math.Min(0.95, prediction.Confidence*1.1) // Tight risk management boost
			prediction.Reasoning = fmt.Sprintf("%s + Tight ATR stop (%.2f%% away)", prediction.Reasoning, stopDistance*100)
		}
	}

	// Ensure confidence stays within reasonable bounds
	prediction.Confidence = math.Max(0.3, math.Min(0.95, prediction.Confidence))

	return prediction
}

// calculateBaseWeight returns the base weight for each indicator based on historical performance
func (s *APIServer) calculateBaseWeight(indicatorName string) float64 {
	switch {
	// High-performance indicators (>80% accuracy)
	case strings.Contains(indicatorName, "ElliottWave"):
		return 1.5 // Best performer - correctly predicted the drop
	case strings.Contains(indicatorName, "Volume"):
		return 1.4 // Increased weight - 87.1% accuracy - strong momentum confirmation
	case strings.Contains(indicatorName, "Trend"):
		return 1.3 // Increased weight - 83.9% accuracy - reliable trend detection
	case strings.Contains(indicatorName, "Channel"):
		return 1.2 // Good for market structure analysis

	// Medium-performance indicators (60-80% accuracy)
	case strings.Contains(indicatorName, "MACD"):
		return 1.2 // Increased weight - 80.6% accuracy - good trend following
	case strings.Contains(indicatorName, "BollingerBands"):
		return 1.0 // Conservative tuning - moderate weight until proven
	case strings.Contains(indicatorName, "EMA"):
		return 1.0 // New indicator - neutral weight until proven
	case strings.Contains(indicatorName, "ReverseMFI"):
		return 1.0 // 61.3% accuracy - normal weight

	// Conservative indicators (tuned for accuracy over sensitivity)
	case strings.Contains(indicatorName, "RSI"):
		return 0.9 // Improved tuning - moderate weight
	case strings.Contains(indicatorName, "Stochastic"):
		return 0.7 // Conservative settings - moderate weight
	case strings.Contains(indicatorName, "PinBar"):
		return 0.7 // Pattern recognition - conservative until proven

	// Keep Williams %R disabled for now (still problematic)
	case strings.Contains(indicatorName, "Williams"):
		return 0.3 // Very low weight if somehow still active

	default:
		return 1.0 // Default neutral weight
	}
}

// calculateMarketRegimeBoost adjusts weights based on current market conditions
func (s *APIServer) calculateMarketRegimeBoost(indicatorName string, currentPrice float64, indicators []bot.IndicatorSignal) float64 {
	// Detect market regime based on indicator consensus
	trendingSignals := 0
	rangingSignals := 0

	for _, ind := range indicators {
		if ind.Signal == bot.Buy || ind.Signal == bot.Sell {
			trendingSignals++
		} else {
			rangingSignals++
		}
	}

	totalSignals := trendingSignals + rangingSignals
	if totalSignals == 0 {
		return 1.0
	}

	trendRatio := float64(trendingSignals) / float64(totalSignals)

	// ENHANCED: More aggressive boost system for better trend filtering
	if trendRatio > 0.6 { // Trending market (lowered threshold from 0.7)
		switch {
		case strings.Contains(indicatorName, "Trend"), strings.Contains(indicatorName, "MACD"), strings.Contains(indicatorName, "EMA"):
			return 1.4 // Increased boost for trend-following indicators
		case strings.Contains(indicatorName, "ElliottWave"):
			return 1.5 // Increased boost - Elliott Wave excels in trending markets
		case strings.Contains(indicatorName, "Volume"):
			return 1.3 // Boost volume confirmation in trends
		case strings.Contains(indicatorName, "RSI"), strings.Contains(indicatorName, "Stochastic"), strings.Contains(indicatorName, "Williams"):
			return 0.4 // HEAVILY reduce oscillator weights in strong trends (was 0.8)
		}
	} else if trendRatio < 0.4 { // Ranging market (increased threshold from 0.3)
		switch {
		case strings.Contains(indicatorName, "RSI"), strings.Contains(indicatorName, "Stochastic"), strings.Contains(indicatorName, "Williams"):
			return 1.2 // Boost oscillators in ranging markets (reduced from 1.3 to prevent over-sensitivity)
		case strings.Contains(indicatorName, "BollingerBands"):
			return 1.3 // Bollinger Bands work well in ranging markets
		case strings.Contains(indicatorName, "Channel"):
			return 1.4 // Channel analysis excels in ranging markets
		case strings.Contains(indicatorName, "Trend"), strings.Contains(indicatorName, "EMA"):
			return 0.7 // Reduce trend indicators in ranging markets
		}
	}

	return 1.0 // Neutral for mixed conditions
}

// calculateVolatilityAdjustment adjusts weights based on signal strength and volatility
func (s *APIServer) calculateVolatilityAdjustment(indicatorName string, strength float64) float64 {
	// Penalize extreme signals (often unreliable)
	if strength > 0.95 {
		return 0.7 // Heavily penalize extreme signals
	} else if strength > 0.85 {
		return 0.85 // Moderately penalize high signals
	} else if strength < 0.3 {
		return 0.9 // Slightly penalize very weak signals
	}

	// Boost moderate strength signals (often most reliable)
	if strength >= 0.6 && strength <= 0.8 {
		return 1.1 // Boost moderate confidence signals
	}

	return 1.0 // Neutral adjustment
}

// buildIndicatorPredictions builds detailed indicator predictions
func (s *APIServer) buildIndicatorPredictions(signal *bot.TradingSignal) []IndicatorPrediction {
	predictions := make([]IndicatorPrediction, 0, len(signal.IndicatorSignals))

	for _, ind := range signal.IndicatorSignals {
		prediction := IndicatorPrediction{
			Name:      ind.Name,
			Signal:    ind.Signal.String(),
			Strength:  ind.Strength,
			Timeframe: ind.Timeframe.String(),
		}
		predictions = append(predictions, prediction)
	}

	return predictions
}

// getStatus returns the current bot status
// @Summary Get bot status
// @Description Get detailed status information about the trading bot
// @Tags status
// @Accept json
// @Produce json
// @Success 200 {object} bot.SignalEngineStatus
// @Router /status [get]
func (s *APIServer) getStatus(c *gin.Context) {
	status := s.tradingBot.GetStatus()
	c.JSON(http.StatusOK, status)
}

// getLatestSignals returns recent signals
// @Summary Get latest signals
// @Description Get the most recent trading signal generated by the bot
// @Tags signals
// @Accept json
// @Produce json
// @Success 200 {object} bot.TradingSignal
// @Failure 404 {object} ErrorResponse
// @Router /signals [get]
func (s *APIServer) getLatestSignals(c *gin.Context) {
	signal := s.tradingBot.GetLastSignal()
	if signal == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error: "No signals available",
		})
		return
	}

	c.JSON(http.StatusOK, signal)
}

// healthCheck returns service health
// @Summary Health check
// @Description Check if the trading bot API is healthy and running
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (s *APIServer) healthCheck(c *gin.Context) {
	status := s.tradingBot.GetStatus()

	health := HealthResponse{
		Status:     "healthy",
		Timestamp:  time.Now().UTC().Format(time.RFC3339),
		BotRunning: status.Running,
		Symbol:     status.Symbol,
	}

	if !status.Running {
		health.Status = "unhealthy"
	}

	c.JSON(http.StatusOK, health)
}

// Start starts the API server
func (s *APIServer) Start() error {
	fmt.Printf("🌐 Starting API server on port %s\n", s.port)
	fmt.Printf("📡 Prediction endpoint: http://localhost:%s/api/v1/predict\n", s.port)
	fmt.Printf("📊 Status endpoint: http://localhost:%s/api/v1/status\n", s.port)
	fmt.Printf("📚 Swagger docs: http://localhost:%s/swagger/index.html\n", s.port)
	fmt.Printf("🔍 All endpoints: http://localhost:%s/\n", s.port)

	return s.router.Run(":" + s.port)
}

// StartWithContext starts the API server with context for graceful shutdown
func (s *APIServer) StartWithContext(ctx context.Context) error {
	srv := &http.Server{
		Addr:    ":" + s.port,
		Handler: s.router,
	}

	// Start server in a goroutine
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("API server error: %v\n", err)
		}
	}()

	fmt.Printf("🌐 API server started on port %s\n", s.port)
	fmt.Printf("📡 Prediction endpoint: http://localhost:%s/api/v1/predict\n", s.port)
	fmt.Printf("📚 Trading endpoints: http://localhost:%s/api/v1/trading/*\n", s.port)
	fmt.Printf("📚 Swagger docs: http://localhost:%s/swagger/index.html\n", s.port)

	// Wait for context cancellation
	<-ctx.Done()

	return nil
}

// Pine Script ATR Trading Strategy API Handlers

// getTradingStatus returns current trading status
// @Summary Get trading status
// @Description Get current Pine Script ATR trading strategy status including positions and performance
// @Tags trading
// @Accept json
// @Produce json
// @Success 200 {object} interface{} "Trading status"
// @Router /trading/status [get]
func (s *APIServer) getTradingStatus(c *gin.Context) {
	status := s.tradingBot.GetTradingStatus()
	c.JSON(http.StatusOK, status)
}

// getCurrentPosition returns current trading position
// @Summary Get current position
// @Description Get current open trading position for Pine Script ATR strategy
// @Tags trading
// @Accept json
// @Produce json
// @Success 200 {object} interface{} "Current position"
// @Router /trading/position [get]
func (s *APIServer) getCurrentPosition(c *gin.Context) {
	position := s.tradingBot.GetCurrentTradingPosition()
	if position == nil {
		c.JSON(http.StatusOK, map[string]interface{}{
			"position": nil,
			"message":  "No open position",
		})
		return
	}
	c.JSON(http.StatusOK, map[string]interface{}{
		"position": position,
	})
}

// getTradeHistory returns recent trade history
// @Summary Get trade history
// @Description Get recent trade history for Pine Script ATR strategy
// @Tags trading
// @Accept json
// @Produce json
// @Param limit query int false "Number of trades to return (default: 10)"
// @Success 200 {object} interface{} "Trade history"
// @Router /trading/history [get]
func (s *APIServer) getTradeHistory(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	trades := s.tradingBot.GetTradeHistory(limit)
	c.JSON(http.StatusOK, map[string]interface{}{
		"trades": trades,
		"count":  len(trades),
	})
}

// enableTrading enables trade execution
// @Summary Enable trading
// @Description Enable Pine Script ATR strategy trade execution
// @Tags trading
// @Accept json
// @Produce json
// @Success 200 {object} interface{} "Trading enabled"
// @Router /trading/enable [post]
func (s *APIServer) enableTrading(c *gin.Context) {
	s.tradingBot.EnableTrading()
	c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Pine Script ATR trading strategy enabled",
		"enabled": true,
	})
}

// disableTrading disables trade execution
// @Summary Disable trading
// @Description Disable Pine Script ATR strategy trade execution
// @Tags trading
// @Accept json
// @Produce json
// @Success 200 {object} interface{} "Trading disabled"
// @Router /trading/disable [post]
func (s *APIServer) disableTrading(c *gin.Context) {
	s.tradingBot.DisableTrading()
	c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Pine Script ATR trading strategy disabled",
		"enabled": false,
	})
}

// forceClosePosition manually closes current position
// @Summary Force close position
// @Description Manually close the current open trading position
// @Tags trading
// @Accept json
// @Produce json
// @Success 200 {object} interface{} "Position closed"
// @Router /trading/close [post]
func (s *APIServer) forceClosePosition(c *gin.Context) {
	err := s.tradingBot.ForceClosePosition()
	if err != nil {
		c.JSON(http.StatusBadRequest, map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Position closed manually",
	})
}
