package indicator

import (
	"fmt"
	"math"
	"time"
)

// ReverseMFI Indicator
type ReverseMFI struct {
	config    MFIConfig
	timeframe Timeframe
}

// NewReverseMFI creates a new Reverse-MFI indicator
func NewReverseMFI(config MFIConfig, timeframe Timeframe) *ReverseMFI {
	return &ReverseMFI{
		config:    config,
		timeframe: timeframe,
	}
}

// Calculate computes MFI values for given candles
func (mfi *ReverseMFI) Calculate(candles []Candle) []float64 {
	if len(candles) < mfi.config.Period+1 {
		return []float64{}
	}

	values := make([]float64, len(candles)-mfi.config.Period)

	for i := mfi.config.Period; i < len(candles); i++ {
		// Calculate raw money flow for the period
		positiveFlow := 0.0
		negativeFlow := 0.0

		for j := i - mfi.config.Period + 1; j <= i; j++ {
			typicalPrice := (candles[j].High + candles[j].Low + candles[j].Close) / 3.0
			rawMoneyFlow := typicalPrice * candles[j].Volume

			if j > 0 {
				prevTypicalPrice := (candles[j-1].High + candles[j-1].Low + candles[j-1].Close) / 3.0

				if typicalPrice > prevTypicalPrice {
					positiveFlow += rawMoneyFlow
				} else if typicalPrice < prevTypicalPrice {
					negativeFlow += rawMoneyFlow
				}
				// If typical price is unchanged, money flow is neither positive nor negative
			}
		}

		// Calculate Money Flow Index
		var mfiValue float64
		if negativeFlow == 0 {
			mfiValue = 100.0
		} else {
			moneyRatio := positiveFlow / negativeFlow
			mfiValue = 100.0 - (100.0 / (1.0 + moneyRatio))
		}

		values[i-mfi.config.Period] = mfiValue
	}

	return values
}

// GetSignal generates a REVERSE trading signal based on MFI values
// In Reverse-MFI strategy: overbought levels become buy signals, oversold levels become sell signals
func (mfi *ReverseMFI) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      mfi.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: mfi.timeframe,
		}
	}

	currentMFI := values[len(values)-1]
	var signal SignalType
	var strength float64

	// REVERSE MFI Logic: Contrarian approach
	if currentMFI >= mfi.config.Overbought {
		// Traditional: Sell (overbought)
		// Reverse: Buy (assuming strong buying pressure continues)
		signal = Buy
		strength = math.Min(1.0, (currentMFI-mfi.config.Overbought)/(100-mfi.config.Overbought))
	} else if currentMFI <= mfi.config.Oversold {
		// Traditional: Buy (oversold)
		// Reverse: Sell (assuming strong selling pressure continues)
		signal = Sell
		strength = math.Min(1.0, (mfi.config.Oversold-currentMFI)/mfi.config.Oversold)
	} else {
		signal = Hold
		// Strength based on distance from neutral (50)
		neutralDistance := math.Abs(currentMFI - 50)
		strength = neutralDistance / 50.0
	}

	return IndicatorSignal{
		Name:      mfi.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentMFI,
		Timestamp: time.Now(),
		Timeframe: mfi.timeframe,
	}
}

// GetName returns the indicator name
func (mfi *ReverseMFI) GetName() string {
	return fmt.Sprintf("ReverseMFI_%s", mfi.timeframe.String())
}
