package indicator

import (
	"fmt"
	"math"
	"time"
)

// ChannelType represents the type of channel detected
type ChannelType int

const (
	HoldChannel ChannelType = iota
	IncreasingChannel
	DecreasingChannel
)

func (c ChannelType) String() string {
	switch c {
	case IncreasingChannel:
		return "INCREASING"
	case DecreasingChannel:
		return "DECREASING"
	case HoldChannel:
		return "HOLD"
	default:
		return "UNKNOWN"
	}
}

// ChannelAnalysisConfig holds channel analysis parameters
type ChannelAnalysisConfig struct {
	Enabled          bool    `json:"enabled"`           // Feature flag
	LookbackPeriod   int     `json:"lookback_period"`   // Period to analyze (default: 20)
	ChannelThreshold float64 `json:"channel_threshold"` // Min % change to confirm channel (default: 0.5%)
	SignalBoost      float64 `json:"signal_boost"`      // Boost factor for channel signals (default: 1.2)
}

// PivotPoint represents a high or low point
type PivotPointData struct {
	Price     float64
	Index     int
	Timestamp time.Time
	IsHigh    bool // true for high, false for low
}

// ChannelAnalysis indicator
type ChannelAnalysis struct {
	config    ChannelAnalysisConfig
	timeframe Timeframe
	highs     []PivotPointData
	lows      []PivotPointData
}

// NewChannelAnalysis creates a new Channel Analysis indicator
func NewChannelAnalysis(config ChannelAnalysisConfig, timeframe Timeframe) *ChannelAnalysis {
	return &ChannelAnalysis{
		config:    config,
		timeframe: timeframe,
		highs:     make([]PivotPointData, 0),
		lows:      make([]PivotPointData, 0),
	}
}

// Calculate identifies pivot points and channel direction
func (ca *ChannelAnalysis) Calculate(candles []Candle) []float64 {
	// CRITICAL: Need minimum candles for reliable channel analysis
	// 3 (pivot buffer) + 15 (lookback) + 3 (pivot buffer) = 21 candles minimum
	minRequiredCandles := ca.config.LookbackPeriod + 6 // Add 6 for pivot detection buffers

	if len(candles) < minRequiredCandles {
		// Not enough data for reliable channel analysis
		return []float64{}
	}

	// Find pivot points (highs and lows)
	ca.findPivotPoints(candles)

	// Validate we have sufficient pivot points for analysis
	if len(ca.highs) < 2 || len(ca.lows) < 2 {
		// Need at least 2 highs and 2 lows for channel trend analysis
		return []float64{}
	}

	// Analyze channel direction for each valid point
	results := make([]float64, len(candles)-ca.config.LookbackPeriod+1)

	for i := 0; i < len(results); i++ {
		endIndex := ca.config.LookbackPeriod + i - 1
		channelType := ca.analyzeChannel(candles, endIndex)
		results[i] = float64(channelType) // 0=Hold, 1=Increasing, 2=Decreasing
	}

	return results
}

// findPivotPoints identifies significant highs and lows
func (ca *ChannelAnalysis) findPivotPoints(candles []Candle) {
	ca.highs = ca.highs[:0] // Clear previous data
	ca.lows = ca.lows[:0]

	lookback := 3 // Look 3 candles back and forward for pivot confirmation

	for i := lookback; i < len(candles)-lookback; i++ {
		current := candles[i]

		// Check for pivot high
		isHigh := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].High >= current.High {
				isHigh = false
				break
			}
		}

		if isHigh {
			ca.highs = append(ca.highs, PivotPointData{
				Price:     current.High,
				Index:     i,
				Timestamp: current.Timestamp,
				IsHigh:    true,
			})
		}

		// Check for pivot low
		isLow := true
		for j := i - lookback; j <= i+lookback; j++ {
			if j != i && candles[j].Low <= current.Low {
				isLow = false
				break
			}
		}

		if isLow {
			ca.lows = append(ca.lows, PivotPointData{
				Price:     current.Low,
				Index:     i,
				Timestamp: current.Timestamp,
				IsHigh:    false,
			})
		}
	}
}

