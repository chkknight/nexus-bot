package indicator

import (
	"fmt"
	"math"
	"time"
)

// ElliottWaveConfig holds Elliott Wave configuration
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

// WaveType represents different Elliott Wave types
type WaveType int

const (
	NoWave WaveType = iota
	ImpulseWave
	CorrectionWave
	DiagonalWave
	ExtensionWave
)

func (w WaveType) String() string {
	switch w {
	case ImpulseWave:
		return "Impulse"
	case CorrectionWave:
		return "Correction"
	case DiagonalWave:
		return "Diagonal"
	case ExtensionWave:
		return "Extension"
	default:
		return "None"
	}
}

// WaveCount represents the current wave count
type WaveCount int

const (
	Wave1 WaveCount = iota + 1
	Wave2
	Wave3
	Wave4
	Wave5
	WaveA
	WaveB
	WaveC
	WaveUndefined
)

func (w WaveCount) String() string {
	switch w {
	case Wave1:
		return "1"
	case Wave2:
		return "2"
	case Wave3:
		return "3"
	case Wave4:
		return "4"
	case Wave5:
		return "5"
	case WaveA:
		return "A"
	case WaveB:
		return "B"
	case WaveC:
		return "C"
	default:
		return "?"
	}
}

// WavePattern represents an identified wave pattern
type WavePattern struct {
	Type       WaveType
	Count      WaveCount
	StartIndex int
	EndIndex   int
	StartPrice float64
	EndPrice   float64
	Length     float64
	FibRatio   float64
	Confidence float64
	IsComplete bool
	Target     float64
	StopLoss   float64
}

// ElliottWave represents the Elliott Wave indicator
type ElliottWave struct {
	config         ElliottWaveConfig
	timeframe      Timeframe
	candles        []Candle
	pivotHighs     []int
	pivotLows      []int
	waves          []WavePattern
	currentWave    WavePattern
	lastSignal     SignalType
	lastStrength   float64
	lastWaveType   WaveType
	lastWaveCount  WaveCount
	fibonacciLevel float64
	initialized    bool
}

// NewElliottWave creates a new Elliott Wave indicator
func NewElliottWave(config ElliottWaveConfig, timeframe Timeframe) *ElliottWave {
	return &ElliottWave{
		config:         config,
		timeframe:      timeframe,
		candles:        make([]Candle, 0),
		pivotHighs:     make([]int, 0),
		pivotLows:      make([]int, 0),
		waves:          make([]WavePattern, 0),
		currentWave:    WavePattern{Type: NoWave, Count: WaveUndefined},
		lastSignal:     Hold,
		lastStrength:   0.0,
		lastWaveType:   NoWave,
		lastWaveCount:  WaveUndefined,
		fibonacciLevel: 0.0,
		initialized:    false,
	}
}

// Update processes new candle data
func (ew *ElliottWave) Update(candle Candle) {
	ew.candles = append(ew.candles, candle)

	// Maintain buffer size
	if len(ew.candles) > ew.config.MaxLookback {
		ew.candles = ew.candles[1:]
		// Adjust pivot indices
		for i := range ew.pivotHighs {
			if ew.pivotHighs[i] > 0 {
				ew.pivotHighs[i]--
			}
		}
		for i := range ew.pivotLows {
			if ew.pivotLows[i] > 0 {
				ew.pivotLows[i]--
			}
		}
	}

	// Identify pivot points
	if len(ew.candles) >= ew.config.MinWaveLength*2 {
		ew.identifyPivots()
		ew.analyzeWaves()
		ew.initialized = true
	}
}

