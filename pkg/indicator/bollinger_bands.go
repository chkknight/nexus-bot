package indicator

import (
	"math"
	"time"
)

// BollingerBands represents a Bollinger Bands indicator
type BollingerBands struct {
	config    BollingerBandsConfig
	timeframe Timeframe
}

// BollingerBandsValues holds all calculated values
type BollingerBandsValues struct {
	UpperBand  []float64 // Upper Bollinger Band
	LowerBand  []float64 // Lower Bollinger Band
	MiddleBand []float64 // Middle Band (SMA)
	Position   []float64 // Price position within bands (0-1)
	Bandwidth  []float64 // Band width (volatility measure)
	Squeeze    []bool    // Bollinger Band squeeze detection
}

// NewBollingerBands creates a new Bollinger Bands indicator
func NewBollingerBands(config BollingerBandsConfig, timeframe Timeframe) *BollingerBands {
	return &BollingerBands{
		config:    config,
		timeframe: timeframe,
	}
}

// GetName returns the indicator name
func (bb *BollingerBands) GetName() string {
	return "BollingerBands_" + bb.timeframe.String()
}

// Calculate computes Bollinger Bands values
func (bb *BollingerBands) Calculate(candles []Candle) []float64 {
	if len(candles) < bb.config.Period {
		return []float64{}
	}

	values := bb.CalculateAll(candles)

	// Return position within bands as the main signal
	return values.Position
}

// CalculateAll computes all Bollinger Bands values
func (bb *BollingerBands) CalculateAll(candles []Candle) BollingerBandsValues {
	length := len(candles)
	if length < bb.config.Period {
		return BollingerBandsValues{}
	}

	upperBand := make([]float64, length)
	lowerBand := make([]float64, length)
	middleBand := make([]float64, length)
	position := make([]float64, length)
	bandwidth := make([]float64, length)
	squeeze := make([]bool, length)

	// Calculate for each position
	for i := bb.config.Period - 1; i < length; i++ {
		// Calculate SMA (Middle Band)
		sum := 0.0
		for j := i - bb.config.Period + 1; j <= i; j++ {
			sum += candles[j].Close
		}
		sma := sum / float64(bb.config.Period)
		middleBand[i] = sma

		// Calculate Standard Deviation
		sumSquares := 0.0
		for j := i - bb.config.Period + 1; j <= i; j++ {
			diff := candles[j].Close - sma
			sumSquares += diff * diff
		}
		stdDev := math.Sqrt(sumSquares / float64(bb.config.Period))

		// Calculate Upper and Lower Bands
		upperBand[i] = sma + (bb.config.StandardDev * stdDev)
		lowerBand[i] = sma - (bb.config.StandardDev * stdDev)

		// Calculate price position within bands (0 = lower band, 1 = upper band)
		bandRange := upperBand[i] - lowerBand[i]
		if bandRange > 0 {
			position[i] = (candles[i].Close - lowerBand[i]) / bandRange
		} else {
			position[i] = 0.5 // Default to middle if no range
		}

		// Calculate bandwidth (volatility measure)
		bandwidth[i] = (upperBand[i] - lowerBand[i]) / middleBand[i]

		// Detect squeeze (low volatility)
		if i > 0 {
			avgBandwidth := (bandwidth[i] + bandwidth[i-1]) / 2
			squeeze[i] = avgBandwidth < 0.1 // Squeeze when bandwidth < 10%
		}
	}

	return BollingerBandsValues{
		UpperBand:  upperBand,
		LowerBand:  lowerBand,
		MiddleBand: middleBand,
		Position:   position,
		Bandwidth:  bandwidth,
		Squeeze:    squeeze,
	}
}

// GetSignal generates trading signals based on Bollinger Bands
func (bb *BollingerBands) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      bb.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: bb.timeframe,
		}
	}

	// Get current position within bands
	currentPosition := values[len(values)-1]

	// Calculate strength and signal based on position
	var signal SignalType
	var strength float64

	// Enhanced 5-minute specific logic
	if bb.timeframe == FiveMinute {
		strength = bb.calculate5MinuteStrength(currentPosition, values)
		signal = bb.calculate5MinuteSignal(currentPosition, values)
	} else {
		// Standard logic for other timeframes
		if currentPosition <= bb.config.OversoldStd {
			signal = Buy
			strength = (bb.config.OversoldStd - currentPosition) / bb.config.OversoldStd
		} else if currentPosition >= bb.config.OverboughtStd {
			signal = Sell
			strength = (currentPosition - bb.config.OverboughtStd) / (1.0 - bb.config.OverboughtStd)
		} else {
			signal = Hold
			strength = 0.3 // Default hold strength
		}
	}

	// Normalize strength to 0-1 range
	if strength > 1.0 {
		strength = 1.0
	} else if strength < 0.1 {
		strength = 0.1
	}

	return IndicatorSignal{
		Name:      bb.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentPosition,
		Timestamp: time.Now(),
		Timeframe: bb.timeframe,
	}
}

