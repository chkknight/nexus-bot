package indicator

import (
	"fmt"
	"math"
	"time"
)

// Ichimoku Cloud Indicator with 5-minute optimization
type Ichimoku struct {
	config    IchimokuConfig
	timeframe Timeframe
}

// NewIchimoku creates a new Ichimoku indicator
func NewIchimoku(config IchimokuConfig, timeframe Timeframe) *Ichimoku {
	return &Ichimoku{
		config:    config,
		timeframe: timeframe,
	}
}

// get5MinuteOptimizedConfig returns optimized parameters for 5-minute trading
func (ich *Ichimoku) get5MinuteOptimizedConfig() IchimokuConfig {
	if ich.timeframe == FiveMinute {
		// Optimized for 5-minute trading - shorter, more responsive periods
		return IchimokuConfig{
			Enabled:      ich.config.Enabled,
			TenkanPeriod: 6,  // Reduced from 9 for faster response
			KijunPeriod:  18, // Reduced from 26 for better short-term signals
			SenkouPeriod: 36, // Reduced from 52 for 5-minute relevance
			Displacement: 18, // Reduced from 26 for shorter-term analysis
		}
	}
	return ich.config
}

// Calculate computes all Ichimoku Cloud components with 5-minute optimization
func (ich *Ichimoku) Calculate(candles []Candle) []float64 {
	optimizedConfig := ich.get5MinuteOptimizedConfig()

	if len(candles) < optimizedConfig.SenkouPeriod {
		return []float64{}
	}

	values := ich.calculateAllWithConfig(candles, optimizedConfig)

	// Enhanced 5-minute signal calculation
	cloudSignals := make([]float64, len(values.CloudTop))
	for i := 0; i < len(cloudSignals); i++ {
		if i >= len(candles) {
			continue
		}

		currentPrice := candles[i].Close
		cloudTop := values.CloudTop[i]
		cloudBottom := values.CloudBottom[i]

		// 5-minute specific signal calculation with gradual strength
		if ich.timeframe == FiveMinute {
			cloudSignals[i] = ich.calculate5MinuteSignal(currentPrice, cloudTop, cloudBottom, values, i)
		} else {
			// Original logic for other timeframes
			if currentPrice > cloudTop {
				cloudSignals[i] = 1.0
			} else if currentPrice < cloudBottom {
				cloudSignals[i] = -1.0
			} else {
				cloudSignals[i] = 0.0
			}
		}
	}

	return cloudSignals
}

// calculate5MinuteSignal provides enhanced signal calculation for 5-minute trading
func (ich *Ichimoku) calculate5MinuteSignal(currentPrice, cloudTop, cloudBottom float64, values IchimokuValues, index int) float64 {
	cloudThickness := cloudTop - cloudBottom

	// Enhanced signal strength based on multiple factors
	var signal float64

	if currentPrice > cloudTop {
		// Price above cloud - bullish signal
		if cloudThickness > 0 {
			// Distance above cloud affects strength
			distanceAbove := (currentPrice - cloudTop) / cloudThickness
			signal = 0.6 + math.Min(0.4, distanceAbove*0.2) // 0.6 to 1.0 range
		} else {
			signal = 0.8 // Strong bullish when cloud is thin
		}

		// Enhance with Tenkan-Kijun analysis for 5-minute
		if index < len(values.TenkanSen) && index < len(values.KijunSen) {
			tenkan := values.TenkanSen[index]
			kijun := values.KijunSen[index]

			if tenkan > kijun {
				signal = math.Min(1.0, signal+0.1) // Boost for aligned signals
			}
		}

	} else if currentPrice < cloudBottom {
		// Price below cloud - bearish signal
		if cloudThickness > 0 {
			// Distance below cloud affects strength
			distanceBelow := (cloudBottom - currentPrice) / cloudThickness
			signal = -0.6 - math.Min(0.4, distanceBelow*0.2) // -0.6 to -1.0 range
		} else {
			signal = -0.8 // Strong bearish when cloud is thin
		}

		// Enhance with Tenkan-Kijun analysis for 5-minute
		if index < len(values.TenkanSen) && index < len(values.KijunSen) {
			tenkan := values.TenkanSen[index]
			kijun := values.KijunSen[index]

			if tenkan < kijun {
				signal = math.Max(-1.0, signal-0.1) // Boost for aligned signals
			}
		}

	} else {
		// Price inside cloud - neutral with position-based strength
		if cloudThickness > 0 {
			position := (currentPrice - cloudBottom) / cloudThickness
			// Scale from -0.3 to 0.3 based on position within cloud
			signal = (position - 0.5) * 0.6
		} else {
			signal = 0.0
		}
	}

	return signal
}

