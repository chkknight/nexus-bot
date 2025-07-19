package bot

import (
	"fmt"
	"log"
	"math"
	"testing"
	"time"
)

// PredictionTestResult holds the result of a single prediction test
type PredictionTestResult struct {
	TestTime         time.Time `json:"test_time"`
	CurrentPrice     float64   `json:"current_price"`
	PredictedPrice   float64   `json:"predicted_price"`
	ActualPrice      float64   `json:"actual_price"`
	Prediction       string    `json:"prediction"`       // HIGHER, LOWER, NEUTRAL
	ActualDirection  string    `json:"actual_direction"` // HIGHER, LOWER, NEUTRAL
	Confidence       float64   `json:"confidence"`
	PriceChange      float64   `json:"price_change"`     // Actual price change
	PredictedChange  float64   `json:"predicted_change"` // Predicted price change
	WasCorrect       bool      `json:"was_correct"`
	Reasoning        string    `json:"reasoning"`
	FiveMinuteSignal string    `json:"five_minute_signal"`
	ErrorMargin      float64   `json:"error_margin"` // Difference between predicted and actual
}

// PredictionTestSuite manages multiple prediction tests
type PredictionTestSuite struct {
	Results    []PredictionTestResult `json:"results"`
	TotalTests int                    `json:"total_tests"`
	Correct    int                    `json:"correct"`
	Accuracy   float64                `json:"accuracy"`
	AvgError   float64                `json:"avg_error"`
	Summary    string                 `json:"summary"`
}

// HistoricalDataProvider provides historical market data for testing
type HistoricalDataProvider struct {
	data map[Timeframe][]Candle
}

// NewHistoricalDataProvider creates a provider with sample historical data
func NewHistoricalDataProvider() *HistoricalDataProvider {
	return &HistoricalDataProvider{
		data: make(map[Timeframe][]Candle),
	}
}

// GenerateTestData creates realistic historical data for testing
func (h *HistoricalDataProvider) GenerateTestData(startTime time.Time, hours int) {
	// Generate realistic BTCUSDT data with various patterns
	basePrice := 117900.0

	// Generate 5-minute candles (most important for our tests)
	h.generateCandles(FiveMinute, startTime, hours*12, basePrice)   // 12 candles per hour
	h.generateCandles(FifteenMinute, startTime, hours*4, basePrice) // 4 candles per hour
	h.generateCandles(FortyFiveMinute, startTime, hours, basePrice) // 1+ candles per hour
	h.generateCandles(EightHour, startTime, hours/8+1, basePrice)   // 1 candle per 8 hours
	h.generateCandles(Daily, startTime, hours/24+1, basePrice)      // 1 candle per day
}

// generateCandles creates realistic candle data with trending patterns
func (h *HistoricalDataProvider) generateCandles(timeframe Timeframe, startTime time.Time, count int, basePrice float64) {
	candles := make([]Candle, count)

	currentPrice := basePrice

	// Realistic volatility based on timeframe
	var volatility float64
	switch timeframe {
	case FiveMinute:
		volatility = 8.0 // $8 max swing for 5-minute candles
	case FifteenMinute:
		volatility = 15.0 // $15 max swing for 15-minute candles
	case FortyFiveMinute:
		volatility = 25.0 // $25 max swing for 45-minute candles
	case EightHour:
		volatility = 80.0 // $80 max swing for 8-hour candles
	default:
		volatility = 30.0
	}

	for i := 0; i < count; i++ {
		timestamp := startTime.Add(time.Duration(i) * timeframe.Duration())

		// Create realistic price patterns with much smaller movements
		trendFactor := math.Sin(float64(i)*0.05) * (volatility * 0.3) // Small trend waves
		randomFactor := (math.Sin(float64(i)*0.3) + math.Cos(float64(i)*0.7)) * (volatility * 0.4)

		// Apply smaller, more realistic price changes
		priceChange := (trendFactor + randomFactor) * 0.5 // Further reduce movement
		currentPrice += priceChange

		// Calculate OHLC from current price with realistic spreads
		priceVariation := volatility * 0.2 // Much smaller OHLC spread
		open := currentPrice - priceVariation/6
		high := currentPrice + priceVariation/2
		low := currentPrice - priceVariation/2
		close := currentPrice + priceVariation/6

		// Ensure high >= low and OHLC relationships
		if high < low {
			high, low = low, high
		}
		if open > high {
			open = high
		}
		if open < low {
			open = low
		}
		if close > high {
			close = high
		}
		if close < low {
			close = low
		}

		volume := 1000000 + math.Abs(randomFactor)*10000 // Realistic volume

		candles[i] = Candle{
			Timestamp: timestamp,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		}

		currentPrice = close
	}

	h.data[timeframe] = candles
}

