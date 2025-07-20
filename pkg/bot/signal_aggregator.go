package bot

import (
	"fmt"
	"math"
	"strings"
	"time"

	"trading-bot/pkg/indicator"
)

// SignalAggregator combines signals from multiple indicators and timeframes
type SignalAggregator struct {
	config     Config
	indicators map[Timeframe][]indicator.TechnicalIndicator
}

// NewSignalAggregator creates a new signal aggregator
func NewSignalAggregator(config Config) *SignalAggregator {
	sa := &SignalAggregator{
		config:     config,
		indicators: make(map[Timeframe][]indicator.TechnicalIndicator),
	}

	// Initialize indicators for each timeframe
	sa.initializeIndicators()
	return sa
}

// GetActiveIndicatorCount returns the number of active indicators for a timeframe
func (sa *SignalAggregator) GetActiveIndicatorCount(timeframe Timeframe) int {
	return len(sa.indicators[timeframe])
}

// GetTotalActiveIndicators returns the total number of active indicators across all timeframes
func (sa *SignalAggregator) GetTotalActiveIndicators() int {
	enabledIndicators := 0

	if sa.config.RSI.Enabled {
		enabledIndicators++
	}
	if sa.config.MACD.Enabled {
		enabledIndicators++
	}
	if sa.config.Volume.Enabled {
		enabledIndicators++
	}
	if sa.config.Trend.Enabled {
		enabledIndicators++
	}
	if sa.config.SupportResistance.Enabled {
		enabledIndicators++
	}
	if sa.config.Ichimoku.Enabled {
		enabledIndicators++
	}
	if sa.config.MFI.Enabled {
		enabledIndicators++
	}
	if sa.config.BollingerBands.Enabled {
		enabledIndicators++
	}
	if sa.config.Stochastic.Enabled {
		enabledIndicators++
	}
	if sa.config.WilliamsR.Enabled {
		enabledIndicators++
	}
	if sa.config.PinBar.Enabled {
		enabledIndicators++
	}
	if sa.config.EMA.Enabled {
		enabledIndicators++
	}
	if sa.config.ElliottWave.Enabled {
		enabledIndicators++
	}
	if sa.config.ChannelAnalysis.Enabled {
		enabledIndicators++
	}
	if sa.config.ATR.Enabled {
		enabledIndicators++
	}

	return enabledIndicators
}

// GetActiveIndicatorNames returns the names of active indicators
func (sa *SignalAggregator) GetActiveIndicatorNames() []string {
	var names []string

	if sa.config.RSI.Enabled {
		names = append(names, "RSI")
	}
	if sa.config.MACD.Enabled {
		names = append(names, "MACD")
	}
	if sa.config.Volume.Enabled {
		names = append(names, "Volume")
	}
	if sa.config.Trend.Enabled {
		names = append(names, "Trend")
	}
	if sa.config.SupportResistance.Enabled {
		names = append(names, "Support/Resistance")
	}
	if sa.config.Ichimoku.Enabled {
		names = append(names, "Ichimoku")
	}
	if sa.config.MFI.Enabled {
		names = append(names, "Reverse-MFI")
	}
	if sa.config.BollingerBands.Enabled {
		names = append(names, "Bollinger Bands")
	}
	if sa.config.Stochastic.Enabled {
		names = append(names, "Stochastic")
	}
	if sa.config.WilliamsR.Enabled {
		names = append(names, "Williams %R")
	}
	if sa.config.PinBar.Enabled {
		names = append(names, "Pin Bar")
	}
	if sa.config.EMA.Enabled {
		names = append(names, "EMA")
	}
	if sa.config.ElliottWave.Enabled {
		names = append(names, "Elliott Wave")
	}
	if sa.config.ChannelAnalysis.Enabled {
		names = append(names, "Channel Analysis")
	}
	if sa.config.ATR.Enabled {
		names = append(names, "ATR")
	}

	return names
}

