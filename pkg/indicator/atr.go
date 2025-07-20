package indicator

import (
	"fmt"
	"math"
	"time"
)

// ATR represents Average True Range indicator with Pine Script trailing stops strategy
type ATR struct {
	config        ATRConfig
	timeframe     Timeframe
	atrValues     []float64 // ATR values
	trueRanges    []float64 // True Range values
	trailingStops []float64 // ATR Trailing Stop values (xATRTrailingStop in Pine Script)
	positions     []int     // Position tracking: 1=long, -1=short, 0=neutral
	candles       []Candle  // Price data
	initialized   bool
	lastSignal    SignalType
	lastStrength  float64
	prevClose     float64 // Previous close for Pine Script logic
	prevTrailStop float64 // Previous trailing stop for Pine Script logic
}

// ATRConfig holds ATR configuration
type ATRConfig struct {
	Enabled    bool    `json:"enabled"`    // Feature flag to enable/disable ATR
	Period     int     `json:"period"`     // ATR calculation period (Pine Script: nATRPeriod)
	Multiplier float64 `json:"multiplier"` // ATR multiplier for trailing stop (Pine Script: nATRMultip)
	UseShorts  bool    `json:"use_shorts"` // Allow short signals (Pine Script: useShorts)
}

// NewATR creates a new ATR indicator with Pine Script strategy
func NewATR(config ATRConfig, timeframe Timeframe) *ATR {
	return &ATR{
		config:        config,
		timeframe:     timeframe,
		atrValues:     make([]float64, 0),
		trueRanges:    make([]float64, 0),
		trailingStops: make([]float64, 0),
		positions:     make([]int, 0),
		candles:       make([]Candle, 0),
		initialized:   false,
		lastSignal:    Hold,
		lastStrength:  0.0,
		prevClose:     0.0,
		prevTrailStop: 0.0,
	}
}

// GetName returns the indicator name
func (atr *ATR) GetName() string {
	return fmt.Sprintf("ATR_%s", atr.timeframe.String())
}

// Update processes new candle data using Pine Script ATR Trailing Stops logic
func (atr *ATR) Update(candle Candle) {
	atr.candles = append(atr.candles, candle)

	// Maintain buffer size
	maxSize := atr.config.Period * 3
	if len(atr.candles) > maxSize {
		atr.candles = atr.candles[1:]
	}

	// Calculate True Range
	if len(atr.candles) >= 2 {
		atr.calculateTrueRange()

		// Calculate ATR when we have enough true range values
		if len(atr.trueRanges) >= atr.config.Period {
			atr.calculateATR()
			atr.calculatePineScriptTrailingStop(candle.Close)
			atr.calculatePineScriptPosition(candle.Close)
			atr.initialized = true
		}
	}
}

// calculateTrueRange calculates the True Range for the latest candle
func (atr *ATR) calculateTrueRange() {
	if len(atr.candles) < 2 {
		return
	}

	current := atr.candles[len(atr.candles)-1]
	previous := atr.candles[len(atr.candles)-2]

	// True Range = max(H-L, |H-PC|, |L-PC|)
	tr1 := current.High - current.Low
	tr2 := math.Abs(current.High - previous.Close)
	tr3 := math.Abs(current.Low - previous.Close)

	trueRange := math.Max(tr1, math.Max(tr2, tr3))
	atr.trueRanges = append(atr.trueRanges, trueRange)

	// Maintain buffer
	if len(atr.trueRanges) > atr.config.Period+5 {
		atr.trueRanges = atr.trueRanges[1:]
	}
}

// calculateATR calculates the ATR value (Simple Moving Average of True Range)
func (atr *ATR) calculateATR() {
	if len(atr.trueRanges) < atr.config.Period {
		return
	}

	// Calculate SMA of True Range for the period
	start := len(atr.trueRanges) - atr.config.Period
	sum := 0.0
	for i := start; i < len(atr.trueRanges); i++ {
		sum += atr.trueRanges[i]
	}

	atrValue := sum / float64(atr.config.Period)
	atr.atrValues = append(atr.atrValues, atrValue)

	// Maintain buffer
	if len(atr.atrValues) > atr.config.Period+5 {
		atr.atrValues = atr.atrValues[1:]
	}
}

