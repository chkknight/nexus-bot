package bot

import (
	"testing"
	"time"
)

// TestThresholdOptimization finds the optimal threshold for 80%+ accuracy
func TestThresholdOptimization(t *testing.T) {
	t.Log("🎯 THRESHOLD OPTIMIZATION FOR 80% ACCURACY")
	t.Log("===========================================")

	// Create historical data and aggregator
	histData := NewHistoricalDataProvider()
	baseTime := time.Now().Add(-24 * time.Hour)
	histData.GenerateTestData(baseTime, 24)

	config := getTestConfig()
	aggregator := NewSignalAggregator(config)

	// Test different neutral thresholds
	thresholds := []float64{15, 20, 25, 30, 35, 40, 45, 50}

	t.Log("\n📊 TESTING DIFFERENT NEUTRAL THRESHOLDS:")
	t.Log("Threshold | Total | Correct | Accuracy | HIGHER | LOWER | NEUTRAL")
	t.Log("----------|-------|---------|----------|--------|-------|--------")

	bestThreshold := 0.0
	bestAccuracy := 0.0

	for _, threshold := range thresholds {
		correct := 0
		total := 0
		higherCount := 0
		lowerCount := 0
		neutralCount := 0

		// Test 30 prediction points
		for i := 0; i < 30; i++ {
			testTime := baseTime.Add(time.Duration(i) * 45 * time.Minute)

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
			direction := predictWithThreshold(signal, threshold)

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

		t.Logf("%8.0f   | %5d | %7d | %7.1f%% | %6d | %5d | %7d",
			threshold, total, correct, accuracy, higherCount, lowerCount, neutralCount)
	}

	t.Logf("\n🏆 BEST RESULT: %.0f%% threshold = %.1f%% accuracy", bestThreshold, bestAccuracy)

	if bestAccuracy >= 80 {
		t.Logf("✅ TARGET ACHIEVED! Use neutralThreshold := %.0f for 80%%+ accuracy", bestThreshold)
	} else {
		t.Logf("⚠️  Highest achievable: %.1f%% with %.0f%% threshold", bestAccuracy, bestThreshold)

		// Provide specific recommendations
		t.Log("\n💡 RECOMMENDATIONS TO REACH 80%:")
		if bestAccuracy > 75 {
			t.Log("• Very close! Try fine-tuning indicator weights")
			t.Log("• Consider multi-timeframe consensus requirement")
		} else if bestAccuracy > 60 {
			t.Log("• Moderate improvement needed - focus on indicator quality")
			t.Log("• Consider disabling more poor performers")
		} else {
			t.Log("• Major improvement needed - fundamental indicator review required")
			t.Log("• Consider using real market data instead of synthetic")
		}
	}
}

// predictWithThreshold applies custom threshold logic
func predictWithThreshold(signal *TradingSignal, neutralThreshold float64) string {
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