// initializeIndicators sets up all indicators for each timeframe
func (sa *SignalAggregator) initializeIndicators() {
	// FOCUSED: Only initialize 5-minute timeframe for ultra-fast trading
	timeframes := []Timeframe{FiveMinute}

	for _, tf := range timeframes {
		var indicators []indicator.TechnicalIndicator

		// Add RSI (if enabled)
		if sa.config.RSI.Enabled {
			indicators = append(indicators, indicator.NewRSI(convertRSIConfig(sa.config.RSI), convertTimeframe(tf)))
		}

		// Add MACD (if enabled)
		if sa.config.MACD.Enabled {
			indicators = append(indicators, indicator.NewMACD(convertMACDConfig(sa.config.MACD), convertTimeframe(tf)))
		}

		// Add Volume (if enabled)
		if sa.config.Volume.Enabled {
			indicators = append(indicators, indicator.NewVolume(convertVolumeConfig(sa.config.Volume), convertTimeframe(tf)))
		}

		// Add Trend (if enabled)
		if sa.config.Trend.Enabled {
			indicators = append(indicators, indicator.NewTrend(convertTrendConfig(sa.config.Trend), convertTimeframe(tf)))
		}

		// Add Support/Resistance (if enabled)
		if sa.config.SupportResistance.Enabled {
			indicators = append(indicators, indicator.NewSupportResistance(convertSupportResistanceConfig(sa.config.SupportResistance), convertTimeframe(tf)))
		}

		// Add Ichimoku (if enabled)
		if sa.config.Ichimoku.Enabled {
			indicators = append(indicators, indicator.NewIchimoku(convertIchimokuConfig(sa.config.Ichimoku), convertTimeframe(tf)))
		}

		// Add Reverse-MFI (if enabled)
		if sa.config.MFI.Enabled {
			indicators = append(indicators, indicator.NewReverseMFI(convertMFIConfig(sa.config.MFI), convertTimeframe(tf)))
		}

		// Add Bollinger Bands (if enabled)
		if sa.config.BollingerBands.Enabled {
			indicators = append(indicators, indicator.NewBollingerBands(convertBollingerBandsConfig(sa.config.BollingerBands), convertTimeframe(tf)))
		}

		// Add Stochastic (if enabled)
		if sa.config.Stochastic.Enabled {
			indicators = append(indicators, indicator.NewStochastic(convertStochasticConfig(sa.config.Stochastic), convertTimeframe(tf)))
		}

		// Add Williams %R (if enabled)
		if sa.config.WilliamsR.Enabled {
			indicators = append(indicators, indicator.NewWilliamsR(convertWilliamsRConfig(sa.config.WilliamsR), convertTimeframe(tf)))
		}

		// Add Pin Bar (if enabled)
		if sa.config.PinBar.Enabled {
			indicators = append(indicators, indicator.NewPinBar(convertPinBarConfig(sa.config.PinBar), convertTimeframe(tf)))
		}

		// Add EMA (if enabled)
		if sa.config.EMA.Enabled {
			indicators = append(indicators, indicator.NewEMA(convertEMAConfig(sa.config.EMA), convertTimeframe(tf)))
		}

		// Add Elliott Wave (if enabled)
		if sa.config.ElliottWave.Enabled {
			indicators = append(indicators, indicator.NewElliottWave(convertElliottWaveConfig(sa.config.ElliottWave), convertTimeframe(tf)))
		}

		// Add Channel Analysis (if enabled) - Works best on 5min and 15min timeframes
		if sa.config.ChannelAnalysis.Enabled && (tf == FiveMinute) {
			indicators = append(indicators, indicator.NewChannelAnalysis(convertChannelAnalysisConfig(sa.config.ChannelAnalysis), convertTimeframe(tf)))
		}

		// Add ATR (if enabled)
		if sa.config.ATR.Enabled {
			indicators = append(indicators, indicator.NewATR(convertATRConfig(sa.config.ATR), convertTimeframe(tf)))
		}

		sa.indicators[tf] = indicators
	}
}

// Helper functions to convert between bot and indicator package types
func convertTimeframe(tf Timeframe) indicator.Timeframe {
	switch tf {
	case FiveMinute:
		return indicator.FiveMinute
	case FifteenMinute:
		return indicator.FifteenMinute
	case FortyFiveMinute:
		return indicator.FortyFiveMinute
	case EightHour:
		return indicator.EightHour
	case Daily:
		return indicator.Daily
	default:
		return indicator.FiveMinute
	}
}