// calculatePineScriptTrailingStop implements the exact Pine Script ATR trailing stop logic
func (atr *ATR) calculatePineScriptTrailingStop(close float64) {
	if len(atr.atrValues) == 0 {
		return
	}

	// Pine Script: xATR = atr(nATRPeriod)
	xATR := atr.atrValues[len(atr.atrValues)-1]

	// Pine Script: nLoss = nATRMultip * xATR
	nLoss := atr.config.Multiplier * xATR

	var xATRTrailingStop float64

	// Pine Script ATR Trailing Stop Logic:
	// xATRTrailingStop :=
	//  iff(close > nz(xATRTrailingStop[1], 0) and close[1] > nz(xATRTrailingStop[1], 0), max(nz(xATRTrailingStop[1]), close - nLoss),
	//   iff(close < nz(xATRTrailingStop[1], 0) and close[1] < nz(xATRTrailingStop[1], 0), min(nz(xATRTrailingStop[1]), close + nLoss),
	//    iff(close > nz(xATRTrailingStop[1], 0), close - nLoss, close + nLoss)))

	if len(atr.trailingStops) == 0 {
		// First calculation - initialize
		if close > 0 {
			xATRTrailingStop = close - nLoss
		} else {
			xATRTrailingStop = close + nLoss
		}
	} else {
		// Implement Pine Script logic exactly
		prevTrailStop := atr.prevTrailStop
		prevClose := atr.prevClose

		if close > prevTrailStop && prevClose > prevTrailStop {
			// Both current and previous close above trailing stop -> uptrend continuation
			xATRTrailingStop = math.Max(prevTrailStop, close-nLoss)
		} else if close < prevTrailStop && prevClose < prevTrailStop {
			// Both current and previous close below trailing stop -> downtrend continuation
			xATRTrailingStop = math.Min(prevTrailStop, close+nLoss)
		} else if close > prevTrailStop {
			// Price crossed above trailing stop -> new uptrend
			xATRTrailingStop = close - nLoss
		} else {
			// Price crossed below trailing stop -> new downtrend
			xATRTrailingStop = close + nLoss
		}
	}

	atr.trailingStops = append(atr.trailingStops, xATRTrailingStop)

	// Update previous values for next calculation
	atr.prevClose = close
	atr.prevTrailStop = xATRTrailingStop

	// Maintain buffer
	if len(atr.trailingStops) > atr.config.Period+5 {
		atr.trailingStops = atr.trailingStops[1:]
	}
}

// calculatePineScriptPosition implements Pine Script position tracking logic
func (atr *ATR) calculatePineScriptPosition(close float64) {
	if len(atr.trailingStops) < 2 {
		atr.positions = append(atr.positions, 0)
		return
	}

	// Pine Script position logic:
	// pos :=
	//  iff(close[1] < nz(xATRTrailingStop[1], 0) and close > nz(xATRTrailingStop[1], 0), 1,
	//   iff(close[1] > nz(xATRTrailingStop[1], 0) and close < nz(xATRTrailingStop[1], 0), -1, nz(pos[1], 0)))

	var position int
	prevClose := atr.prevClose
	prevTrailStop := len(atr.trailingStops) >= 2 && atr.trailingStops[len(atr.trailingStops)-2] != 0
	currentTrailStop := atr.trailingStops[len(atr.trailingStops)-1]
	var prevTrailStopValue float64
	if len(atr.trailingStops) >= 2 {
		prevTrailStopValue = atr.trailingStops[len(atr.trailingStops)-2]
	}

	if prevTrailStop && prevClose < prevTrailStopValue && close > currentTrailStop {
		// Crossover from below -> Long position
		position = 1
	} else if prevTrailStop && prevClose > prevTrailStopValue && close < currentTrailStop {
		// Crossunder from above -> Short position (if enabled)
		if atr.config.UseShorts {
			position = -1
		} else {
			position = 0 // No shorts allowed
		}
	} else {
		// Maintain previous position
		if len(atr.positions) > 0 {
			position = atr.positions[len(atr.positions)-1]
		} else {
			position = 0
		}
	}

	atr.positions = append(atr.positions, position)

	// Maintain buffer
	if len(atr.positions) > atr.config.Period+5 {
		atr.positions = atr.positions[1:]
	}
}

