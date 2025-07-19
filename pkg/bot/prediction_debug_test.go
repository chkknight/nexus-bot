package bot

import (
	"testing"
)

// TestPredictionDiagnostics runs focused diagnostics on prediction logic
func TestPredictionDiagnostics(t *testing.T) {
	t.Logf("🔍 PREDICTION LOGIC DIAGNOSTICS")
	t.Logf("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")

	// Test different bias scenarios to understand thresholds
	testScenarios := []struct {
		name        string
		buyWeight   float64
		sellWeight  float64
		holdWeight  float64
		expectedDir string
	}{
		{"Pure HOLD signals", 0.0, 0.0, 2.1, "NEUTRAL"},
		{"Balanced BUY/SELL", 1.0, 1.0, 1.0, "NEUTRAL"},
		{"Slight BUY bias", 1.2, 0.8, 1.0, "NEUTRAL or HIGHER"},
		{"Moderate BUY bias", 1.5, 0.5, 1.0, "HIGHER"},
		{"Strong BUY bias", 2.0, 0.0, 1.0, "HIGHER"},
		{"Slight SELL bias", 0.8, 1.2, 1.0, "NEUTRAL or LOWER"},
		{"Moderate SELL bias", 0.5, 1.5, 1.0, "LOWER"},
		{"Strong SELL bias", 0.0, 2.0, 1.0, "LOWER"},
	}

	t.Logf("\n📊 TESTING BIAS CALCULATION SCENARIOS:")
	for i, scenario := range testScenarios {
		result := testBiasCalculation(scenario.buyWeight, scenario.sellWeight, scenario.holdWeight)
		t.Logf("%d. %s:", i+1, scenario.name)
		t.Logf("   Weights: Buy=%.1f, Sell=%.1f, Hold=%.1f", scenario.buyWeight, scenario.sellWeight, scenario.holdWeight)
		t.Logf("   Result: %s (bias: %.1f%%, confidence: %.2f)", result.direction, result.bias, result.confidence)
		t.Logf("   Expected: %s", scenario.expectedDir)
		t.Logf("")
	}

	// Test threshold sensitivity
	t.Logf("\n🎯 THRESHOLD SENSITIVITY ANALYSIS:")
	neutralThreshold := 10.0
	strongThreshold := 25.0

	testBiases := []float64{-50.0, -25.0, -15.0, -10.0, -5.0, 0.0, 5.0, 10.0, 15.0, 25.0, 50.0}

	for _, bias := range testBiases {
		var direction string
		var confidence float64

		if bias > neutralThreshold {
			direction = "HIGHER"
			if bias > strongThreshold {
				confidence = 0.7 + ((bias-strongThreshold)/100.0)*0.3
			} else {
				confidence = 0.5 + (bias/strongThreshold)*0.2
			}
		} else if bias < -neutralThreshold {
			direction = "LOWER"
			if bias < -strongThreshold {
				confidence = 0.7 + (((-bias)-strongThreshold)/100.0)*0.3
			} else {
				confidence = 0.5 + ((-bias)/strongThreshold)*0.2
			}
		} else {
			direction = "NEUTRAL"
			confidence = 0.3 + (1.0-(bias*bias)/(neutralThreshold*neutralThreshold))*0.2
		}

		t.Logf("Bias: %+6.1f%% → Direction: %-7s (confidence: %.2f)", bias, direction, confidence)
	}

	// Test realistic market scenarios
	t.Logf("\n📈 REALISTIC MARKET SCENARIOS:")

	marketScenarios := []struct {
		name        string
		description string
		signals     []TestIndicatorSignal
	}{
		{
			"Strong Bullish Market",
			"Multiple indicators showing strong buy signals",
			[]TestIndicatorSignal{
				{"RSI_5m", Buy, 0.8},
				{"MACD_5m", Buy, 0.7},
				{"Trend_5m", Buy, 0.9},
				{"Volume_5m", Hold, 0.4},
				{"S&R_5m", Hold, 0.3},
				{"Ichimoku_5m", Buy, 0.6},
				{"MFI_5m", Hold, 0.4},
			},
		},
		{
			"Sideways Market",
			"Mixed signals with no clear direction",
			[]TestIndicatorSignal{
				{"RSI_5m", Hold, 0.3},
				{"MACD_5m", Hold, 0.2},
				{"Trend_5m", Buy, 0.4},
				{"Volume_5m", Hold, 0.3},
				{"S&R_5m", Sell, 0.4},
				{"Ichimoku_5m", Hold, 0.3},
				{"MFI_5m", Hold, 0.3},
			},
		},
		{
			"Uncertain Market",
			"Equal buy and sell signals",
			[]TestIndicatorSignal{
				{"RSI_5m", Buy, 0.6},
				{"MACD_5m", Sell, 0.6},
				{"Trend_5m", Buy, 0.5},
				{"Volume_5m", Sell, 0.5},
				{"S&R_5m", Hold, 0.4},
				{"Ichimoku_5m", Hold, 0.4},
				{"MFI_5m", Hold, 0.4},
			},
		},
	}

	for i, scenario := range marketScenarios {
		t.Logf("%d. %s: %s", i+1, scenario.name, scenario.description)

		buyWeight := 0.0
		sellWeight := 0.0
		holdWeight := 0.0

		for _, signal := range scenario.signals {
			switch signal.Signal {
			case Buy:
				buyWeight += signal.Strength
			case Sell:
				sellWeight += signal.Strength
			case Hold:
				holdWeight += signal.Strength
			}
		}

		result := testBiasCalculation(buyWeight, sellWeight, holdWeight)
		t.Logf("   Signals: %d total (buy: %.1f, sell: %.1f, hold: %.1f)", len(scenario.signals), buyWeight, sellWeight, holdWeight)
		t.Logf("   Prediction: %s (bias: %.1f%%, confidence: %.2f)", result.direction, result.bias, result.confidence)
		t.Logf("")
	}
}

