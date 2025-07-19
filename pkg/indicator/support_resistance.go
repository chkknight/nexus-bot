package indicator

import (
	"fmt"
	"math"
	"time"
)

// SupportResistance Indicator
type SupportResistance struct {
	config    SupportResistanceConfig
	timeframe Timeframe
}

// NewSupportResistance creates a new Support and Resistance indicator
func NewSupportResistance(config SupportResistanceConfig, timeframe Timeframe) *SupportResistance {
	return &SupportResistance{
		config:    config,
		timeframe: timeframe,
	}
}

// Calculate computes support and resistance levels for given candles
func (sr *SupportResistance) Calculate(candles []Candle) []float64 {
	if len(candles) < sr.config.Period {
		return []float64{}
	}

	// Find pivot points (local highs and lows)
	pivots := sr.findPivotPoints(candles)

	// Calculate support and resistance levels
	levels := make([]float64, len(candles))

	for i := 0; i < len(candles); i++ {
		// Find the most relevant support/resistance level for current price
		currentPrice := candles[i].Close
		closestLevel := sr.findClosestLevel(pivots, currentPrice, i)
		levels[i] = closestLevel
	}

	return levels
}

// findPivotPoints identifies local highs and lows
func (sr *SupportResistance) findPivotPoints(candles []Candle) []PivotPoint {
	if len(candles) < 5 {
		return []PivotPoint{}
	}

	var pivots []PivotPoint
	lookback := 2 // Look 2 candles back and forward

	for i := lookback; i < len(candles)-lookback; i++ {
		current := candles[i]

		// Check for local high
		isHigh := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].High >= current.High {
				isHigh = false
				break
			}
		}

		if isHigh {
			pivots = append(pivots, PivotPoint{
				Price:     current.High,
				Timestamp: current.Timestamp,
				Type:      "resistance",
				Index:     i,
			})
		}

		// Check for local low
		isLow := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].Low <= current.Low {
				isLow = false
				break
			}
		}

		if isLow {
			pivots = append(pivots, PivotPoint{
				Price:     current.Low,
				Timestamp: current.Timestamp,
				Type:      "support",
				Index:     i,
			})
		}
	}

	return pivots
}

// findClosestLevel finds the most relevant support/resistance level
func (sr *SupportResistance) findClosestLevel(pivots []PivotPoint, currentPrice float64, currentIndex int) float64 {
	if len(pivots) == 0 {
		return currentPrice
	}

	var closestLevel float64
	minDistance := math.Inf(1)

	for _, pivot := range pivots {
		// Only consider recent pivots
		if currentIndex-pivot.Index > sr.config.Period {
			continue
		}

		distance := math.Abs(pivot.Price - currentPrice)
		if distance < minDistance {
			minDistance = distance
			closestLevel = pivot.Price
		}
	}

	if closestLevel == 0 {
		return currentPrice
	}

	return closestLevel
}

// GetSignal generates a trading signal based on support/resistance analysis
func (sr *SupportResistance) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      sr.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: sr.timeframe,
		}
	}

	currentLevel := values[len(values)-1]
	var signal SignalType
	var strength float64

	// Calculate distance from support/resistance level
	distance := math.Abs(currentPrice - currentLevel)
	relativeDistance := distance / currentPrice

	// If price is very close to support/resistance level
	if relativeDistance < sr.config.Threshold {
		// Determine if it's support or resistance based on price position
		if currentPrice < currentLevel {
			// Price below resistance, potential rejection/bounce down
			signal = Sell
			strength = math.Min(1.0, (sr.config.Threshold-relativeDistance)/sr.config.Threshold)
		} else {
			// Price above support, potential bounce up
			signal = Buy
			strength = math.Min(1.0, (sr.config.Threshold-relativeDistance)/sr.config.Threshold)
		}
	} else {
		signal = Hold
		strength = 0.3 // Low strength when away from levels
	}

	return IndicatorSignal{
		Name:      sr.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentLevel,
		Timestamp: time.Now(),
		Timeframe: sr.timeframe,
	}
}

// GetName returns the indicator name
func (sr *SupportResistance) GetName() string {
	return fmt.Sprintf("S&R_%s", sr.timeframe.String())
}