// identifyPivots identifies pivot highs and lows
func (ew *ElliottWave) identifyPivots() {
	if len(ew.candles) < ew.config.MinWaveLength*2+1 {
		return
	}

	lookback := ew.config.MinWaveLength
	currentIndex := len(ew.candles) - lookback - 1

	if currentIndex < lookback {
		return
	}

	current := ew.candles[currentIndex]

	// Check for pivot high
	isPivotHigh := true
	for i := currentIndex - lookback; i <= currentIndex+lookback; i++ {
		if i == currentIndex || i < 0 || i >= len(ew.candles) {
			continue
		}
		if ew.candles[i].High >= current.High {
			isPivotHigh = false
			break
		}
	}

	// Check for pivot low
	isPivotLow := true
	for i := currentIndex - lookback; i <= currentIndex+lookback; i++ {
		if i == currentIndex || i < 0 || i >= len(ew.candles) {
			continue
		}
		if ew.candles[i].Low <= current.Low {
			isPivotLow = false
			break
		}
	}

	// Add pivot points
	if isPivotHigh {
		ew.pivotHighs = append(ew.pivotHighs, currentIndex)
		// Keep only recent pivots
		if len(ew.pivotHighs) > 20 {
			ew.pivotHighs = ew.pivotHighs[1:]
		}
	}

	if isPivotLow {
		ew.pivotLows = append(ew.pivotLows, currentIndex)
		// Keep only recent pivots
		if len(ew.pivotLows) > 20 {
			ew.pivotLows = ew.pivotLows[1:]
		}
	}
}

// analyzeWaves analyzes wave patterns from pivot points
func (ew *ElliottWave) analyzeWaves() {
	if len(ew.pivotHighs) < 3 && len(ew.pivotLows) < 3 {
		return
	}

	// Combine and sort pivots
	pivots := make([]struct {
		index  int
		price  float64
		isHigh bool
	}, 0)

	for _, high := range ew.pivotHighs {
		if high < len(ew.candles) {
			pivots = append(pivots, struct {
				index  int
				price  float64
				isHigh bool
			}{high, ew.candles[high].High, true})
		}
	}

	for _, low := range ew.pivotLows {
		if low < len(ew.candles) {
			pivots = append(pivots, struct {
				index  int
				price  float64
				isHigh bool
			}{low, ew.candles[low].Low, false})
		}
	}

	// Sort by index
	for i := 0; i < len(pivots)-1; i++ {
		for j := i + 1; j < len(pivots); j++ {
			if pivots[i].index > pivots[j].index {
				pivots[i], pivots[j] = pivots[j], pivots[i]
			}
		}
	}

	// Analyze wave patterns
	if len(pivots) >= 5 {
		ew.identifyWavePatterns(pivots)
	}
}

// identifyWavePatterns identifies Elliott Wave patterns
func (ew *ElliottWave) identifyWavePatterns(pivots []struct {
	index  int
	price  float64
	isHigh bool
}) {
	if len(pivots) < 5 {
		return
	}

	// Look for 5-wave impulse patterns
	for i := 0; i <= len(pivots)-5; i++ {
		pattern := ew.analyzeImpulsePattern(pivots[i : i+5])
		if pattern.Type != NoWave {
			ew.waves = append(ew.waves, pattern)
			ew.currentWave = pattern

			// Keep only recent waves
			if len(ew.waves) > 10 {
				ew.waves = ew.waves[1:]
			}
			break
		}
	}

	// Look for 3-wave corrective patterns
	for i := 0; i <= len(pivots)-3; i++ {
		pattern := ew.analyzeCorrectionPattern(pivots[i : i+3])
		if pattern.Type != NoWave {
			ew.waves = append(ew.waves, pattern)
			ew.currentWave = pattern

			// Keep only recent waves
			if len(ew.waves) > 10 {
				ew.waves = ew.waves[1:]
			}
			break
		}
	}
}

