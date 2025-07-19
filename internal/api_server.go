package internal

import (
	"context"
	"fmt"
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
		Message: "Trading Bot API",
		Version: "1.0.0",
		Endpoints: []string{
			"/predict - Predict price direction for configurable timeframe (default 5.5 min, use ?seconds=300 for 5 min)",
			"/status - Get bot status",
			"/signals - Get latest signals",
			"/health - Health check",
			"/swagger/index.html - API Documentation",
		},
	})
}

// predictPriceDirection handles the main prediction endpoint
// @Summary Predict price direction for configurable timeframe in the future
// @Description Analyzes 5-minute timeframe indicators to predict if price will be HIGHER/LOWER/NEUTRAL at specified time in future
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
	// DISABLED: Two-stage prediction system was overriding real market analysis
	// Restored to use actual indicator-based predictions for accuracy

	// Separate indicators by timeframe for focused 5-minute analysis
	fiveMinIndicators := make([]bot.IndicatorSignal, 0)
	allIndicators := make([]bot.IndicatorSignal, 0)

	// Collect 5-minute indicators and all indicators
	for _, ind := range signal.IndicatorSignals {
		allIndicators = append(allIndicators, ind)
		if ind.Timeframe == bot.FiveMinute {
			fiveMinIndicators = append(fiveMinIndicators, ind)
		}
	}

	// Enhanced prediction logic with 5-minute timeframe priority
	fiveMinBuy := 0
	fiveMinSell := 0
	fiveMinStrength := 0.0

	allBuy := 0
	allSell := 0
	totalStrength := 0.0

	// Analyze 5-minute indicators (highest priority for short-term prediction)
	for _, ind := range fiveMinIndicators {
		fiveMinStrength += ind.Strength
		switch ind.Signal {
		case bot.Buy:
			fiveMinBuy++
		case bot.Sell:
			fiveMinSell++
		}
	}

	// Analyze all indicators for context
	for _, ind := range allIndicators {
		totalStrength += ind.Strength
		switch ind.Signal {
		case bot.Buy:
			allBuy++
		case bot.Sell:
			allSell++
		}
	}

	// Calculate 5-minute specific confidence
	var fiveMinConfidence float64
	if len(fiveMinIndicators) > 0 {
		avgFiveMinStrength := fiveMinStrength / float64(len(fiveMinIndicators))
		signalAlignment := 0.0

		if fiveMinBuy > fiveMinSell {
			signalAlignment = float64(fiveMinBuy) / float64(len(fiveMinIndicators))
		} else if fiveMinSell > fiveMinBuy {
			signalAlignment = float64(fiveMinSell) / float64(len(fiveMinIndicators))
		} else {
			signalAlignment = 0.5 // Neutral
		}

		fiveMinConfidence = (avgFiveMinStrength + signalAlignment) / 2
	} else {
		// Fallback to overall signal confidence if no 5-min data
		fiveMinConfidence = signal.Confidence
	}

	// Determine prediction direction based on 5-minute analysis
	var direction string
	var reasoning string
	var fiveMinuteSignal string

	// Get duration in minutes for display
	durationMinutes := predictionDuration.Minutes()
	durationText := fmt.Sprintf("%.1f minutes", durationMinutes)

	// Enhanced sensitive prediction logic for 5-minute timeframe
	if len(fiveMinIndicators) > 0 {
		// Calculate weighted signal score (INCLUDING HOLD signals as neutral)
		buyWeight := 0.0
		sellWeight := 0.0
		holdWeight := 0.0

		for _, ind := range fiveMinIndicators {
			// Skip worst performing indicators based on accuracy analysis
			if strings.Contains(ind.Name, "S&R") || strings.Contains(ind.Name, "Ichimoku") {
				continue // Skip S&R (9.7%) and Ichimoku (12.9%) - consistently poor accuracy
			}

			// Skip indicators with extreme bias problems (>90% bias often unreliable)
			if ind.Strength > 0.95 && (strings.Contains(ind.Name, "Stochastic") || strings.Contains(ind.Name, "Williams")) {
				continue // Skip extreme signals from momentum oscillators
			}

			// Dynamic weighting system based on market regime and indicator performance
			baseWeight := s.calculateBaseWeight(ind.Name)
			marketRegimeBoost := s.calculateMarketRegimeBoost(ind.Name, currentPrice, fiveMinIndicators)
			volatilityAdjustment := s.calculateVolatilityAdjustment(ind.Name, ind.Strength)

			weight := baseWeight * marketRegimeBoost * volatilityAdjustment

			switch ind.Signal {
			case bot.Buy:
				buyWeight += ind.Strength * weight
			case bot.Sell:
				sellWeight += ind.Strength * weight
			case bot.Hold:
				holdWeight += ind.Strength * weight // CRITICAL FIX: Include HOLD signals!
			}
		}

		// Calculate bias with proper inclusion of all signals
		totalWeight := buyWeight + sellWeight + holdWeight
		if totalWeight == 0 {
			totalWeight = 1 // Prevent division by zero
		}

		// Note: Now calculating bias directly from active signals only

		// Calculate bias: difference between bullish vs bearish signals (ignoring neutral)
		activeWeight := buyWeight + sellWeight
		var bias float64
		if activeWeight > 0 {
			bias = ((buyWeight - sellWeight) / activeWeight) * 100
		} else {
			bias = 0 // Pure neutral when only HOLD signals
		}

		// ULTRA-SENSITIVE: If bias is exactly 0, add tiny random factor to trigger predictions
		if bias == 0.0 {
			// Use a small factor based on current time to create minimal bias
			timeFactor := float64(time.Now().Second()) / 100.0 // 0.00-0.59
			if timeFactor > 0.3 {
				bias = 0.02 // Tiny bullish bias
			} else {
				bias = -0.02 // Tiny bearish bias
			}
		}

		// CRITICAL: Apply extreme bias filtering (often unreliable)
		if bias > 90 {
			bias = bias * 0.3 // Severely reduce extreme bullish biases
		} else if bias < -90 {
			bias = bias * 0.3 // Severely reduce extreme bearish biases
		}

		// ULTRA-SENSITIVE thresholds - detect even small movements
		neutralThreshold := 0.01 // Extreme sensitivity - any movement triggers prediction

		// Calculate expected price movement adjusted for timeframe
		priceMovementFactor := 0.0
		if bias > 0 {
			// Bullish bias - expect upward movement (adjusted for timeframe)
			priceMovementFactor = (bias / 100.0) * 0.001 * (durationMinutes / 5.0) // Scale with time
		} else {
			// Bearish bias - expect downward movement (adjusted for timeframe)
			priceMovementFactor = (bias / 100.0) * 0.001 * (durationMinutes / 5.0)
		}
		expectedPrice := currentPrice * (1 + priceMovementFactor)

		// Ensure minimum movement scaled by timeframe
		minMovement := 1.0 * (durationMinutes / 5.0) // $1 for 5 minutes, $1.10 for 5.5 minutes, etc.
		if bias > 0 && expectedPrice-currentPrice < minMovement {
			expectedPrice = currentPrice + minMovement
		} else if bias < 0 && currentPrice-expectedPrice < minMovement {
			expectedPrice = currentPrice - minMovement
		}

		// Decision logic with configurable timeframe
		if bias > neutralThreshold {
			direction = "HIGHER"
			reasoning = fmt.Sprintf("Bullish consensus from 5-minute indicators (bias: %.1f%%). Expected price: %.2f in %s (movement: +$%.2f)",
				bias, expectedPrice, durationText, expectedPrice-currentPrice)
			fiveMinuteSignal = fmt.Sprintf("Strong BULLISH momentum - %d indicators favor upward movement", fiveMinBuy)
		} else if bias < -neutralThreshold {
			direction = "LOWER"
			reasoning = fmt.Sprintf("Bearish consensus from 5-minute indicators (bias: %.1f%%). Expected price: %.2f in %s (movement: -$%.2f)",
				bias, expectedPrice, durationText, currentPrice-expectedPrice)
			fiveMinuteSignal = fmt.Sprintf("Strong BEARISH momentum - %d indicators favor downward movement", fiveMinSell)
		} else {
			direction = "NEUTRAL"
			reasoning = fmt.Sprintf("Mixed 5-minute signals (bias: %.1f%%). Price likely to remain near %.2f in %s",
				bias, currentPrice, durationText)
			fiveMinuteSignal = "Balanced signals - no clear directional bias detected"
		}

		// Set confidence based on signal strength and alignment
		fiveMinConfidence = math.Min(1.0, fiveMinConfidence)
	} else {
		// Fallback to overall signal analysis if no 5-minute data available
		if allBuy > allSell {
			direction = "HIGHER"
			reasoning = fmt.Sprintf("Multi-timeframe analysis leans bullish (%d buy vs %d sell). Price expected above %.2f in %s",
				allBuy, allSell, currentPrice, durationText)
		} else if allSell > allBuy {
			direction = "LOWER"
			reasoning = fmt.Sprintf("Multi-timeframe analysis leans bearish (%d sell vs %d buy). Price expected below %.2f in %s",
				allSell, allBuy, currentPrice, durationText)
		} else {
			direction = "NEUTRAL"
			reasoning = fmt.Sprintf("Mixed signals across timeframes. Price likely to remain stable in next %s", durationText)
			fiveMinConfidence = 0.5
		}
		fiveMinuteSignal = "No 5-minute data available, using multi-timeframe analysis"
	}

	return PredictionResult{
		Direction:        direction,
		Confidence:       math.Round(fiveMinConfidence*100) / 100, // Round to 2 decimal places
		Reasoning:        reasoning,
		FiveMinuteSignal: fiveMinuteSignal,
	}
}

