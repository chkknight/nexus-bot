package bot

import (
	"testing"
	"time"
)

// TestIndicatorDataAlignment checks if synthetic data creates proper indicator signals
func TestIndicatorDataAlignment(t *testing.T) {
	t.Log("🔍 INDICATOR-DATA ALIGNMENT ANALYSIS")
	t.Log("=====================================")

	// Create historical data provider with controlled movements
	histData := NewHistoricalDataProvider()
	baseTime := time.Now().Add(-24 * time.Hour)
	histData.GenerateTestData(baseTime, 24)

	// Create signal aggregator
	config := getTestConfig()
	aggregator := NewSignalAggregator(config)

	// Test 5 specific scenarios
	testTimes := []time.Time{
		baseTime.Add(1 * time.Hour),  // Early data
		baseTime.Add(6 * time.Hour),  // Morning
		baseTime.Add(12 * time.Hour), // Midday
		baseTime.Add(18 * time.Hour), // Evening
		baseTime.Add(23 * time.Hour), // Late
	}

	for i, testTime := range testTimes {
		t.Logf("\n📊 SCENARIO %d: Testing at %s", i+1, testTime.Format("15:04:05"))

		// Get price movement over next 5 minutes
		currentPrice := histData.GetActualPriceAt(testTime)
		futurePrice := histData.GetActualPriceAt(testTime.Add(5 * time.Minute))
		priceChange := futurePrice - currentPrice
		actualDirection := "NEUTRAL"
		if priceChange > 3 {
			actualDirection = "HIGHER"
		} else if priceChange < -3 {
			actualDirection = "LOWER"
		}

		t.Logf("   Price Movement: $%.2f -> $%.2f (change: $%.2f, direction: %s)",
			currentPrice, futurePrice, priceChange, actualDirection)

		// Generate indicators for this time point
		fiveMinCandles := histData.GetCandles(FiveMinute, testTime, 100)
		if len(fiveMinCandles) < 50 {
			t.Logf("   ⚠️  Insufficient data for analysis")
			continue
		}

		// Create context and generate signals
		ctx := &MultiTimeframeContext{
			Symbol:              "BTCUSDT",
			FiveMinCandles:      fiveMinCandles,
			FifteenMinCandles:   histData.GetCandles(FifteenMinute, testTime, 50),
			FortyFiveMinCandles: histData.GetCandles(FortyFiveMinute, testTime, 30),
			EightHourCandles:    histData.GetCandles(EightHour, testTime, 10),
			DailyCandles:        histData.GetCandles(Daily, testTime, 5),
			LastUpdate:          testTime,
		}

		signal, err := aggregator.GenerateSignal(ctx)
		if err != nil {
			t.Logf("   ❌ Signal generation error: %v", err)
			continue
		}

		// Analyze 5-minute indicators specifically
		fiveMinIndicators := make([]IndicatorSignal, 0)
		buyCount, sellCount, holdCount := 0, 0, 0
		buyWeight, sellWeight, holdWeight := 0.0, 0.0, 0.0

		for _, ind := range signal.IndicatorSignals {
			if ind.Timeframe == FiveMinute {
				fiveMinIndicators = append(fiveMinIndicators, ind)
				switch ind.Signal {
				case Buy:
					buyCount++
					buyWeight += ind.Strength
				case Sell:
					sellCount++
					sellWeight += ind.Strength
				case Hold:
					holdCount++
					holdWeight += ind.Strength
				}
			}
		}

		// Calculate bias
		activeWeight := buyWeight + sellWeight
		var bias float64
		if activeWeight > 0 {
			bias = ((buyWeight - sellWeight) / activeWeight) * 100
		} else {
			bias = 0
		}

		// Determine prediction
		prediction := "NEUTRAL"
		if bias > 12 {
			prediction = "HIGHER"
		} else if bias < -12 {
			prediction = "LOWER"
		}

		t.Logf("   Indicators: %d total (BUY: %d/%.2f, SELL: %d/%.2f, HOLD: %d/%.2f)",
			len(fiveMinIndicators), buyCount, buyWeight, sellCount, sellWeight, holdCount, holdWeight)
		t.Logf("   Calculated Bias: %.1f%% → Prediction: %s", bias, prediction)
		t.Logf("   Match: %s (Prediction: %s, Actual: %s)",
			map[bool]string{true: "✅ CORRECT", false: "❌ WRONG"}[prediction == actualDirection],
			prediction, actualDirection)

		// Detailed indicator breakdown
		t.Logf("   📋 Individual Indicators:")
		for _, ind := range fiveMinIndicators {
			t.Logf("      - %s: %s (%.2f strength, value: %.2f)",
				ind.Name, ind.Signal.String(), ind.Strength, ind.Value)
		}
	}

	t.Log("\n💡 ALIGNMENT ANALYSIS COMPLETE")
	t.Log("Look for patterns where indicators consistently predict opposite to actual price movement")
}
