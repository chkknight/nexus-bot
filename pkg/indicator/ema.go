package indicator

import (
	"fmt"
	"math"
	"time"
)

// EMAConfig holds EMA configuration
type EMAConfig struct {
	Enabled        bool    `json:"enabled"`         // Feature flag to enable/disable EMA
	FastPeriod     int     `json:"fast_period"`     // Fast EMA period (default: 12)
	SlowPeriod     int     `json:"slow_period"`     // Slow EMA period (default: 26)
	SignalPeriod   int     `json:"signal_period"`   // Signal EMA period (default: 9)
	TrendPeriod    int     `json:"trend_period"`    // Trend EMA period (default: 50)
	SlopeThreshold float64 `json:"slope_threshold"` // Minimum slope for trend signals (default: 0.0001)
	CrossoverBoost float64 `json:"crossover_boost"` // Boost for crossover signals (default: 1.3)
	TrendBoost     float64 `json:"trend_boost"`     // Boost for trend alignment (default: 1.2)
	VolumeConfirm  bool    `json:"volume_confirm"`  // Require volume confirmation (default: false)
}

// EMASignalType represents different EMA signal types
type EMASignalType int

const (
	EMANeutral EMASignalType = iota
	EMABullishCrossover
	EMABearishCrossover
	EMABullishTrend
	EMABearishTrend
	EMABullishMomentum
	EMABearishMomentum
)

func (e EMASignalType) String() string {
	switch e {
	case EMABullishCrossover:
		return "Bullish Crossover"
	case EMABearishCrossover:
		return "Bearish Crossover"
	case EMABullishTrend:
		return "Bullish Trend"
	case EMABearishTrend:
		return "Bearish Trend"
	case EMABullishMomentum:
		return "Bullish Momentum"
	case EMABearishMomentum:
		return "Bearish Momentum"
	default:
		return "Neutral"
	}
}

// EMA represents the Exponential Moving Average indicator
type EMA struct {
	config        EMAConfig
	timeframe     Timeframe
	prices        []float64
	fastEMA       []float64
	slowEMA       []float64
	signalEMA     []float64
	trendEMA      []float64
	lastSignal    SignalType
	lastStrength  float64
	lastEMASignal EMASignalType
	initialized   bool
}

// NewEMA creates a new EMA indicator
func NewEMA(config EMAConfig, timeframe Timeframe) *EMA {
	return &EMA{
		config:        config,
		timeframe:     timeframe,
		prices:        make([]float64, 0),
		fastEMA:       make([]float64, 0),
		slowEMA:       make([]float64, 0),
		signalEMA:     make([]float64, 0),
		trendEMA:      make([]float64, 0),
		lastSignal:    Hold,
		lastStrength:  0.0,
		lastEMASignal: EMANeutral,
		initialized:   false,
	}
}

// Update processes new price data
func (ema *EMA) Update(candle Candle) {
	price := candle.Close
	ema.prices = append(ema.prices, price)

	// Maintain price buffer
	maxSize := int(math.Max(float64(ema.config.TrendPeriod), float64(ema.config.SlowPeriod))) + 50
	if len(ema.prices) > maxSize {
		ema.prices = ema.prices[1:]
	}

	// Calculate EMAs
	ema.calculateEMAs()

	// Mark as initialized when we have enough data
	if len(ema.fastEMA) >= 3 && len(ema.slowEMA) >= 3 && len(ema.signalEMA) >= 3 && len(ema.trendEMA) >= 3 {
		ema.initialized = true
	}
}

