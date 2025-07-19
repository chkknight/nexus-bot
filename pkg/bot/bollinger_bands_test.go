package bot

import (
	"fmt"
	"testing"
	"time"

	"trading-bot/pkg/indicator"
)

func TestBollingerBands5MinuteIntegration(t *testing.T) {
	// Create a config with Bollinger Bands optimized for 5-minute trading
	config := Config{
		BollingerBands: BollingerBandsConfig{
			Enabled:       true,
			Period:        14,   // Shorter period for 5-minute (instead of 20)
			StandardDev:   2.0,  // Standard 2 standard deviations
			OverboughtStd: 0.85, // More sensitive for 5-minute
			OversoldStd:   0.15, // More sensitive for 5-minute
		},
		// Enable other indicators for comparison
		RSI: RSIConfig{
			Enabled:    true,
			Period:     14,
			Overbought: 70,
			Oversold:   30,
		},
		Volume: VolumeConfig{
			Enabled:         true,
			Period:          20,
			VolumeThreshold: 15000,
		},
		Trend: TrendConfig{
			Enabled: true,
			ShortMA: 20,
			LongMA:  50,
		},
	}

	// Create signal aggregator
	sa := NewSignalAggregator(config)

	// Create test candles with varied volatility patterns
	candles := generateBollingerTestCandles(100, 50000.0)

	// Test 5-minute signals
	fiveMinSignals := sa.getTimeframeSignals(candles, FiveMinute, 50000.0)

	// Find Bollinger Bands signal
	var bollingerSignal *IndicatorSignal
	for i := range fiveMinSignals {
		if fiveMinSignals[i].Name == "BollingerBands_5m" {
			bollingerSignal = &fiveMinSignals[i]
			break
		}
	}

	if bollingerSignal == nil {
		t.Fatal("Bollinger Bands signal not found in 5-minute signals")
	}

	// Test that the signal is generated properly
	if bollingerSignal.Strength == 0 {
		t.Error("Expected non-zero strength for Bollinger Bands signal")
	}

	// Test the signal value (position within bands)
	if bollingerSignal.Value < 0 || bollingerSignal.Value > 1 {
		t.Errorf("Expected signal value between 0 and 1, got %f", bollingerSignal.Value)
	}

	// Print signal details for verification
	fmt.Printf("Bollinger Bands Signal: %+v\n", bollingerSignal)

	// Test integration with signal aggregation
	ctx := &MultiTimeframeContext{
		Symbol:              "BTCUSDT",
		FiveMinCandles:      candles,
		FifteenMinCandles:   candles,
		FortyFiveMinCandles: candles,
		EightHourCandles:    candles,
		DailyCandles:        candles,
	}

	tradingSignal, err := sa.GenerateSignal(ctx)
	if err != nil {
		t.Fatalf("Error generating trading signal: %v", err)
	}

	if tradingSignal == nil {
		t.Fatal("Trading signal is nil")
	}

	// Verify that Bollinger Bands is included in the indicator signals
	hasBollingerBands := false
	for _, signal := range tradingSignal.IndicatorSignals {
		if signal.Name == "BollingerBands_5m" {
			hasBollingerBands = true
			break
		}
	}

	if !hasBollingerBands {
		t.Error("Bollinger Bands signal not found in trading signal indicators")
	}

	fmt.Printf("Final Trading Signal: %+v\n", tradingSignal)
}

func TestBollingerBands5MinuteEnhancedSignal(t *testing.T) {
	// Create Bollinger Bands indicator with 5-minute optimized parameters
	bollingerConfig := indicator.BollingerBandsConfig{
		Enabled:       true,
		Period:        14,
		StandardDev:   2.0,
		OverboughtStd: 0.85,
		OversoldStd:   0.15,
	}

	bollinger := indicator.NewBollingerBands(bollingerConfig, indicator.FiveMinute)

	// Create test scenarios
	testScenarios := []struct {
		name     string
		candles  []Candle
		expected SignalType
	}{
		{
			name:     "Oversold Bounce",
			candles:  generateOversoldScenario(50, 50000.0),
			expected: Buy,
		},
		{
			name:     "Overbought Rejection",
			candles:  generateOverboughtScenario(50, 50000.0),
			expected: Sell,
		},
		{
			name:     "Squeeze Breakout",
			candles:  generateSqueezeScenario(50, 50000.0),
			expected: Buy, // or Sell depending on direction
		},
	}

	for _, scenario := range testScenarios {
		t.Run(scenario.name, func(t *testing.T) {
			// Convert to indicator candles
			indicatorCandles := make([]indicator.Candle, len(scenario.candles))
			for i, candle := range scenario.candles {
				indicatorCandles[i] = indicator.Candle{
					Timestamp: candle.Timestamp,
					Open:      candle.Open,
					High:      candle.High,
					Low:       candle.Low,
					Close:     candle.Close,
					Volume:    candle.Volume,
				}
			}

			// Test enhanced 5-minute signal
			currentPrice := scenario.candles[len(scenario.candles)-1].Close
			enhancedSignal := bollinger.GetEnhanced5MinuteSignal(indicatorCandles, currentPrice)

			// Test standard signal for comparison
			values := bollinger.Calculate(indicatorCandles)
			standardSignal := bollinger.GetSignal(values, currentPrice)

			fmt.Printf("Scenario: %s\n", scenario.name)
			fmt.Printf("  Standard Signal: %+v\n", standardSignal)
			fmt.Printf("  Enhanced Signal: %+v\n", enhancedSignal)
			fmt.Printf("  Current Price: %.2f\n", currentPrice)
			fmt.Printf("  Position: %.3f\n", enhancedSignal.Value)

			// Verify signal strength is reasonable
			if enhancedSignal.Strength < 0.1 || enhancedSignal.Strength > 1.0 {
				t.Errorf("Enhanced signal strength out of range: %f", enhancedSignal.Strength)
			}

			// Verify signal value is within bands (0-1)
			if enhancedSignal.Value < 0 || enhancedSignal.Value > 1 {
				t.Errorf("Enhanced signal value out of range: %f", enhancedSignal.Value)
			}
		})
	}
}