// analyzeImpulsePattern analyzes a 5-wave impulse pattern
func (ew *ElliottWave) analyzeImpulsePattern(pivots []struct {
	index  int
	price  float64
	isHigh bool
}) WavePattern {
	if len(pivots) < 5 {
		return WavePattern{Type: NoWave}
	}

	// Check for alternating highs and lows
	expectedPattern := []bool{false, true, false, true, false} // Start with low
	if pivots[0].isHigh {
		expectedPattern = []bool{true, false, true, false, true} // Start with high
	}

	validPattern := true
	for i, pivot := range pivots {
		if pivot.isHigh != expectedPattern[i] {
			validPattern = false
			break
		}
	}

	if !validPattern {
		return WavePattern{Type: NoWave}
	}

	// Calculate wave measurements
	wave1 := math.Abs(pivots[1].price - pivots[0].price)
	wave2 := math.Abs(pivots[2].price - pivots[1].price)
	wave3 := math.Abs(pivots[3].price - pivots[2].price)
	wave4 := math.Abs(pivots[4].price - pivots[3].price)

	// Elliott Wave rules validation
	// Rule 1: Wave 3 cannot be the shortest
	if wave3 < wave1 && wave3 < wave4 {
		return WavePattern{Type: NoWave}
	}

	// Rule 2: Wave 2 cannot retrace more than 100% of Wave 1
	retracement2 := wave2 / wave1
	if retracement2 > 1.0 {
		return WavePattern{Type: NoWave}
	}

	// Rule 3: Wave 4 cannot overlap Wave 1 price territory
	if pivots[0].isHigh { // Downtrend
		if pivots[3].price > pivots[1].price {
			return WavePattern{Type: NoWave}
		}
	} else { // Uptrend
		if pivots[3].price < pivots[1].price {
			return WavePattern{Type: NoWave}
		}
	}

	// Calculate Fibonacci relationships
	fibRatio := ew.calculateFibonacciRatio(wave1, wave3)

	// Determine wave type and confidence
	waveType := ImpulseWave
	if wave3 > wave1*1.618 {
		waveType = ExtensionWave
	}

	confidence := ew.calculateWaveConfidence(wave1, wave2, wave3, wave4, fibRatio)

	// Calculate targets
	target := ew.calculateWaveTarget(pivots, waveType)
	stopLoss := ew.calculateStopLoss(pivots, waveType)

	return WavePattern{
		Type:       waveType,
		Count:      Wave5, // Assume we're looking at completion of wave 5
		StartIndex: pivots[0].index,
		EndIndex:   pivots[4].index,
		StartPrice: pivots[0].price,
		EndPrice:   pivots[4].price,
		Length:     math.Abs(pivots[4].price - pivots[0].price),
		FibRatio:   fibRatio,
		Confidence: confidence,
		IsComplete: true,
		Target:     target,
		StopLoss:   stopLoss,
	}
}

// analyzeCorrectionPattern analyzes a 3-wave corrective pattern
func (ew *ElliottWave) analyzeCorrectionPattern(pivots []struct {
	index  int
	price  float64
	isHigh bool
}) WavePattern {
	if len(pivots) < 3 {
		return WavePattern{Type: NoWave}
	}

	// Check for alternating highs and lows
	expectedPattern := []bool{false, true, false} // Start with low
	if pivots[0].isHigh {
		expectedPattern = []bool{true, false, true} // Start with high
	}

	validPattern := true
	for i, pivot := range pivots {
		if pivot.isHigh != expectedPattern[i] {
			validPattern = false
			break
		}
	}

	if !validPattern {
		return WavePattern{Type: NoWave}
	}

	// Calculate wave measurements
	waveA := math.Abs(pivots[1].price - pivots[0].price)
	waveB := math.Abs(pivots[2].price - pivots[1].price)

	// Common correction ratios
	fibRatio := waveB / waveA

	// Validate common Fibonacci ratios for corrections
	validFibRatio := false
	commonRatios := []float64{0.382, 0.5, 0.618, 0.786, 1.0, 1.272, 1.618}
	for _, ratio := range commonRatios {
		if math.Abs(fibRatio-ratio) < ew.config.FibonacciTolerance {
			validFibRatio = true
			break
		}
	}

	if !validFibRatio {
		return WavePattern{Type: NoWave}
	}

	confidence := ew.calculateCorrectionConfidence(waveA, waveB, fibRatio)
	target := ew.calculateCorrectionTarget(pivots)
	stopLoss := ew.calculateStopLoss(pivots, CorrectionWave)

	return WavePattern{
		Type:       CorrectionWave,
		Count:      WaveC, // Assume we're looking at completion of wave C
		StartIndex: pivots[0].index,
		EndIndex:   pivots[2].index,
		StartPrice: pivots[0].price,
		EndPrice:   pivots[2].price,
		Length:     math.Abs(pivots[2].price - pivots[0].price),
		FibRatio:   fibRatio,
		Confidence: confidence,
		IsComplete: true,
		Target:     target,
		StopLoss:   stopLoss,
	}
}

