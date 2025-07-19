package bot

import (
	"context"
	"fmt"
	"log"
	"math"
	"testing"
	"time"
)

// RealDataPredictionTest tests prediction accuracy using real Binance data
func TestRealDataPredictionAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping real data test in short mode")
	}

	// Load configuration for Binance testing
	config, err := LoadConfig("config.json")
	if err != nil {
		t.Skip("Skipping real data test - config.json not available")
	}

	// Only run if Binance credentials are configured
	if config.Binance.APIKey == "" || config.DataProvider != "binance" {
		t.Skip("Skipping real data test - Binance not configured")
	}

	log.Println("Starting real Binance data prediction accuracy test...")

	// Create Binance data provider
	dataProvider := NewBinanceFuturesDataProvider(config.Binance.APIKey, config.Binance.SecretKey)

	// Test with the last 24 hours of real data
	results := testRealDataPredictions(t, dataProvider, config, 12) // 12 test points

	// Analyze results
	analyzeRealDataResults(t, results)
}

// testRealDataPredictions runs prediction tests using real historical data
func testRealDataPredictions(t *testing.T, dataProvider DataProvider, config Config, testCount int) []PredictionTestResult {
	var results []PredictionTestResult
	signalAggregator := NewSignalAggregator(config)

	// Get current time and work backwards
	endTime := time.Now()

	// Test at 2-hour intervals going back
	for i := 0; i < testCount; i++ {
		testTime := endTime.Add(-time.Duration(i*2) * time.Hour)

		// Skip if test time is too recent (need data for future verification)
		if time.Since(testTime) < 10*time.Minute {
			continue
		}

		result := testRealPredictionAtTime(t, signalAggregator, dataProvider, testTime, config.Symbol)
		if result.WasCorrect || result.Reasoning != "No data available" {
			results = append(results, result)
		}

		// Add delay to respect rate limits
		time.Sleep(100 * time.Millisecond)
	}

	return results
}

