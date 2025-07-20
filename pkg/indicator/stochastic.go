package indicator

import (
	"fmt"
	"math"
	"time"
)

// StochasticConfig holds Stochastic Oscillator configuration
type StochasticConfig struct {
	Enabled         bool    `json:"enabled"`          // Feature flag to enable/disable Stochastic
	KPeriod         int     `json:"k_period"`         // Period for %K calculation (default: 14)
	DPeriod         int     `json:"d_period"`         // Period for %D smoothing (default: 3)
	SlowPeriod      int     `json:"slow_period"`      // Period for slow %K (default: 3)
	Overbought      float64 `json:"overbought"`       // Overbought threshold (default: 80)
	Oversold        float64 `json:"oversold"`         // Oversold threshold (default: 20)
	MomentumBoost   float64 `json:"momentum_boost"`   // Boost factor for momentum signals (default: 1.2)
	DivergenceBoost float64 `json:"divergence_boost"` // Boost for divergence signals (default: 1.3)
}

// Stochastic represents the Stochastic Oscillator
type Stochastic struct {
	config       StochasticConfig
	timeframe    Timeframe
	kValues      []float64 // Fast %K values
	dValues      []float64 // %D values (smoothed %K)
	slowKValues  []float64 // Slow %K values
	highPrices   []float64 // High prices buffer
	lowPrices    []float64 // Low prices buffer
	closePrices  []float64 // Close prices buffer
	lastSignal   SignalType
	lastStrength float64
	initialized  bool
}

// NewStochastic creates a new Stochastic Oscillator
func NewStochastic(config StochasticConfig, timeframe Timeframe) *Stochastic {
	return &Stochastic{
		config:       config,
		timeframe:    timeframe,
		kValues:      make([]float64, 0),
		dValues:      make([]float64, 0),
		slowKValues:  make([]float64, 0),
		highPrices:   make([]float64, 0),
		lowPrices:    make([]float64, 0),
		closePrices:  make([]float64, 0),
		lastSignal:   Hold,
		lastStrength: 0.0,
		initialized:  false,
	}
}

// Update processes new price data and updates the Stochastic values
func (s *Stochastic) Update(data Candle) {
	// Add new price data
	s.highPrices = append(s.highPrices, data.High)
	s.lowPrices = append(s.lowPrices, data.Low)
	s.closePrices = append(s.closePrices, data.Close)

	// Maintain buffer size
	maxSize := s.config.KPeriod + s.config.DPeriod + s.config.SlowPeriod + 10
	if len(s.highPrices) > maxSize {
		s.highPrices = s.highPrices[1:]
		s.lowPrices = s.lowPrices[1:]
		s.closePrices = s.closePrices[1:]
	}

	// Calculate %K if we have enough data
	if len(s.highPrices) >= s.config.KPeriod {
		s.calculateFastK()
		s.calculateSlowK()
		s.calculateD()
		s.initialized = true
	}
}

// calculateFastK calculates the fast %K values
func (s *Stochastic) calculateFastK() {
	if len(s.highPrices) < s.config.KPeriod {
		return
	}

	// Get the last K periods
	start := len(s.highPrices) - s.config.KPeriod
	highs := s.highPrices[start:]
	lows := s.lowPrices[start:]
	currentClose := s.closePrices[len(s.closePrices)-1]

	// Find highest high and lowest low
	highestHigh := highs[0]
	lowestLow := lows[0]

	for i := 1; i < len(highs); i++ {
		if highs[i] > highestHigh {
			highestHigh = highs[i]
		}
		if lows[i] < lowestLow {
			lowestLow = lows[i]
		}
	}

	// Calculate %K
	var fastK float64
	if highestHigh-lowestLow != 0 {
		fastK = ((currentClose - lowestLow) / (highestHigh - lowestLow)) * 100
	} else {
		fastK = 50 // Neutral when no range
	}

	s.kValues = append(s.kValues, fastK)

	// Maintain buffer
	if len(s.kValues) > s.config.KPeriod+s.config.DPeriod+5 {
		s.kValues = s.kValues[1:]
	}
}

