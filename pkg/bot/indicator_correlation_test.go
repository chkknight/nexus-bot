package bot

import (
	"fmt"
	"testing"
	"time"
)

func TestIndicatorCorrelationAnalysis(t *testing.T) {
	t.Log("\n🎯 INDICATOR CORRELATION ANALYSIS FOR 5-MINUTE TRADING")
	t.Log("=======================================================")

	// Test different market scenarios
	scenarios := []struct {
		name        string
		description string
		candles     []Candle
	}{
		{
			name:        "Strong Uptrend",
			description: "Consistent upward movement with volume",
			candles:     generateCorrelationTrendingCandles(60, 50000.0, 200.0, true),
		},
		{
			name:        "Strong Downtrend",
			description: "Consistent downward movement with volume",
			candles:     generateCorrelationTrendingCandles(60, 50000.0, 200.0, false),
		},
		{
			name:        "Sideways Range",
			description: "Consolidation with low volatility",
			candles:     generateCorrelationRangingCandles(60, 50000.0, 100.0),
		},
		{
			name:        "High Volatility",
			description: "Large price swings with high volume",
			candles:     generateCorrelationVolatileCandles(60, 50000.0, 500.0),
		},
		{
			name:        "Breakout",
			description: "Squeeze followed by explosive move",
			candles:     generateCorrelationBreakoutCandles(60, 50000.0),
		},
	}

	// Full indicator configuration
	config := Config{
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
		BollingerBands: BollingerBandsConfig{
			Enabled:       true,
			Period:        14,
			StandardDev:   2.0,
			OverboughtStd: 0.85,
			OversoldStd:   0.15,
		},
		MFI: MFIConfig{
			Enabled:    true,
			Period:     14,
			Overbought: 80,
			Oversold:   20,
		},
		Ichimoku: IchimokuConfig{
			Enabled:      true,
			TenkanPeriod: 6,
			KijunPeriod:  18,
			SenkouPeriod: 36,
			Displacement: 18,
		},
		SupportResistance: SupportResistanceConfig{
			Enabled:   false, // Disabled due to poor accuracy
			Period:    20,
			Threshold: 0.02,
		},
	}

	sa := NewSignalAggregator(config)

	for _, scenario := range scenarios {
		t.Run(scenario.name, func(t *testing.T) {
			t.Logf("\n📊 SCENARIO: %s", scenario.name)
			t.Logf("Description: %s", scenario.description)
			t.Logf("===========================================")

			// Get signals from all indicators
			signals := sa.getTimeframeSignals(scenario.candles, FiveMinute, 50000.0)

			// Analyze signal correlation
			analyzeSignalCorrelation(t, signals, scenario.name)
		})
	}
}

func analyzeSignalCorrelation(t *testing.T, signals []IndicatorSignal, scenario string) {
	t.Logf("\n🔍 SIGNAL ANALYSIS:")

	// Group signals by type
	buySignals := []IndicatorSignal{}
	sellSignals := []IndicatorSignal{}
	holdSignals := []IndicatorSignal{}

	for _, signal := range signals {
		switch signal.Signal {
		case Buy:
			buySignals = append(buySignals, signal)
		case Sell:
			sellSignals = append(sellSignals, signal)
		case Hold:
			holdSignals = append(holdSignals, signal)
		}
	}

	t.Logf("BUY Signals (%d):", len(buySignals))
	for _, s := range buySignals {
		t.Logf("  %s: %.3f strength", s.Name, s.Strength)
	}

	t.Logf("SELL Signals (%d):", len(sellSignals))
	for _, s := range sellSignals {
		t.Logf("  %s: %.3f strength", s.Name, s.Strength)
	}

	t.Logf("HOLD Signals (%d):", len(holdSignals))
	for _, s := range holdSignals {
		t.Logf("  %s: %.3f strength", s.Name, s.Strength)
	}

	// Calculate consensus
	consensus := calculateConsensus(buySignals, sellSignals, holdSignals)
	t.Logf("📈 CONSENSUS: %s", consensus)
}