func convertRSIConfig(config RSIConfig) indicator.RSIConfig {
	return indicator.RSIConfig{
		Enabled:    config.Enabled,
		Period:     config.Period,
		Overbought: config.Overbought,
		Oversold:   config.Oversold,
	}
}

func convertMACDConfig(config MACDConfig) indicator.MACDConfig {
	return indicator.MACDConfig{
		Enabled:      config.Enabled,
		FastPeriod:   config.FastPeriod,
		SlowPeriod:   config.SlowPeriod,
		SignalPeriod: config.SignalPeriod,
	}
}

func convertVolumeConfig(config VolumeConfig) indicator.VolumeConfig {
	return indicator.VolumeConfig{
		Enabled:         config.Enabled,
		Period:          config.Period,
		VolumeThreshold: config.VolumeThreshold,
	}
}

func convertTrendConfig(config TrendConfig) indicator.TrendConfig {
	return indicator.TrendConfig{
		Enabled: config.Enabled,
		ShortMA: config.ShortMA,
		LongMA:  config.LongMA,
	}
}

func convertSupportResistanceConfig(config SupportResistanceConfig) indicator.SupportResistanceConfig {
	return indicator.SupportResistanceConfig{
		Enabled:   config.Enabled,
		Period:    config.Period,
		Threshold: config.Threshold,
	}
}

func convertIchimokuConfig(config IchimokuConfig) indicator.IchimokuConfig {
	return indicator.IchimokuConfig{
		Enabled:      config.Enabled,
		TenkanPeriod: config.TenkanPeriod,
		KijunPeriod:  config.KijunPeriod,
		SenkouPeriod: config.SenkouPeriod,
		Displacement: config.Displacement,
	}
}

func convertMFIConfig(config MFIConfig) indicator.MFIConfig {
	return indicator.MFIConfig{
		Enabled:    config.Enabled,
		Period:     config.Period,
		Overbought: config.Overbought,
		Oversold:   config.Oversold,
	}
}

func convertBollingerBandsConfig(config BollingerBandsConfig) indicator.BollingerBandsConfig {
	return indicator.BollingerBandsConfig{
		Enabled:       config.Enabled,
		Period:        config.Period,
		StandardDev:   config.StandardDev,
		OverboughtStd: config.OverboughtStd,
		OversoldStd:   config.OversoldStd,
	}
}

func convertStochasticConfig(config StochasticConfig) indicator.StochasticConfig {
	return indicator.StochasticConfig{
		Enabled:         config.Enabled,
		KPeriod:         config.KPeriod,
		DPeriod:         config.DPeriod,
		SlowPeriod:      config.SlowPeriod,
		Overbought:      config.Overbought,
		Oversold:        config.Oversold,
		MomentumBoost:   config.MomentumBoost,
		DivergenceBoost: config.DivergenceBoost,
	}
}

func convertWilliamsRConfig(config WilliamsRConfig) indicator.WilliamsRConfig {
	return indicator.WilliamsRConfig{
		Enabled:       config.Enabled,
		Period:        config.Period,
		Overbought:    config.Overbought,
		Oversold:      config.Oversold,
		FastResponse:  config.FastResponse,
		MomentumBoost: config.MomentumBoost,
		ReversalBoost: config.ReversalBoost,
	}
}

func convertPinBarConfig(config PinBarConfig) indicator.PinBarConfig {
	return indicator.PinBarConfig{
		Enabled:              config.Enabled,
		MinWickRatio:         config.MinWickRatio,
		MaxBodyRatio:         config.MaxBodyRatio,
		MinRangePercent:      config.MinRangePercent,
		SupportResistance:    config.SupportResistance,
		TrendConfirmation:    config.TrendConfirmation,
		PatternStrengthBoost: config.PatternStrengthBoost,
	}
}

func convertCandles(candles []Candle) []indicator.Candle {
	indicatorCandles := make([]indicator.Candle, len(candles))
	for i, candle := range candles {
		indicatorCandles[i] = indicator.Candle{
			Timestamp: candle.Timestamp,
			Open:      candle.Open,
			High:      candle.High,
			Low:       candle.Low,
			Close:     candle.Close,
			Volume:    candle.Volume,
		}
	}
	return indicatorCandles
}