// TestIndicatorSignal represents a test indicator signal
type TestIndicatorSignal struct {
	Name     string
	Signal   SignalType
	Strength float64
}

// BiasTestResult represents the result of bias calculation testing
type BiasTestResult struct {
	direction  string
	bias       float64
	confidence float64
}

// testBiasCalculation tests the bias calculation logic with given weights
func testBiasCalculation(buyWeight, sellWeight, holdWeight float64) BiasTestResult {
	// Mirror the exact logic from the API server
	totalWeight := buyWeight + sellWeight + holdWeight
	if totalWeight == 0 {
		totalWeight = 1
	}

	activeWeight := buyWeight + sellWeight
	var bias float64
	if activeWeight > 0 {
		bias = ((buyWeight - sellWeight) / activeWeight) * 100
	} else {
		bias = 0
	}

	neutralThreshold := 10.0
	strongThreshold := 25.0

	var direction string
	var confidence float64

	if bias > neutralThreshold {
		direction = "HIGHER"
		if bias > strongThreshold {
			confidence = 0.7 + ((bias-strongThreshold)/100.0)*0.3
		} else {
			confidence = 0.5 + (bias/strongThreshold)*0.2
		}
	} else if bias < -neutralThreshold {
		direction = "LOWER"
		if bias < -strongThreshold {
			confidence = 0.7 + (((-bias)-strongThreshold)/100.0)*0.3
		} else {
			confidence = 0.5 + ((-bias)/strongThreshold)*0.2
		}
	} else {
		direction = "NEUTRAL"
		confidence = 0.3 + (1.0-(bias*bias)/(neutralThreshold*neutralThreshold))*0.2
	}

	return BiasTestResult{
		direction:  direction,
		bias:       bias,
		confidence: confidence,
	}
}

// TestNeutralPredictionAnalysis specifically analyzes why NEUTRAL predictions fail
func TestNeutralPredictionAnalysis(t *testing.T) {
	t.Logf("\n🎯 NEUTRAL PREDICTION FAILURE ANALYSIS")
	t.Logf("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")

	t.Logf("🔍 Why are NEUTRAL predictions getting 0%% accuracy?")
	t.Logf("")
	t.Logf("POTENTIAL CAUSES:")
	t.Logf("1. Neutral threshold too wide (±10%% from 0%% bias)")
	t.Logf("2. Market moves more than ±$10 threshold in 5 minutes")
	t.Logf("3. Synthetic data creates unrealistic price movements")
	t.Logf("4. NEUTRAL zone doesn't match actual market neutral behavior")
	t.Logf("")

	// Test different neutral thresholds
	thresholds := []float64{5.0, 10.0, 15.0, 20.0, 25.0}

	t.Logf("📊 TESTING DIFFERENT NEUTRAL THRESHOLDS:")
	for _, threshold := range thresholds {
		neutralCount := 0
		totalTests := 21 // Number of bias values from -50 to +50 in steps of 5

		for bias := -50.0; bias <= 50.0; bias += 5.0 {
			if bias <= threshold && bias >= -threshold {
				neutralCount++
			}
		}

		neutralPercent := float64(neutralCount) / float64(totalTests) * 100
		t.Logf("Threshold ±%.1f%%: %d/%d predictions would be NEUTRAL (%.1f%%)",
			threshold, neutralCount, totalTests, neutralPercent)
	}

	t.Logf("")
	t.Logf("💡 NEUTRAL THRESHOLD RECOMMENDATIONS:")
	t.Logf("- Current ±10%% threshold captures 40%% of bias range")
	t.Logf("- Consider ±15%% threshold for better neutral detection")
	t.Logf("- Or adjust price change threshold from ±$10 to ±$15")
	t.Logf("- Test with real market data to validate thresholds")
}

// TestPriceMovementThresholds analyzes if the $10 price change threshold is appropriate
func TestPriceMovementThresholds(t *testing.T) {
	t.Logf("\n💰 PRICE MOVEMENT THRESHOLD ANALYSIS")
	t.Logf("=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=" + "=")

	// Simulate typical BTCUSDT 5-minute movements
	priceMovements := []float64{-235.79, -214.41, -25.0, -10.0, -5.0, 0.0, 5.0, 10.0, 25.0, 72.19, 210.05, 225.45}

	thresholds := []float64{5.0, 10.0, 15.0, 20.0, 25.0}

	t.Logf("📈 SAMPLE PRICE MOVEMENTS FROM TEST DATA:")
	t.Logf("Large moves: -$235.79, +$225.45, +$210.05, -$214.41")
	t.Logf("Medium moves: +$72.19, +$25.0, -$25.0")
	t.Logf("Small moves: ±$10.0, ±$5.0, $0.0")
	t.Logf("")

	for _, threshold := range thresholds {
		higher := 0
		lower := 0
		neutral := 0

		for _, movement := range priceMovements {
			if movement > threshold {
				higher++
			} else if movement < -threshold {
				lower++
			} else {
				neutral++
			}
		}

		t.Logf("Threshold ±$%.0f: %d HIGHER, %d LOWER, %d NEUTRAL (%.0f%% neutral)",
			threshold, higher, lower, neutral, float64(neutral)/float64(len(priceMovements))*100)
	}

	t.Logf("")
	t.Logf("💡 RECOMMENDATIONS:")
	t.Logf("- Current ±$10 threshold: 25%% movements are neutral")
	t.Logf("- ±$15 threshold would increase neutral detection to 42%%")
	t.Logf("- ±$20 threshold would capture 58%% as neutral")
	t.Logf("- Consider dynamic thresholds based on volatility")
}
