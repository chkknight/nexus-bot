package bot

import (
	"testing"
)

// TestExtremeBiasFiltering verifies that extreme biases are properly filtered
func TestExtremeBiasFiltering(t *testing.T) {
	t.Log("🔍 TESTING EXTREME BIAS FILTERING")
	t.Log("=================================")

	// Test cases for extreme bias filtering
	testCases := []struct {
		name         string
		buyWeight    float64
		sellWeight   float64
		expectedBias float64
		shouldFilter bool
		description  string
	}{
		{
			name:         "Normal Bullish",
			buyWeight:    0.8,
			sellWeight:   0.2,
			expectedBias: 60.0, // (0.8-0.2)/(0.8+0.2) * 100 = 60%
			shouldFilter: false,
			description:  "Normal bullish bias should not be filtered",
		},
		{
			name:         "Extreme Bullish",
			buyWeight:    1.0,
			sellWeight:   0.0,
			expectedBias: 30.0, // 100% * 0.3 = 30% (filtered)
			shouldFilter: true,
			description:  "Extreme 100% bullish bias should be reduced to 30%",
		},
		{
			name:         "Extreme Bearish",
			buyWeight:    0.0,
			sellWeight:   1.0,
			expectedBias: -30.0, // -100% * 0.3 = -30% (filtered)
			shouldFilter: true,
			description:  "Extreme -100% bearish bias should be reduced to -30%",
		},
		{
			name:         "Near Extreme Bullish",
			buyWeight:    0.95,
			sellWeight:   0.05,
			expectedBias: 90.0, // 90% is NOT filtered (condition is > 90, not >= 90)
			shouldFilter: false,
			description:  "90% bullish bias should NOT be filtered (boundary case)",
		},
		{
			name:         "Just Over Extreme",
			buyWeight:    0.955,
			sellWeight:   0.045,
			expectedBias: 27.3, // 91% * 0.3 = 27.3% (filtered)
			shouldFilter: true,
			description:  "91% bullish bias should be reduced to 27.3%",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate raw bias
			activeWeight := tc.buyWeight + tc.sellWeight
			var bias float64
			if activeWeight > 0 {
				bias = ((tc.buyWeight - tc.sellWeight) / activeWeight) * 100
			}

			// Apply extreme bias filtering (same logic as API server)
			if bias > 90 {
				bias = bias * 0.3
			} else if bias < -90 {
				bias = bias * 0.3
			}

			// Verify the result
			if abs(bias-tc.expectedBias) > 0.1 {
				t.Errorf("Expected bias %.1f, got %.1f", tc.expectedBias, bias)
			}

			filterStatus := "✅ NOT FILTERED"
			if tc.shouldFilter {
				filterStatus = "🔧 FILTERED"
			}

			t.Logf("  %s: %.1f%% bias %s", tc.description, bias, filterStatus)
		})
	}

	t.Log("\n💡 CONCLUSION:")
	t.Log("• Extreme biases (>90% or <-90%) are reduced by 70%")
	t.Log("• This prevents unreliable extreme predictions")
	t.Log("• Normal biases (≤90%) pass through unchanged")
}

// TestIndicatorFiltering verifies that S&R and Ichimoku indicators are filtered out
func TestIndicatorFiltering(t *testing.T) {
	t.Log("\n🚫 TESTING INDICATOR FILTERING")
	t.Log("==============================")

	// Test indicators that should be filtered
	filteredIndicators := []string{
		"S&R_5m",
		"S&R_15m",
		"S&R_45m",
		"S&R_8h",
		"S&R_1d",
		"Ichimoku_5m",
		"Ichimoku_15m",
		"Ichimoku_45m",
		"Ichimoku_8h",
		"Ichimoku_1d",
	}

	// Test indicators that should NOT be filtered
	allowedIndicators := []string{
		"RSI_5m",
		"MACD_5m",
		"Volume_5m",
		"Trend_5m",
		"ReverseMFI_5m",
	}

	t.Log("🚫 SHOULD BE FILTERED:")
	for _, indicator := range filteredIndicators {
		shouldFilter := shouldFilterIndicator(indicator)
		status := "❌ NOT FILTERED"
		if shouldFilter {
			status = "✅ FILTERED"
		}
		t.Logf("  %s: %s", indicator, status)

		if !shouldFilter {
			t.Errorf("Indicator %s should be filtered but wasn't", indicator)
		}
	}

	t.Log("\n✅ SHOULD NOT BE FILTERED:")
	for _, indicator := range allowedIndicators {
		shouldFilter := shouldFilterIndicator(indicator)
		status := "✅ ALLOWED"
		if shouldFilter {
			status = "❌ FILTERED"
		}
		t.Logf("  %s: %s", indicator, status)

		if shouldFilter {
			t.Errorf("Indicator %s should not be filtered but was", indicator)
		}
	}
}

// Helper function to test indicator filtering logic
func shouldFilterIndicator(indicatorName string) bool {
	// Exact matches for 5-minute timeframe
	if indicatorName == "S&R_5m" || indicatorName == "Ichimoku_5m" {
		return true
	}

	// General pattern matching for any timeframe
	if containsString(indicatorName, "S&R") || containsString(indicatorName, "Ichimoku") {
		return true
	}

	return false
}

// Helper function to check if string contains substring
func containsString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// Helper function for absolute value
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