func convertIndicatorSignal(signal indicator.IndicatorSignal) IndicatorSignal {
	return IndicatorSignal{
		Name:      signal.Name,
		Signal:    SignalType(signal.Signal),
		Strength:  signal.Strength,
		Value:     signal.Value,
		Timestamp: signal.Timestamp,
		Timeframe: convertIndicatorTimeframe(signal.Timeframe),
	}
}

// convertChannelAnalysisConfig converts bot config to indicator config
func convertChannelAnalysisConfig(config ChannelAnalysisConfig) indicator.ChannelAnalysisConfig {
	return indicator.ChannelAnalysisConfig{
		Enabled:          config.Enabled,
		LookbackPeriod:   config.LookbackPeriod,
		ChannelThreshold: config.ChannelThreshold,
		SignalBoost:      config.SignalBoost,
	}
}

// convertATRConfig converts bot config to indicator config
func convertATRConfig(config ATRConfig) indicator.ATRConfig {
	return indicator.ATRConfig{
		Enabled:    config.Enabled,
		Period:     config.Period,
		Multiplier: config.Multiplier,
		UseShorts:  config.UseShorts,
	}
}

func convertIndicatorTimeframe(tf indicator.Timeframe) Timeframe {
	switch tf {
	case indicator.FiveMinute:
		return FiveMinute
	case indicator.FifteenMinute:
		return FifteenMinute
	case indicator.FortyFiveMinute:
		return FortyFiveMinute
	case indicator.EightHour:
		return EightHour
	case indicator.Daily:
		return Daily
	default:
		return FiveMinute
	}
}

func convertEMAConfig(config EMAConfig) indicator.EMAConfig {
	return indicator.EMAConfig{
		Enabled:        config.Enabled,
		FastPeriod:     config.FastPeriod,
		SlowPeriod:     config.SlowPeriod,
		SignalPeriod:   config.SignalPeriod,
		TrendPeriod:    config.TrendPeriod,
		SlopeThreshold: config.SlopeThreshold,
		CrossoverBoost: config.CrossoverBoost,
		TrendBoost:     config.TrendBoost,
		VolumeConfirm:  config.VolumeConfirm,
	}
}

func convertElliottWaveConfig(config ElliottWaveConfig) indicator.ElliottWaveConfig {
	return indicator.ElliottWaveConfig{
		Enabled:            config.Enabled,
		MinWaveLength:      config.MinWaveLength,
		FibonacciTolerance: config.FibonacciTolerance,
		TrendStrength:      config.TrendStrength,
		ImpulseBoost:       config.ImpulseBoost,
		CorrectionBoost:    config.CorrectionBoost,
		CompletionBoost:    config.CompletionBoost,
		MaxLookback:        config.MaxLookback,
	}
}

// GenerateSignal creates a trading signal from multi-timeframe analysis
func (sa *SignalAggregator) GenerateSignal(ctx *MultiTimeframeContext) (*TradingSignal, error) {
	if ctx == nil {
		return nil, fmt.Errorf("context is nil")
	}

	currentPrice := ctx.GetCurrentPrice()
	if currentPrice == 0 {
		return nil, fmt.Errorf("invalid current price")
	}

	// FOCUSED: Only get 5-minute signals for ultra-fast response
	fiveMinSignals := sa.getTimeframeSignals(ctx.FiveMinCandles, FiveMinute, currentPrice)

	// Apply focused 5-minute logic
	finalSignal := sa.applyFocused5MinuteLogic(fiveMinSignals, currentPrice)

	return &TradingSignal{
		Symbol:           ctx.Symbol,
		Signal:           finalSignal.Signal,
		Confidence:       finalSignal.Confidence,
		Timestamp:        time.Now(),
		IndicatorSignals: fiveMinSignals,
		Reasoning:        finalSignal.Reasoning,
		TargetPrice:      finalSignal.TargetPrice,
		StopLoss:         finalSignal.StopLoss,
	}, nil
}