// calculateSlowK calculates the slow %K (smoothed fast %K)
func (s *Stochastic) calculateSlowK() {
	if len(s.kValues) < s.config.SlowPeriod {
		return
	}

	// Calculate slow %K as SMA of fast %K
	start := len(s.kValues) - s.config.SlowPeriod
	sum := 0.0
	for i := start; i < len(s.kValues); i++ {
		sum += s.kValues[i]
	}
	slowK := sum / float64(s.config.SlowPeriod)

	s.slowKValues = append(s.slowKValues, slowK)

	// Maintain buffer
	if len(s.slowKValues) > s.config.DPeriod+5 {
		s.slowKValues = s.slowKValues[1:]
	}
}

// calculateD calculates the %D values (smoothed slow %K)
func (s *Stochastic) calculateD() {
	if len(s.slowKValues) < s.config.DPeriod {
		return
	}

	// Calculate %D as SMA of slow %K
	start := len(s.slowKValues) - s.config.DPeriod
	sum := 0.0
	for i := start; i < len(s.slowKValues); i++ {
		sum += s.slowKValues[i]
	}
	d := sum / float64(s.config.DPeriod)

	s.dValues = append(s.dValues, d)

	// Maintain buffer
	if len(s.dValues) > 10 {
		s.dValues = s.dValues[1:]
	}
}

// GetCurrentSignal returns the current Stochastic signal
func (s *Stochastic) GetCurrentSignal() (SignalType, float64) {
	if !s.initialized || len(s.slowKValues) < 2 || len(s.dValues) < 2 {
		return Hold, 0.0
	}

	// Get current and previous values
	currentK := s.slowKValues[len(s.slowKValues)-1]
	previousK := s.slowKValues[len(s.slowKValues)-2]
	currentD := s.dValues[len(s.dValues)-1]
	previousD := s.dValues[len(s.dValues)-2]

	// Calculate signal strength based on position and momentum
	strength := s.calculateSignalStrength(currentK, currentD, previousK, previousD)

	// Determine signal based on crossovers and levels
	signal := s.determineSignal(currentK, currentD, previousK, previousD, strength)

	s.lastSignal = signal
	s.lastStrength = strength

	return signal, strength
}

// calculateSignalStrength calculates the strength of the signal - FIXED: More conservative
func (s *Stochastic) calculateSignalStrength(currentK, currentD, previousK, previousD float64) float64 {
	baseStrength := 0.0

	// Position-based strength - FIXED: More conservative values
	if currentK > s.config.Overbought && currentD > s.config.Overbought {
		baseStrength = 0.6 // FIXED: Reduced from 0.8 - not maximum confidence
	} else if currentK < s.config.Oversold && currentD < s.config.Oversold {
		baseStrength = 0.6 // FIXED: Reduced from 0.8 - not maximum confidence
	} else if currentK > 70 || currentD > 70 {
		baseStrength = 0.4 // FIXED: Reduced from 0.6 - moderate
	} else if currentK < 30 || currentD < 30 {
		baseStrength = 0.4 // FIXED: Reduced from 0.6 - moderate
	} else {
		baseStrength = 0.3 // FIXED: Reduced from 0.4 - neutral zone
	}

	// Crossover strength - FIXED: Smaller contribution
	crossoverStrength := 0.0
	if (currentK > currentD && previousK <= previousD) || (currentK < currentD && previousK >= previousD) {
		crossoverStrength = 0.15 // FIXED: Reduced from 0.3
	}

	// Momentum strength - FIXED: Smaller contribution
	momentumStrength := 0.0
	kMomentum := math.Abs(currentK - previousK)
	dMomentum := math.Abs(currentD - previousD)
	if kMomentum > 5 || dMomentum > 3 {
		momentumStrength = 0.1 * s.config.MomentumBoost // FIXED: Reduced from 0.2
	}

	// FIXED: Use weighted average instead of sum to prevent exceeding reasonable strength
	totalStrength := (baseStrength * 0.6) + (crossoverStrength * 0.3) + (momentumStrength * 0.1)

	// Cap at reasonable maximum - FIXED: Lower maximum
	if totalStrength > 0.85 {
		totalStrength = 0.85 // FIXED: Max 0.85 instead of 1.0
	}

	return totalStrength
}