func calculateConsensus(buy, sell, hold []IndicatorSignal) string {
	if len(buy) > len(sell) && len(buy) > len(hold) {
		return fmt.Sprintf("BULLISH (%d BUY vs %d SELL)", len(buy), len(sell))
	} else if len(sell) > len(buy) && len(sell) > len(hold) {
		return fmt.Sprintf("BEARISH (%d SELL vs %d BUY)", len(sell), len(buy))
	} else {
		return fmt.Sprintf("NEUTRAL (%d BUY, %d SELL, %d HOLD)", len(buy), len(sell), len(hold))
	}
}

func TestOptimalIndicatorCombinations(t *testing.T) {
	t.Log("\n🎯 OPTIMAL INDICATOR COMBINATIONS")
	t.Log("=================================")

	// Define indicator categories
	trendFollowing := []string{"Trend", "MACD", "Ichimoku"}
	meanReverting := []string{"RSI", "BollingerBands", "ReverseMFI"}
	volumeBased := []string{"Volume", "ReverseMFI"}
	volatility := []string{"BollingerBands"}

	t.Log("\n📊 INDICATOR CATEGORIZATION:")
	t.Log("============================")
	t.Logf("📈 TREND-FOLLOWING: %v", trendFollowing)
	t.Logf("🔄 MEAN-REVERTING: %v", meanReverting)
	t.Logf("📊 VOLUME-BASED: %v", volumeBased)
	t.Logf("⚡ VOLATILITY: %v", volatility)

	t.Log("\n🎯 RECOMMENDED COMBINATIONS:")
	t.Log("=============================")

	combinations := []struct {
		name        string
		indicators  []string
		description string
		bestFor     string
	}{
		{
			name:        "Trend Confirmation",
			indicators:  []string{"Volume", "Trend", "MACD"},
			description: "High-accuracy trending combination",
			bestFor:     "Strong directional moves",
		},
		{
			name:        "Mean Reversion",
			indicators:  []string{"BollingerBands", "RSI", "ReverseMFI"},
			description: "Oversold/overbought signals",
			bestFor:     "Range-bound markets",
		},
		{
			name:        "Breakout Detection",
			indicators:  []string{"Volume", "BollingerBands", "MACD"},
			description: "Volume + volatility + momentum",
			bestFor:     "Squeeze breakouts",
		},
		{
			name:        "High-Accuracy Core",
			indicators:  []string{"Volume", "Trend", "MACD", "BollingerBands"},
			description: "Top performers only",
			bestFor:     "Consistent profitability",
		},
		{
			name:        "Divergence Detection",
			indicators:  []string{"Volume", "RSI", "MACD"},
			description: "Price vs indicator divergence",
			bestFor:     "Early reversal signals",
		},
	}

	for _, combo := range combinations {
		t.Logf("\n🔧 %s:", combo.name)
		t.Logf("   Indicators: %v", combo.indicators)
		t.Logf("   Description: %s", combo.description)
		t.Logf("   Best for: %s", combo.bestFor)
	}
}

func TestIndicatorStrengthMatrix(t *testing.T) {
	t.Log("\n📊 INDICATOR STRENGTH MATRIX")
	t.Log("=============================")

	// Current accuracy rankings
	indicators := []struct {
		name     string
		accuracy float64
		weight   float64
		category string
		lag      string
		strength string
	}{
		{"Volume", 87.1, 1.3, "Volume", "Coincident", "Confirmation"},
		{"Trend", 83.9, 1.2, "Trend", "Lagging", "Direction"},
		{"MACD", 80.6, 1.1, "Momentum", "Lagging", "Crossover"},
		{"BollingerBands", 75.0, 1.0, "Volatility", "Coincident", "Mean Reversion"}, // Estimated
		{"ReverseMFI", 61.3, 1.0, "Volume", "Leading", "Contrarian"},
		{"RSI", 41.9, 0.9, "Momentum", "Leading", "Overbought/Oversold"},
		{"Ichimoku", 40.0, 0.8, "Trend", "Mixed", "Cloud Analysis"}, // Estimated enhanced
		{"S&R", 9.7, 0.0, "Price", "Lagging", "Levels"},             // Disabled
	}

	t.Log("\n🎯 PERFORMANCE RANKING:")
	t.Log("Rank | Indicator      | Accuracy | Weight | Category   | Timing     | Strength")
	t.Log("-----|----------------|----------|--------|------------|------------|----------------")

	for i, ind := range indicators {
		status := "✅"
		if ind.accuracy < 50 {
			status = "⚠️"
		}
		if ind.weight == 0 {
			status = "❌"
		}

		t.Logf("%s %2d | %-14s | %5.1f%%  | %5.1fx | %-10s | %-10s | %s",
			status, i+1, ind.name, ind.accuracy, ind.weight, ind.category, ind.lag, ind.strength)
	}
}