// getTimeframeSignals calculates signals for a specific timeframe
func (sa *SignalAggregator) getTimeframeSignals(candles []Candle, timeframe Timeframe, currentPrice float64) []IndicatorSignal {
	var signals []IndicatorSignal

	indicators := sa.indicators[timeframe]

	for _, ind := range indicators {
		var signal indicator.IndicatorSignal

		// Enhanced 5-minute Ichimoku signal processing
		if timeframe == FiveMinute && strings.Contains(ind.GetName(), "Ichimoku") {
			// Use enhanced 5-minute signal for Ichimoku on 5-minute timeframe
			if ichimokuIndicator, ok := ind.(*indicator.Ichimoku); ok {
				signal = ichimokuIndicator.GetEnhanced5MinuteSignal(convertCandles(candles), currentPrice)
			} else {
				// Fallback to standard calculation
				values := ind.Calculate(convertCandles(candles))
				signal = ind.GetSignal(values, currentPrice)
			}
		} else if timeframe == FiveMinute && strings.Contains(ind.GetName(), "BollingerBands") {
			// Use enhanced 5-minute signal for Bollinger Bands on 5-minute timeframe
			if bollingerIndicator, ok := ind.(*indicator.BollingerBands); ok {
				signal = bollingerIndicator.GetEnhanced5MinuteSignal(convertCandles(candles), currentPrice)
			} else {
				// Fallback to standard calculation
				values := ind.Calculate(convertCandles(candles))
				signal = ind.GetSignal(values, currentPrice)
			}
		} else {
			// Standard signal calculation for all other cases
			values := ind.Calculate(convertCandles(candles))
			signal = ind.GetSignal(values, currentPrice)
		}

		signals = append(signals, convertIndicatorSignal(signal))
	}

	return signals
}

// MultiTimeframeResult holds the final trading decision
type MultiTimeframeResult struct {
	Signal      SignalType
	Confidence  float64
	Reasoning   string
	TargetPrice float64
	StopLoss    float64
}

// applyMultiTimeframeLogic combines signals using multi-timeframe analysis
func (sa *SignalAggregator) applyMultiTimeframeLogic(dailySignals, eightHourSignals, fortyFiveMinSignals, fifteenMinSignals, fiveMinSignals []IndicatorSignal, currentPrice float64) MultiTimeframeResult {
	// Rebalanced weights prioritizing 5-minute timeframe for short-term predictions
	// while maintaining higher timeframe context
	dailyContext := sa.analyzeTimeframeContext(dailySignals, 0.25)               // 25% weight for daily (reduced from 35%)
	eightHourContext := sa.analyzeTimeframeContext(eightHourSignals, 0.20)       // 20% weight for 8H (reduced from 25%)
	fortyFiveMinContext := sa.analyzeTimeframeContext(fortyFiveMinSignals, 0.20) // 20% weight for 45min (same)
	fifteenMinContext := sa.analyzeTimeframeContext(fifteenMinSignals, 0.20)     // 20% weight for 15min (increased from 15%)
	fiveMinContext := sa.analyzeTimeframeContext(fiveMinSignals, 0.15)           // 15% weight for 5min (increased from 5%) - MOST IMPORTANT for accuracy

	// Higher timeframe bias (Daily + 8H)
	higherTimeframeBias := sa.calculateTimeframeBias(dailyContext, eightHourContext)

	// Final signal logic with 5-timeframe analysis
	var finalSignal SignalType
	var confidence float64
	var reasoning strings.Builder

	// Enhanced multi-timeframe confirmation logic
	totalBullishSignals := 0
	totalBearishSignals := 0
	totalConfidence := 0.0

	// Count signals from all timeframes
	contexts := []TimeframeContext{dailyContext, eightHourContext, fortyFiveMinContext, fifteenMinContext, fiveMinContext}
	for _, ctx := range contexts {
		totalConfidence += ctx.Confidence
		if ctx.Signal == Buy {
			totalBullishSignals++
		} else if ctx.Signal == Sell {
			totalBearishSignals++
		}
	}

	// Determine overall signal based on majority and higher timeframe bias
	if totalBullishSignals > totalBearishSignals && higherTimeframeBias.Signal == Buy {
		// Strong bullish confluence
		finalSignal = Buy
		confidence = math.Min(1.0, totalConfidence/5.0*1.2) // Boost for alignment
		reasoning.WriteString("BULLISH: Multi-timeframe bullish confluence")
	} else if totalBearishSignals > totalBullishSignals && higherTimeframeBias.Signal == Sell {
		// Strong bearish confluence
		finalSignal = Sell
		confidence = math.Min(1.0, totalConfidence/5.0*1.2) // Boost for alignment
		reasoning.WriteString("BEARISH: Multi-timeframe bearish confluence")
	} else if totalBullishSignals > totalBearishSignals {
		// Bullish majority but higher timeframes neutral/bearish
		finalSignal = Buy
		confidence = math.Min(1.0, totalConfidence/5.0*0.8) // Reduce for conflict
		reasoning.WriteString("CAUTIOUS BULLISH: Lower timeframes bullish")
	} else if totalBearishSignals > totalBullishSignals {
		// Bearish majority but higher timeframes neutral/bullish
		finalSignal = Sell
		confidence = math.Min(1.0, totalConfidence/5.0*0.8) // Reduce for conflict
		reasoning.WriteString("CAUTIOUS BEARISH: Lower timeframes bearish")
	} else {
		// No clear majority or conflicting signals
		finalSignal = Hold
		confidence = 0.3
		reasoning.WriteString("HOLD: Mixed signals across timeframes")
	}

	// Apply minimum confidence threshold
	if confidence < sa.config.MinConfidence {
		finalSignal = Hold
		confidence = 0.2
		reasoning.WriteString(" - Below minimum confidence threshold")
	}

	// Calculate target price and stop loss using higher timeframes
	targetPrice, stopLoss := sa.calculateTargetAndStopLoss(finalSignal, currentPrice, dailySignals, eightHourSignals, fortyFiveMinSignals)

	return MultiTimeframeResult{
		Signal:      finalSignal,
		Confidence:  confidence,
		Reasoning:   reasoning.String(),
		TargetPrice: targetPrice,
		StopLoss:    stopLoss,
	}
}