// calculateFibonacciRatio calculates Fibonacci ratio between waves
func (ew *ElliottWave) calculateFibonacciRatio(wave1, wave3 float64) float64 {
	if wave1 == 0 {
		return 0
	}
	return wave3 / wave1
}

// calculateWaveConfidence calculates confidence for impulse waves
func (ew *ElliottWave) calculateWaveConfidence(wave1, wave2, wave3, wave4, fibRatio float64) float64 {
	confidence := 0.0

	// Base confidence from wave 3 being strongest
	if wave3 > wave1 && wave3 > wave4 {
		confidence += 0.3
	}

	// Fibonacci ratio confidence
	commonRatios := []float64{0.618, 1.0, 1.618, 2.618}
	for _, ratio := range commonRatios {
		if math.Abs(fibRatio-ratio) < ew.config.FibonacciTolerance {
			confidence += 0.3
			break
		}
	}

	// Wave 2 retracement confidence
	retracement := wave2 / wave1
	if retracement >= 0.382 && retracement <= 0.786 {
		confidence += 0.2
	}

	// Wave 4 retracement confidence
	retracement4 := wave4 / wave3
	if retracement4 >= 0.236 && retracement4 <= 0.5 {
		confidence += 0.2
	}

	return math.Min(confidence, 1.0)
}

// calculateCorrectionConfidence calculates confidence for correction waves
func (ew *ElliottWave) calculateCorrectionConfidence(waveA, waveB, fibRatio float64) float64 {
	confidence := 0.3 // Base confidence

	// Fibonacci ratio confidence
	commonRatios := []float64{0.382, 0.5, 0.618, 0.786, 1.0, 1.272, 1.618}
	for _, ratio := range commonRatios {
		if math.Abs(fibRatio-ratio) < ew.config.FibonacciTolerance {
			confidence += 0.4
			break
		}
	}

	// Wave relationship confidence
	if waveA > 0 && waveB > 0 {
		confidence += 0.3
	}

	return math.Min(confidence, 1.0)
}

// calculateWaveTarget calculates price target for wave completion
func (ew *ElliottWave) calculateWaveTarget(pivots []struct {
	index  int
	price  float64
	isHigh bool
}, waveType WaveType) float64 {
	if len(pivots) < 3 {
		return 0
	}

	// Use Fibonacci extensions for targets
	wave1 := pivots[1].price - pivots[0].price

	if pivots[0].isHigh { // Downtrend
		return pivots[2].price + wave1*1.618 // Common extension
	} else { // Uptrend
		return pivots[2].price + wave1*1.618 // Common extension
	}
}

// calculateCorrectionTarget calculates target for correction completion
func (ew *ElliottWave) calculateCorrectionTarget(pivots []struct {
	index  int
	price  float64
	isHigh bool
}) float64 {
	if len(pivots) < 3 {
		return 0
	}

	// Use common retracement levels
	totalMove := pivots[2].price - pivots[0].price

	if pivots[0].isHigh { // Downward correction
		return pivots[0].price + totalMove*0.618 // 61.8% retracement
	} else { // Upward correction
		return pivots[0].price + totalMove*0.618 // 61.8% retracement
	}
}

// calculateStopLoss calculates stop loss for wave patterns
func (ew *ElliottWave) calculateStopLoss(pivots []struct {
	index  int
	price  float64
	isHigh bool
}, waveType WaveType) float64 {
	if len(pivots) < 2 {
		return 0
	}

	// Use previous wave extreme as stop loss
	if pivots[0].isHigh { // Downtrend
		return pivots[0].price * 1.02 // 2% above high
	} else { // Uptrend
		return pivots[0].price * 0.98 // 2% below low
	}
}