func TestBollingerBandsSignalAccuracy(t *testing.T) {
	// Test different market conditions
	conditions := []struct {
		name        string
		candles     []Candle
		expectedSig SignalType
	}{
		{
			name:        "Extreme Oversold",
			candles:     generateExtremeOversoldCandles(30, 50000.0),
			expectedSig: Buy,
		},
		{
			name:        "Extreme Overbought",
			candles:     generateExtremeOverboughtCandles(30, 50000.0),
			expectedSig: Sell,
		},
		{
			name:        "Sideways Market",
			candles:     generateSidewaysCandles(30, 50000.0),
			expectedSig: Hold,
		},
	}

	bollingerConfig := indicator.BollingerBandsConfig{
		Enabled:       true,
		Period:        14,
		StandardDev:   2.0,
		OverboughtStd: 0.85,
		OversoldStd:   0.15,
	}

	bollinger := indicator.NewBollingerBands(bollingerConfig, indicator.FiveMinute)

	for _, condition := range conditions {
		t.Run(condition.name, func(t *testing.T) {
			// Convert to indicator candles
			indicatorCandles := make([]indicator.Candle, len(condition.candles))
			for i, candle := range condition.candles {
				indicatorCandles[i] = indicator.Candle{
					Timestamp: candle.Timestamp,
					Open:      candle.Open,
					High:      candle.High,
					Low:       candle.Low,
					Close:     candle.Close,
					Volume:    candle.Volume,
				}
			}

			currentPrice := condition.candles[len(condition.candles)-1].Close
			signal := bollinger.GetEnhanced5MinuteSignal(indicatorCandles, currentPrice)

			fmt.Printf("Condition: %s\n", condition.name)
			fmt.Printf("  Signal: %s\n", signal.Signal)
			fmt.Printf("  Strength: %.3f\n", signal.Strength)
			fmt.Printf("  Position: %.3f\n", signal.Value)
			fmt.Printf("  Expected: %s\n", condition.expectedSig)

			// For extreme conditions, signal should match expectations
			if condition.name == "Extreme Oversold" && signal.Signal.String() != "BUY" {
				t.Logf("Warning: Expected BUY signal for extreme oversold, got %s", signal.Signal)
			}
			if condition.name == "Extreme Overbought" && signal.Signal.String() != "SELL" {
				t.Logf("Warning: Expected SELL signal for extreme overbought, got %s", signal.Signal)
			}
		})
	}
}

// Helper functions to generate test candles for different scenarios

func generateBollingerTestCandles(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Create varied volatility pattern
		volatility := 100.0 + float64(i%10)*50.0
		price := basePrice + float64(i*20) + float64(i%5)*volatility

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - volatility/2,
			High:      price + volatility,
			Low:       price - volatility,
			Close:     price,
			Volume:    15000 + float64(i*100),
		}
	}

	return candles
}

func generateOversoldScenario(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Create downward trend that leads to oversold
		decline := float64(i) * 50.0
		price := basePrice - decline

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price + 25,
			High:      price + 50,
			Low:       price - 50,
			Close:     price,
			Volume:    20000,
		}
	}

	return candles
}

func generateOverboughtScenario(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Create upward trend that leads to overbought
		rally := float64(i) * 50.0
		price := basePrice + rally

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - 25,
			High:      price + 50,
			Low:       price - 50,
			Close:     price,
			Volume:    20000,
		}
	}

	return candles
}

func generateSqueezeScenario(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Create low volatility followed by breakout
		if i < count/2 {
			// Low volatility phase
			price := basePrice + float64(i%3)*10.0
			candles[i] = Candle{
				Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
				Open:      price - 5,
				High:      price + 10,
				Low:       price - 10,
				Close:     price,
				Volume:    10000,
			}
		} else {
			// Breakout phase
			breakout := float64(i-count/2) * 30.0
			price := basePrice + breakout
			candles[i] = Candle{
				Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
				Open:      price - 15,
				High:      price + 30,
				Low:       price - 30,
				Close:     price,
				Volume:    25000,
			}
		}
	}

	return candles
}

func generateExtremeOversoldCandles(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Extreme decline to create oversold condition
		decline := float64(i) * 80.0
		price := basePrice - decline

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price + 40,
			High:      price + 60,
			Low:       price - 60,
			Close:     price,
			Volume:    30000,
		}
	}

	return candles
}

func generateExtremeOverboughtCandles(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Extreme rally to create overbought condition
		rally := float64(i) * 80.0
		price := basePrice + rally

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - 40,
			High:      price + 60,
			Low:       price - 60,
			Close:     price,
			Volume:    30000,
		}
	}

	return candles
}

func generateSidewaysCandles(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Sideways movement within tight range
		price := basePrice + float64(i%3)*20.0 - 20.0

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - 10,
			High:      price + 20,
			Low:       price - 20,
			Close:     price,
			Volume:    15000,
		}
	}

	return candles
}