// determineSignal determines the buy/sell/hold signal
func (s *Stochastic) determineSignal(currentK, currentD, previousK, previousD, strength float64) SignalType {
	// Bullish crossover (K crosses above D) in oversold territory
	if currentK > currentD && previousK <= previousD && currentK < 50 {
		return Buy
	}

	// Bearish crossover (K crosses below D) in overbought territory
	if currentK < currentD && previousK >= previousD && currentK > 50 {
		return Sell
	}

	// Strong overbought condition
	if currentK > s.config.Overbought && currentD > s.config.Overbought && strength > 0.7 {
		return Sell
	}

	// Strong oversold condition
	if currentK < s.config.Oversold && currentD < s.config.Oversold && strength > 0.7 {
		return Buy
	}

	return Hold
}

// GetEnhanced5MinuteSignal returns enhanced signal for 5-minute trading
func (s *Stochastic) GetEnhanced5MinuteSignal() (SignalType, float64) {
	if s.timeframe != FiveMinute {
		return s.GetCurrentSignal()
	}

	signal, strength := s.GetCurrentSignal()

	// Enhanced 5-minute specific logic
	if len(s.slowKValues) >= 3 && len(s.dValues) >= 3 {
		currentK := s.slowKValues[len(s.slowKValues)-1]
		_ = s.dValues[len(s.dValues)-1] // currentD not used

		// Quick momentum detection for 5-minute - FIXED: More conservative strength
		if len(s.kValues) >= 3 {
			recent := s.kValues[len(s.kValues)-3:]
			if recent[2] > recent[1] && recent[1] > recent[0] && currentK < 40 {
				// Strong upward momentum in lower territory
				strength = math.Min(strength*1.15, 0.85) // FIXED: Max 0.85 instead of 1.0
				signal = Buy
			} else if recent[2] < recent[1] && recent[1] < recent[0] && currentK > 60 {
				// Strong downward momentum in upper territory
				strength = math.Min(strength*1.15, 0.85) // FIXED: Max 0.85 instead of 1.0
				signal = Sell
			}
		}
	}

	return signal, strength
}

// GetCurrentValues returns current Stochastic values for debugging
func (s *Stochastic) GetCurrentValues() (float64, float64, float64) {
	if !s.initialized {
		return 0, 0, 0
	}

	var fastK, slowK, d float64

	if len(s.kValues) > 0 {
		fastK = s.kValues[len(s.kValues)-1]
	}
	if len(s.slowKValues) > 0 {
		slowK = s.slowKValues[len(s.slowKValues)-1]
	}
	if len(s.dValues) > 0 {
		d = s.dValues[len(s.dValues)-1]
	}

	return fastK, slowK, d
}

// Calculate implements TechnicalIndicator interface - calculates Stochastic values from candles
func (s *Stochastic) Calculate(candles []Candle) []float64 {
	if len(candles) < s.config.KPeriod {
		return []float64{}
	}

	values := make([]float64, 0, len(candles))

	// Process each candle
	for _, candle := range candles {
		s.Update(candle)
		if s.initialized {
			_, slowK, _ := s.GetCurrentValues()
			values = append(values, slowK)
		}
	}

	return values
}

// GetSignal implements TechnicalIndicator interface - returns signal from calculated values
func (s *Stochastic) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      s.GetName(),
			Signal:    Hold,
			Strength:  0.0,
			Value:     0.0,
			Timestamp: time.Now(),
			Timeframe: s.timeframe,
		}
	}

	// Use the current signal from the indicator
	signal, strength := s.GetEnhanced5MinuteSignal()
	currentValue := values[len(values)-1]

	return IndicatorSignal{
		Name:      s.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentValue,
		Timestamp: time.Now(),
		Timeframe: s.timeframe,
	}
}

// GetName returns the indicator name
func (s *Stochastic) GetName() string {
	return "Stochastic"
}

// GetLastSignal returns the last signal and strength
func (s *Stochastic) GetLastSignal() (SignalType, float64) {
	return s.lastSignal, s.lastStrength
}

// String returns a string representation
func (s *Stochastic) String() string {
	if !s.initialized {
		return "Stochastic: Not initialized"
	}

	fastK, slowK, d := s.GetCurrentValues()
	return fmt.Sprintf("Stochastic: FastK=%.2f, SlowK=%.2f, D=%.2f, Signal=%s, Strength=%.2f",
		fastK, slowK, d, s.lastSignal, s.lastStrength)
}