// calculateAllWithConfig computes all Ichimoku components using provided config
func (ich *Ichimoku) calculateAllWithConfig(candles []Candle, config IchimokuConfig) IchimokuValues {
	if len(candles) < config.SenkouPeriod {
		return IchimokuValues{}
	}

	// Calculate Tenkan-sen (Conversion Line)
	tenkanSen := ich.calculateHighLowAverageWithPeriod(candles, config.TenkanPeriod)

	// Calculate Kijun-sen (Base Line)
	kijunSen := ich.calculateHighLowAverageWithPeriod(candles, config.KijunPeriod)

	// Calculate Senkou Span A (Leading Span A)
	senkouSpanA := ich.calculateSenkouSpanA(tenkanSen, kijunSen)

	// Calculate Senkou Span B (Leading Span B)
	senkouSpanB := ich.calculateHighLowAverageWithPeriod(candles, config.SenkouPeriod)

	// Calculate Chikou Span (Lagging Span)
	chikouSpan := ich.calculateChikouSpanWithDisplacement(candles, config.Displacement)

	// Calculate cloud boundaries
	cloudTop, cloudBottom := ich.calculateCloudBoundaries(senkouSpanA, senkouSpanB)

	return IchimokuValues{
		TenkanSen:   tenkanSen,
		KijunSen:    kijunSen,
		SenkouSpanA: senkouSpanA,
		SenkouSpanB: senkouSpanB,
		ChikouSpan:  chikouSpan,
		CloudTop:    cloudTop,
		CloudBottom: cloudBottom,
	}
}

// calculateHighLowAverageWithPeriod calculates (highest high + lowest low) / 2 for given period
func (ich *Ichimoku) calculateHighLowAverageWithPeriod(candles []Candle, period int) []float64 {
	if len(candles) < period {
		return []float64{}
	}

	values := make([]float64, len(candles)-period+1)

	for i := 0; i < len(values); i++ {
		high := candles[i].High
		low := candles[i].Low

		// Find highest high and lowest low in the period
		for j := 1; j < period; j++ {
			if candles[i+j].High > high {
				high = candles[i+j].High
			}
			if candles[i+j].Low < low {
				low = candles[i+j].Low
			}
		}

		values[i] = (high + low) / 2
	}

	return values
}

// calculateChikouSpanWithDisplacement calculates the Lagging Span with custom displacement
func (ich *Ichimoku) calculateChikouSpanWithDisplacement(candles []Candle, displacement int) []float64 {
	if len(candles) < displacement {
		return []float64{}
	}

	values := make([]float64, len(candles)-displacement)

	for i := 0; i < len(values); i++ {
		// Chikou Span = Close price displaced backward
		values[i] = candles[i+displacement].Close
	}

	return values
}

// calculateSenkouSpanA calculates Leading Span A
func (ich *Ichimoku) calculateSenkouSpanA(tenkanSen, kijunSen []float64) []float64 {
	minLen := len(tenkanSen)
	if len(kijunSen) < minLen {
		minLen = len(kijunSen)
	}

	if minLen == 0 {
		return []float64{}
	}

	values := make([]float64, minLen)

	for i := 0; i < minLen; i++ {
		// Senkou Span A = (Tenkan-sen + Kijun-sen) / 2
		values[i] = (tenkanSen[i] + kijunSen[i]) / 2
	}

	return values
}

// calculateCloudBoundaries determines the upper and lower boundaries of the cloud
func (ich *Ichimoku) calculateCloudBoundaries(senkouSpanA, senkouSpanB []float64) ([]float64, []float64) {
	minLen := len(senkouSpanA)
	if len(senkouSpanB) < minLen {
		minLen = len(senkouSpanB)
	}

	if minLen == 0 {
		return []float64{}, []float64{}
	}

	cloudTop := make([]float64, minLen)
	cloudBottom := make([]float64, minLen)

	for i := 0; i < minLen; i++ {
		if senkouSpanA[i] > senkouSpanB[i] {
			cloudTop[i] = senkouSpanA[i]
			cloudBottom[i] = senkouSpanB[i]
		} else {
			cloudTop[i] = senkouSpanB[i]
			cloudBottom[i] = senkouSpanA[i]
		}
	}

	return cloudTop, cloudBottom
}