// applyFocused5MinuteLogic applies focused 5-minute trading logic for ultra-fast response
func (sa *SignalAggregator) applyFocused5MinuteLogic(fiveMinSignals []IndicatorSignal, currentPrice float64) MultiTimeframeResult {
	buyCount := 0
	sellCount := 0
	holdCount := 0
	totalStrength := 0.0

	// Analyze 5-minute signals with focused weighting
	for _, signal := range fiveMinSignals {
		totalStrength += signal.Strength
		switch signal.Signal {
		case Buy:
			buyCount++
		case Sell:
			sellCount++
		case Hold:
			holdCount++
		}
	}

	// Calculate focused confidence
	avgStrength := totalStrength / float64(len(fiveMinSignals))
	var confidence float64
	var finalSignal SignalType
	var reasoning string

	// Determine signal based on 5-minute consensus
	if buyCount > sellCount {
		finalSignal = Buy
		confidence = math.Min(0.95, 0.75+(avgStrength*0.2)) // High base confidence
		reasoning = fmt.Sprintf("5-minute BULLISH consensus: %d buy vs %d sell signals (avg strength: %.1f%%)",
			buyCount, sellCount, avgStrength*100)
	} else if sellCount > buyCount {
		finalSignal = Sell
		confidence = math.Min(0.95, 0.75+(avgStrength*0.2)) // High base confidence
		reasoning = fmt.Sprintf("5-minute BEARISH consensus: %d sell vs %d buy signals (avg strength: %.1f%%)",
			sellCount, buyCount, avgStrength*100)
	} else {
		finalSignal = Hold
		confidence = math.Min(0.9, 0.7+(avgStrength*0.15)) // Strong confidence for consolidation
		reasoning = fmt.Sprintf("5-minute CONSOLIDATION: Balanced signals with %.1f%% average strength",
			avgStrength*100)
	}

	// Calculate target price based on 5-minute momentum
	var targetPrice, stopLoss float64
	priceChange := currentPrice * 0.001 * float64(buyCount-sellCount) // 0.1% per signal difference

	if finalSignal == Buy {
		targetPrice = currentPrice + math.Abs(priceChange)
		stopLoss = currentPrice - (math.Abs(priceChange) * 0.5)
	} else if finalSignal == Sell {
		targetPrice = currentPrice - math.Abs(priceChange)
		stopLoss = currentPrice + (math.Abs(priceChange) * 0.5)
	}

	return MultiTimeframeResult{
		Signal:      finalSignal,
		Confidence:  confidence,
		Reasoning:   reasoning,
		TargetPrice: targetPrice,
		StopLoss:    stopLoss,
	}
}