// calculate5MinuteStrength provides enhanced strength calculation for 5-minute trading
func (bb *BollingerBands) calculate5MinuteStrength(position float64, values []float64) float64 {
	if len(values) < 3 {
		return 0.3
	}

	// Base strength from position
	var baseStrength float64
	if position <= 0.1 {
		// Very oversold - strong buy signal
		baseStrength = 0.9
	} else if position <= 0.2 {
		// Oversold - moderate buy signal
		baseStrength = 0.7
	} else if position >= 0.9 {
		// Very overbought - strong sell signal
		baseStrength = 0.9
	} else if position >= 0.8 {
		// Overbought - moderate sell signal
		baseStrength = 0.7
	} else if position >= 0.4 && position <= 0.6 {
		// Middle range - neutral
		baseStrength = 0.3
	} else {
		// Transitional zones
		baseStrength = 0.5
	}

	// Add trend component (momentum)
	if len(values) >= 3 {
		recent := values[len(values)-1]
		previous := values[len(values)-3]

		// Trend strength boost
		trendStrength := math.Abs(recent-previous) * 2
		if trendStrength > 0.3 {
			baseStrength += 0.1 // Boost for strong trends
		}
	}

	// Add volatility component (band width consideration)
	// For 5-minute, we want to react quickly to volatility changes
	volatilityBoost := 0.0
	if len(values) >= 5 {
		recentVol := calculateVolatility(values[len(values)-5:])
		if recentVol > 0.15 { // High volatility threshold
			volatilityBoost = 0.15 // Boost signals in volatile conditions
		}
	}

	finalStrength := baseStrength + volatilityBoost
	if finalStrength > 1.0 {
		finalStrength = 1.0
	}

	return finalStrength
}

// calculate5MinuteSignal provides enhanced signal calculation for 5-minute trading
func (bb *BollingerBands) calculate5MinuteSignal(position float64, values []float64) SignalType {
	// More sensitive thresholds for 5-minute trading
	oversoldThreshold := 0.15   // More sensitive than default 0.2
	overboughtThreshold := 0.85 // More sensitive than default 0.8

	if position <= oversoldThreshold {
		return Buy
	} else if position >= overboughtThreshold {
		return Sell
	}

	// Add momentum consideration for 5-minute signals
	if len(values) >= 3 {
		recent := values[len(values)-1]
		previous := values[len(values)-3]

		// Trend-following signals in middle zone
		if recent < previous-0.1 && position < 0.4 {
			return Buy // Bouncing off lower area
		} else if recent > previous+0.1 && position > 0.6 {
			return Sell // Rejection from upper area
		}
	}

	return Hold
}

// calculateVolatility calculates volatility from position values
func calculateVolatility(positions []float64) float64 {
	if len(positions) < 2 {
		return 0
	}

	sum := 0.0
	for i := 1; i < len(positions); i++ {
		change := math.Abs(positions[i] - positions[i-1])
		sum += change
	}

	return sum / float64(len(positions)-1)
}

// GetEnhanced5MinuteSignal provides the most optimized signal for 5-minute trading
func (bb *BollingerBands) GetEnhanced5MinuteSignal(candles []Candle, currentPrice float64) IndicatorSignal {
	if len(candles) < bb.config.Period {
		return IndicatorSignal{
			Name:      bb.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: bb.timeframe,
		}
	}

	values := bb.CalculateAll(candles)
	if len(values.Position) == 0 {
		return IndicatorSignal{
			Name:      bb.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: bb.timeframe,
		}
	}

	// Get current position and calculate enhanced metrics
	currentPosition := values.Position[len(values.Position)-1]
	currentBandwidth := values.Bandwidth[len(values.Bandwidth)-1]
	isSqueezing := values.Squeeze[len(values.Squeeze)-1]

	// Enhanced 5-minute analysis
	var signal SignalType
	var strength float64

	// Base signal from position
	if currentPosition <= 0.1 {
		// Very oversold - strong buy opportunity
		signal = Buy
		strength = 0.9
	} else if currentPosition <= 0.2 {
		// Oversold - buy opportunity
		signal = Buy
		strength = 0.7
	} else if currentPosition >= 0.9 {
		// Very overbought - strong sell opportunity
		signal = Sell
		strength = 0.9
	} else if currentPosition >= 0.8 {
		// Overbought - sell opportunity
		signal = Sell
		strength = 0.7
	} else {
		// Middle range - check for other signals
		signal = Hold
		strength = 0.3
	}

	// Squeeze breakout detection (high priority for 5-minute)
	if isSqueezing && len(values.Position) >= 3 {
		recent := values.Position[len(values.Position)-1]
		previous := values.Position[len(values.Position)-3]

		if recent > previous+0.1 && recent > 0.5 {
			signal = Buy
			strength = 0.8 // Strong breakout signal
		} else if recent < previous-0.1 && recent < 0.5 {
			signal = Sell
			strength = 0.8 // Strong breakdown signal
		}
	}

	// Volatility adjustment for 5-minute
	if currentBandwidth > 0.05 { // High volatility
		strength += 0.1 // Boost signals in volatile conditions
	}

	// Mean reversion in extreme zones (5-minute specific)
	if currentPosition <= 0.05 {
		signal = Buy
		strength = 1.0 // Maximum strength for extreme oversold
	} else if currentPosition >= 0.95 {
		signal = Sell
		strength = 1.0 // Maximum strength for extreme overbought
	}

	// Normalize strength
	if strength > 1.0 {
		strength = 1.0
	} else if strength < 0.1 {
		strength = 0.1
	}

	return IndicatorSignal{
		Name:      bb.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentPosition,
		Timestamp: time.Now(),
		Timeframe: bb.timeframe,
	}
}