// calculateNuancedStrength calculates strength based on multiple Ichimoku components
func (ich *Ichimoku) calculateNuancedStrength(currentPrice float64, cloudSignal float64) float64 {
	// Base strength from cloud position
	var baseStrength float64

	if math.Abs(cloudSignal) > 0.5 {
		// Price is clearly above/below cloud
		baseStrength = 0.4
	} else {
		// Price is inside cloud - weak signal
		baseStrength = 0.2
	}

	// Add strength factors based on timeframe
	switch ich.timeframe {
	case FiveMinute:
		// 5-minute signals are less reliable, reduce strength
		baseStrength *= 0.7
	case FifteenMinute:
		// 15-minute signals are more reliable
		baseStrength *= 0.9
	case FortyFiveMinute:
		// 45-minute signals are quite reliable
		baseStrength *= 1.0
	case EightHour:
		// 8-hour signals are very reliable
		baseStrength *= 1.1
	case Daily:
		// Daily signals are most reliable
		baseStrength *= 1.2
	}

	// Ensure strength is within bounds
	strength := math.Min(0.85, math.Max(0.1, baseStrength)) // FIXED: Cap at 0.85 instead of 1.0

	return strength
}

// CalculateAll computes all Ichimoku components and returns detailed values
func (ich *Ichimoku) CalculateAll(candles []Candle) IchimokuValues {
	optimizedConfig := ich.get5MinuteOptimizedConfig()
	return ich.calculateAllWithConfig(candles, optimizedConfig)
}

// calculateNuanced5MinuteStrength calculates enhanced strength for 5-minute trading
func (ich *Ichimoku) calculateNuanced5MinuteStrength(currentPrice float64, cloudSignal float64, values IchimokuValues) float64 {
	if ich.timeframe != FiveMinute {
		return ich.calculateNuancedStrength(currentPrice, cloudSignal)
	}

	// Enhanced 5-minute strength calculation
	baseStrength := math.Abs(cloudSignal)

	// 5-minute specific adjustments
	if baseStrength > 0.8 {
		// Strong signals - good for 5-minute trading
		baseStrength = 0.7 + (baseStrength-0.8)*1.5 // Scale 0.8-1.0 to 0.7-1.0
	} else if baseStrength > 0.6 {
		// Moderate signals - enhanced for 5-minute
		baseStrength = 0.5 + (baseStrength-0.6)*1.0 // Scale 0.6-0.8 to 0.5-0.7
	} else if baseStrength > 0.3 {
		// Weak signals - still usable for 5-minute
		baseStrength = 0.3 + (baseStrength-0.3)*0.7 // Scale 0.3-0.6 to 0.3-0.5
	} else {
		// Very weak signals - minimal strength
		baseStrength = baseStrength * 0.5 // Scale 0.0-0.3 to 0.0-0.15
	}

	// Additional validation with Tenkan-Kijun for 5-minute
	if len(values.TenkanSen) > 0 && len(values.KijunSen) > 0 {
		lastIdx := len(values.TenkanSen) - 1
		if lastIdx >= 0 && lastIdx < len(values.KijunSen) {
			tenkan := values.TenkanSen[lastIdx]
			kijun := values.KijunSen[lastIdx]

			// Boost strength when Tenkan-Kijun alignment matches cloud signal
			if (cloudSignal > 0 && tenkan > kijun) || (cloudSignal < 0 && tenkan < kijun) {
				baseStrength = math.Min(0.85, baseStrength*1.2) // FIXED: Cap to 0.85
			}
		}
	}

	return math.Min(0.85, math.Max(0.1, baseStrength)) // FIXED: Cap to 0.85
}