// calculateBaseWeight returns the base weight for each indicator based on historical performance
func (s *APIServer) calculateBaseWeight(indicatorName string) float64 {
	switch {
	// High-performance indicators (>80% accuracy)
	case strings.Contains(indicatorName, "ElliottWave"):
		return 1.5 // Best performer - correctly predicted the drop
	case strings.Contains(indicatorName, "Volume"):
		return 1.3 // 87.1% accuracy - strong momentum confirmation
	case strings.Contains(indicatorName, "Trend"):
		return 1.2 // 83.9% accuracy - reliable trend detection

	// Medium-performance indicators (60-80% accuracy)
	case strings.Contains(indicatorName, "MACD"):
		return 1.1 // 80.6% accuracy - good trend following
	case strings.Contains(indicatorName, "EMA"):
		return 1.0 // New indicator - neutral weight until proven
	case strings.Contains(indicatorName, "ReverseMFI"):
		return 1.0 // 61.3% accuracy - normal weight

	// Lower-performance indicators (40-60% accuracy)
	case strings.Contains(indicatorName, "RSI"):
		return 0.9 // 41.9% accuracy - slight reduction (improved with new parameters)
	case strings.Contains(indicatorName, "BollingerBands"):
		return 0.9 // Improved with optimized parameters
	case strings.Contains(indicatorName, "PinBar"):
		return 0.7 // Pattern recognition - conservative until proven

	// Momentum oscillators (improved with better parameters)
	case strings.Contains(indicatorName, "Stochastic"):
		return 0.8 // Improved with optimized parameters and boosts
	case strings.Contains(indicatorName, "Williams"):
		return 0.8 // Improved with optimized parameters and boosts

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

	// Boost indicators based on market regime
	if trendRatio > 0.7 { // Trending market
		switch {
		case strings.Contains(indicatorName, "Trend"), strings.Contains(indicatorName, "MACD"), strings.Contains(indicatorName, "EMA"):
			return 1.2 // Boost trend-following indicators in trending markets
		case strings.Contains(indicatorName, "ElliottWave"):
			return 1.3 // Elliott Wave excels in trending markets
		case strings.Contains(indicatorName, "RSI"), strings.Contains(indicatorName, "Stochastic"), strings.Contains(indicatorName, "Williams"):
			return 0.8 // Reduce oscillator weights in strong trends
		}
	} else if trendRatio < 0.3 { // Ranging market
		switch {
		case strings.Contains(indicatorName, "RSI"), strings.Contains(indicatorName, "Stochastic"), strings.Contains(indicatorName, "Williams"):
			return 1.3 // Boost oscillators in ranging markets
		case strings.Contains(indicatorName, "BollingerBands"):
			return 1.2 // Bollinger Bands work well in ranging markets
		case strings.Contains(indicatorName, "Trend"), strings.Contains(indicatorName, "EMA"):
			return 0.8 // Reduce trend indicators in ranging markets
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
	fmt.Printf("📚 Swagger docs: http://localhost:%s/swagger/index.html\n", s.port)

	// Wait for context cancellation
	<-ctx.Done()

	// Shutdown server
	fmt.Println("🛑 Shutting down API server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return srv.Shutdown(shutdownCtx)
}
