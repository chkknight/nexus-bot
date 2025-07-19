package indicator

import (
	"fmt"
	"math"
	"time"
)

// Volume Indicator
type Volume struct {
	config    VolumeConfig
	timeframe Timeframe
}

// NewVolume creates a new Volume indicator
func NewVolume(config VolumeConfig, timeframe Timeframe) *Volume {
	return &Volume{
		config:    config,
		timeframe: timeframe,
	}
}

// Calculate computes volume analysis for given candles
func (v *Volume) Calculate(candles []Candle) []float64 {
	if len(candles) < v.config.Period {
		return []float64{}
	}

	// Calculate volume moving average
	volumeMA := make([]float64, len(candles)-v.config.Period+1)

	for i := 0; i < len(volumeMA); i++ {
		var sum float64
		for j := 0; j < v.config.Period; j++ {
			sum += candles[i+j].Volume
		}
		volumeMA[i] = sum / float64(v.config.Period)
	}

	return volumeMA
}

// GetSignal generates a trading signal based on volume analysis
func (v *Volume) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      v.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: v.timeframe,
		}
	}

	currentVolume := values[len(values)-1]
	var signal SignalType
	var strength float64

	// Volume threshold analysis
	if currentVolume > v.config.VolumeThreshold {
		// High volume - confirmation signal
		signal = Hold // Volume confirms other signals
		strength = math.Min(1.0, currentVolume/v.config.VolumeThreshold)
	} else {
		// Low volume - weak signal
		signal = Hold
		strength = 0.2
	}

	return IndicatorSignal{
		Name:      v.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentVolume,
		Timestamp: time.Now(),
		Timeframe: v.timeframe,
	}
}

// GetName returns the indicator name
func (v *Volume) GetName() string {
	return fmt.Sprintf("Volume_%s", v.timeframe.String())
}