// GetCurrentSignal returns the current Elliott Wave signal
func (ew *ElliottWave) GetCurrentSignal() (SignalType, float64) {
	if !ew.initialized || ew.currentWave.Type == NoWave {
		return Hold, 0.0
	}

	signal, strength := ew.analyzeCurrentWave()

	ew.lastSignal = signal
	ew.lastStrength = strength
	ew.lastWaveType = ew.currentWave.Type
	ew.lastWaveCount = ew.currentWave.Count

	return signal, strength
}

// analyzeCurrentWave analyzes the current wave for trading signals
func (ew *ElliottWave) analyzeCurrentWave() (SignalType, float64) {
	if ew.currentWave.Type == NoWave {
		return Hold, 0.0
	}

	confidence := ew.currentWave.Confidence

	// Signal based on wave completion and type
	switch ew.currentWave.Type {
	case ImpulseWave:
		if ew.currentWave.Count == Wave5 && ew.currentWave.IsComplete {
			// End of impulse wave - expect reversal
			if ew.currentWave.EndPrice > ew.currentWave.StartPrice {
				return Sell, confidence * ew.config.ImpulseBoost // Bullish impulse ending
			} else {
				return Buy, confidence * ew.config.ImpulseBoost // Bearish impulse ending
			}
		}

	case CorrectionWave:
		if ew.currentWave.Count == WaveC && ew.currentWave.IsComplete {
			// End of correction - expect resumption of trend
			if ew.currentWave.EndPrice > ew.currentWave.StartPrice {
				return Sell, confidence * ew.config.CorrectionBoost // Upward correction ending
			} else {
				return Buy, confidence * ew.config.CorrectionBoost // Downward correction ending
			}
		}

	case ExtensionWave:
		if ew.currentWave.IsComplete {
			// Extended wave completion - strong reversal signal
			if ew.currentWave.EndPrice > ew.currentWave.StartPrice {
				return Sell, confidence * ew.config.CompletionBoost
			} else {
				return Buy, confidence * ew.config.CompletionBoost
			}
		}
	}

	return Hold, 0.0
}

// Calculate implements TechnicalIndicator interface
func (ew *ElliottWave) Calculate(candles []Candle) []float64 {
	if len(candles) < ew.config.MinWaveLength*2 {
		return []float64{}
	}

	values := make([]float64, 0, len(candles))

	// Process each candle
	for _, candle := range candles {
		ew.Update(candle)
		if ew.initialized {
			// Return wave confidence as value
			values = append(values, ew.currentWave.Confidence)
		}
	}

	return values
}

// GetSignal implements TechnicalIndicator interface
func (ew *ElliottWave) GetSignal(values []float64, currentPrice float64) IndicatorSignal {
	signal, strength := ew.GetCurrentSignal()

	var value float64
	if len(values) > 0 {
		value = values[len(values)-1]
	}

	return IndicatorSignal{
		Name:      ew.GetName(),
		Signal:    signal,
		Strength:  strength,
		Value:     value,
		Timestamp: time.Now(),
		Timeframe: ew.timeframe,
	}
}

// GetName returns the indicator name
func (ew *ElliottWave) GetName() string {
	return "ElliottWave"
}

// GetLastSignal returns the last signal and strength
func (ew *ElliottWave) GetLastSignal() (SignalType, float64) {
	return ew.lastSignal, ew.lastStrength
}

// GetCurrentWave returns the current wave information
func (ew *ElliottWave) GetCurrentWave() WavePattern {
	return ew.currentWave
}

// GetWaveCount returns the current wave count
func (ew *ElliottWave) GetWaveCount() WaveCount {
	return ew.lastWaveCount
}

// GetFibonacciLevel returns the current Fibonacci level
func (ew *ElliottWave) GetFibonacciLevel() float64 {
	return ew.currentWave.FibRatio
}

// String returns a string representation
func (ew *ElliottWave) String() string {
	if !ew.initialized {
		return "ElliottWave: Not initialized"
	}

	return fmt.Sprintf("ElliottWave: Type=%s, Count=%s, FibRatio=%.3f, Signal=%s, Strength=%.2f",
		ew.lastWaveType.String(), ew.lastWaveCount.String(), ew.currentWave.FibRatio,
		ew.lastSignal, ew.lastStrength)
}
