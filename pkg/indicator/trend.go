package indicator

import (
	"fmt"
	"math"
	"time"
)

// Trend Indicator with adaptive sensitivity
type Trend struct {
	config    TrendConfig
	timeframe Timeframe
}

// NewTrend creates a new Trend indicator
func NewTrend(config TrendConfig, timeframe Timeframe) *Trend {
	return &Trend{
		config:    config,
		timeframe: timeframe,
	}
}

// getAdaptiveMAPeriods returns optimized MA periods based on timeframe
func (t *Trend) getAdaptiveMAPeriods() (int, int) {
	switch t.timeframe {
	case FiveMinute:
		// Higher sensitivity for 5-minute - catch quick reversals
		return 12, 26 // More sensitive than default 20/50
	case FifteenMinute:
		// Balanced sensitivity for 15-minute
		return t.config.ShortMA, t.config.LongMA // Use config defaults
	case FortyFiveMinute:
		// Medium-term trend - slightly less sensitive
		return 25, 60
	case EightHour:
		// Long-term trend - less sensitive, smoother
		return 30, 80
	case Daily:
		// Very long-term - much less sensitive
		return 50, 200
	default:
		return t.config.ShortMA, t.config.LongMA
	}
}

// Calculate computes trend analysis for given candles with adaptive sensitivity
func (t *Trend) Calculate(candles []Candle) []float64 {
	shortPeriod, longPeriod := t.getAdaptiveMAPeriods()

	if len(candles) < longPeriod {
		return []float64{}
	}

	// Calculate adaptive MAs
	shortMA := calculateSMA(candles, shortPeriod)
	longMA := calculateSMA(candles, longPeriod)

	// Calculate trend signal (short MA - long MA)
	trendSignal := make([]float64, len(longMA))
	for i := 0; i < len(longMA); i++ {
		trendSignal[i] = shortMA[i+len(shortMA)-len(longMA)] - longMA[i]
	}

	return trendSignal
}

// GetSignal generates a trading signal with adaptive sensitivity
func (t *Trend) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) < 2 {
		return IndicatorSignal{
			Name:      t.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: t.timeframe,
		}
	}

	current := values[len(values)-1]
	previous := values[len(values)-2]

	var signal SignalType
	var strength float64

	// Adaptive sensitivity thresholds based on timeframe
	sensitivityThreshold := t.getSensitivityThreshold(currentPrice)

	// Trend crossover signals with adaptive sensitivity
	if current > sensitivityThreshold && previous <= sensitivityThreshold {
		// Bullish trend
		signal = Buy
		strength = t.calculateAdaptiveStrength(current, currentPrice)
	} else if current < -sensitivityThreshold && previous >= -sensitivityThreshold {
		// Bearish trend
		signal = Sell
		strength = t.calculateAdaptiveStrength(current, currentPrice)
	} else {
		signal = Hold
		strength = 0.4 // Moderate strength for trend continuation
	}

	return IndicatorSignal{
		Name:      t.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     current,
		Timestamp: time.Now(),
		Timeframe: t.timeframe,
	}
}

// getSensitivityThreshold returns adaptive threshold based on timeframe
func (t *Trend) getSensitivityThreshold(currentPrice float64) float64 {
	baseThreshold := currentPrice * 0.0001 // 0.01% of price

	switch t.timeframe {
	case FiveMinute:
		return baseThreshold * 0.5 // More sensitive for 5-minute
	case FifteenMinute:
		return baseThreshold * 1.0 // Normal sensitivity
	case FortyFiveMinute:
		return baseThreshold * 1.5 // Less sensitive
	case EightHour:
		return baseThreshold * 2.0 // Much less sensitive
	case Daily:
		return baseThreshold * 3.0 // Least sensitive
	default:
		return baseThreshold
	}
}

// calculateAdaptiveStrength calculates strength with timeframe consideration
func (t *Trend) calculateAdaptiveStrength(trendValue, currentPrice float64) float64 {
	baseStrength := math.Min(1.0, math.Abs(trendValue)/currentPrice*100)

	// Timeframe-based strength adjustment
	switch t.timeframe {
	case FiveMinute:
		return baseStrength * 0.8 // Reduce strength for short-term signals
	case FifteenMinute:
		return baseStrength * 1.0 // Normal strength
	case FortyFiveMinute:
		return baseStrength * 1.1 // Slight boost for medium-term
	case EightHour:
		return baseStrength * 1.2 // Higher strength for long-term signals
	case Daily:
		return baseStrength * 1.3 // Highest strength for daily signals
	default:
		return baseStrength
	}
}

// GetName returns the indicator name
func (t *Trend) GetName() string {
	return fmt.Sprintf("Trend_%s", t.timeframe.String())
}

// Helper function to calculate SMA
func calculateSMA(candles []Candle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	values := make([]float64, len(candles)-period+1)

	for i := 0; i < len(values); i++ {
		var sum float64
		for j := 0; j < period; j++ {
			sum += candles[i+j].Close
		}
		values[i] = sum / float64(period)
	}

	return values
}