// TimeframeContext represents the overall signal context for a timeframe
type TimeframeContext struct {
	Signal      SignalType
	Confidence  float64
	BuyCount    int
	SellCount   int
	HoldCount   int
	AvgStrength float64
}

// getIndicatorWeight returns the performance-based weight for each indicator
func (sa *SignalAggregator) getIndicatorWeight(indicatorName string) float64 {
	switch {
	// TIER 1: Elite performers (>80% accuracy) - HIGHEST WEIGHTS
	case strings.Contains(indicatorName, "ElliottWave"):
		return 10.0 // Best performer - correctly predicted drops
	case strings.Contains(indicatorName, "Volume"):
		return 8.7 // 87.1% accuracy - excellent momentum confirmation
	case strings.Contains(indicatorName, "Trend"):
		return 8.4 // 83.9% accuracy - reliable trend detection

	// TIER 2: Good performers (60-80% accuracy) - HIGH WEIGHTS
	case strings.Contains(indicatorName, "MACD"):
		return 8.1 // 80.6% accuracy - solid trend following
	case strings.Contains(indicatorName, "EMA"):
		return 6.0 // New indicator - moderate weight until proven
	case strings.Contains(indicatorName, "ReverseMFI"):
		return 6.1 // 61.3% accuracy - moderate performance

	// TIER 3: Moderate performers (40-60% accuracy) - MEDIUM WEIGHTS
	case strings.Contains(indicatorName, "RSI"):
		return 4.2 // 41.9% accuracy - improved with new parameters
	case strings.Contains(indicatorName, "BollingerBands"):
		return 4.5 // Moderate performance with optimized parameters
	case strings.Contains(indicatorName, "PinBar"):
		return 3.5 // Pattern recognition - conservative weight

	// TIER 4: Momentum oscillators (improved parameters) - LOW-MEDIUM WEIGHTS
	case strings.Contains(indicatorName, "Stochastic"):
		return 2.9 // 29% accuracy - low weight despite improvements
	case strings.Contains(indicatorName, "Williams"):
		return 2.9 // Similar to Stochastic - low weight

	// TIER 5: Poor performers - MINIMAL WEIGHTS (but not zero to allow for rare good signals)
	case strings.Contains(indicatorName, "Ichimoku"):
		return 1.3 // 12.9% accuracy - minimal weight
	case strings.Contains(indicatorName, "S&R"):
		return 1.0 // 9.7% accuracy - lowest weight
	case strings.Contains(indicatorName, "ATR"):
		return 2.0 // 20% accuracy - moderate performance

	default:
		return 3.0 // Default moderate weight for unknown indicators
	}
}

// analyzeTimeframeContext analyzes signals with performance-based weighted scoring
func (sa *SignalAggregator) analyzeTimeframeContext(signals []IndicatorSignal, timeframeWeight float64) TimeframeContext {
	if len(signals) == 0 {
		return TimeframeContext{Signal: Hold, Confidence: 0}
	}

	var buyCount, sellCount, holdCount int
	var buyWeightedScore, sellWeightedScore, holdWeightedScore float64
	var totalStrength, totalWeight float64

	// Calculate weighted scores based on indicator performance
	for _, signal := range signals {
		indicatorWeight := sa.getIndicatorWeight(signal.Name)
		weightedStrength := signal.Strength * indicatorWeight

		switch signal.Signal {
		case Buy:
			buyCount++
			buyWeightedScore += weightedStrength
		case Sell:
			sellCount++
			sellWeightedScore += weightedStrength
		case Hold:
			holdCount++
			holdWeightedScore += weightedStrength
		}

		totalStrength += signal.Strength
		totalWeight += indicatorWeight
	}

	avgStrength := totalStrength / float64(len(signals))
	totalWeightedScore := buyWeightedScore + sellWeightedScore + holdWeightedScore

	var dominantSignal SignalType
	var confidence float64

	if totalWeightedScore == 0 {
		return TimeframeContext{Signal: Hold, Confidence: 0.1 * timeframeWeight}
	}

	// Determine dominant signal based on WEIGHTED SCORES, not just counts
	buyPercentage := buyWeightedScore / totalWeightedScore
	sellPercentage := sellWeightedScore / totalWeightedScore
	holdPercentage := holdWeightedScore / totalWeightedScore

	// Use weighted scores for decision (this heavily favors high-performing indicators)
	if buyPercentage > sellPercentage && buyPercentage > holdPercentage {
		dominantSignal = Buy
		confidence = buyPercentage * avgStrength * timeframeWeight

		// Boost confidence if elite indicators agree
		if buyWeightedScore > 15.0 { // Elliott Wave (10.0) + Volume (8.7) would exceed this
			confidence *= 1.3 // Boost for elite indicator consensus
		}

	} else if sellPercentage > buyPercentage && sellPercentage > holdPercentage {
		dominantSignal = Sell
		confidence = sellPercentage * avgStrength * timeframeWeight

		// Boost confidence if elite indicators agree
		if sellWeightedScore > 15.0 {
			confidence *= 1.3 // Boost for elite indicator consensus
		}

	} else {
		dominantSignal = Hold
		confidence = holdPercentage * avgStrength * timeframeWeight

		// Reduce confidence for HOLD when it's due to poor indicators
		if holdWeightedScore < 5.0 { // Only poor indicators giving HOLD
			confidence *= 0.5
		}
	}

	// Cap confidence at reasonable levels
	confidence = math.Min(1.0, confidence)

	return TimeframeContext{
		Signal:      dominantSignal,
		Confidence:  confidence,
		BuyCount:    buyCount,
		SellCount:   sellCount,
		HoldCount:   holdCount,
		AvgStrength: avgStrength,
	}
}

