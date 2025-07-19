package bot

import (
	"fmt"
	"testing"

	"trading-bot/pkg/indicator"
)

func TestIchimokuIntegration5MinuteEnhanced(t *testing.T) {
	// Create a config with enhanced 5-minute Ichimoku parameters
	config := Config{
		Ichimoku: IchimokuConfig{
			Enabled:      true,
			TenkanPeriod: 6,  // Optimized for 5-minute
			KijunPeriod:  18, // Optimized for 5-minute
			SenkouPeriod: 36, // Optimized for 5-minute
			Displacement: 18, // Optimized for 5-minute
		},
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
			VolumeThreshold: 15000,
		},
		Trend: TrendConfig{
			Enabled: true,
			ShortMA: 20,
			LongMA:  50,
		},
		SupportResistance: SupportResistanceConfig{
			Enabled:   true,
			Period:    20,
			Threshold: 0.02,
		},
		MFI: MFIConfig{
			Enabled:    true,
			Period:     14,
			Overbought: 80,
			Oversold:   20,
		},
	}

	// Create signal aggregator
	sa := NewSignalAggregator(config)

	// Create test candles - simulating a scenario where price moves above cloud
	candles := generateTestCandles(100, 45000.0)

	// Test 5-minute signals
	fiveMinSignals := sa.getTimeframeSignals(candles, FiveMinute, 50000.0)

	// Find Ichimoku signal
	var ichimokuSignal *IndicatorSignal
	for i := range fiveMinSignals {
		if fiveMinSignals[i].Name == "Ichimoku_5m" {
			ichimokuSignal = &fiveMinSignals[i]
			break
		}
	}

	if ichimokuSignal == nil {
		t.Fatal("Ichimoku signal not found in 5-minute signals")
	}

	// Test that the signal is generated (should not be zero)
	if ichimokuSignal.Strength == 0 {
		t.Error("Expected non-zero strength for Ichimoku signal")
	}

	// Test different timeframes to ensure enhanced signal is only used for 5-minute
	fifteenMinSignals := sa.getTimeframeSignals(candles, FifteenMinute, 50000.0)

	var ichimoku15mSignal *IndicatorSignal
	for i := range fifteenMinSignals {
		if fifteenMinSignals[i].Name == "Ichimoku_15m" {
			ichimoku15mSignal = &fifteenMinSignals[i]
			break
		}
	}

	if ichimoku15mSignal == nil {
		t.Fatal("Ichimoku signal not found in 15-minute signals")
	}

	// Print signal details for verification
	fmt.Printf("5-minute Ichimoku Signal: %+v\n", ichimokuSignal)
	fmt.Printf("15-minute Ichimoku Signal: %+v\n", ichimoku15mSignal)

	// Test that the enhanced signal calculation is being used
	// The enhanced signal should provide more nuanced strength values
	if ichimokuSignal.Strength < 0.1 || ichimokuSignal.Strength > 1.0 {
		t.Errorf("Expected strength between 0.1 and 1.0, got %f", ichimokuSignal.Strength)
	}

	// Test the signal aggregation end-to-end
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

	// Verify that Ichimoku is included in the indicator signals
	hasIchimoku := false
	for _, signal := range tradingSignal.IndicatorSignals {
		if signal.Name == "Ichimoku_5m" {
			hasIchimoku = true
			break
		}
	}

	if !hasIchimoku {
		t.Error("Ichimoku signal not found in trading signal indicators")
	}

	fmt.Printf("Final Trading Signal: %+v\n", tradingSignal)
}

func TestIchimokuEnhancedVsStandardSignal(t *testing.T) {
	// Create test candles
	candles := generateTestCandles(100, 45000.0)
	currentPrice := 50000.0

	// Create Ichimoku indicator with optimized 5-minute parameters
	ichimokuConfig := indicator.IchimokuConfig{
		Enabled:      true,
		TenkanPeriod: 6,
		KijunPeriod:  18,
		SenkouPeriod: 36,
		Displacement: 18,
	}

	ichimoku := indicator.NewIchimoku(ichimokuConfig, indicator.FiveMinute)

	// Convert candles to indicator format
	indicatorCandles := make([]indicator.Candle, len(candles))
	for i, candle := range candles {
		indicatorCandles[i] = indicator.Candle{
			Timestamp: candle.Timestamp,
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
		}
	}

	// Get standard signal
	values := ichimoku.Calculate(indicatorCandles)
	standardSignal := ichimoku.GetSignal(values, currentPrice)

	// Get enhanced 5-minute signal
	enhancedSignal := ichimoku.GetEnhanced5MinuteSignal(indicatorCandles, currentPrice)

	// Compare signals
	fmt.Printf("Standard Signal: %+v\n", standardSignal)
	fmt.Printf("Enhanced Signal: %+v\n", enhancedSignal)

	// Both should be valid signals
	if standardSignal.Name == "" || enhancedSignal.Name == "" {
		t.Error("Both signals should have names")
	}

	// Enhanced signal should provide more nuanced strength
	if enhancedSignal.Strength == 0 {
		t.Error("Enhanced signal should have non-zero strength for this test case")
	}

	// Test that the enhanced signal is being used in the main flow
	config := Config{
		Ichimoku: IchimokuConfig{
			Enabled:      true,
			TenkanPeriod: 6,
			KijunPeriod:  18,
			SenkouPeriod: 36,
			Displacement: 18,
		},
	}

	sa := NewSignalAggregator(config)
	signals := sa.getTimeframeSignals(candles, FiveMinute, currentPrice)

	// Find the Ichimoku signal from the aggregator
	var aggregatorSignal *IndicatorSignal
	for i := range signals {
		if signals[i].Name == "Ichimoku_5m" {
			aggregatorSignal = &signals[i]
			break
		}
	}

	if aggregatorSignal == nil {
		t.Fatal("Ichimoku signal not found in aggregator output")
	}

	// The aggregator should be using the enhanced signal
	// We can verify this by comparing the strength values
	fmt.Printf("Aggregator Signal: %+v\n", aggregatorSignal)

	// Test that the enhanced signal is indeed being used
	if aggregatorSignal.Strength != enhancedSignal.Strength {
		t.Errorf("Expected aggregator to use enhanced signal strength %f, got %f",
			enhancedSignal.Strength, aggregatorSignal.Strength)
	}
}