// testRealPredictionAtTime tests prediction at a specific time using real data
func testRealPredictionAtTime(t *testing.T, aggregator *SignalAggregator, dataProvider DataProvider, testTime time.Time, symbol string) PredictionTestResult {
	log.Printf("Testing prediction at %s", testTime.Format("2006-01-02 15:04:05"))

	// Get historical data up to test time
	ctx := context.Background()

	// Fetch data for each timeframe (ending at test time)
	fiveMinCandles, err := fetchHistoricalCandles(dataProvider, ctx, symbol, FiveMinute, testTime, 100)
	if err != nil {
		return PredictionTestResult{
			TestTime:   testTime,
			WasCorrect: false,
			Reasoning:  fmt.Sprintf("Failed to fetch 5m data: %v", err),
		}
	}

	fifteenMinCandles, _ := fetchHistoricalCandles(dataProvider, ctx, symbol, FifteenMinute, testTime, 50)
	fortyFiveMinCandles, _ := fetchHistoricalCandles(dataProvider, ctx, symbol, FortyFiveMinute, testTime, 30)
	eightHourCandles, _ := fetchHistoricalCandles(dataProvider, ctx, symbol, EightHour, testTime, 10)
	dailyCandles, _ := fetchHistoricalCandles(dataProvider, ctx, symbol, Daily, testTime, 5)

	if len(fiveMinCandles) == 0 {
		return PredictionTestResult{
			TestTime:   testTime,
			WasCorrect: false,
			Reasoning:  "No data available",
		}
	}

	// Get current price at test time
	currentPrice := fiveMinCandles[len(fiveMinCandles)-1].Close

	// Create context for prediction
	mtfCtx := &MultiTimeframeContext{
		Symbol:              symbol,
		FiveMinCandles:      fiveMinCandles,
		FifteenMinCandles:   fifteenMinCandles,
		FortyFiveMinCandles: fortyFiveMinCandles,
		EightHourCandles:    eightHourCandles,
		DailyCandles:        dailyCandles,
		LastUpdate:          testTime,
	}

	// Generate prediction
	signal, err := aggregator.GenerateSignal(mtfCtx)
	if err != nil {
		return PredictionTestResult{
			TestTime:   testTime,
			WasCorrect: false,
			Reasoning:  fmt.Sprintf("Signal generation failed: %v", err),
		}
	}

	// Convert to prediction
	prediction := convertTestSignalToPrediction(signal, currentPrice)

	// Get actual price 5 minutes later
	targetTime := testTime.Add(5 * time.Minute)
	actualPrice, err := getRealPriceAtTime(dataProvider, ctx, symbol, targetTime)
	if err != nil {
		return PredictionTestResult{
			TestTime:   testTime,
			WasCorrect: false,
			Reasoning:  fmt.Sprintf("Failed to get actual price: %v", err),
		}
	}

	// Calculate actual direction
	priceChange := actualPrice - currentPrice
	actualDirection := "NEUTRAL"
	threshold := 15.0 // Slightly higher threshold for real data noise

	if priceChange > threshold {
		actualDirection = "HIGHER"
	} else if priceChange < -threshold {
		actualDirection = "LOWER"
	}

	// Determine accuracy
	wasCorrect := prediction.Direction == actualDirection
	errorMargin := math.Abs(priceChange - prediction.PredictedChange)

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

// fetchHistoricalCandles gets historical data ending at a specific time
func fetchHistoricalCandles(dataProvider DataProvider, ctx context.Context, symbol string, timeframe Timeframe, endTime time.Time, limit int) ([]Candle, error) {
	// For real testing, we'd need to implement a method that fetches historical data
	// up to a specific end time. For now, this is a placeholder that would need
	// to be implemented based on the specific data provider interface.

	// This is a simplified approach - in a real implementation, you'd need to:
	// 1. Convert the timeframe to the provider's format
	// 2. Calculate the start time based on limit and timeframe
	// 3. Fetch candles from start time to end time
	// 4. Filter to ensure no candles are after endTime

	if binanceProvider, ok := dataProvider.(*BinanceFuturesDataProvider); ok {
		// Use a mock implementation for now
		return fetchBinanceHistoricalData(binanceProvider, symbol, timeframe, endTime, limit)
	}

	return nil, fmt.Errorf("data provider not supported for historical testing")
}

// fetchBinanceHistoricalData is a placeholder for fetching historical Binance data
func fetchBinanceHistoricalData(provider *BinanceFuturesDataProvider, symbol string, timeframe Timeframe, endTime time.Time, limit int) ([]Candle, error) {
	// This would need to be implemented to fetch historical data up to endTime
	// For now, return empty to indicate this is a placeholder
	log.Printf("Would fetch %d %s candles for %s ending at %s", limit, timeframe.String(), symbol, endTime.Format("15:04:05"))
	return []Candle{}, fmt.Errorf("historical data fetching not yet implemented")
}

// getRealPriceAtTime gets the actual price at a specific historical time
func getRealPriceAtTime(dataProvider DataProvider, ctx context.Context, symbol string, targetTime time.Time) (float64, error) {
	// Get 5-minute candle that contains the target time
	candles, err := fetchHistoricalCandles(dataProvider, ctx, symbol, FiveMinute, targetTime.Add(5*time.Minute), 1)
	if err != nil {
		return 0, err
	}

	if len(candles) == 0 {
		return 0, fmt.Errorf("no candle data available for target time")
	}

	return candles[0].Close, nil
}

// analyzeRealDataResults provides analysis specific to real data testing
func analyzeRealDataResults(t *testing.T, results []PredictionTestResult) {
	if len(results) == 0 {
		t.Logf("❌ No real data test results available")
		return
	}

	correct := 0
	totalError := 0.0

	for _, result := range results {
		if result.WasCorrect {
			correct++
		}
		totalError += math.Abs(result.ErrorMargin)
	}

	accuracy := float64(correct) / float64(len(results)) * 100
	avgError := totalError / float64(len(results))

	t.Logf("\n=== REAL DATA PREDICTION ANALYSIS ===")
	t.Logf("Total Real Data Tests: %d", len(results))
	t.Logf("Correct Predictions: %d", correct)
	t.Logf("Real Data Accuracy: %.2f%%", accuracy)
	t.Logf("Average Error Margin: $%.2f", avgError)

	// Compare with expected performance
	if accuracy >= 65 {
		t.Logf("✅ Excellent real-world performance (≥65%% accuracy)")
	} else if accuracy >= 55 {
		t.Logf("✅ Good real-world performance (≥55%% accuracy)")
	} else if accuracy >= 45 {
		t.Logf("⚠️  Acceptable real-world performance (≥45%% accuracy)")
	} else {
		t.Logf("❌ Below expected real-world performance (<45%% accuracy)")
	}

	// Show sample results
	t.Logf("\n=== SAMPLE REAL DATA RESULTS ===")
	for i, result := range results {
		if i < 3 { // Show first 3 results
			t.Logf("Test %d at %s:", i+1, result.TestTime.Format("15:04"))
			t.Logf("  Predicted: %s | Actual: %s | Correct: %v",
				result.Prediction, result.ActualDirection, result.WasCorrect)
			t.Logf("  Price: $%.2f → $%.2f (change: $%.2f)",
				result.CurrentPrice, result.ActualPrice, result.PriceChange)
		}
	}

	// Recommendations for real data
	t.Logf("\n=== REAL DATA RECOMMENDATIONS ===")
	if avgError > 75 {
		t.Logf("⚠️  High error margin with real data - consider market noise filtering")
	}

	if accuracy < 50 {
		t.Logf("🔧 Consider adjusting thresholds for real market conditions")
		t.Logf("🔧 Real markets have more noise than synthetic data")
		t.Logf("🔧 May need longer timeframes or stronger signal confirmation")
	}
}

// TestComparesyntheticVsRealData compares synthetic and real data performance
func TestCompareSyntheticVsRealData(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping comparison test in short mode")
	}

	t.Logf("\n=== SYNTHETIC VS REAL DATA COMPARISON ===")
	t.Logf("📊 Run both TestPredictionAccuracy and TestRealDataPredictionAccuracy")
	t.Logf("📊 Compare accuracy percentages between synthetic and real data")
	t.Logf("📊 Expected: Synthetic data accuracy should be higher due to controlled patterns")
	t.Logf("📊 Expected: Real data accuracy 10-20%% lower due to market noise")
	t.Logf("📊 If real data accuracy is much lower, consider:")
	t.Logf("   - Adjusting sensitivity thresholds")
	t.Logf("   - Adding noise filtering")
	t.Logf("   - Using longer confirmation periods")
	t.Logf("   - Implementing market volatility adjustments")
}