// calculateEMAs calculates all EMA lines
func (ema *EMA) calculateEMAs() {
	if len(ema.prices) == 0 {
		return
	}

	// Calculate Fast EMA
	ema.fastEMA = ema.calculateEMA(ema.prices, ema.config.FastPeriod, ema.fastEMA)

	// Calculate Slow EMA
	ema.slowEMA = ema.calculateEMA(ema.prices, ema.config.SlowPeriod, ema.slowEMA)

	// Calculate Signal EMA (EMA of the difference between Fast and Slow)
	if len(ema.fastEMA) > 0 && len(ema.slowEMA) > 0 {
		macdLine := ema.fastEMA[len(ema.fastEMA)-1] - ema.slowEMA[len(ema.slowEMA)-1]
		macdValues := []float64{macdLine}
		if len(ema.signalEMA) > 0 {
			// Get recent MACD values for signal EMA calculation
			recentMACD := make([]float64, 0)
			start := int(math.Max(0, float64(len(ema.fastEMA)-ema.config.SignalPeriod)))
			for i := start; i < len(ema.fastEMA); i++ {
				if i < len(ema.slowEMA) {
					recentMACD = append(recentMACD, ema.fastEMA[i]-ema.slowEMA[i])
				}
			}
			macdValues = recentMACD
		}
		ema.signalEMA = ema.calculateEMA(macdValues, ema.config.SignalPeriod, ema.signalEMA)
	}

	// Calculate Trend EMA
	ema.trendEMA = ema.calculateEMA(ema.prices, ema.config.TrendPeriod, ema.trendEMA)
}

// calculateEMA calculates EMA for given data and period
func (ema *EMA) calculateEMA(data []float64, period int, existing []float64) []float64 {
	if len(data) == 0 {
		return existing
	}

	alpha := 2.0 / (float64(period) + 1.0)

	if len(existing) == 0 {
		// Initialize with SMA for the first value
		if len(data) >= period {
			sum := 0.0
			for i := 0; i < period; i++ {
				sum += data[i]
			}
			sma := sum / float64(period)
			return []float64{sma}
		}
		return []float64{}
	}

	// Calculate new EMA value
	currentPrice := data[len(data)-1]
	previousEMA := existing[len(existing)-1]
	newEMA := alpha*currentPrice + (1-alpha)*previousEMA

	result := append(existing, newEMA)

	// Maintain buffer size
	if len(result) > period+20 {
		result = result[1:]
	}

	return result
}

// GetCurrentSignal returns the current EMA signal
func (ema *EMA) GetCurrentSignal() (SignalType, float64) {
	if !ema.initialized {
		return Hold, 0.0
	}

	signal, strength, emaSignalType := ema.analyzeEMASignals()

	ema.lastSignal = signal
	ema.lastStrength = strength
	ema.lastEMASignal = emaSignalType

	return signal, strength
}

// analyzeEMASignals analyzes all EMA signals and returns the strongest
func (ema *EMA) analyzeEMASignals() (SignalType, float64, EMASignalType) {
	if len(ema.fastEMA) < 3 || len(ema.slowEMA) < 3 || len(ema.signalEMA) < 3 || len(ema.trendEMA) < 3 {
		return Hold, 0.0, EMANeutral
	}

	// Get current and previous values
	currentFast := ema.fastEMA[len(ema.fastEMA)-1]
	previousFast := ema.fastEMA[len(ema.fastEMA)-2]
	currentSlow := ema.slowEMA[len(ema.slowEMA)-1]
	previousSlow := ema.slowEMA[len(ema.slowEMA)-2]
	currentSignal := ema.signalEMA[len(ema.signalEMA)-1]
	previousSignal := ema.signalEMA[len(ema.signalEMA)-2]
	currentTrend := ema.trendEMA[len(ema.trendEMA)-1]
	previousTrend := ema.trendEMA[len(ema.trendEMA)-2]

	// Check for crossover signals
	if crossoverSignal, crossoverStrength := ema.checkCrossover(currentFast, previousFast, currentSlow, previousSlow); crossoverSignal != Hold {
		if crossoverSignal == Buy {
			return crossoverSignal, crossoverStrength, EMABullishCrossover
		} else {
			return crossoverSignal, crossoverStrength, EMABearishCrossover
		}
	}

	// Check for MACD-style signals
	if macdSignal, macdStrength := ema.checkMACDSignals(currentFast, currentSlow, currentSignal, previousSignal); macdSignal != Hold {
		if macdSignal == Buy {
			return macdSignal, macdStrength, EMABullishMomentum
		} else {
			return macdSignal, macdStrength, EMABearishMomentum
		}
	}

	// Check for trend signals
	if trendSignal, trendStrength := ema.checkTrendSignals(currentFast, currentSlow, currentTrend, previousTrend); trendSignal != Hold {
		if trendSignal == Buy {
			return trendSignal, trendStrength, EMABullishTrend
		} else {
			return trendSignal, trendStrength, EMABearishTrend
		}
	}

	return Hold, 0.0, EMANeutral
}

