package indicator

import (
	"fmt"
	"math"
	"time"
)

// PinBarConfig holds Pin Bar pattern detection configuration
type PinBarConfig struct {
	Enabled              bool    `json:"enabled"`                // Feature flag to enable/disable Pin Bar
	MinWickRatio         float64 `json:"min_wick_ratio"`         // Minimum wick to body ratio (default: 2.0)
	MaxBodyRatio         float64 `json:"max_body_ratio"`         // Maximum body to total range ratio (default: 0.33)
	MinRangePercent      float64 `json:"min_range_percent"`      // Minimum range as % of price (default: 0.001)
	SupportResistance    bool    `json:"support_resistance"`     // Consider S/R levels for strength
	TrendConfirmation    bool    `json:"trend_confirmation"`     // Require trend confirmation
	PatternStrengthBoost float64 `json:"pattern_strength_boost"` // Boost for strong patterns (default: 1.2)
}

// PinBarPattern represents different pin bar pattern types
type PinBarPattern int

const (
	NoPinBar PinBarPattern = iota
	BullishPinBar
	BearishPinBar
	Doji
	Hammer
	InvertedHammer
	ShootingStar
	HangingMan
	Engulfing
)

func (p PinBarPattern) String() string {
	switch p {
	case BullishPinBar:
		return "Bullish Pin Bar"
	case BearishPinBar:
		return "Bearish Pin Bar"
	case Doji:
		return "Doji"
	case Hammer:
		return "Hammer"
	case InvertedHammer:
		return "Inverted Hammer"
	case ShootingStar:
		return "Shooting Star"
	case HangingMan:
		return "Hanging Man"
	case Engulfing:
		return "Engulfing"
	default:
		return "No Pattern"
	}
}

// PinBar represents the Pin Bar pattern detector
type PinBar struct {
	config           PinBarConfig
	timeframe        Timeframe
	candles          []Candle
	patterns         []PinBarPattern
	patternStrengths []float64
	lastSignal       SignalType
	lastStrength     float64
	lastPattern      PinBarPattern
	initialized      bool
}

// NewPinBar creates a new Pin Bar pattern detector
func NewPinBar(config PinBarConfig, timeframe Timeframe) *PinBar {
	return &PinBar{
		config:           config,
		timeframe:        timeframe,
		candles:          make([]Candle, 0),
		patterns:         make([]PinBarPattern, 0),
		patternStrengths: make([]float64, 0),
		lastSignal:       Hold,
		lastStrength:     0.0,
		lastPattern:      NoPinBar,
		initialized:      false,
	}
}

// Update processes new candle data
func (pb *PinBar) Update(candle Candle) {
	pb.candles = append(pb.candles, candle)

	// Maintain buffer size
	maxSize := 50
	if len(pb.candles) > maxSize {
		pb.candles = pb.candles[1:]
	}

	// Detect pattern if we have enough data
	if len(pb.candles) >= 3 {
		pattern, strength := pb.detectPattern()
		pb.patterns = append(pb.patterns, pattern)
		pb.patternStrengths = append(pb.patternStrengths, strength)

		// Maintain pattern buffers
		if len(pb.patterns) > 20 {
			pb.patterns = pb.patterns[1:]
			pb.patternStrengths = pb.patternStrengths[1:]
		}

		pb.lastPattern = pattern
		pb.initialized = true
	}
}

// detectPattern detects pin bar and similar candlestick patterns
func (pb *PinBar) detectPattern() (PinBarPattern, float64) {
	if len(pb.candles) < 3 {
		return NoPinBar, 0.0
	}

	current := pb.candles[len(pb.candles)-1]
	previous := pb.candles[len(pb.candles)-2]

	// Calculate basic measurements
	body := math.Abs(current.Close - current.Open)
	totalRange := current.High - current.Low
	upperWick := current.High - math.Max(current.Open, current.Close)
	lowerWick := math.Min(current.Open, current.Close) - current.Low

	// Skip if range is too small
	if totalRange < current.Close*pb.config.MinRangePercent {
		return NoPinBar, 0.0
	}

	// Calculate ratios
	bodyRatio := body / totalRange
	upperWickRatio := upperWick / totalRange
	lowerWickRatio := lowerWick / totalRange

	// Detect Pin Bar patterns
	if pattern, strength := pb.detectPinBarPattern(current, bodyRatio, upperWickRatio, lowerWickRatio); pattern != NoPinBar {
		return pattern, strength
	}

	// Detect Doji pattern
	if pattern, strength := pb.detectDojiPattern(current, bodyRatio, upperWickRatio, lowerWickRatio); pattern != NoPinBar {
		return pattern, strength
	}

	// Detect Hammer/Hanging Man patterns
	if pattern, strength := pb.detectHammerPattern(current, previous, bodyRatio, upperWickRatio, lowerWickRatio); pattern != NoPinBar {
		return pattern, strength
	}

	// Detect Shooting Star/Inverted Hammer patterns
	if pattern, strength := pb.detectShootingStarPattern(current, previous, bodyRatio, upperWickRatio, lowerWickRatio); pattern != NoPinBar {
		return pattern, strength
	}

	// Detect Engulfing patterns
	if pattern, strength := pb.detectEngulfingPattern(current, previous); pattern != NoPinBar {
		return pattern, strength
	}

	return NoPinBar, 0.0
}