// analyzeChannel determines the channel type based on pivot points
func (ca *ChannelAnalysis) analyzeChannel(candles []Candle, endIndex int) ChannelType {
	if len(ca.highs) < 2 || len(ca.lows) < 2 {
		return HoldChannel
	}

	// Get recent pivot points within our lookback period
	startIndex := endIndex - ca.config.LookbackPeriod + 1
	recentHighs := ca.getRecentPivots(ca.highs, startIndex, endIndex)
	recentLows := ca.getRecentPivots(ca.lows, startIndex, endIndex)

	if len(recentHighs) < 2 || len(recentLows) < 2 {
		return HoldChannel
	}

	// Analyze trend in highs and lows
	highsTrend := ca.calculateTrend(recentHighs)
	lowsTrend := ca.calculateTrend(recentLows)

	currentPrice := candles[endIndex].Close
	priceChangeThreshold := currentPrice * ca.config.ChannelThreshold / 100

	// Determine channel type
	if highsTrend > priceChangeThreshold && lowsTrend > priceChangeThreshold {
		return IncreasingChannel // Both highs and lows trending up
	} else if highsTrend < -priceChangeThreshold && lowsTrend < -priceChangeThreshold {
		return DecreasingChannel // Both highs and lows trending down
	} else {
		return HoldChannel // Sideways or unclear trend
	}
}

// getRecentPivots filters pivot points within the specified range
func (ca *ChannelAnalysis) getRecentPivots(pivots []PivotPointData, startIndex, endIndex int) []PivotPointData {
	var recent []PivotPointData
	for _, pivot := range pivots {
		if pivot.Index >= startIndex && pivot.Index <= endIndex {
			recent = append(recent, pivot)
		}
	}
	return recent
}

// calculateTrend calculates the overall trend of pivot points
func (ca *ChannelAnalysis) calculateTrend(pivots []PivotPointData) float64 {
	if len(pivots) < 2 {
		return 0
	}

	// Simple linear trend calculation
	firstPrice := pivots[0].Price
	lastPrice := pivots[len(pivots)-1].Price

	return lastPrice - firstPrice
}

// GetSignal generates trading signals based on channel analysis
func (ca *ChannelAnalysis) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	// VALIDATION: Ensure we have sufficient data and pivot points
	if len(values) == 0 || len(ca.highs) < 2 || len(ca.lows) < 2 {
		return IndicatorSignal{
			Name:      ca.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: ca.timeframe,
		}
	}

	currentChannel := ChannelType(values[len(values)-1])
	var signal SignalType
	var strength float64

	switch currentChannel {
	case IncreasingChannel:
		signal = Buy
		strength = 0.7 * ca.config.SignalBoost // FIXED: Reduced from 0.8 to 0.7
	case DecreasingChannel:
		signal = Sell
		strength = 0.7 * ca.config.SignalBoost // FIXED: Reduced from 0.8 to 0.7
	case HoldChannel:
		signal = Hold
		strength = 0.3 // Low strength for sideways market
	}

	return IndicatorSignal{
		Name:      ca.GetName(),
		Signal:    signal,
		Strength:  math.Min(strength, 0.85), // FIXED: Cap at 0.85 instead of 1.0
		Value:     float64(currentChannel),
		Timestamp: time.Now(),
		Timeframe: ca.timeframe,
	}
}

// GetName returns the indicator name
func (ca *ChannelAnalysis) GetName() string {
	return fmt.Sprintf("Channel_%s", ca.timeframe.String())
}

// GetChannelAnalysis returns detailed channel information
func (ca *ChannelAnalysis) GetChannelAnalysis(values []float64) string {
	if len(values) == 0 {
		return "❌ Insufficient data for channel analysis"
	}

	currentChannel := ChannelType(values[len(values)-1])

	switch currentChannel {
	case IncreasingChannel:
		return "📈 INCREASING CHANNEL: Market showing higher highs and higher lows - Bullish trend confirmed"
	case DecreasingChannel:
		return "📉 DECREASING CHANNEL: Market showing lower highs and lower lows - Bearish trend confirmed"
	case HoldChannel:
		return "➡️ HOLD CHANNEL: Market in consolidation/sideways movement - Wait for breakout"
	default:
		return "❓ UNKNOWN CHANNEL: Unable to determine market direction"
	}
}

// GetDataStatus returns the status of data requirements for channel analysis
func (ca *ChannelAnalysis) GetDataStatus(candleCount int) string {
	minRequired := ca.config.LookbackPeriod + 6

	if candleCount < minRequired {
		return fmt.Sprintf("⏳ WAITING: Need %d candles, have %d (%.1f%% ready)",
			minRequired, candleCount, float64(candleCount)/float64(minRequired)*100)
	}

	if len(ca.highs) < 2 || len(ca.lows) < 2 {
		return fmt.Sprintf("⚠️  DATA OK: %d candles available, but insufficient pivot points (highs: %d, lows: %d)",
			candleCount, len(ca.highs), len(ca.lows))
	}

	return fmt.Sprintf("✅ READY: %d candles, %d highs, %d lows - Channel analysis active",
		candleCount, len(ca.highs), len(ca.lows))
}