// checkCrossover checks for EMA crossover signals
func (ema *EMA) checkCrossover(currentFast, previousFast, currentSlow, previousSlow float64) (SignalType, float64) {
	// Bullish crossover: Fast EMA crosses above Slow EMA
	if currentFast > currentSlow && previousFast <= previousSlow {
		strength := ema.calculateCrossoverStrength(currentFast, currentSlow, true)
		return Buy, strength * ema.config.CrossoverBoost
	}

	// Bearish crossover: Fast EMA crosses below Slow EMA
	if currentFast < currentSlow && previousFast >= previousSlow {
		strength := ema.calculateCrossoverStrength(currentFast, currentSlow, false)
		return Sell, strength * ema.config.CrossoverBoost
	}

	return Hold, 0.0
}

// checkMACDSignals checks for MACD-style signals using EMA difference
func (ema *EMA) checkMACDSignals(currentFast, currentSlow, currentSignal, previousSignal float64) (SignalType, float64) {
	currentMACD := currentFast - currentSlow
	previousMACD := currentFast - currentSlow // Approximation for previous MACD

	// Bullish signal: MACD line crosses above signal line
	if currentMACD > currentSignal && previousMACD <= previousSignal {
		strength := ema.calculateMACDStrength(currentMACD, currentSignal, true)
		return Buy, strength
	}

	// Bearish signal: MACD line crosses below signal line
	if currentMACD < currentSignal && previousMACD >= previousSignal {
		strength := ema.calculateMACDStrength(currentMACD, currentSignal, false)
		return Sell, strength
	}

	return Hold, 0.0
}

// checkTrendSignals checks for trend-based signals
func (ema *EMA) checkTrendSignals(currentFast, currentSlow, currentTrend, previousTrend float64) (SignalType, float64) {
	// Calculate trend slope
	trendSlope := (currentTrend - previousTrend) / previousTrend

	// Strong uptrend: All EMAs aligned bullishly with good slope
	if currentFast > currentSlow && currentSlow > currentTrend && trendSlope > ema.config.SlopeThreshold {
		strength := ema.calculateTrendStrength(currentFast, currentSlow, currentTrend, trendSlope, true)
		return Buy, strength * ema.config.TrendBoost
	}

	// Strong downtrend: All EMAs aligned bearishly with good slope
	if currentFast < currentSlow && currentSlow < currentTrend && trendSlope < -ema.config.SlopeThreshold {
		strength := ema.calculateTrendStrength(currentFast, currentSlow, currentTrend, trendSlope, false)
		return Sell, strength * ema.config.TrendBoost
	}

	return Hold, 0.0
}

// calculateCrossoverStrength calculates the strength of a crossover signal
func (ema *EMA) calculateCrossoverStrength(fast, slow float64, bullish bool) float64 {
	separation := math.Abs(fast-slow) / slow
	baseStrength := math.Min(separation*100, 0.8) // Max 80% from separation

	// Add momentum component
	if len(ema.fastEMA) >= 3 {
		fastMomentum := (ema.fastEMA[len(ema.fastEMA)-1] - ema.fastEMA[len(ema.fastEMA)-3]) / ema.fastEMA[len(ema.fastEMA)-3]
		momentumStrength := math.Min(math.Abs(fastMomentum)*50, 0.2) // Max 20% from momentum
		baseStrength += momentumStrength
	}

	return math.Min(baseStrength, 1.0)
}