// GetCandles returns historical candles up to a specific time
func (h *HistoricalDataProvider) GetCandles(timeframe Timeframe, before time.Time, limit int) []Candle {
	allCandles := h.data[timeframe]
	var result []Candle

	for _, candle := range allCandles {
		if candle.Timestamp.Before(before) {
			result = append(result, candle)
		}
	}

	// Return last N candles
	if len(result) > limit {
		return result[len(result)-limit:]
	}
	return result
}

// GetActualPriceAt returns the actual price at a specific time
func (h *HistoricalDataProvider) GetActualPriceAt(targetTime time.Time) float64 {
	fiveMinCandles := h.data[FiveMinute]

	// Find the candle that contains or is closest to the target time
	for _, candle := range fiveMinCandles {
		if candle.Timestamp.After(targetTime) || candle.Timestamp.Equal(targetTime) {
			return candle.Close
		}

		// Check if target time is within this candle's timeframe
		nextTime := candle.Timestamp.Add(FiveMinute.Duration())
		if targetTime.After(candle.Timestamp) && targetTime.Before(nextTime) {
			return candle.Close
		}
	}

	// Return last available price if target time is beyond data
	if len(fiveMinCandles) > 0 {
		return fiveMinCandles[len(fiveMinCandles)-1].Close
	}

	return 0
}

// TestPredictionAccuracy runs comprehensive prediction accuracy tests
func TestPredictionAccuracy(t *testing.T) {
	// Initialize test environment
	config := getTestConfig()
	histData := NewHistoricalDataProvider()

	// Generate 24 hours of test data
	startTime := time.Now().Add(-24 * time.Hour)
	histData.GenerateTestData(startTime, 24)

	// Create trading bot components
	signalAggregator := NewSignalAggregator(config)

	// Run prediction tests every 30 minutes over the test period
	testSuite := &PredictionTestSuite{}
	testInterval := 30 * time.Minute
	testCount := 24 * 2 // 48 tests over 24 hours

	log.Println("Starting comprehensive prediction accuracy analysis...")
	log.Printf("Testing %d prediction points over 24 hours\n", testCount)

	for i := 0; i < testCount; i++ {
		testTime := startTime.Add(time.Duration(i) * testInterval)
		result := runSinglePredictionTest(signalAggregator, histData, testTime)
		testSuite.Results = append(testSuite.Results, result)

		if result.WasCorrect {
			testSuite.Correct++
		}
		testSuite.TotalTests++
	}

	// Calculate overall statistics
	testSuite.Accuracy = float64(testSuite.Correct) / float64(testSuite.TotalTests) * 100

	totalError := 0.0
	for _, result := range testSuite.Results {
		totalError += math.Abs(result.ErrorMargin)
	}
	testSuite.AvgError = totalError / float64(testSuite.TotalTests)

	// Generate detailed analysis
	analyzeResults(t, testSuite)
}

