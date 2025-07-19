package indicator

import (
	"time"
)

// Timeframe represents different time intervals
type Timeframe int

const (
	FiveMinute Timeframe = iota
	FifteenMinute
	FortyFiveMinute
	EightHour
	Daily
)

func (t Timeframe) String() string {
	switch t {
	case FiveMinute:
		return "5m"
	case FifteenMinute:
		return "15m"
	case FortyFiveMinute:
		return "45m"
	case EightHour:
		return "8h"
	case Daily:
		return "1d"
	default:
		return "unknown"
	}
}

func (t Timeframe) Duration() time.Duration {
	switch t {
	case FiveMinute:
		return 5 * time.Minute
	case FifteenMinute:
		return 15 * time.Minute
	case FortyFiveMinute:
		return 45 * time.Minute
	case EightHour:
		return 8 * time.Hour
	case Daily:
		return 24 * time.Hour
	default:
		return time.Hour
	}
}

// Candle represents OHLCV data for a specific time period
type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// SignalType represents the type of trading signal
type SignalType int

const (
	Hold SignalType = iota
	Buy
	Sell
)

func (s SignalType) String() string {
	switch s {
	case Hold:
		return "HOLD"
	case Buy:
		return "BUY"
	case Sell:
		return "SELL"
	default:
		return "UNKNOWN"
	}
}

// IndicatorSignal represents a signal from a technical indicator
type IndicatorSignal struct {
	Name      string     `json:"name"`
	Signal    SignalType `json:"signal"`
	Strength  float64    `json:"strength"` // 0-1 confidence
	Value     float64    `json:"value"`    // actual indicator value
	Timestamp time.Time  `json:"timestamp"`
	Timeframe Timeframe  `json:"timeframe"`
}

// TechnicalIndicator interface that all indicators must implement
type TechnicalIndicator interface {
	Calculate(candles []Candle) []float64
	GetSignal(values []float64, currentPrice float64) IndicatorSignal
	GetName() string
}

// Configuration types for each indicator

// RSIConfig represents configuration for RSI indicator
type RSIConfig struct {
	Enabled    bool    `json:"enabled"` // Feature flag to enable/disable RSI
	Period     int     `json:"period"`
	Overbought float64 `json:"overbought"`
	Oversold   float64 `json:"oversold"`
}

// MACDConfig represents configuration for MACD indicator
type MACDConfig struct {
	Enabled      bool `json:"enabled"` // Feature flag to enable/disable MACD
	FastPeriod   int  `json:"fast_period"`
	SlowPeriod   int  `json:"slow_period"`
	SignalPeriod int  `json:"signal_period"`
}

// VolumeConfig represents configuration for Volume indicator
type VolumeConfig struct {
	Enabled         bool    `json:"enabled"` // Feature flag to enable/disable Volume analysis
	Period          int     `json:"period"`
	VolumeThreshold float64 `json:"volume_threshold"`
}

// TrendConfig represents configuration for Trend indicator
type TrendConfig struct {
	Enabled bool `json:"enabled"` // Feature flag to enable/disable Trend analysis
	ShortMA int  `json:"short_ma"`
	LongMA  int  `json:"long_ma"`
}

// SupportResistanceConfig represents configuration for Support/Resistance indicator
type SupportResistanceConfig struct {
	Enabled   bool    `json:"enabled"` // Feature flag to enable/disable Support/Resistance
	Period    int     `json:"period"`
	Threshold float64 `json:"threshold"`
}

// IchimokuConfig represents configuration for Ichimoku Cloud indicator
type IchimokuConfig struct {
	Enabled      bool `json:"enabled"`       // Feature flag to enable/disable Ichimoku Cloud
	TenkanPeriod int  `json:"tenkan_period"` // Conversion Line period (default: 9)
	KijunPeriod  int  `json:"kijun_period"`  // Base Line period (default: 26)
	SenkouPeriod int  `json:"senkou_period"` // Leading Span B period (default: 52)
	Displacement int  `json:"displacement"`  // Cloud displacement (default: 26)
}

// MFIConfig represents configuration for Money Flow Index indicator
type MFIConfig struct {
	Enabled    bool    `json:"enabled"`    // Feature flag to enable/disable Reverse-MFI
	Period     int     `json:"period"`     // Period for MFI calculation (default: 14)
	Overbought float64 `json:"overbought"` // Overbought level (default: 80)
	Oversold   float64 `json:"oversold"`   // Oversold level (default: 20)
}

// BollingerBandsConfig represents configuration for Bollinger Bands indicator
type BollingerBandsConfig struct {
	Enabled       bool    `json:"enabled"`        // Feature flag to enable/disable Bollinger Bands
	Period        int     `json:"period"`         // Period for moving average and std dev (default: 20)
	StandardDev   float64 `json:"standard_dev"`   // Standard deviation multiplier (default: 2.0)
	OverboughtStd float64 `json:"overbought_std"` // Overbought threshold (default: 0.8)
	OversoldStd   float64 `json:"oversold_std"`   // Oversold threshold (default: 0.2)
}

// PivotPoint represents a support or resistance level
type PivotPoint struct {
	Price     float64   `json:"price"`
	Timestamp time.Time `json:"timestamp"`
	Type      string    `json:"type"` // "support" or "resistance"
	Index     int       `json:"index"`
}

// IchimokuValues holds all calculated Ichimoku lines
type IchimokuValues struct {
	TenkanSen   []float64 // Conversion Line
	KijunSen    []float64 // Base Line
	SenkouSpanA []float64 // Leading Span A
	SenkouSpanB []float64 // Leading Span B
	ChikouSpan  []float64 // Lagging Span
	CloudTop    []float64 // Upper cloud boundary
	CloudBottom []float64 // Lower cloud boundary
}

// IchimokuSignal represents a detailed Ichimoku signal
type IchimokuSignal struct {
	Signal    SignalType     `json:"signal"`
	Strength  float64        `json:"strength"`
	Reasoning []string       `json:"reasoning"`
	Values    IchimokuValues `json:"values"`
}