// calculateTimeframeBias combines daily and 8H signals for higher timeframe bias
func (sa *SignalAggregator) calculateTimeframeBias(dailyCtx, eightHourCtx TimeframeContext) TimeframeContext {
	// Daily has higher weight than 8H
	dailyWeight := 0.6
	eightHourWeight := 0.4

	var bias SignalType
	var confidence float64

	// If both timeframes agree
	if dailyCtx.Signal == eightHourCtx.Signal {
		bias = dailyCtx.Signal
		confidence = (dailyCtx.Confidence * dailyWeight) + (eightHourCtx.Confidence * eightHourWeight)
	} else {
		// Conflicting signals - use daily with reduced confidence
		bias = dailyCtx.Signal
		confidence = dailyCtx.Confidence * dailyWeight * 0.7 // Reduce for conflict
	}

	return TimeframeContext{
		Signal:     bias,
		Confidence: confidence,
	}
}

// calculateTargetAndStopLoss calculates target price and stop loss based on signals
func (sa *SignalAggregator) calculateTargetAndStopLoss(signal SignalType, currentPrice float64, dailySignals, eightHourSignals, fortyFiveMinSignals []IndicatorSignal) (float64, float64) {
	var targetPrice, stopLoss float64

	// Find support/resistance levels from higher timeframes
	var supportLevel, resistanceLevel float64

	// Get S&R levels from daily, 8H, and 45min timeframes
	allSignals := append(dailySignals, eightHourSignals...)
	allSignals = append(allSignals, fortyFiveMinSignals...)

	for _, sig := range allSignals {
		if strings.Contains(sig.Name, "S&R") {
			if supportLevel == 0 || sig.Value < supportLevel {
				supportLevel = sig.Value
			}
			if resistanceLevel == 0 || sig.Value > resistanceLevel {
				resistanceLevel = sig.Value
			}
		}
	}

	switch signal {
	case Buy:
		// Target: Next resistance level or 2% above current price
		if resistanceLevel > currentPrice {
			targetPrice = resistanceLevel
		} else {
			targetPrice = currentPrice * 1.02 // 2% target
		}

		// Stop loss: Support level or 1% below current price
		if supportLevel > 0 && supportLevel < currentPrice {
			stopLoss = supportLevel
		} else {
			stopLoss = currentPrice * 0.99 // 1% stop loss
		}

	case Sell:
		// Target: Next support level or 2% below current price
		if supportLevel > 0 && supportLevel < currentPrice {
			targetPrice = supportLevel
		} else {
			targetPrice = currentPrice * 0.98 // 2% target
		}

		// Stop loss: Resistance level or 1% above current price
		if resistanceLevel > currentPrice {
			stopLoss = resistanceLevel
		} else {
			stopLoss = currentPrice * 1.01 // 1% stop loss
		}
	}

	return targetPrice, stopLoss
}