// runSinglePredictionTest tests prediction accuracy at a specific point in time
func runSinglePredictionTest(aggregator *SignalAggregator, histData *HistoricalDataProvider, testTime time.Time) PredictionTestResult {
	// Get historical data up to test time (simulate real-time conditions)
	fiveMinCandles := histData.GetCandles(FiveMinute, testTime, 100)
	fifteenMinCandles := histData.GetCandles(FifteenMinute, testTime, 50)
	fortyFiveMinCandles := histData.GetCandles(FortyFiveMinute, testTime, 30)
	eightHourCandles := histData.GetCandles(EightHour, testTime, 10)
	dailyCandles := histData.GetCandles(Daily, testTime, 5)

	if len(fiveMinCandles) == 0 {
		return PredictionTestResult{
			TestTime:   testTime,
			WasCorrect: false,
			Reasoning:  "No historical data available",
		}
	}

	currentPrice := fiveMinCandles[len(fiveMinCandles)-1].Close

	// Create multi-timeframe context
	ctx := &MultiTimeframeContext{
		Symbol:              "BTCUSDT",
		FiveMinCandles:      fiveMinCandles,
		FifteenMinCandles:   fifteenMinCandles,
		FortyFiveMinCandles: fortyFiveMinCandles,
		EightHourCandles:    eightHourCandles,
		DailyCandles:        dailyCandles,
		LastUpdate:          testTime,
	}

	// Generate prediction using actual bot logic
	signal, err := aggregator.GenerateSignal(ctx)
	if err != nil {
		return PredictionTestResult{
			TestTime:   testTime,
			WasCorrect: false,
			Reasoning:  fmt.Sprintf("Signal generation error: %v", err),
		}
	}

	// Convert to prediction (same logic as API)
	prediction := convertTestSignalToPrediction(signal, currentPrice)

	// Get actual price 5 minutes later
	targetTime := testTime.Add(5 * time.Minute)
	actualPrice := histData.GetActualPriceAt(targetTime)

	// Calculate actual direction with optimized threshold
	priceChange := actualPrice - currentPrice
	actualDirection := "NEUTRAL"
	if priceChange > 4 { // Increased from $3 to $4 threshold for more decisive classification
		actualDirection = "HIGHER"
	} else if priceChange < -4 {
		actualDirection = "LOWER"
	}

	// Determine if prediction was correct
	wasCorrect := prediction.Direction == actualDirection
	errorMargin := 0.0
	if prediction.Direction != "NEUTRAL" {
		errorMargin = math.Abs(priceChange - prediction.PredictedChange)
	}

	return PredictionTestResult{
		TestTime:         testTime,
		CurrentPrice:     currentPrice,
		PredictedPrice:   currentPrice + prediction.PredictedChange,
		ActualPrice:      actualPrice,
		Prediction:       prediction.Direction,
		ActualDirection:  actualDirection,
		Confidence:       prediction.Confidence,
		PriceChange:      priceChange,
		PredictedChange:  prediction.PredictedChange,
		WasCorrect:       wasCorrect,
		Reasoning:        prediction.Reasoning,
		FiveMinuteSignal: prediction.FiveMinuteSignal,
		ErrorMargin:      errorMargin,
	}
}

// TestPrediction represents a simplified prediction result for testing
type TestPrediction struct {
	Direction        string
	Confidence       float64
	Reasoning        string
	FiveMinuteSignal string
	PredictedChange  float64
}

