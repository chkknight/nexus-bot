package bot

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

// MarketData represents market data for multiple timeframes
type MarketData struct {
	Symbol     string                 `json:"symbol"`
	Timeframes map[Timeframe][]Candle `json:"timeframes"`
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

// TradingSignal represents a final trading decision
type TradingSignal struct {
	Symbol           string            `json:"symbol"`
	Signal           SignalType        `json:"signal"`
	Confidence       float64           `json:"confidence"`
	Timestamp        time.Time         `json:"timestamp"`
	IndicatorSignals []IndicatorSignal `json:"indicator_signals"`
	Reasoning        string            `json:"reasoning"`
	TargetPrice      float64           `json:"target_price,omitempty"`
	StopLoss         float64           `json:"stop_loss,omitempty"`
}

// RSIConfig holds RSI parameters
type RSIConfig struct {
	Enabled    bool    `json:"enabled"` // Feature flag to enable/disable RSI
	Period     int     `json:"period"`
	Overbought float64 `json:"overbought"`
	Oversold   float64 `json:"oversold"`
}

// MACDConfig holds MACD parameters
type MACDConfig struct {
	Enabled      bool `json:"enabled"` // Feature flag to enable/disable MACD
	FastPeriod   int  `json:"fast_period"`
	SlowPeriod   int  `json:"slow_period"`
	SignalPeriod int  `json:"signal_period"`
}

// VolumeConfig holds volume analysis parameters
type VolumeConfig struct {
	Enabled         bool    `json:"enabled"` // Feature flag to enable/disable Volume analysis
	Period          int     `json:"period"`
	VolumeThreshold float64 `json:"volume_threshold"`
}

// TrendConfig holds trend analysis parameters
type TrendConfig struct {
	Enabled bool `json:"enabled"` // Feature flag to enable/disable Trend analysis
	ShortMA int  `json:"short_ma"`
	LongMA  int  `json:"long_ma"`
}

// SupportResistanceConfig holds S/R parameters
type SupportResistanceConfig struct {
	Enabled   bool    `json:"enabled"` // Feature flag to enable/disable Support/Resistance
	Period    int     `json:"period"`
	Threshold float64 `json:"threshold"`
}

// IchimokuConfig holds Ichimoku Cloud parameters
type IchimokuConfig struct {
	Enabled      bool `json:"enabled"`       // Feature flag to enable/disable Ichimoku Cloud
	TenkanPeriod int  `json:"tenkan_period"` // Conversion Line period (default: 9)
	KijunPeriod  int  `json:"kijun_period"`  // Base Line period (default: 26)
	SenkouPeriod int  `json:"senkou_period"` // Leading Span B period (default: 52)
	Displacement int  `json:"displacement"`  // Cloud displacement (default: 26)
}

// MFIConfig holds Money Flow Index parameters
type MFIConfig struct {
	Enabled    bool    `json:"enabled"`    // Feature flag to enable/disable Reverse-MFI
	Period     int     `json:"period"`     // Period for MFI calculation (default: 14)
	Overbought float64 `json:"overbought"` // Overbought level (default: 80)
	Oversold   float64 `json:"oversold"`   // Oversold level (default: 20)
}

// BollingerBandsConfig holds Bollinger Bands parameters
type BollingerBandsConfig struct {
	Enabled       bool    `json:"enabled"`        // Feature flag to enable/disable Bollinger Bands
	Period        int     `json:"period"`         // Period for moving average and std dev (default: 20)
	StandardDev   float64 `json:"standard_dev"`   // Standard deviation multiplier (default: 2.0)
	OverboughtStd float64 `json:"overbought_std"` // Overbought threshold (default: 0.8)
	OversoldStd   float64 `json:"oversold_std"`   // Oversold threshold (default: 0.2)
}

// StochasticConfig holds Stochastic Oscillator parameters
type StochasticConfig struct {
	Enabled         bool    `json:"enabled"`          // Feature flag to enable/disable Stochastic
	KPeriod         int     `json:"k_period"`         // Period for %K calculation (default: 14)
	DPeriod         int     `json:"d_period"`         // Period for %D smoothing (default: 3)
	SlowPeriod      int     `json:"slow_period"`      // Period for slow %K (default: 3)
	Overbought      float64 `json:"overbought"`       // Overbought threshold (default: 80)
	Oversold        float64 `json:"oversold"`         // Oversold threshold (default: 20)
	MomentumBoost   float64 `json:"momentum_boost"`   // Boost factor for momentum signals (default: 1.2)
	DivergenceBoost float64 `json:"divergence_boost"` // Boost for divergence signals (default: 1.3)
}

// WilliamsRConfig holds Williams %R parameters
type WilliamsRConfig struct {
	Enabled       bool    `json:"enabled"`        // Feature flag to enable/disable Williams %R
	Period        int     `json:"period"`         // Lookback period (default: 10 for 5-minute)
	Overbought    float64 `json:"overbought"`     // Overbought threshold (default: -20)
	Oversold      float64 `json:"oversold"`       // Oversold threshold (default: -80)
	FastResponse  bool    `json:"fast_response"`  // Enable fast response for 5-minute trading
	MomentumBoost float64 `json:"momentum_boost"` // Boost factor for momentum signals (default: 1.3)
	ReversalBoost float64 `json:"reversal_boost"` // Boost for reversal signals (default: 1.4)
}

// PinBarConfig holds Pin Bar pattern detection parameters
type PinBarConfig struct {
	Enabled              bool    `json:"enabled"`                // Feature flag to enable/disable Pin Bar
	MinWickRatio         float64 `json:"min_wick_ratio"`         // Minimum wick to body ratio (default: 2.0)
	MaxBodyRatio         float64 `json:"max_body_ratio"`         // Maximum body to total range ratio (default: 0.33)
	MinRangePercent      float64 `json:"min_range_percent"`      // Minimum range as % of price (default: 0.001)
	SupportResistance    bool    `json:"support_resistance"`     // Consider S/R levels for strength
	TrendConfirmation    bool    `json:"trend_confirmation"`     // Require trend confirmation
	PatternStrengthBoost float64 `json:"pattern_strength_boost"` // Boost for strong patterns (default: 1.2)
}

// EMAConfig holds EMA configuration parameters
type EMAConfig struct {
	Enabled        bool    `json:"enabled"`         // Feature flag to enable/disable EMA
	FastPeriod     int     `json:"fast_period"`     // Fast EMA period (default: 12)
	SlowPeriod     int     `json:"slow_period"`     // Slow EMA period (default: 26)
	SignalPeriod   int     `json:"signal_period"`   // Signal EMA period (default: 9)
	TrendPeriod    int     `json:"trend_period"`    // Trend EMA period (default: 50)
	SlopeThreshold float64 `json:"slope_threshold"` // Minimum slope for trend signals (default: 0.0001)
	CrossoverBoost float64 `json:"crossover_boost"` // Boost for crossover signals (default: 1.3)
	TrendBoost     float64 `json:"trend_boost"`     // Boost for trend alignment (default: 1.2)
	VolumeConfirm  bool    `json:"volume_confirm"`  // Require volume confirmation (default: false)
}

// ElliottWaveConfig holds Elliott Wave configuration parameters
type ElliottWaveConfig struct {
	Enabled            bool    `json:"enabled"`             // Feature flag to enable/disable Elliott Wave
	MinWaveLength      int     `json:"min_wave_length"`     // Minimum candles for wave identification (default: 5)
	FibonacciTolerance float64 `json:"fibonacci_tolerance"` // Tolerance for Fibonacci relationships (default: 0.1)
	TrendStrength      float64 `json:"trend_strength"`      // Minimum trend strength for wave validation (default: 0.02)
	ImpulseBoost       float64 `json:"impulse_boost"`       // Boost for impulse wave signals (default: 1.4)
	CorrectionBoost    float64 `json:"correction_boost"`    // Boost for correction wave signals (default: 1.2)
	CompletionBoost    float64 `json:"completion_boost"`    // Boost for wave completion signals (default: 1.5)
	MaxLookback        int     `json:"max_lookback"`        // Maximum lookback period (default: 100)
}

// BinanceConfig holds Binance API configuration
type BinanceConfig struct {
	APIKey     string `json:"api_key"`
	SecretKey  string `json:"secret_key"`
	UseTestnet bool   `json:"use_testnet"`
}

// Config holds all configuration parameters
type Config struct {
	RSI               RSIConfig               `json:"rsi"`
	MACD              MACDConfig              `json:"macd"`
	Volume            VolumeConfig            `json:"volume"`
	Trend             TrendConfig             `json:"trend"`
	SupportResistance SupportResistanceConfig `json:"support_resistance"`
	Ichimoku          IchimokuConfig          `json:"ichimoku"`
	MFI               MFIConfig               `json:"mfi"`
	BollingerBands    BollingerBandsConfig    `json:"bollinger_bands"`
	Stochastic        StochasticConfig        `json:"stochastic"`
	WilliamsR         WilliamsRConfig         `json:"williams_r"`
	PinBar            PinBarConfig            `json:"pin_bar"`
	EMA               EMAConfig               `json:"ema"`
	ElliottWave       ElliottWaveConfig       `json:"elliott_wave"`
	MinConfidence     float64                 `json:"min_confidence"`
	Symbol            string                  `json:"symbol"`
	Binance           BinanceConfig           `json:"binance"`
	DataProvider      string                  `json:"data_provider"` // "sample" or "binance"
}
