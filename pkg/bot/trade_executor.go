package bot

import (
	"fmt"
	"log"
	"math"
	"sync"
	"time"
)

// TradeExecutor handles actual trade execution based on Pine Script ATR strategy
type TradeExecutor struct {
	config           Config
	enabled          bool
	currentPosition  *Position
	openOrders       map[string]*Order
	tradeHistory     []*Trade
	balance          float64
	mutex            sync.RWMutex
	riskManager      *RiskManager
	performanceStats *PerformanceStats
}

// Position represents an open trading position
type Position struct {
	ID           string    `json:"id"`
	Symbol       string    `json:"symbol"`
	Side         string    `json:"side"` // "LONG" or "SHORT"
	EntryPrice   float64   `json:"entry_price"`
	Quantity     float64   `json:"quantity"`
	CurrentPrice float64   `json:"current_price"`
	PnL          float64   `json:"pnl"`
	PnLPercent   float64   `json:"pnl_percent"`
	StopLoss     float64   `json:"stop_loss"`
	TakeProfit   float64   `json:"take_profit"`
	ATRTrailStop float64   `json:"atr_trail_stop"` // Pine Script ATR trailing stop
	OpenTime     time.Time `json:"open_time"`
	Strategy     string    `json:"strategy"` // "ATR_PINE_SCRIPT"
	Confidence   float64   `json:"confidence"`
}

// Order represents a trading order
type Order struct {
	ID          string    `json:"id"`
	Symbol      string    `json:"symbol"`
	Side        string    `json:"side"`
	Type        string    `json:"type"` // "MARKET", "LIMIT", "STOP"
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Status      string    `json:"status"` // "PENDING", "FILLED", "CANCELLED"
	CreatedTime time.Time `json:"created_time"`
	FilledTime  time.Time `json:"filled_time"`
	Strategy    string    `json:"strategy"`
	Confidence  float64   `json:"confidence"`
}

