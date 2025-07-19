package indicator

import (
	"fmt"
	"math"
	"time"
)

// RSI Indicator
type RSI struct {
	config    RSIConfig
	timeframe Timeframe
}

// NewRSI creates a new RSI indicator
func NewRSI(config RSIConfig, timeframe Timeframe) *RSI {
	return &RSI{
		config:    config,
		timeframe: timeframe,
	}
}

// Calculate computes RSI values for given candles
func (rsi *RSI) Calculate(candles []Candle) []float64 {
	if len(candles) < rsi.config.Period+1 {
		return []float64{}
	}

	values := make([]float64, len(candles)-rsi.config.Period)

	// Calculate price changes
	priceChanges := make([]float64, len(candles)-1)
	for i := 1; i < len(candles); i++ {
		priceChanges[i-1] = candles[i].Close - candles[i-1].Close
	}

	// Calculate initial average gain and loss
	var avgGain, avgLoss float64
	for i := 0; i < rsi.config.Period; i++ {
		if priceChanges[i] > 0 {
			avgGain += priceChanges[i]
		} else {
			avgLoss += math.Abs(priceChanges[i])
		}
	}
	avgGain /= float64(rsi.config.Period)
	avgLoss /= float64(rsi.config.Period)

	// Calculate first RSI value
	if avgLoss == 0 {
		values[0] = 100
	} else {
		rs := avgGain / avgLoss
		values[0] = 100 - (100 / (1 + rs))
	}

	// Calculate subsequent RSI values using smoothed averages
	for i := rsi.config.Period; i < len(priceChanges); i++ {
		change := priceChanges[i]

		var gain, loss float64
		if change > 0 {
			gain = change
		} else {
			loss = math.Abs(change)
		}

		// Smoothed moving average
		avgGain = (avgGain*float64(rsi.config.Period-1) + gain) / float64(rsi.config.Period)
		avgLoss = (avgLoss*float64(rsi.config.Period-1) + loss) / float64(rsi.config.Period)

		if avgLoss == 0 {
			values[i-rsi.config.Period+1] = 100
		} else {
			rs := avgGain / avgLoss
			values[i-rsi.config.Period+1] = 100 - (100 / (1 + rs))
		}
	}

	return values
}

// GetSignal generates a trading signal based on RSI values
func (rsi *RSI) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      rsi.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: rsi.timeframe,
		}
	}

	currentRSI := values[len(values)-1]
	var signal SignalType
	var strength float64

	if currentRSI <= rsi.config.Oversold {
		signal = Buy
		// Strength increases as RSI gets more oversold
		strength = math.Min(1.0, (rsi.config.Oversold-currentRSI)/rsi.config.Oversold)
	} else if currentRSI >= rsi.config.Overbought {
		signal = Sell
		// Strength increases as RSI gets more overbought
		strength = math.Min(1.0, (currentRSI-rsi.config.Overbought)/(100-rsi.config.Overbought))
	} else {
		signal = Hold
		// Neutral zone strength based on distance from center
		centerDistance := math.Abs(currentRSI - 50)
		strength = 1.0 - (centerDistance / 50.0)
	}

	return IndicatorSignal{
		Name:      rsi.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentRSI,
		Timestamp: time.Now(),
		Timeframe: rsi.timeframe,
	}
}

// GetName returns the indicator name
func (rsi *RSI) GetName() string {
	return fmt.Sprintf("RSI_%s", rsi.timeframe.String())
}
