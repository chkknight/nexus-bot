package bot

import (
	"testing"
	"time"
)

// TestFineTuneThreshold finds the exact optimal threshold between 30-40% for 80% accuracy
func TestFineTuneThreshold(t *testing.T) {
	t.Log("🎯 FINE-TUNING THRESHOLD FOR EXACTLY 80% ACCURACY")
	t.Log("================================================")

	// Create historical data and aggregator
	histData := NewHistoricalDataProvider()
	baseTime := time.Now().Add(-24 * time.Hour)
	histData.GenerateTestData(baseTime, 24)

	config := getTestConfig()
	aggregator := NewSignalAggregator(config)

	// Test fine-grained thresholds between 30-40%
	thresholds := []float64{30, 31, 32, 33, 34, 35, 36, 37, 38, 39, 40}

	t.Log("\n📊 FINE-TUNING NEUTRAL THRESHOLDS:")
	t.Log("Threshold | Total | Correct | Accuracy | HIGHER | LOWER | NEUTRAL | Target?")
	t.Log("----------|-------|---------|----------|--------|-------|---------|--------")

	bestThreshold := 0.0
	bestAccuracy := 0.0
	targetThreshold := 0.0

	for _, threshold := range thresholds {
		correct := 0
		total := 0
		higherCount := 0
		lowerCount := 0
		neutralCount := 0

		// Test 40 prediction points for better statistical significance
		for i := 0; i < 40; i++ {
			testTime := baseTime.Add(time.Duration(i) * 30 * time.Minute)

			// Get actual price movement
			currentPrice := histData.GetActualPriceAt(testTime)
			futurePrice := histData.GetActualPriceAt(testTime.Add(5 * time.Minute))
			priceChange := futurePrice - currentPrice

			actualDirection := "NEUTRAL"
			if priceChange > 4 {
				actualDirection = "HIGHER"
			} else if priceChange < -4 {
				actualDirection = "LOWER"
			}

			// Generate prediction with this threshold
			fiveMinCandles := histData.GetCandles(FiveMinute, testTime, 100)
			if len(fiveMinCandles) < 50 {
				continue
			}

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
				continue
			}

			// Apply custom threshold prediction logic
			direction := predictWithFineTunedThreshold(signal, threshold)

			// Count prediction types
			switch direction {
			case "HIGHER":
				higherCount++
			case "LOWER":
				lowerCount++
			case "NEUTRAL":
				neutralCount++
			}

			// Check accuracy
			if direction == actualDirection {
				correct++
			}
			total++
		}

		accuracy := float64(correct) / float64(total) * 100
		if accuracy > bestAccuracy {
			bestAccuracy = accuracy
			bestThreshold = threshold
		}

		// Check if we hit exactly 80% or very close
		isTarget := ""
		if accuracy >= 79.5 && accuracy <= 80.5 {
			isTarget = "🎯 YES!"
			targetThreshold = threshold
		}

		t.Logf("%8.0f   | %5d | %7d | %7.1f%% | %6d | %5d | %7d | %s",
			threshold, total, correct, accuracy, higherCount, lowerCount, neutralCount, isTarget)
	}

	t.Logf("\n🏆 BEST OVERALL: %.0f%% threshold = %.1f%% accuracy", bestThreshold, bestAccuracy)

	if targetThreshold > 0 {
		t.Logf("🎯 TARGET HIT: Use neutralThreshold := %.0f for ~80%% accuracy", targetThreshold)
	} else if bestAccuracy >= 80 {
		t.Logf("✅ TARGET EXCEEDED: Use neutralThreshold := %.0f for %.1f%% accuracy", bestThreshold, bestAccuracy)
	} else {
		t.Logf("⚠️  Closest to target: %.1f%% with %.0f%% threshold", bestAccuracy, bestThreshold)

		// Calculate how close we are
		gap := 80.0 - bestAccuracy
		if gap <= 5 {
			t.Log("💡 Very close! Consider:")
			t.Log("• Testing with real market data instead of synthetic")
			t.Log("• Fine-tuning indicator weights further")
			t.Log("• Adding more sophisticated multi-timeframe consensus")
		}
	}
}

// predictWithFineTunedThreshold applies the fine-tuned threshold logic
func predictWithFineTunedThreshold(signal *TradingSignal, neutralThreshold float64) string {
	// Calculate bias using same logic as main prediction
	fiveMinIndicators := make([]IndicatorSignal, 0)
	for _, ind := range signal.IndicatorSignals {
		if ind.Timeframe == FiveMinute {
			fiveMinIndicators = append(fiveMinIndicators, ind)
		}
	}

	if len(fiveMinIndicators) == 0 {
		return "NEUTRAL"
	}

	buyWeight, sellWeight, holdWeight := 0.0, 0.0, 0.0
	for _, ind := range fiveMinIndicators {
		// Skip worst performers
		if ind.Name == "S&R_5m" || ind.Name == "Ichimoku_5m" {
			continue
		}

		weight := 1.0
		switch ind.Name {
		case "Volume_5m":
			weight = 1.3
		case "Trend_5m":
			weight = 1.2
		case "MACD_5m":
			weight = 1.1
		case "RSI_5m":
			weight = 0.9
		}

		switch ind.Signal {
		case Buy:
			buyWeight += ind.Strength * weight
		case Sell:
			sellWeight += ind.Strength * weight
		case Hold:
			holdWeight += ind.Strength * weight
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

	// Apply extreme bias filtering
	if bias > 90 {
		bias = bias * 0.3
	} else if bias < -90 {
		bias = bias * 0.3
	}

	// Apply threshold
	if bias > neutralThreshold {
		return "HIGHER"
	} else if bias < -neutralThreshold {
		return "LOWER"
	}
	return "NEUTRAL"
}