// GetCurrentSignal returns the current signal based on Pine Script logic
func (atr *ATR) GetCurrentSignal() (SignalType, float64) {
	if !atr.initialized || len(atr.positions) < 2 || len(atr.candles) < 2 {
		return Hold, 0.0
	}

	currentClose := atr.candles[len(atr.candles)-1].Close
	currentTrailStop := atr.trailingStops[len(atr.trailingStops)-1]
	currentPosition := atr.positions[len(atr.positions)-1]
	prevPosition := atr.positions[len(atr.positions)-2]

	var signal SignalType
	var strength float64

	// Pine Script signals:
	// buy = crossover(close, xATRTrailingStop)
	// sell = crossunder(close, xATRTrailingStop)

	// Detect position changes (crossovers)
	if prevPosition <= 0 && currentPosition == 1 {
		// Crossover -> BUY signal
		signal = Buy
		// Calculate strength based on distance from trailing stop
		if currentTrailStop > 0 {
			distance := (currentClose - currentTrailStop) / currentTrailStop
			strength = math.Min(0.85, 0.6+distance*10) // Base 60% + distance component
		} else {
			strength = 0.6
		}
	} else if prevPosition >= 0 && currentPosition == -1 && atr.config.UseShorts {
		// Crossunder -> SELL signal (only if shorts enabled)
		signal = Sell
		// Calculate strength based on distance from trailing stop
		if currentClose > 0 {
			distance := (currentTrailStop - currentClose) / currentClose
			strength = math.Min(0.85, 0.6+distance*10) // Base 60% + distance component
		} else {
			strength = 0.6
		}
	} else {
		// No crossover - HOLD
		signal = Hold
		if currentPosition == 1 {
			// In long position - moderate hold strength
			strength = 0.4
		} else if currentPosition == -1 {
			// In short position - moderate hold strength
			strength = 0.4
		} else {
			// Neutral position - low strength
			strength = 0.2
		}
	}

	atr.lastSignal = signal
	atr.lastStrength = strength

	return signal, strength
}

// Calculate implements TechnicalIndicator interface
func (atr *ATR) Calculate(candles []Candle) []float64 {
	if len(candles) < atr.config.Period {
		return []float64{}
	}

	values := make([]float64, 0, len(candles))

	// Process each candle
	for _, candle := range candles {
		atr.Update(candle)
		if atr.initialized && len(atr.trailingStops) > 0 {
			// Return trailing stop value
			values = append(values, atr.trailingStops[len(atr.trailingStops)-1])
		}
	}

	return values
}

// GetSignal implements TechnicalIndicator interface
func (atr *ATR) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	signal, strength := atr.GetCurrentSignal()

	var value float64
	if len(values) > 0 {
		value = values[len(values)-1]
	}

	return IndicatorSignal{
		Name:      atr.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     value,
		Timestamp: time.Now(),
		Timeframe: atr.timeframe,
	}
}

// GetLastSignal returns the last signal and strength
func (atr *ATR) GetLastSignal() (SignalType, float64) {
	return atr.lastSignal, atr.lastStrength
}

// GetCurrentTrailingStop returns the current trailing stop value
func (atr *ATR) GetCurrentTrailingStop() float64 {
	if !atr.initialized || len(atr.trailingStops) == 0 {
		return 0.0
	}
	return atr.trailingStops[len(atr.trailingStops)-1]
}

// GetCurrentPosition returns the current position (1=long, -1=short, 0=neutral)
func (atr *ATR) GetCurrentPosition() int {
	if !atr.initialized || len(atr.positions) == 0 {
		return 0
	}
	return atr.positions[len(atr.positions)-1]
}

// String returns a string representation
func (atr *ATR) String() string {
	if !atr.initialized {
		return "ATR Pine Script: Not initialized"
	}

	trailingStop := atr.GetCurrentTrailingStop()
	position := atr.GetCurrentPosition()
	posStr := "Neutral"
	if position == 1 {
		posStr = "Long"
	} else if position == -1 {
		posStr = "Short"
	}

	return fmt.Sprintf("ATR Pine Script: TrailingStop=%.2f, Position=%s, Signal=%s, Strength=%.2f",
		trailingStop, posStr, atr.lastSignal, atr.lastStrength)
}
