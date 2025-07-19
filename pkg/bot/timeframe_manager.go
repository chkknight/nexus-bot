package bot

import (
	"fmt"
	"sync"
	"time"
)

// TimeframeManager handles multi-timeframe data coordination
type TimeframeManager struct {
	marketData *MarketData
	mutex      sync.RWMutex
	lastUpdate map[Timeframe]time.Time
	minCandles map[Timeframe]int
}

// NewTimeframeManager creates a new timeframe manager
func NewTimeframeManager(symbol string) *TimeframeManager {
	return &TimeframeManager{
		marketData: &MarketData{
			Symbol:     symbol,
			Timeframes: make(map[Timeframe][]Candle),
		},
		lastUpdate: make(map[Timeframe]time.Time),
		minCandles: map[Timeframe]int{
			FiveMinute:      100, // Need enough 5-min candles for indicators
			FifteenMinute:   80,  // Need enough 15-min candles for short-term analysis
			FortyFiveMinute: 60,  // Need enough 45-min candles for medium-term analysis
			EightHour:       50,  // Need enough 8H candles for trend
			Daily:           30,  // Need enough daily candles for S/R
		},
	}
}

// AddCandle adds a new candle to the specified timeframe
func (tm *TimeframeManager) AddCandle(timeframe Timeframe, candle Candle) {
	tm.mutex.Lock()
	defer tm.mutex.Unlock()

	// Initialize timeframe if it doesn't exist
	if tm.marketData.Timeframes[timeframe] == nil {
		tm.marketData.Timeframes[timeframe] = make([]Candle, 0)
	}

	// Check if we need to update or append
	candles := tm.marketData.Timeframes[timeframe]
	if len(candles) > 0 {
		lastCandle := candles[len(candles)-1]

		// If the timestamp matches, update the last candle
		if lastCandle.Timestamp.Equal(candle.Timestamp) {
			candles[len(candles)-1] = candle
			tm.marketData.Timeframes[timeframe] = candles
		} else if candle.Timestamp.After(lastCandle.Timestamp) {
			// New candle, append it
			tm.marketData.Timeframes[timeframe] = append(candles, candle)
		}
	} else {
		// First candle for this timeframe
		tm.marketData.Timeframes[timeframe] = append(candles, candle)
	}

	tm.lastUpdate[timeframe] = time.Now()
}

// GetCandles returns candles for a specific timeframe
func (tm *TimeframeManager) GetCandles(timeframe Timeframe) ([]Candle, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	candles, exists := tm.marketData.Timeframes[timeframe]
	if !exists {
		return nil, fmt.Errorf("no data for timeframe %s", timeframe.String())
	}

	return candles, nil
}

// GetLatestCandles returns the most recent N candles for a timeframe
func (tm *TimeframeManager) GetLatestCandles(timeframe Timeframe, count int) ([]Candle, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	candles, exists := tm.marketData.Timeframes[timeframe]
	if !exists {
		return nil, fmt.Errorf("no data for timeframe %s", timeframe.String())
	}

	if len(candles) < count {
		return candles, nil
	}

	return candles[len(candles)-count:], nil
}

// GetCurrentPrice returns the latest close price from 5-minute timeframe
func (tm *TimeframeManager) GetCurrentPrice() (float64, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	candles, exists := tm.marketData.Timeframes[FiveMinute]
	if !exists || len(candles) == 0 {
		return 0, fmt.Errorf("no 5-minute data available")
	}

	return candles[len(candles)-1].Close, nil
}

// IsReady checks if we have enough data for analysis
func (tm *TimeframeManager) IsReady() bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	for timeframe, minCount := range tm.minCandles {
		candles, exists := tm.marketData.Timeframes[timeframe]
		if !exists || len(candles) < minCount {
			return false
		}
	}

	return true
}

// GetReadyStatus returns detailed status of data availability
func (tm *TimeframeManager) GetReadyStatus() map[Timeframe]bool {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	status := make(map[Timeframe]bool)
	for timeframe, minCount := range tm.minCandles {
		candles, exists := tm.marketData.Timeframes[timeframe]
		status[timeframe] = exists && len(candles) >= minCount
	}

	return status
}