func TestConflictingSignalAnalysis(t *testing.T) {
	t.Log("\n⚠️  CONFLICTING SIGNAL ANALYSIS")
	t.Log("==============================")

	conflicts := []struct {
		situation  string
		indicators []string
		reason     string
		resolution string
	}{
		{
			situation:  "Trend vs Mean Reversion",
			indicators: []string{"Trend=BUY", "BollingerBands=SELL"},
			reason:     "Trend up but price overbought",
			resolution: "Wait for pullback or use higher timeframe",
		},
		{
			situation:  "Volume vs Price",
			indicators: []string{"Volume=HOLD", "MACD=BUY"},
			reason:     "No volume confirmation for move",
			resolution: "Reduce position size or wait",
		},
		{
			situation:  "RSI vs Momentum",
			indicators: []string{"RSI=SELL", "MACD=BUY"},
			reason:     "Overbought but momentum strong",
			resolution: "Trust higher accuracy indicator (MACD)",
		},
		{
			situation:  "Short-term vs Long-term",
			indicators: []string{"5m=BUY", "15m=SELL"},
			reason:     "Timeframe divergence",
			resolution: "Use higher timeframe bias",
		},
	}

	t.Log("\n🔍 COMMON CONFLICTS:")
	for _, conflict := range conflicts {
		t.Logf("\n📊 %s:", conflict.situation)
		t.Logf("   Signals: %v", conflict.indicators)
		t.Logf("   Reason: %s", conflict.reason)
		t.Logf("   Resolution: %s", conflict.resolution)
	}
}

// Helper functions for test data generation

func generateCorrelationTrendingCandles(count int, basePrice, increment float64, uptrend bool) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		var priceChange float64
		if uptrend {
			priceChange = float64(i) * increment / 10
		} else {
			priceChange = -float64(i) * increment / 10
		}

		price := basePrice + priceChange
		volatility := 50.0 + float64(i%5)*20.0

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - volatility/2,
			High:      price + volatility,
			Low:       price - volatility,
			Close:     price,
			Volume:    15000 + float64(i*200), // Increasing volume
		}
	}

	return candles
}

func generateCorrelationRangingCandles(count int, basePrice, rangeSize float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Oscillate within range
		priceOffset := rangeSize*float64(i%8)/8 - rangeSize/2
		price := basePrice + priceOffset
		volatility := 30.0

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - volatility/2,
			High:      price + volatility,
			Low:       price - volatility,
			Close:     price,
			Volume:    12000 + float64(i%3)*1000, // Low, steady volume
		}
	}

	return candles
}

func generateCorrelationVolatileCandles(count int, basePrice, volatility float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		// Random large moves
		priceChange := volatility * (float64(i%7) - 3) / 3
		price := basePrice + priceChange
		vol := volatility + float64(i%4)*100

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - vol/2,
			High:      price + vol,
			Low:       price - vol,
			Close:     price,
			Volume:    20000 + float64(i*300), // High volume
		}
	}

	return candles
}

func generateCorrelationBreakoutCandles(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	baseTime := time.Now().Add(-time.Duration(count) * 5 * time.Minute)

	for i := 0; i < count; i++ {
		var price float64
		var vol float64
		var volume float64

		if i < count/2 {
			// Compression phase
			price = basePrice + float64(i%3)*10.0
			vol = 20.0 - float64(i)*0.3    // Decreasing volatility
			volume = 10000 - float64(i)*50 // Decreasing volume
		} else {
			// Breakout phase
			breakoutStart := count / 2
			breakoutMagnitude := float64(i-breakoutStart) * 100.0
			price = basePrice + breakoutMagnitude
			vol = 200.0 + float64(i-breakoutStart)*50.0    // Increasing volatility
			volume = 25000 + float64(i-breakoutStart)*1000 // Surging volume
		}

		candles[i] = Candle{
			Timestamp: baseTime.Add(time.Duration(i) * 5 * time.Minute),
			Open:      price - vol/2,
			High:      price + vol,
			Low:       price - vol,
			Close:     price,
			Volume:    volume,
		}
	}

	return candles
}
