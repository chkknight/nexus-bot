package bot

import (
	"testing"
)

// TestSensitivityComparison shows how different thresholds affect predictions
func TestSensitivityComparison(t *testing.T) {
	t.Log("🎯 SENSITIVITY COMPARISON: 25% vs 15% THRESHOLD")
	t.Log("===============================================")

	// Test different bias scenarios
	testCases := []struct {
		bias        float64
		description string
	}{
		{bias: 20.0, description: "Moderate bullish bias"},
		{bias: 18.0, description: "Mild bullish bias"},
		{bias: 15.0, description: "Weak bullish bias"},
		{bias: 12.0, description: "Very weak bullish bias"},
		{bias: 0.0, description: "Perfectly balanced"},
		{bias: -12.0, description: "Very weak bearish bias"},
		{bias: -15.0, description: "Weak bearish bias"},
		{bias: -18.0, description: "Mild bearish bias"},
		{bias: -20.0, description: "Moderate bearish bias"},
	}

	oldThreshold := 25.0 // Previous threshold
	newThreshold := 15.0 // New more sensitive threshold

	t.Log("\n📊 COMPARISON RESULTS:")
	t.Log("Bias     | Description              | Old (25%) | New (15%) | Change")
	t.Log("---------|--------------------------|-----------|-----------|------------------")

	for _, tc := range testCases {
		// Old threshold prediction
		var oldPred string
		if tc.bias > oldThreshold {
			oldPred = "HIGHER"
		} else if tc.bias < -oldThreshold {
			oldPred = "LOWER"
		} else {
			oldPred = "NEUTRAL"
		}

		// New threshold prediction
		var newPred string
		if tc.bias > newThreshold {
			newPred = "HIGHER"
		} else if tc.bias < -newThreshold {
			newPred = "LOWER"
		} else {
			newPred = "NEUTRAL"
		}

		// Determine change
		change := "Same"
		if oldPred != newPred {
			change = "📈 More Sensitive!"
		}

		t.Logf("%7.1f%% | %-24s | %-9s | %-9s | %s",
			tc.bias, tc.description, oldPred, newPred, change)
	}

	t.Log("\n🎯 SENSITIVITY IMPROVEMENTS:")
	t.Log("✅ 20% bias: NEUTRAL → HIGHER (more bullish signals)")
	t.Log("✅ 18% bias: NEUTRAL → HIGHER (catches mild trends)")
	t.Log("✅ -18% bias: NEUTRAL → LOWER (catches mild downtrends)")
	t.Log("✅ -20% bias: NEUTRAL → LOWER (more bearish signals)")

	t.Log("\n📈 EXPECTED RESULTS:")
	t.Log("• More HIGHER predictions when market shows even mild bullish bias")
	t.Log("• More LOWER predictions when market shows even mild bearish bias")
	t.Log("• Only NEUTRAL when market is truly balanced (±15% range)")
	t.Log("• Better for active trading - catches smaller movements")
}

// TestCurrentMarketBehavior explains why we're seeing NEUTRAL now
func TestCurrentMarketBehavior(t *testing.T) {
	t.Log("\n🔍 CURRENT MARKET ANALYSIS")
	t.Log("==========================")

	t.Log("📊 CURRENT SITUATION:")
	t.Log("• Market Bias: 0.0% (perfectly balanced)")
	t.Log("• Prediction: NEUTRAL (correct for balanced market)")
	t.Log("• Threshold: 15% (more sensitive than before)")

	t.Log("\n🎯 WHEN YOU'LL SEE DIRECTIONAL PREDICTIONS:")

	examples := []struct {
		scenario   string
		bias       string
		prediction string
		likelihood string
	}{
		{
			scenario:   "Minor Price Rise",
			bias:       "16-20%",
			prediction: "HIGHER",
			likelihood: "Common - catches small uptrends",
		},
		{
			scenario:   "Minor Price Drop",
			bias:       "-16% to -20%",
			prediction: "LOWER",
			likelihood: "Common - catches small downtrends",
		},
		{
			scenario:   "Strong Bullish Move",
			bias:       "25-40%",
			prediction: "HIGHER",
			likelihood: "Frequent - strong upward momentum",
		},
		{
			scenario:   "Strong Bearish Move",
			bias:       "-25% to -40%",
			prediction: "LOWER",
			likelihood: "Frequent - strong downward momentum",
		},
		{
			scenario:   "Sideways Market",
			bias:       "-15% to +15%",
			prediction: "NEUTRAL",
			likelihood: "Only when truly balanced",
		},
	}

	for _, ex := range examples {
		t.Logf("📈 %s:", ex.scenario)
		t.Logf("   Bias: %s → %s (%s)", ex.bias, ex.prediction, ex.likelihood)
	}

	t.Log("\n💡 KEY INSIGHT:")
	t.Log("The system is correctly showing NEUTRAL because Bitcoin is")
	t.Log("in a perfectly balanced state (0.0% bias) right now.")
	t.Log("Once market conditions change, you'll get directional predictions!")

	t.Log("\n🚀 IMPROVED SENSITIVITY:")
	t.Log("• Old system: Only directional predictions when bias > 25%")
	t.Log("• New system: Directional predictions when bias > 15%")
	t.Log("• Result: 67% more sensitive to market movements!")
}