// GetDataSummary returns a summary of available data
func (tm *TimeframeManager) GetDataSummary() map[Timeframe]int {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	summary := make(map[Timeframe]int)
	for timeframe, candles := range tm.marketData.Timeframes {
		summary[timeframe] = len(candles)
	}

	return summary
}

// GetMultiTimeframeContext returns context from all timeframes for analysis
func (tm *TimeframeManager) GetMultiTimeframeContext() (*MultiTimeframeContext, error) {
	tm.mutex.RLock()
	defer tm.mutex.RUnlock()

	// Get latest candles from each timeframe
	dailyCandles, err := tm.getLatestCandlesInternal(Daily, 30)
	if err != nil {
		return nil, fmt.Errorf("failed to get daily candles: %w", err)
	}

	eightHourCandles, err := tm.getLatestCandlesInternal(EightHour, 50)
	if err != nil {
		return nil, fmt.Errorf("failed to get 8H candles: %w", err)
	}

	fortyFiveMinCandles, err := tm.getLatestCandlesInternal(FortyFiveMinute, 60)
	if err != nil {
		return nil, fmt.Errorf("failed to get 45-minute candles: %w", err)
	}

	fifteenMinCandles, err := tm.getLatestCandlesInternal(FifteenMinute, 80)
	if err != nil {
		return nil, fmt.Errorf("failed to get 15-minute candles: %w", err)
	}

	fiveMinCandles, err := tm.getLatestCandlesInternal(FiveMinute, 100)
	if err != nil {
		return nil, fmt.Errorf("failed to get 5-minute candles: %w", err)
	}

	return &MultiTimeframeContext{
		Symbol:              tm.marketData.Symbol,
		DailyCandles:        dailyCandles,
		EightHourCandles:    eightHourCandles,
		FortyFiveMinCandles: fortyFiveMinCandles,
		FifteenMinCandles:   fifteenMinCandles,
		FiveMinCandles:      fiveMinCandles,
		LastUpdate:          time.Now(),
	}, nil
}

// getLatestCandlesInternal is internal helper (assumes lock is held)
func (tm *TimeframeManager) getLatestCandlesInternal(timeframe Timeframe, count int) ([]Candle, error) {
	candles, exists := tm.marketData.Timeframes[timeframe]
	if !exists {
		return nil, fmt.Errorf("no data for timeframe %s", timeframe.String())
	}

	if len(candles) < count {
		return candles, nil
	}

	return candles[len(candles)-count:], nil
}

// MultiTimeframeContext holds data from all timeframes for analysis
type MultiTimeframeContext struct {
	Symbol              string    `json:"symbol"`
	DailyCandles        []Candle  `json:"daily_candles"`
	EightHourCandles    []Candle  `json:"eight_hour_candles"`
	FortyFiveMinCandles []Candle  `json:"forty_five_min_candles"`
	FifteenMinCandles   []Candle  `json:"fifteen_min_candles"`
	FiveMinCandles      []Candle  `json:"five_min_candles"`
	LastUpdate          time.Time `json:"last_update"`
}

// GetCurrentPrice returns the latest price from 5-minute data
func (ctx *MultiTimeframeContext) GetCurrentPrice() float64 {
	if len(ctx.FiveMinCandles) == 0 {
		return 0
	}
	return ctx.FiveMinCandles[len(ctx.FiveMinCandles)-1].Close
}

// GetDailyTrend returns basic trend from daily data
func (ctx *MultiTimeframeContext) GetDailyTrend() string {
	if len(ctx.DailyCandles) < 2 {
		return "UNKNOWN"
	}

	current := ctx.DailyCandles[len(ctx.DailyCandles)-1].Close
	previous := ctx.DailyCandles[len(ctx.DailyCandles)-2].Close

	if current > previous {
		return "BULLISH"
	} else if current < previous {
		return "BEARISH"
	}
	return "NEUTRAL"
}

// GetEightHourTrend returns basic trend from 8H data
func (ctx *MultiTimeframeContext) GetEightHourTrend() string {
	if len(ctx.EightHourCandles) < 2 {
		return "UNKNOWN"
	}

	current := ctx.EightHourCandles[len(ctx.EightHourCandles)-1].Close
	previous := ctx.EightHourCandles[len(ctx.EightHourCandles)-2].Close

	if current > previous {
		return "BULLISH"
	} else if current < previous {
		return "BEARISH"
	}
	return "NEUTRAL"
}