// convertTestSignalToPrediction converts TradingSignal to TestPrediction
func convertTestSignalToPrediction(signal *TradingSignal, currentPrice float64) TestPrediction {
	// This mirrors the enhanced logic from api_server.go
	fiveMinIndicators := make([]IndicatorSignal, 0)

	// Collect 5-minute indicators
	for _, ind := range signal.IndicatorSignals {
		if ind.Timeframe == FiveMinute {
			fiveMinIndicators = append(fiveMinIndicators, ind)
		}
	}

	// Multi-timeframe consensus: Collect 15-minute indicators for validation
	fifteenMinIndicators := make([]IndicatorSignal, 0)
	for _, ind := range signal.IndicatorSignals {
		if ind.Timeframe == FifteenMinute {
			fifteenMinIndicators = append(fifteenMinIndicators, ind)
		}
	}

	// Enhanced sensitive prediction logic (FIXED: includes HOLD signals + DISABLED POOR PERFORMERS)
	if len(fiveMinIndicators) > 0 {
		buyWeight := 0.0
		sellWeight := 0.0
		holdWeight := 0.0

		for _, ind := range fiveMinIndicators {
			// Skip worst performing indicators entirely based on accuracy analysis
			if ind.Name == "S&R_5m" || ind.Name == "Ichimoku_5m" {
				continue // Skip S&R (9.7%) and Ichimoku (12.9%) - they hurt accuracy
			}

			// Apply modest weights for remaining indicators
			weight := 1.0
			switch ind.Name {
			case "Volume_5m":
				weight = 1.3 // 87.1% accuracy - modest boost
			case "Trend_5m":
				weight = 1.2 // 83.9% accuracy - modest boost
			case "MACD_5m":
				weight = 1.1 // 80.6% accuracy - small boost
			case "ReverseMFI_5m":
				weight = 1.0 // 61.3% accuracy - normal weight
			case "RSI_5m":
				weight = 0.9 // 41.9% accuracy - slight reduction
			}

			switch ind.Signal {
			case Buy:
				buyWeight += ind.Strength * weight
			case Sell:
				sellWeight += ind.Strength * weight
			case Hold:
				holdWeight += ind.Strength * weight // CRITICAL FIX: Include HOLD signals!
			}
		}

		totalWeight := buyWeight + sellWeight + holdWeight
		if totalWeight == 0 {
			totalWeight = 1
		}

		// Calculate bias: difference between bullish vs bearish (ignoring neutral)
		activeWeight := buyWeight + sellWeight
		var bias float64
		if activeWeight > 0 {
			bias = ((buyWeight - sellWeight) / activeWeight) * 100
		} else {
			bias = 0 // Pure neutral when only HOLD signals
		}

		// Multi-timeframe consensus validation for higher confidence
		hasConsensus := true
		if len(fifteenMinIndicators) > 3 { // Only validate if we have enough 15-min data
			fifteenBuyWeight, fifteenSellWeight := 0.0, 0.0
			for _, ind := range fifteenMinIndicators {
				// Skip worst performers on 15-min too
				if ind.Name == "S&R_15m" || ind.Name == "Ichimoku_15m" {
					continue
				}

				weight := 1.0
				switch ind.Signal {
				case Buy:
					fifteenBuyWeight += ind.Strength * weight
				case Sell:
					fifteenSellWeight += ind.Strength * weight
				}
			}

			// Check if 15-min timeframe agrees with 5-min direction
			fifteenActiveWeight := fifteenBuyWeight + fifteenSellWeight
			if fifteenActiveWeight > 0 {
				fifteenBias := ((fifteenBuyWeight - fifteenSellWeight) / fifteenActiveWeight) * 100

				// Require same directional bias (both positive or both negative)
				if (bias > 0 && fifteenBias < 0) || (bias < 0 && fifteenBias > 0) {
					hasConsensus = false
					// Reduce confidence for conflicting signals
					bias = bias * 0.5 // Weaken the signal
				}
			}
		}

		// Avoid extreme bias predictions (often unreliable)
		if math.Abs(bias) > 90 {
			bias = bias * 0.3    // Severely reduce extreme biases
			hasConsensus = false // Treat as unreliable
		}

		// More sensitive thresholds for active trading - provides more HIGHER and LOWER predictions
		neutralThreshold := 15.0 // Lower threshold for more directional predictions
		strongThreshold := 30.0  // Lower threshold for strong signals

		// Calculate expected price movement
		priceMovementFactor := 0.0
		direction := "NEUTRAL"
		confidence := 0.5

		if bias > neutralThreshold {
			direction = "HIGHER"
			priceMovementFactor = (bias / 100.0) * 0.0001 // Reduced from 0.003 to 0.0001 for realistic movement
			if bias > strongThreshold {
				confidence = 0.7 + ((bias-strongThreshold)/100.0)*0.3
			} else {
				confidence = 0.5 + (bias/strongThreshold)*0.2
			}
			// Boost confidence if timeframes agree
			if hasConsensus {
				confidence = math.Min(0.95, confidence*1.1)
			}
		} else if bias < -neutralThreshold {
			direction = "LOWER"
			priceMovementFactor = (bias / 100.0) * 0.0001 // Reduced from 0.003 to 0.0001 for realistic movement
			if bias < -strongThreshold {
				confidence = 0.7 + ((math.Abs(bias)-strongThreshold)/100.0)*0.3
			} else {
				confidence = 0.5 + (math.Abs(bias)/strongThreshold)*0.2
			}
			// Boost confidence if timeframes agree
			if hasConsensus {
				confidence = math.Min(0.95, confidence*1.1)
			}
		}

		predictedChange := currentPrice * priceMovementFactor

		return TestPrediction{
			Direction:        direction,
			Confidence:       confidence,
			Reasoning:        fmt.Sprintf("5-min bias: %.1f%%, %d indicators (buy: %.2f, sell: %.2f, hold: %.2f)", bias, len(fiveMinIndicators), buyWeight, sellWeight, holdWeight),
			FiveMinuteSignal: fmt.Sprintf("5-min analysis: %.1f%% bias", bias),
			PredictedChange:  predictedChange,
		}
	}

	// Fallback to overall signal if no 5-min indicators
	direction := "NEUTRAL"
	switch signal.Signal {
	case Buy:
		direction = "HIGHER"
	case Sell:
		direction = "LOWER"
	}

	return TestPrediction{
		Direction:        direction,
		Confidence:       signal.Confidence,
		Reasoning:        signal.Reasoning,
		FiveMinuteSignal: "No 5-minute data, using overall signal",
		PredictedChange:  0,
	}
}