// detectPinBarPattern detects classic pin bar patterns
func (pb *PinBar) detectPinBarPattern(current Candle, bodyRatio, upperWickRatio, lowerWickRatio float64) (PinBarPattern, float64) {
	// Pin bar criteria:
	// 1. Small body (body ratio < max_body_ratio)
	// 2. Long wick on one side (wick ratio > min_wick_ratio)
	// 3. Short wick on other side

	if bodyRatio > pb.config.MaxBodyRatio {
		return NoPinBar, 0.0
	}

	// Bullish pin bar (long lower wick)
	if lowerWickRatio > pb.config.MinWickRatio && upperWickRatio < 0.1 {
		strength := pb.calculatePatternStrength(lowerWickRatio, bodyRatio, true)
		return BullishPinBar, strength
	}

	// Bearish pin bar (long upper wick)
	if upperWickRatio > pb.config.MinWickRatio && lowerWickRatio < 0.1 {
		strength := pb.calculatePatternStrength(upperWickRatio, bodyRatio, false)
		return BearishPinBar, strength
	}

	return NoPinBar, 0.0
}

// detectDojiPattern detects doji patterns
func (pb *PinBar) detectDojiPattern(current Candle, bodyRatio, upperWickRatio, lowerWickRatio float64) (PinBarPattern, float64) {
	// Doji: very small body, wicks on both sides
	if bodyRatio < 0.05 && upperWickRatio > 0.2 && lowerWickRatio > 0.2 {
		strength := pb.calculatePatternStrength(math.Max(upperWickRatio, lowerWickRatio), bodyRatio, true)
		return Doji, strength * 0.8 // Doji is more neutral
	}

	return NoPinBar, 0.0
}

// detectHammerPattern detects hammer and hanging man patterns
func (pb *PinBar) detectHammerPattern(current, previous Candle, bodyRatio, upperWickRatio, lowerWickRatio float64) (PinBarPattern, float64) {
	// Hammer/Hanging Man: small body, long lower wick, short upper wick
	if bodyRatio < 0.4 && lowerWickRatio > 0.6 && upperWickRatio < 0.2 {
		strength := pb.calculatePatternStrength(lowerWickRatio, bodyRatio, true)

		// Determine if it's hammer or hanging man based on trend
		if pb.isInDowntrend(len(pb.candles) - 1) {
			return Hammer, strength // Bullish reversal
		} else {
			return HangingMan, strength * 0.9 // Bearish reversal (less reliable)
		}
	}

	return NoPinBar, 0.0
}

// detectShootingStarPattern detects shooting star and inverted hammer patterns
func (pb *PinBar) detectShootingStarPattern(current, previous Candle, bodyRatio, upperWickRatio, lowerWickRatio float64) (PinBarPattern, float64) {
	// Shooting Star/Inverted Hammer: small body, long upper wick, short lower wick
	if bodyRatio < 0.4 && upperWickRatio > 0.6 && lowerWickRatio < 0.2 {
		strength := pb.calculatePatternStrength(upperWickRatio, bodyRatio, false)

		// Determine if it's shooting star or inverted hammer based on trend
		if pb.isInUptrend(len(pb.candles) - 1) {
			return ShootingStar, strength // Bearish reversal
		} else {
			return InvertedHammer, strength * 0.9 // Bullish reversal (less reliable)
		}
	}

	return NoPinBar, 0.0
}

// detectEngulfingPattern detects engulfing patterns
func (pb *PinBar) detectEngulfingPattern(current, previous Candle) (PinBarPattern, float64) {
	// Engulfing: current candle body completely engulfs previous candle body
	currentBody := math.Abs(current.Close - current.Open)
	previousBody := math.Abs(previous.Close - previous.Open)

	if currentBody > previousBody*1.2 {
		// Bullish engulfing
		if current.Close > current.Open && previous.Close < previous.Open &&
			current.Open < previous.Close && current.Close > previous.Open {
			strength := pb.calculatePatternStrength(currentBody/previousBody, 0.1, true)
			return Engulfing, strength
		}

		// Bearish engulfing
		if current.Close < current.Open && previous.Close > previous.Open &&
			current.Open > previous.Close && current.Close < previous.Open {
			strength := pb.calculatePatternStrength(currentBody/previousBody, 0.1, false)
			return Engulfing, strength
		}
	}

	return NoPinBar, 0.0
}

