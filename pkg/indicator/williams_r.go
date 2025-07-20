package indicator

import (
	"fmt"
	"math"
	"time"
)

// WilliamsRConfig holds Williams %R configuration
type WilliamsRConfig struct {
	Enabled       bool    `json:"enabled"`        // Feature flag to enable/disable Williams %R
	Period        int     `json:"period"`         // Lookback period (default: 14)
	Overbought    float64 `json:"overbought"`     // Overbought threshold (default: -20)
	Oversold      float64 `json:"oversold"`       // Oversold threshold (default: -80)
	FastResponse  bool    `json:"fast_response"`  // Enable fast response for 5-minute trading
	MomentumBoost float64 `json:"momentum_boost"` // Boost factor for momentum signals (default: 1.3)
	ReversalBoost float64 `json:"reversal_boost"` // Boost for reversal signals (default: 1.4)
}

// WilliamsR represents the Williams %R indicator
type WilliamsR struct {
	config       WilliamsRConfig
	timeframe    Timeframe
	values       []float64 // Williams %R values
	highPrices   []float64 // High prices buffer
	lowPrices    []float64 // Low prices buffer
	closePrices  []float64 // Close prices buffer
	lastSignal   SignalType
	lastStrength float64
	initialized  bool
}

// NewWilliamsR creates a new Williams %R indicator
func NewWilliamsR(config WilliamsRConfig, timeframe Timeframe) *WilliamsR {
	return &WilliamsR{
		config:       config,
		timeframe:    timeframe,
		values:       make([]float64, 0),
		highPrices:   make([]float64, 0),
		lowPrices:    make([]float64, 0),
		closePrices:  make([]float64, 0),
		lastSignal:   Hold,
		lastStrength: 0.0,
		initialized:  false,
	}
}

// Update processes new price data and updates Williams %R values
func (wr *WilliamsR) Update(data Candle) {
	// Add new price data
	wr.highPrices = append(wr.highPrices, data.High)
	wr.lowPrices = append(wr.lowPrices, data.Low)
	wr.closePrices = append(wr.closePrices, data.Close)

	// Maintain buffer size
	maxSize := wr.config.Period + 10
	if len(wr.highPrices) > maxSize {
		wr.highPrices = wr.highPrices[1:]
		wr.lowPrices = wr.lowPrices[1:]
		wr.closePrices = wr.closePrices[1:]
	}

	// Calculate Williams %R if we have enough data
	if len(wr.highPrices) >= wr.config.Period {
		wr.calculateWilliamsR()
		wr.initialized = true
	}
}

// calculateWilliamsR calculates the Williams %R value
func (wr *WilliamsR) calculateWilliamsR() {
	if len(wr.highPrices) < wr.config.Period {
		return
	}

	// Get the last N periods
	start := len(wr.highPrices) - wr.config.Period
	highs := wr.highPrices[start:]
	lows := wr.lowPrices[start:]
	currentClose := wr.closePrices[len(wr.closePrices)-1]

	// Find highest high and lowest low over the period
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

	// Calculate Williams %R
	var williamsR float64
	if highestHigh-lowestLow != 0 {
		williamsR = ((highestHigh - currentClose) / (highestHigh - lowestLow)) * -100
	} else {
		williamsR = -50 // Neutral when no range
	}

	wr.values = append(wr.values, williamsR)

	// Maintain buffer
	if len(wr.values) > wr.config.Period+5 {
		wr.values = wr.values[1:]
	}
}

// GetCurrentSignal returns the current Williams %R signal
func (wr *WilliamsR) GetCurrentSignal() (SignalType, float64) {
	if !wr.initialized || len(wr.values) < 2 {
		return Hold, 0.0
	}

	current := wr.values[len(wr.values)-1]
	previous := wr.values[len(wr.values)-2]

	// Calculate signal strength
	strength := wr.calculateSignalStrength(current, previous)

	// Determine signal
	signal := wr.determineSignal(current, previous, strength)

	wr.lastSignal = signal
	wr.lastStrength = strength

	return signal, strength
}

// calculateSignalStrength calculates the strength of the signal
func (wr *WilliamsR) calculateSignalStrength(current, previous float64) float64 {
	baseStrength := 0.0

	// Position-based strength
	if current <= wr.config.Overbought {
		// Overbought territory
		baseStrength = 0.7 + (math.Abs(current-wr.config.Overbought)/20.0)*0.3
	} else if current >= wr.config.Oversold {
		// Oversold territory
		baseStrength = 0.7 + (math.Abs(current-wr.config.Oversold)/20.0)*0.3
	} else {
		// Neutral zone
		baseStrength = 0.4
	}

	// Momentum strength
	momentum := math.Abs(current - previous)
	momentumStrength := 0.0
	if momentum > 5 {
		momentumStrength = 0.2 * wr.config.MomentumBoost
	}

	// Reversal strength (when crossing thresholds)
	reversalStrength := 0.0
	if (current > wr.config.Overbought && previous <= wr.config.Overbought) ||
		(current < wr.config.Oversold && previous >= wr.config.Oversold) {
		reversalStrength = 0.3 * wr.config.ReversalBoost
	}

	// Fast response boost for 5-minute trading
	fastResponseBoost := 0.0
	if wr.config.FastResponse && wr.timeframe == FiveMinute {
		fastResponseBoost = 0.1
	}

	// Combine strengths
	totalStrength := baseStrength + momentumStrength + reversalStrength + fastResponseBoost

	// FIXED: Cap to prevent exceeding 1.0 from boost combinations
	if totalStrength > 0.85 {
		totalStrength = 0.85
	}

	return totalStrength
}