// Trade represents a completed trade
type Trade struct {
	ID         string    `json:"id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Quantity   float64   `json:"quantity"`
	PnL        float64   `json:"pnl"`
	PnLPercent float64   `json:"pnl_percent"`
	EntryTime  time.Time `json:"entry_time"`
	ExitTime   time.Time `json:"exit_time"`
	Duration   string    `json:"duration"`
	Strategy   string    `json:"strategy"`
	ExitReason string    `json:"exit_reason"` // "ATR_STOP", "TAKE_PROFIT", "MANUAL", "SIGNAL_CHANGE"
	Confidence float64   `json:"confidence"`
}

// RiskManager handles position sizing and risk controls
type RiskManager struct {
	MaxPositionSize   float64   `json:"max_position_size"`   // Max % of balance per trade
	MaxDailyLoss      float64   `json:"max_daily_loss"`      // Max daily loss %
	MaxDrawdown       float64   `json:"max_drawdown"`        // Max portfolio drawdown %
	ATRStopMultiplier float64   `json:"atr_stop_multiplier"` // ATR multiplier for stops
	MinConfidence     float64   `json:"min_confidence"`      // Min signal confidence to trade
	DailyLossUsed     float64   `json:"daily_loss_used"`     // Current daily loss
	LastResetTime     time.Time `json:"last_reset_time"`
}

// PerformanceStats tracks trading performance
type PerformanceStats struct {
	TotalTrades     int       `json:"total_trades"`
	WinningTrades   int       `json:"winning_trades"`
	LosingTrades    int       `json:"losing_trades"`
	WinRate         float64   `json:"win_rate"`
	TotalPnL        float64   `json:"total_pnl"`
	TotalPnLPercent float64   `json:"total_pnl_percent"`
	MaxWin          float64   `json:"max_win"`
	MaxLoss         float64   `json:"max_loss"`
	AverageWin      float64   `json:"average_win"`
	AverageLoss     float64   `json:"average_loss"`
	ProfitFactor    float64   `json:"profit_factor"`
	SharpeRatio     float64   `json:"sharpe_ratio"`
	MaxDrawdown     float64   `json:"max_drawdown"`
	ATRTradeCount   int       `json:"atr_trade_count"` // Pine Script ATR trades
	LastUpdated     time.Time `json:"last_updated"`
}

// NewTradeExecutor creates a new trade executor
func NewTradeExecutor(config Config, initialBalance float64) *TradeExecutor {
	return &TradeExecutor{
		config:          config,
		enabled:         true, // Enable by default for Pine Script ATR strategy
		currentPosition: nil,
		openOrders:      make(map[string]*Order),
		tradeHistory:    make([]*Trade, 0),
		balance:         initialBalance,
		riskManager: &RiskManager{
			MaxPositionSize:   0.02,                  // 2% of balance per trade (conservative)
			MaxDailyLoss:      0.05,                  // 5% max daily loss
			MaxDrawdown:       0.15,                  // 15% max drawdown
			ATRStopMultiplier: config.ATR.Multiplier, // Use Pine Script ATR multiplier
			MinConfidence:     config.MinConfidence,
			DailyLossUsed:     0,
			LastResetTime:     time.Now(),
		},
		performanceStats: &PerformanceStats{
			LastUpdated: time.Now(),
		},
	}
}

// ExecuteSignal processes a trading signal from Pine Script ATR strategy
func (te *TradeExecutor) ExecuteSignal(signal *TradingSignal, currentPrice float64, atrTrailStop float64) error {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	if !te.enabled {
		log.Printf("🚫 Trade execution disabled - skipping signal: %s", signal.Signal.String())
		return nil
	}

	// Check risk management
	if !te.checkRiskManagement(signal) {
		log.Printf("🛑 Risk management blocked trade: %s", signal.Signal.String())
		return nil
	}

	// Get ATR-specific signal (prioritize ATR indicator)
	var atrStrength float64
	for _, indSig := range signal.IndicatorSignals {
		if indSig.Name == "ATR_5m" {
			atrStrength = indSig.Strength
			break
		}
	}

	// Execute based on Pine Script ATR strategy logic
	switch signal.Signal {
	case Buy:
		return te.executeLongEntry(signal, currentPrice, atrTrailStop, atrStrength)
	case Sell:
		if te.config.ATR.UseShorts {
			return te.executeShortEntry(signal, currentPrice, atrTrailStop, atrStrength)
		} else {
			// Close long position if open (spot trading)
			if te.currentPosition != nil && te.currentPosition.Side == "LONG" {
				return te.closePosition("SIGNAL_CHANGE", currentPrice, atrTrailStop)
			}
		}
	case Hold:
		// Update trailing stops for open positions
		return te.updateTrailingStops(currentPrice, atrTrailStop)
	}

	return nil
}

// executeLongEntry executes a long position entry
func (te *TradeExecutor) executeLongEntry(signal *TradingSignal, currentPrice, atrTrailStop, atrStrength float64) error {
	// Close any short position first
	if te.currentPosition != nil && te.currentPosition.Side == "SHORT" {
		if err := te.closePosition("SIGNAL_CHANGE", currentPrice, atrTrailStop); err != nil {
			return err
		}
	}

	// Don't open new long if already long
	if te.currentPosition != nil && te.currentPosition.Side == "LONG" {
		// Update trailing stop
		return te.updateTrailingStops(currentPrice, atrTrailStop)
	}

	// Calculate position size based on risk management
	quantity := te.calculatePositionSize(currentPrice, atrTrailStop)
	if quantity == 0 {
		return fmt.Errorf("position size calculation resulted in 0 quantity")
	}

	// Create new long position
	position := &Position{
		ID:           fmt.Sprintf("pos_%d", time.Now().UnixNano()),
		Symbol:       te.config.Symbol,
		Side:         "LONG",
		EntryPrice:   currentPrice,
		Quantity:     quantity,
		CurrentPrice: currentPrice,
		PnL:          0,
		PnLPercent:   0,
		StopLoss:     atrTrailStop,
		TakeProfit:   0, // No fixed take profit for ATR strategy
		ATRTrailStop: atrTrailStop,
		OpenTime:     time.Now(),
		Strategy:     "ATR_PINE_SCRIPT",
		Confidence:   signal.Confidence,
	}

	te.currentPosition = position

	// Log the trade
	log.Printf("🟢 LONG ENTRY: %s at $%.2f", te.config.Symbol, currentPrice)
	log.Printf("   📊 Quantity: %.6f", quantity)
	log.Printf("   🛡️ ATR Stop: $%.2f", atrTrailStop)
	log.Printf("   📈 Confidence: %.1f%%", signal.Confidence*100)
	log.Printf("   ⚡ ATR Strength: %.3f", atrStrength)
	log.Printf("   🎯 Strategy: Pine Script ATR (Length=%d, Mult=%.1f)", te.config.ATR.Period, te.config.ATR.Multiplier)

	return nil
}

// executeShortEntry executes a short position entry (futures only)
func (te *TradeExecutor) executeShortEntry(signal *TradingSignal, currentPrice, atrTrailStop, atrStrength float64) error {
	// Close any long position first
	if te.currentPosition != nil && te.currentPosition.Side == "LONG" {
		if err := te.closePosition("SIGNAL_CHANGE", currentPrice, atrTrailStop); err != nil {
			return err
		}
	}

	// Don't open new short if already short
	if te.currentPosition != nil && te.currentPosition.Side == "SHORT" {
		// Update trailing stop
		return te.updateTrailingStops(currentPrice, atrTrailStop)
	}

	// Calculate position size based on risk management
	quantity := te.calculatePositionSize(currentPrice, atrTrailStop)
	if quantity == 0 {
		return fmt.Errorf("position size calculation resulted in 0 quantity")
	}

	// Create new short position
	position := &Position{
		ID:           fmt.Sprintf("pos_%d", time.Now().UnixNano()),
		Symbol:       te.config.Symbol,
		Side:         "SHORT",
		EntryPrice:   currentPrice,
		Quantity:     quantity,
		CurrentPrice: currentPrice,
		PnL:          0,
		PnLPercent:   0,
		StopLoss:     atrTrailStop,
		TakeProfit:   0, // No fixed take profit for ATR strategy
		ATRTrailStop: atrTrailStop,
		OpenTime:     time.Now(),
		Strategy:     "ATR_PINE_SCRIPT",
		Confidence:   signal.Confidence,
	}

	te.currentPosition = position

	// Log the trade
	log.Printf("🔴 SHORT ENTRY: %s at $%.2f", te.config.Symbol, currentPrice)
	log.Printf("   📊 Quantity: %.6f", quantity)
	log.Printf("   🛡️ ATR Stop: $%.2f", atrTrailStop)
	log.Printf("   📈 Confidence: %.1f%%", signal.Confidence*100)
	log.Printf("   ⚡ ATR Strength: %.3f", atrStrength)
	log.Printf("   🎯 Strategy: Pine Script ATR (Length=%d, Mult=%.1f)", te.config.ATR.Period, te.config.ATR.Multiplier)

	return nil
}

// updateTrailingStops updates ATR trailing stops for open positions
func (te *TradeExecutor) updateTrailingStops(currentPrice, newATRTrailStop float64) error {
	if te.currentPosition == nil {
		return nil
	}

	// Update current price and PnL
	te.currentPosition.CurrentPrice = currentPrice

	if te.currentPosition.Side == "LONG" {
		// Long position: trailing stop can only move up
		if newATRTrailStop > te.currentPosition.ATRTrailStop {
			te.currentPosition.ATRTrailStop = newATRTrailStop
			te.currentPosition.StopLoss = newATRTrailStop
			log.Printf("📈 ATR Trailing Stop Updated: $%.2f -> $%.2f (LONG)", te.currentPosition.StopLoss, newATRTrailStop)
		}

		// Calculate PnL
		te.currentPosition.PnL = (currentPrice - te.currentPosition.EntryPrice) * te.currentPosition.Quantity
		te.currentPosition.PnLPercent = (currentPrice - te.currentPosition.EntryPrice) / te.currentPosition.EntryPrice * 100

		// Check if stop loss hit
		if currentPrice <= te.currentPosition.ATRTrailStop {
			log.Printf("🛑 ATR STOP TRIGGERED: Price $%.2f <= Stop $%.2f", currentPrice, te.currentPosition.ATRTrailStop)
			return te.closePosition("ATR_STOP", currentPrice, newATRTrailStop)
		}

	} else if te.currentPosition.Side == "SHORT" {
		// Short position: trailing stop can only move down
		if newATRTrailStop < te.currentPosition.ATRTrailStop || te.currentPosition.ATRTrailStop == 0 {
			te.currentPosition.ATRTrailStop = newATRTrailStop
			te.currentPosition.StopLoss = newATRTrailStop
			log.Printf("📉 ATR Trailing Stop Updated: $%.2f -> $%.2f (SHORT)", te.currentPosition.StopLoss, newATRTrailStop)
		}

		// Calculate PnL
		te.currentPosition.PnL = (te.currentPosition.EntryPrice - currentPrice) * te.currentPosition.Quantity
		te.currentPosition.PnLPercent = (te.currentPosition.EntryPrice - currentPrice) / te.currentPosition.EntryPrice * 100

		// Check if stop loss hit
		if currentPrice >= te.currentPosition.ATRTrailStop {
			log.Printf("🛑 ATR STOP TRIGGERED: Price $%.2f >= Stop $%.2f", currentPrice, te.currentPosition.ATRTrailStop)
			return te.closePosition("ATR_STOP", currentPrice, newATRTrailStop)
		}
	}

	return nil
}

// closePosition closes the current position
func (te *TradeExecutor) closePosition(reason string, exitPrice, atrTrailStop float64) error {
	if te.currentPosition == nil {
		return nil
	}

	position := te.currentPosition
	exitTime := time.Now()
	duration := exitTime.Sub(position.OpenTime)

	// Calculate final PnL
	var finalPnL, finalPnLPercent float64
	if position.Side == "LONG" {
		finalPnL = (exitPrice - position.EntryPrice) * position.Quantity
		finalPnLPercent = (exitPrice - position.EntryPrice) / position.EntryPrice * 100
	} else {
		finalPnL = (position.EntryPrice - exitPrice) * position.Quantity
		finalPnLPercent = (position.EntryPrice - exitPrice) / position.EntryPrice * 100
	}

	// Create trade record
	trade := &Trade{
		ID:         fmt.Sprintf("trade_%d", time.Now().UnixNano()),
		Symbol:     position.Symbol,
		Side:       position.Side,
		EntryPrice: position.EntryPrice,
		ExitPrice:  exitPrice,
		Quantity:   position.Quantity,
		PnL:        finalPnL,
		PnLPercent: finalPnLPercent,
		EntryTime:  position.OpenTime,
		ExitTime:   exitTime,
		Duration:   duration.String(),
		Strategy:   position.Strategy,
		ExitReason: reason,
		Confidence: position.Confidence,
	}

	te.tradeHistory = append(te.tradeHistory, trade)
	te.updatePerformanceStats(trade)

	// Log the trade
	pnlSign := "🟢"
	if finalPnL < 0 {
		pnlSign = "🔴"
	}

	log.Printf("%s POSITION CLOSED: %s %s", pnlSign, position.Side, te.config.Symbol)
	log.Printf("   💰 Entry: $%.2f -> Exit: $%.2f", position.EntryPrice, exitPrice)
	log.Printf("   📊 PnL: $%.2f (%.2f%%)", finalPnL, finalPnLPercent)
	log.Printf("   ⏱️ Duration: %s", duration.String())
	log.Printf("   🎯 Reason: %s", reason)
	log.Printf("   📈 Win Rate: %.1f%% (%d/%d trades)", te.performanceStats.WinRate, te.performanceStats.WinningTrades, te.performanceStats.TotalTrades)

	// Clear current position
	te.currentPosition = nil

	return nil
}

// calculatePositionSize calculates position size based on risk management
func (te *TradeExecutor) calculatePositionSize(entryPrice, stopLoss float64) float64 {
	if stopLoss == 0 {
		return 0
	}

	// Calculate risk per share
	var riskPerShare float64
	if stopLoss < entryPrice {
		// Long position or short with stop above entry
		riskPerShare = math.Abs(entryPrice - stopLoss)
	} else {
		// Short position with stop below entry
		riskPerShare = math.Abs(stopLoss - entryPrice)
	}

	if riskPerShare == 0 {
		return 0
	}

	// Calculate position size based on max position risk
	maxRiskAmount := te.balance * te.riskManager.MaxPositionSize
	quantity := maxRiskAmount / riskPerShare

	// Ensure minimum viable quantity (for crypto, typically > 0.00001)
	minQuantity := 0.00001
	if quantity < minQuantity {
		return 0
	}

	return quantity
}

// checkRiskManagement checks if trade passes risk management rules
func (te *TradeExecutor) checkRiskManagement(signal *TradingSignal) bool {
	// Check confidence threshold
	if signal.Confidence < te.riskManager.MinConfidence {
		log.Printf("🚫 Signal confidence %.3f below minimum %.3f", signal.Confidence, te.riskManager.MinConfidence)
		return false
	}

	// Check daily loss limit
	now := time.Now()
	if now.Sub(te.riskManager.LastResetTime) >= 24*time.Hour {
		// Reset daily loss tracking
		te.riskManager.DailyLossUsed = 0
		te.riskManager.LastResetTime = now
	}

	if te.riskManager.DailyLossUsed >= te.riskManager.MaxDailyLoss {
		log.Printf("🚫 Daily loss limit reached: %.2f%% >= %.2f%%", te.riskManager.DailyLossUsed*100, te.riskManager.MaxDailyLoss*100)
		return false
	}

	// Check max drawdown
	if te.performanceStats.MaxDrawdown >= te.riskManager.MaxDrawdown {
		log.Printf("🚫 Max drawdown limit reached: %.2f%% >= %.2f%%", te.performanceStats.MaxDrawdown*100, te.riskManager.MaxDrawdown*100)
		return false
	}

	return true
}

// updatePerformanceStats updates performance statistics
func (te *TradeExecutor) updatePerformanceStats(trade *Trade) {
	stats := te.performanceStats

	stats.TotalTrades++
	stats.TotalPnL += trade.PnL
	stats.TotalPnLPercent += trade.PnLPercent

	if trade.PnL > 0 {
		stats.WinningTrades++
		if trade.PnL > stats.MaxWin {
			stats.MaxWin = trade.PnL
		}
		stats.AverageWin = (stats.AverageWin*float64(stats.WinningTrades-1) + trade.PnL) / float64(stats.WinningTrades)
	} else {
		stats.LosingTrades++
		if trade.PnL < stats.MaxLoss {
			stats.MaxLoss = trade.PnL
		}
		stats.AverageLoss = (stats.AverageLoss*float64(stats.LosingTrades-1) + trade.PnL) / float64(stats.LosingTrades)

		// Update daily loss
		dailyLossPercent := math.Abs(trade.PnL) / te.balance
		te.riskManager.DailyLossUsed += dailyLossPercent
	}

	// Calculate win rate
	stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades) * 100

	// Calculate profit factor
	if stats.AverageLoss != 0 {
		stats.ProfitFactor = math.Abs(stats.AverageWin / stats.AverageLoss)
	}

	// Track ATR trades specifically
	if trade.Strategy == "ATR_PINE_SCRIPT" {
		stats.ATRTradeCount++
	}

	stats.LastUpdated = time.Now()
}

// GetStatus returns current trading status
func (te *TradeExecutor) GetStatus() interface{} {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	return map[string]interface{}{
		"enabled":           te.enabled,
		"balance":           te.balance,
		"current_position":  te.currentPosition,
		"open_orders_count": len(te.openOrders),
		"total_trades":      len(te.tradeHistory),
		"performance":       te.performanceStats,
		"risk_management":   te.riskManager,
		"strategy":          "Pine Script ATR Trailing Stops",
		"atr_config": map[string]interface{}{
			"period":     te.config.ATR.Period,
			"multiplier": te.config.ATR.Multiplier,
			"use_shorts": te.config.ATR.UseShorts,
		},
	}
}

// GetCurrentPosition returns the current open position
func (te *TradeExecutor) GetCurrentPosition() *Position {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	return te.currentPosition
}

// GetTradeHistory returns recent trade history
func (te *TradeExecutor) GetTradeHistory(limit int) []*Trade {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	if limit <= 0 || limit > len(te.tradeHistory) {
		return te.tradeHistory
	}

	startIdx := len(te.tradeHistory) - limit
	return te.tradeHistory[startIdx:]
}

// Enable enables trade execution
func (te *TradeExecutor) Enable() {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	te.enabled = true
	log.Printf("🟢 Trade execution ENABLED - Pine Script ATR strategy active")
}

// Disable disables trade execution
func (te *TradeExecutor) Disable() {
	te.mutex.Lock()
	defer te.mutex.Unlock()
	te.enabled = false
	log.Printf("🔴 Trade execution DISABLED - Pine Script ATR strategy paused")
}

// ForceClosePosition manually closes current position
func (te *TradeExecutor) ForceClosePosition(currentPrice float64) error {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	if te.currentPosition == nil {
		return fmt.Errorf("no open position to close")
	}

	return te.closePosition("MANUAL", currentPrice, te.currentPosition.ATRTrailStop)
}
