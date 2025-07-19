package indicator

import (
	"fmt"
	"math"
	"time"
)

// MACD Indicator
type MACD struct {
	config    MACDConfig
	timeframe Timeframe
}

// NewMACD creates a new MACD indicator
func NewMACD(config MACDConfig, timeframe Timeframe) *MACD {
	return &MACD{
		config:    config,
		timeframe: timeframe,
	}
}

// Calculate computes MACD values for given candles
func (macd *MACD) Calculate(candles []Candle) []float64 {
	if len(candles) < macd.config.SlowPeriod {
		return []float64{}
	}

	// Calculate EMAs
	fastEMA := calculateEMA(candles, macd.config.FastPeriod)
	slowEMA := calculateEMA(candles, macd.config.SlowPeriod)

	// Calculate MACD line
	macdLine := make([]float64, len(slowEMA))
	for i := 0; i < len(slowEMA); i++ {
		macdLine[i] = fastEMA[i+len(fastEMA)-len(slowEMA)] - slowEMA[i]
	}

	return macdLine
}

// GetSignal generates a trading signal based on MACD values
func (macd *MACD) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) < 2 {
		return IndicatorSignal{
			Name:      macd.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: macd.timeframe,
		}
	}

	current := values[len(values)-1]
	previous := values[len(values)-2]

	var signal SignalType
	var strength float64

	// MACD crossover signals
	if current > 0 && previous <= 0 {
		// Bullish crossover
		signal = Buy
		strength = math.Min(1.0, math.Abs(current)/100) // Normalize strength
	} else if current < 0 && previous >= 0 {
		// Bearish crossover
		signal = Sell
		strength = math.Min(1.0, math.Abs(current)/100) // Normalize strength
	} else {
		signal = Hold
		strength = 0.3 // Low strength for trend continuation
	}

	return IndicatorSignal{
		Name:      macd.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     current,
		Timestamp: time.Now(),
		Timeframe: macd.timeframe,
	}
}

// GetName returns the indicator name
func (macd *MACD) GetName() string {
	return fmt.Sprintf("MACD_%s", macd.timeframe.String())
}

// Helper function to calculate EMA
func calculateEMA(candles []Candle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	values := make([]float64, len(candles)-period+1)
	multiplier := 2.0 / (float64(period) + 1.0)

	// Calculate initial SMA
	var sum float64
	for i := 0; i < period; i++ {
		sum += candles[i].Close
	}
	values[0] = sum / float64(period)

	// Calculate EMA
	for i := 1; i < len(values); i++ {
		values[i] = (candles[i+period-1].Close * multiplier) + (values[i-1] * (1 - multiplier))
	}

	return values
}