// determineSignal determines the buy/sell/hold signal
func (wr *WilliamsR) determineSignal(current, previous, strength float64) SignalType {
	// Oversold bounce (Williams %R rising from oversold)
	if current > wr.config.Oversold && previous <= wr.config.Oversold && strength > 0.6 {
		return Buy
	}

	// Overbought decline (Williams %R falling from overbought)
	if current < wr.config.Overbought && previous >= wr.config.Overbought && strength > 0.6 {
		return Sell
	}

	// Strong oversold condition with momentum
	if current <= wr.config.Oversold && current > previous && strength > 0.7 {
		return Buy
	}

	// Strong overbought condition with momentum
	if current >= wr.config.Overbought && current < previous && strength > 0.7 {
		return Sell
	}

	// Fast response signals for 5-minute trading
	if wr.config.FastResponse && wr.timeframe == FiveMinute {
		if current < -70 && current > previous && strength > 0.5 {
			return Buy // Quick oversold bounce
		}
		if current > -30 && current < previous && strength > 0.5 {
			return Sell // Quick overbought decline
		}
	}

	return Hold
}

// GetEnhanced5MinuteSignal returns enhanced signal for 5-minute trading
func (wr *WilliamsR) GetEnhanced5MinuteSignal() (SignalType, float64) {
	if wr.timeframe != FiveMinute {
		return wr.GetCurrentSignal()
	}

	signal, strength := wr.GetCurrentSignal()

	// Enhanced 5-minute specific logic
	if len(wr.values) >= 3 {
		current := wr.values[len(wr.values)-1]
		previous := wr.values[len(wr.values)-2]
		older := wr.values[len(wr.values)-3]

		// Quick momentum detection
		if current > previous && previous > older && current > -70 {
			// Strong upward momentum from oversold
			strength = math.Min(strength*1.2, 0.85) // FIXED: Cap to 0.85
			if signal == Hold && strength > 0.5 {
				signal = Buy
			}
		} else if current < previous && previous < older && current < -30 {
			// Strong downward momentum from overbought
			strength = math.Min(strength*1.2, 0.85) // FIXED: Cap to 0.85
			if signal == Hold && strength > 0.5 {
				signal = Sell
			}
		}

		// Divergence detection for 5-minute
		if len(wr.closePrices) >= 3 {
			priceChange := wr.closePrices[len(wr.closePrices)-1] - wr.closePrices[len(wr.closePrices)-3]
			williamsRChange := current - older

			// Bullish divergence (price down, Williams %R up)
			if priceChange < 0 && williamsRChange > 0 && current < -50 {
				strength = math.Min(strength*1.3, 0.85) // FIXED: Cap to 0.85
				signal = Buy
			}
			// Bearish divergence (price up, Williams %R down)
			if priceChange > 0 && williamsRChange < 0 && current > -50 {
				strength = math.Min(strength*1.3, 0.85) // FIXED: Cap to 0.85
				signal = Sell
			}
		}
	}

	return signal, strength
}

// Calculate implements TechnicalIndicator interface
func (wr *WilliamsR) Calculate(candles []Candle) []float64 {
	if len(candles) < wr.config.Period {
		return []float64{}
	}

	values := make([]float64, 0, len(candles))

	// Process each candle
	for _, candle := range candles {
		wr.Update(candle)
		if wr.initialized {
			current := wr.values[len(wr.values)-1]
			values = append(values, current)
		}
	}

	return values
}

// GetSignal implements TechnicalIndicator interface
func (wr *WilliamsR) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      wr.GetName(),
			Signal:    Hold,
			Strength:  0.0,
			Value:     0.0,
			Timestamp: time.Now(),
			Timeframe: wr.timeframe,
		}
	}

	// Use the enhanced 5-minute signal
	signal, strength := wr.GetEnhanced5MinuteSignal()
	currentValue := values[len(values)-1]

	return IndicatorSignal{
		Name:      wr.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentValue,
		Timestamp: time.Now(),
		Timeframe: wr.timeframe,
	}
}

// GetCurrentValue returns the current Williams %R value
func (wr *WilliamsR) GetCurrentValue() float64 {
	if !wr.initialized || len(wr.values) == 0 {
		return 0.0
	}
	return wr.values[len(wr.values)-1]
}

// GetName returns the indicator name
func (wr *WilliamsR) GetName() string {
	return "Williams %R"
}

// GetLastSignal returns the last signal and strength
func (wr *WilliamsR) GetLastSignal() (SignalType, float64) {
	return wr.lastSignal, wr.lastStrength
}

// String returns a string representation
func (wr *WilliamsR) String() string {
	if !wr.initialized {
		return "Williams %R: Not initialized"
	}

	value := wr.GetCurrentValue()
	return fmt.Sprintf("Williams %%R: %.2f%%, Signal=%s, Strength=%.2f",
		value, wr.lastSignal, wr.lastStrength)
}