// GetSignal generates a trading signal with 5-minute optimization
func (ich *Ichimoku) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	if len(values) == 0 {
		return IndicatorSignal{
			Name:      ich.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: ich.timeframe,
		}
	}

	// Get the cloud signal
	currentSignal := values[len(values)-1]

	// Calculate enhanced strength for 5-minute trading
	var strength float64
	if ich.timeframe == FiveMinute {
		// For 5-minute, we need full candle data for enhanced calculation
		// This is a simplified version - full enhancement requires GetEnhanced5MinuteSignal
		strength = ich.calculateNuanced5MinuteStrength(currentPrice, currentSignal, IchimokuValues{})
	} else {
		strength = ich.calculateNuancedStrength(currentPrice, currentSignal)
	}

	var signal SignalType

	// Enhanced signal logic for 5-minute trading
	if ich.timeframe == FiveMinute {
		// More sensitive thresholds for 5-minute trading
		if currentSignal > 0.4 && strength > 0.25 {
			signal = Buy
		} else if currentSignal < -0.4 && strength > 0.25 {
			signal = Sell
		} else {
			signal = Hold
			strength = 0.3 // Default hold strength
		}
	} else {
		// Original logic for other timeframes
		if currentSignal > 0.5 && strength > 0.3 {
			signal = Buy
		} else if currentSignal < -0.5 && strength > 0.3 {
			signal = Sell
		} else {
			signal = Hold
			strength = 0.3
		}
	}

	return IndicatorSignal{
		Name:      ich.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     currentSignal,
		Timestamp: time.Now(),
		Timeframe: ich.timeframe,
	}
}

// GetEnhanced5MinuteSignal provides the most optimized signal for 5-minute trading
func (ich *Ichimoku) GetEnhanced5MinuteSignal(candles []Candle, currentPrice float64) IndicatorSignal {
	if ich.timeframe != FiveMinute {
		return ich.GetEnhancedSignal(candles, currentPrice)
	}

	optimizedConfig := ich.get5MinuteOptimizedConfig()

	if len(candles) < optimizedConfig.SenkouPeriod {
		return IndicatorSignal{
			Name:      ich.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: ich.timeframe,
		}
	}

	// Get all Ichimoku components with optimized parameters
	values := ich.calculateAllWithConfig(candles, optimizedConfig)

	// Calculate enhanced cloud signal
	cloudSignal := ich.getEnhanced5MinuteCloudSignal(candles, currentPrice, values)

	// Calculate sophisticated strength
	strength := ich.calculateNuanced5MinuteStrength(currentPrice, cloudSignal, values)

	var signal SignalType

	// 5-minute specific signal determination
	if cloudSignal > 0.3 && strength > 0.2 {
		signal = Buy
	} else if cloudSignal < -0.3 && strength > 0.2 {
		signal = Sell
	} else {
		signal = Hold
		strength = 0.25 // Slightly lower hold strength for 5-minute
	}

	return IndicatorSignal{
		Name:      ich.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     cloudSignal,
		Timestamp: time.Now(),
		Timeframe: ich.timeframe,
	}
}

// getEnhanced5MinuteCloudSignal calculates optimized cloud signal for 5-minute trading
func (ich *Ichimoku) getEnhanced5MinuteCloudSignal(candles []Candle, currentPrice float64, values IchimokuValues) float64 {
	if len(values.CloudTop) == 0 || len(values.CloudBottom) == 0 {
		return 0.0
	}

	lastIdx := len(values.CloudTop) - 1
	cloudTop := values.CloudTop[lastIdx]
	cloudBottom := values.CloudBottom[lastIdx]

	return ich.calculate5MinuteSignal(currentPrice, cloudTop, cloudBottom, values, lastIdx)
}

// GetEnhancedSignal provides the most nuanced Ichimoku analysis using full candle data
func (ich *Ichimoku) GetEnhancedSignal(candles []Candle, currentPrice float64) IndicatorSignal {
	if len(candles) < ich.config.SenkouPeriod {
		return IndicatorSignal{
			Name:      ich.GetName(),
			Signal:    Hold,
			Strength:  0,
			Value:     0,
			Timestamp: time.Now(),
			Timeframe: ich.timeframe,
		}
	}

	// Get detailed signal analysis
	detailedSignal := ich.GetDetailedSignal(candles, currentPrice)

	// Convert to standard IndicatorSignal format
	return IndicatorSignal{
		Name:      ich.GetName(),
		Signal:    detailedSignal.Signal,
		Strength:  detailedSignal.Strength,
		Value:     ich.getCloudSignalValue(candles, currentPrice),
		Timestamp: time.Now(),
		Timeframe: ich.timeframe,
	}
}