// calculatePatternStrength calculates the strength of a detected pattern
func (pb *PinBar) calculatePatternStrength(wickRatio, bodyRatio float64, bullish bool) float64 {
	// Base strength from wick dominance
	baseStrength := math.Min(wickRatio*0.8, 0.8)

	// Body size penalty (smaller body = stronger pattern)
	bodyPenalty := bodyRatio * 0.5

	// Pattern strength boost
	patternBoost := 0.0
	if wickRatio > 0.8 {
		patternBoost = 0.2 * pb.config.PatternStrengthBoost
	}

	// Trend confirmation boost
	trendBoost := 0.0
	if pb.config.TrendConfirmation {
		if bullish && pb.isInDowntrend(len(pb.candles)-1) {
			trendBoost = 0.15 // Bullish reversal in downtrend
		} else if !bullish && pb.isInUptrend(len(pb.candles)-1) {
			trendBoost = 0.15 // Bearish reversal in uptrend
		}
	}

	strength := baseStrength - bodyPenalty + patternBoost + trendBoost
	return math.Max(0.0, math.Min(1.0, strength))
}

// isInUptrend checks if the market is in an uptrend
func (pb *PinBar) isInUptrend(index int) bool {
	if index < 5 {
		return false
	}

	// Simple trend detection: compare recent prices with older prices
	recentAvg := 0.0
	oldAvg := 0.0

	for i := 0; i < 3; i++ {
		recentAvg += pb.candles[index-i].Close
		oldAvg += pb.candles[index-i-3].Close
	}

	return recentAvg/3 > oldAvg/3
}

// isInDowntrend checks if the market is in a downtrend
func (pb *PinBar) isInDowntrend(index int) bool {
	if index < 5 {
		return false
	}

	// Simple trend detection: compare recent prices with older prices
	recentAvg := 0.0
	oldAvg := 0.0

	for i := 0; i < 3; i++ {
		recentAvg += pb.candles[index-i].Close
		oldAvg += pb.candles[index-i-3].Close
	}

	return recentAvg/3 < oldAvg/3
}

// GetCurrentSignal returns the current signal based on detected patterns
func (pb *PinBar) GetCurrentSignal() (SignalType, float64) {
	if !pb.initialized || len(pb.patterns) == 0 {
		return Hold, 0.0
	}

	pattern := pb.patterns[len(pb.patterns)-1]
	strength := pb.patternStrengths[len(pb.patternStrengths)-1]

	signal := pb.patternToSignal(pattern)

	pb.lastSignal = signal
	pb.lastStrength = strength

	return signal, strength
}

// patternToSignal converts a pattern to a trading signal
func (pb *PinBar) patternToSignal(pattern PinBarPattern) SignalType {
	switch pattern {
	case BullishPinBar, Hammer, InvertedHammer:
		return Buy
	case BearishPinBar, ShootingStar, HangingMan:
		return Sell
	case Engulfing:
		// Engulfing direction depends on the candle color
		if len(pb.candles) > 0 {
			current := pb.candles[len(pb.candles)-1]
			if current.Close > current.Open {
				return Buy
			} else {
				return Sell
			}
		}
		return Hold
	case Doji:
		return Hold // Doji is neutral
	default:
		return Hold
	}
}

// Calculate implements TechnicalIndicator interface
func (pb *PinBar) Calculate(candles []Candle) []float64 {
	if len(candles) < 3 {
		return []float64{}
	}

	values := make([]float64, 0, len(candles))

	// Process each candle
	for _, candle := range candles {
		pb.Update(candle)
		if pb.initialized {
			// Return pattern strength as value
			if len(pb.patternStrengths) > 0 {
				values = append(values, pb.patternStrengths[len(pb.patternStrengths)-1])
			}
		}
	}

	return values
}

// GetSignal implements TechnicalIndicator interface
func (pb *PinBar) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	signal, strength := pb.GetCurrentSignal()

	var value float64
	if len(values) > 0 {
		value = values[len(values)-1]
	}

	return IndicatorSignal{
		Name:      pb.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     value,
		Timestamp: time.Now(),
		Timeframe: pb.timeframe,
	}
}

// GetName returns the indicator name
func (pb *PinBar) GetName() string {
	return "PinBar"
}

// GetLastSignal returns the last signal and strength
func (pb *PinBar) GetLastSignal() (SignalType, float64) {
	return pb.lastSignal, pb.lastStrength
}

// GetLastPattern returns the last detected pattern
func (pb *PinBar) GetLastPattern() PinBarPattern {
	return pb.lastPattern
}

// String returns a string representation
func (pb *PinBar) String() string {
	if !pb.initialized {
		return "PinBar: Not initialized"
	}

	return fmt.Sprintf("PinBar: Pattern=%s, Signal=%s, Strength=%.2f",
		pb.lastPattern.String(), pb.lastSignal, pb.lastStrength)
}