// calculateMACDStrength calculates the strength of a MACD-style signal
func (ema *EMA) calculateMACDStrength(macd, signal float64, bullish bool) float64 {
	separation := math.Abs(macd-signal) / math.Max(math.Abs(macd), math.Abs(signal))
	return math.Min(separation*2, 0.8) // Max 80% strength
}

// calculateTrendStrength calculates the strength of a trend signal
func (ema *EMA) calculateTrendStrength(fast, slow, trend, slope float64, bullish bool) float64 {
	// EMA alignment strength
	alignmentStrength := 0.0
	if bullish {
		alignmentStrength = math.Min((fast-slow)/slow+(slow-trend)/trend, 0.6)
	} else {
		alignmentStrength = math.Min((slow-fast)/fast+(trend-slow)/slow, 0.6)
	}

	// Slope strength
	slopeStrength := math.Min(math.Abs(slope)*1000, 0.4) // Max 40% from slope

	return alignmentStrength + slopeStrength
}

// Calculate implements TechnicalIndicator interface
func (ema *EMA) Calculate(candles []Candle) []float64 {
	if len(candles) < ema.config.SlowPeriod {
		return []float64{}
	}

	values := make([]float64, 0, len(candles))

	// Process each candle
	for _, candle := range candles {
		ema.Update(candle)
		if ema.initialized && len(ema.fastEMA) > 0 {
			// Return the fast EMA value
			values = append(values, ema.fastEMA[len(ema.fastEMA)-1])
		}
	}

	return values
}

// GetSignal implements TechnicalIndicator interface
func (ema *EMA) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	signal, strength := ema.GetCurrentSignal()

	var value float64
	if len(values) > 0 {
		value = values[len(values)-1]
	}

	return IndicatorSignal{
		Name:      ema.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     value,
		Timestamp: time.Now(),
		Timeframe: ema.timeframe,
	}
}

// GetName returns the indicator name
func (ema *EMA) GetName() string {
	return "EMA"
}

// GetLastSignal returns the last signal and strength
func (ema *EMA) GetLastSignal() (SignalType, float64) {
	return ema.lastSignal, ema.lastStrength
}

// GetLastEMASignal returns the last EMA signal type
func (ema *EMA) GetLastEMASignal() EMASignalType {
	return ema.lastEMASignal
}

// GetCurrentValues returns current EMA values for debugging
func (ema *EMA) GetCurrentValues() (float64, float64, float64, float64) {
	var fast, slow, signal, trend float64

	if len(ema.fastEMA) > 0 {
		fast = ema.fastEMA[len(ema.fastEMA)-1]
	}
	if len(ema.slowEMA) > 0 {
		slow = ema.slowEMA[len(ema.slowEMA)-1]
	}
	if len(ema.signalEMA) > 0 {
		signal = ema.signalEMA[len(ema.signalEMA)-1]
	}
	if len(ema.trendEMA) > 0 {
		trend = ema.trendEMA[len(ema.trendEMA)-1]
	}

	return fast, slow, signal, trend
}

// String returns a string representation
func (ema *EMA) String() string {
	if !ema.initialized {
		return "EMA: Not initialized"
	}

	fast, slow, signal, trend := ema.GetCurrentValues()
	return fmt.Sprintf("EMA: Fast=%.2f, Slow=%.2f, Signal=%.2f, Trend=%.2f, Type=%s, Signal=%s, Strength=%.2f",
		fast, slow, signal, trend, ema.lastEMASignal.String(), ema.lastSignal, ema.lastStrength)
}