// getCloudSignalValue calculates the current cloud signal value
func (ich *Ichimoku) getCloudSignalValue(candles []Candle, currentPrice float64) float64 {
	values := ich.CalculateAll(candles)

	if len(values.CloudTop) == 0 || len(values.CloudBottom) == 0 {
		return 0.0
	}

	lastIdx := len(values.CloudTop) - 1
	cloudTop := values.CloudTop[lastIdx]
	cloudBottom := values.CloudBottom[lastIdx]

	if currentPrice > cloudTop {
		// Calculate relative distance above cloud
		cloudThickness := cloudTop - cloudBottom
		if cloudThickness > 0 {
			relativeDistance := (currentPrice - cloudTop) / cloudThickness
			// Cap at 1.0 but allow for gradual strength
			return math.Min(1.0, 0.5+relativeDistance*0.5)
		}
		return 1.0
	} else if currentPrice < cloudBottom {
		// Calculate relative distance below cloud
		cloudThickness := cloudTop - cloudBottom
		if cloudThickness > 0 {
			relativeDistance := (cloudBottom - currentPrice) / cloudThickness
			// Cap at -1.0 but allow for gradual strength
			return math.Max(-1.0, -0.5-relativeDistance*0.5)
		}
		return -1.0
	} else {
		// Price inside cloud - return position within cloud
		cloudThickness := cloudTop - cloudBottom
		if cloudThickness > 0 {
			position := (currentPrice - cloudBottom) / cloudThickness
			// Scale from -0.5 to 0.5 based on position within cloud
			return (position - 0.5) * 0.5
		}
		return 0.0
	}
}

// GetDetailedSignal provides comprehensive Ichimoku analysis
func (ich *Ichimoku) GetDetailedSignal(candles []Candle, currentPrice float64) IchimokuSignal {
	if len(candles) < ich.config.SenkouPeriod {
		return IchimokuSignal{Signal: Hold, Strength: 0}
	}

	values := ich.CalculateAll(candles)

	// Analyze current conditions
	lastIdx := len(values.TenkanSen) - 1
	if lastIdx < 0 {
		return IchimokuSignal{Signal: Hold, Strength: 0}
	}

	tenkan := values.TenkanSen[lastIdx]
	kijun := values.KijunSen[lastIdx]

	var signal SignalType
	var strength float64
	var reasoning []string

	// 1. Check Tenkan-Kijun cross
	if tenkan > kijun {
		reasoning = append(reasoning, "Tenkan above Kijun (bullish)")
		signal = Buy
		strength += 0.3
	} else if tenkan < kijun {
		reasoning = append(reasoning, "Tenkan below Kijun (bearish)")
		signal = Sell
		strength += 0.3
	}

	// 2. Check price vs cloud
	if lastIdx < len(values.CloudTop) {
		cloudTop := values.CloudTop[lastIdx]
		cloudBottom := values.CloudBottom[lastIdx]

		if currentPrice > cloudTop {
			reasoning = append(reasoning, "Price above cloud (bullish)")
			if signal == Buy {
				strength += 0.4
			} else {
				signal = Buy
				strength = 0.4
			}
		} else if currentPrice < cloudBottom {
			reasoning = append(reasoning, "Price below cloud (bearish)")
			if signal == Sell {
				strength += 0.4
			} else {
				signal = Sell
				strength = 0.4
			}
		} else {
			reasoning = append(reasoning, "Price inside cloud (neutral)")
			strength *= 0.5 // Reduce strength when in cloud
		}
	}

	// 3. Check cloud color (future cloud trend)
	if lastIdx < len(values.SenkouSpanA) && lastIdx < len(values.SenkouSpanB) {
		spanA := values.SenkouSpanA[lastIdx]
		spanB := values.SenkouSpanB[lastIdx]

		if spanA > spanB {
			reasoning = append(reasoning, "Green cloud (bullish trend)")
			if signal == Buy {
				strength += 0.2
			}
		} else if spanA < spanB {
			reasoning = append(reasoning, "Red cloud (bearish trend)")
			if signal == Sell {
				strength += 0.2
			}
		}
	}

	// Normalize strength
	if strength > 0.85 { // FIXED: Cap to 0.85 instead of 1.0
		strength = 0.85
	}

	return IchimokuSignal{
		Signal:    signal,
		Strength:  strength,
		Reasoning: reasoning,
		Values:    values,
	}
}

// GetName returns the indicator name
func (ich *Ichimoku) GetName() string {
	return fmt.Sprintf("Ichimoku_%s", ich.timeframe.String())
}