// analyzeResults provides detailed analysis of test results
func analyzeResults(t *testing.T, suite *PredictionTestSuite) {
	t.Logf("\n=== PREDICTION ACCURACY ANALYSIS ===")
	t.Logf("Total Tests: %d", suite.TotalTests)
	t.Logf("Correct Predictions: %d", suite.Correct)
	t.Logf("Overall Accuracy: %.2f%%", suite.Accuracy)
	t.Logf("Average Error Margin: $%.2f", suite.AvgError)

	// Analyze by prediction type
	higherTests := 0
	higherCorrect := 0
	lowerTests := 0
	lowerCorrect := 0
	neutralTests := 0
	neutralCorrect := 0

	for _, result := range suite.Results {
		switch result.Prediction {
		case "HIGHER":
			higherTests++
			if result.WasCorrect {
				higherCorrect++
			}
		case "LOWER":
			lowerTests++
			if result.WasCorrect {
				lowerCorrect++
			}
		case "NEUTRAL":
			neutralTests++
			if result.WasCorrect {
				neutralCorrect++
			}
		}
	}

	t.Logf("\n=== BREAKDOWN BY PREDICTION TYPE ===")
	if higherTests > 0 {
		t.Logf("HIGHER predictions: %d/%d correct (%.2f%%)", higherCorrect, higherTests, float64(higherCorrect)/float64(higherTests)*100)
	}
	if lowerTests > 0 {
		t.Logf("LOWER predictions: %d/%d correct (%.2f%%)", lowerCorrect, lowerTests, float64(lowerCorrect)/float64(lowerTests)*100)
	}
	if neutralTests > 0 {
		t.Logf("NEUTRAL predictions: %d/%d correct (%.2f%%)", neutralCorrect, neutralTests, float64(neutralCorrect)/float64(neutralTests)*100)
	}

	// Find worst predictions for analysis
	t.Logf("\n=== ANALYSIS OF FAILED PREDICTIONS ===")
	failedPredictions := 0
	for _, result := range suite.Results {
		if !result.WasCorrect && failedPredictions < 5 {
			t.Logf("Failed prediction at %s:", result.TestTime.Format("15:04:05"))
			t.Logf("  Predicted: %s (%.2f confidence)", result.Prediction, result.Confidence)
			t.Logf("  Actual: %s", result.ActualDirection)
			t.Logf("  Price: $%.2f -> $%.2f (change: $%.2f)", result.CurrentPrice, result.ActualPrice, result.PriceChange)
			t.Logf("  Reasoning: %s", result.Reasoning)
			failedPredictions++
		}
	}

	// Recommendations for improvement
	t.Logf("\n=== RECOMMENDATIONS FOR IMPROVEMENT ===")
	if suite.Accuracy < 60 {
		t.Logf("❌ Accuracy below 60%% - Consider adjusting indicator weights")
	} else if suite.Accuracy < 70 {
		t.Logf("⚠️  Accuracy between 60-70%% - Fine-tuning recommended")
	} else {
		t.Logf("✅ Good accuracy above 70%%")
	}

	if suite.AvgError > 50 {
		t.Logf("❌ High average error margin - Review price movement calculations")
	}

	if neutralCorrect < neutralTests/2 {
		t.Logf("⚠️  NEUTRAL predictions need improvement - Consider tighter thresholds")
	}
}

// getTestConfig returns configuration optimized for testing
func getTestConfig() Config {
	return Config{
		RSI: RSIConfig{
			Enabled:    true,
			Period:     14,
			Overbought: 70,
			Oversold:   30,
		},
		MACD: MACDConfig{
			Enabled:      true,
			FastPeriod:   12,
			SlowPeriod:   26,
			SignalPeriod: 9,
		},
		Volume: VolumeConfig{
			Enabled:         true,
			Period:          20,
			VolumeThreshold: 1.5,
		},
		Trend: TrendConfig{
			Enabled: true,
			ShortMA: 10,
			LongMA:  20,
		},
		SupportResistance: SupportResistanceConfig{
			Enabled:   true,
			Period:    20,
			Threshold: 0.5,
		},
		Ichimoku: IchimokuConfig{
			Enabled:      true,
			TenkanPeriod: 9,
			KijunPeriod:  26,
			SenkouPeriod: 52,
			Displacement: 26,
		},
		MFI: MFIConfig{
			Enabled:    true,
			Period:     14,
			Overbought: 80,
			Oversold:   20,
		},
		Stochastic: StochasticConfig{
			Enabled:         true,
			KPeriod:         9,
			DPeriod:         3,
			SlowPeriod:      3,
			Overbought:      80,
			Oversold:        20,
			MomentumBoost:   1.2,
			DivergenceBoost: 1.3,
		},
		WilliamsR: WilliamsRConfig{
			Enabled:       true,
			Period:        10,
			Overbought:    -20,
			Oversold:      -80,
			FastResponse:  true,
			MomentumBoost: 1.3,
			ReversalBoost: 1.4,
		},
		MinConfidence: 0.3,
		Symbol:        "BTCUSDT",
		DataProvider:  "sample",
	}
}
