package bot

import (
	"fmt"
	"math"
	"math/rand"
	"os"
	"time"

	"trading-bot/pkg/indicator"
)

// TestSuite represents a collection of tests
type TestSuite struct {
	name    string
	tests   []Test
	results []TestResult
}

// Test represents a single test
type Test struct {
	name     string
	function func() error
}

// TestResult represents the result of a test
type TestResult struct {
	name     string
	passed   bool
	error    error
	duration time.Duration
}

// NewTestSuite creates a new test suite
func NewTestSuite(name string) *TestSuite {
	return &TestSuite{
		name:    name,
		tests:   make([]Test, 0),
		results: make([]TestResult, 0),
	}
}

// AddTest adds a test to the suite
func (ts *TestSuite) AddTest(name string, function func() error) {
	ts.tests = append(ts.tests, Test{
		name:     name,
		function: function,
	})
}

// Run executes all tests in the suite
func (ts *TestSuite) Run() {
	fmt.Printf("🧪 Running test suite: %s\n", ts.name)
	fmt.Printf("═══════════════════════════════════════\n")

	for _, test := range ts.tests {
		fmt.Printf("Testing: %s... ", test.name)

		start := time.Now()
		err := test.function()
		duration := time.Since(start)

		result := TestResult{
			name:     test.name,
			passed:   err == nil,
			error:    err,
			duration: duration,
		}

		ts.results = append(ts.results, result)

		if result.passed {
			fmt.Printf("✅ PASS (%.2fms)\n", float64(duration.Nanoseconds())/1000000)
		} else {
			fmt.Printf("❌ FAIL (%.2fms): %v\n", float64(duration.Nanoseconds())/1000000, err)
		}
	}

	ts.printSummary()
}

// printSummary prints the test summary
func (ts *TestSuite) printSummary() {
	fmt.Printf("\n📊 Test Summary for %s:\n", ts.name)
	fmt.Printf("═══════════════════════════════════════\n")

	passed := 0
	failed := 0
	var totalDuration time.Duration

	for _, result := range ts.results {
		if result.passed {
			passed++
		} else {
			failed++
		}
		totalDuration += result.duration
	}

	fmt.Printf("Total Tests: %d\n", len(ts.results))
	fmt.Printf("Passed: %d ✅\n", passed)
	fmt.Printf("Failed: %d ❌\n", failed)
	fmt.Printf("Success Rate: %.1f%%\n", float64(passed)/float64(len(ts.results))*100)
	fmt.Printf("Total Duration: %.2fms\n", float64(totalDuration.Nanoseconds())/1000000)
	fmt.Printf("═══════════════════════════════════════\n\n")
}

// RunAllTests runs comprehensive tests for the trading bot
func RunAllTests() {
	fmt.Println("🔬 Starting Trading Bot Test Suite")
	fmt.Println("══════════════════════════════════════")

	// Test Configuration
	testConfig := NewTestSuite("Configuration")
	testConfig.AddTest("Default Config Creation", testDefaultConfig)
	testConfig.AddTest("Config Validation", testConfigValidation)
	testConfig.AddTest("Config Save/Load", testConfigSaveLoad)
	testConfig.Run()

	// Test Technical Indicators
	testIndicators := NewTestSuite("Technical Indicators")
	testIndicators.AddTest("RSI Calculation", testRSICalculation)
	testIndicators.AddTest("MACD Calculation", testMACDCalculation)
	testIndicators.AddTest("Volume Analysis", testVolumeAnalysis)
	testIndicators.AddTest("Trend Detection", testTrendDetection)
	testIndicators.AddTest("Support/Resistance", testSupportResistance)
	testIndicators.AddTest("Ichimoku Cloud", testIchimokuCloud)
	testIndicators.Run()

	// Test Data Provider
	testDataProvider := NewTestSuite("Data Provider")
	testDataProvider.AddTest("Sample Data Generation", testSampleDataGeneration)
	testDataProvider.AddTest("Historical Data", testHistoricalData)
	testDataProvider.AddTest("Real-time Data", testRealTimeData)
	testDataProvider.Run()

	// Test Timeframe Manager
	testTimeframe := NewTestSuite("Timeframe Manager")
	testTimeframe.AddTest("Multi-timeframe Data", testMultiTimeframeData)
	testTimeframe.AddTest("Data Synchronization", testDataSynchronization)
	testTimeframe.AddTest("Context Generation", testContextGeneration)
	testTimeframe.Run()

	// Test Signal Aggregation
	testSignals := NewTestSuite("Signal Aggregation")
	testSignals.AddTest("Signal Generation", testSignalGeneration)
	testSignals.AddTest("Multi-timeframe Logic", testMultiTimeframeLogic)
	testSignals.AddTest("Signal Confidence", testSignalConfidence)
	testSignals.Run()

	fmt.Println("🎉 All tests completed!")
}

// Configuration Tests
func testDefaultConfig() error {
	config := DefaultConfig()
	return ValidateConfig(config)
}

func testConfigValidation() error {
	// Test invalid RSI configuration
	config := DefaultConfig()
	config.RSI.Period = 0
	if ValidateConfig(config) == nil {
		return fmt.Errorf("should have failed validation for invalid RSI period")
	}

	// Test invalid MACD configuration
	config = DefaultConfig()
	config.MACD.FastPeriod = 26
	config.MACD.SlowPeriod = 12
	if ValidateConfig(config) == nil {
		return fmt.Errorf("should have failed validation for invalid MACD periods")
	}

	return nil
}

func testConfigSaveLoad() error {
	config := DefaultConfig()
	config.Symbol = "TESTBTC"

	filename := "test_config.json"
	if err := SaveConfig(config, filename); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	loadedConfig, err := LoadConfig(filename)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	if loadedConfig.Symbol != config.Symbol {
		return fmt.Errorf("config mismatch: expected %s, got %s", config.Symbol, loadedConfig.Symbol)
	}

	return nil
}

// Technical Indicator Tests
func testRSICalculation() error {
	// Generate test data
	candles := generateTestCandles(50, 100.0)

	rsi := indicator.NewRSI(indicator.RSIConfig{Period: 14, Overbought: 70, Oversold: 30}, indicator.FiveMinute)
	values := rsi.Calculate(convertCandlesToIndicator(candles))

	if len(values) == 0 {
		return fmt.Errorf("RSI calculation returned no values")
	}

	// RSI should be between 0 and 100
	for _, value := range values {
		if value < 0 || value > 100 {
			return fmt.Errorf("RSI value out of range: %f", value)
		}
	}

	return nil
}

func testMACDCalculation() error {
	candles := generateTestCandles(50, 100.0)

	macd := indicator.NewMACD(indicator.MACDConfig{FastPeriod: 12, SlowPeriod: 26, SignalPeriod: 9}, indicator.FiveMinute)
	values := macd.Calculate(convertCandlesToIndicator(candles))

	if len(values) == 0 {
		return fmt.Errorf("MACD calculation returned no values")
	}

	return nil
}

func testVolumeAnalysis() error {
	candles := generateTestCandles(30, 100.0)

	volume := indicator.NewVolume(indicator.VolumeConfig{Period: 20, VolumeThreshold: 10000}, indicator.FiveMinute)
	values := volume.Calculate(convertCandlesToIndicator(candles))

	if len(values) == 0 {
		return fmt.Errorf("Volume analysis returned no values")
	}

	return nil
}

func testTrendDetection() error {
	candles := generateTestCandles(60, 100.0)

	trend := indicator.NewTrend(indicator.TrendConfig{ShortMA: 20, LongMA: 50}, indicator.FiveMinute)
	values := trend.Calculate(convertCandlesToIndicator(candles))

	if len(values) == 0 {
		return fmt.Errorf("Trend detection returned no values")
	}

	return nil
}

func testSupportResistance() error {
	candles := generateTestCandles(30, 100.0)

	sr := indicator.NewSupportResistance(indicator.SupportResistanceConfig{Period: 20, Threshold: 0.02}, indicator.FiveMinute)
	values := sr.Calculate(convertCandlesToIndicator(candles))

	if len(values) == 0 {
		return fmt.Errorf("Support/Resistance calculation returned no values")
	}

	return nil
}

func testIchimokuCloud() error {
	candles := generateTestCandles(60, 100.0) // Need more candles for Ichimoku

	ichimoku := indicator.NewIchimoku(indicator.IchimokuConfig{
		TenkanPeriod: 9,
		KijunPeriod:  26,
		SenkouPeriod: 52,
		Displacement: 26,
	}, indicator.FiveMinute)

	values := ichimoku.Calculate(convertCandlesToIndicator(candles))

	if len(values) == 0 {
		return fmt.Errorf("Ichimoku calculation returned no values")
	}

	// Test all components
	allValues := ichimoku.CalculateAll(convertCandlesToIndicator(candles))

	if len(allValues.TenkanSen) == 0 {
		return fmt.Errorf("Ichimoku Tenkan-sen calculation failed")
	}

	if len(allValues.KijunSen) == 0 {
		return fmt.Errorf("Ichimoku Kijun-sen calculation failed")
	}

	if len(allValues.CloudTop) == 0 || len(allValues.CloudBottom) == 0 {
		return fmt.Errorf("Ichimoku cloud calculation failed")
	}

	// Validate cloud signals are within expected range (-1 to 1)
	for _, value := range values {
		if value < -1.1 || value > 1.1 { // Allow small floating point errors
			return fmt.Errorf("Ichimoku signal out of range: %f", value)
		}
	}

	return nil
}

// Data Provider Tests
func testSampleDataGeneration() error {
	provider := NewSampleDataProvider([]string{"BTCUSD"}, 100.0)

	candles, err := provider.GetHistoricalData("BTCUSD", FiveMinute, 10)
	if err != nil {
		return fmt.Errorf("failed to get historical data: %w", err)
	}

	if len(candles) != 10 {
		return fmt.Errorf("expected 10 candles, got %d", len(candles))
	}

	return nil
}

func testHistoricalData() error {
	manager := NewDataProviderManager()
	provider := NewSampleDataProvider([]string{"BTCUSD"}, 100.0)
	manager.AddProvider("sample", provider)

	candles, err := manager.GetHistoricalData("BTCUSD", FiveMinute, 20)
	if err != nil {
		return fmt.Errorf("failed to get historical data: %w", err)
	}

	if len(candles) != 20 {
		return fmt.Errorf("expected 20 candles, got %d", len(candles))
	}

	return nil
}

func testRealTimeData() error {
	manager := NewDataProviderManager()
	provider := NewSampleDataProvider([]string{"BTCUSD"}, 100.0)
	manager.AddProvider("sample", provider)

	candleChan, err := manager.GetRealTimeData("BTCUSD", FiveMinute)
	if err != nil {
		return fmt.Errorf("failed to get real-time data: %w", err)
	}

	// Wait for at least one candle
	select {
	case <-candleChan:
		// Success
	case <-time.After(2 * time.Second):
		return fmt.Errorf("timeout waiting for real-time data")
	}

	return manager.Close()
}

// Timeframe Manager Tests
func testMultiTimeframeData() error {
	tm := NewTimeframeManager("BTCUSD")

	// Add test data for each timeframe
	for _, tf := range []Timeframe{FiveMinute, FifteenMinute, FortyFiveMinute, EightHour, Daily} {
		candles := generateTestCandles(30, 100.0)
		for _, candle := range candles {
			tm.AddCandle(tf, candle)
		}
	}

	summary := tm.GetDataSummary()
	if len(summary) != 5 {
		return fmt.Errorf("expected 5 timeframes, got %d", len(summary))
	}

	return nil
}

func testDataSynchronization() error {
	tm := NewTimeframeManager("BTCUSD")

	// Add sufficient data
	for _, tf := range []Timeframe{FiveMinute, FifteenMinute, FortyFiveMinute, EightHour, Daily} {
		var count int
		switch tf {
		case FiveMinute:
			count = 100
		case FifteenMinute:
			count = 80
		case FortyFiveMinute:
			count = 60
		case EightHour:
			count = 50
		case Daily:
			count = 30
		}

		candles := generateTestCandles(count, 100.0)
		for _, candle := range candles {
			tm.AddCandle(tf, candle)
		}
	}

	if !tm.IsReady() {
		return fmt.Errorf("timeframe manager should be ready")
	}

	return nil
}

func testContextGeneration() error {
	tm := NewTimeframeManager("BTCUSD")

	// Add test data
	for _, tf := range []Timeframe{FiveMinute, FifteenMinute, FortyFiveMinute, EightHour, Daily} {
		var count int
		switch tf {
		case FiveMinute:
			count = 100
		case FifteenMinute:
			count = 80
		case FortyFiveMinute:
			count = 60
		case EightHour:
			count = 50
		case Daily:
			count = 30
		}

		candles := generateTestCandles(count, 100.0)
		for _, candle := range candles {
			tm.AddCandle(tf, candle)
		}
	}

	ctx, err := tm.GetMultiTimeframeContext()
	if err != nil {
		return fmt.Errorf("failed to get context: %w", err)
	}

	if ctx.Symbol != "BTCUSD" {
		return fmt.Errorf("context symbol mismatch")
	}

	// Verify all timeframes are present
	if len(ctx.DailyCandles) == 0 || len(ctx.EightHourCandles) == 0 ||
		len(ctx.FortyFiveMinCandles) == 0 || len(ctx.FifteenMinCandles) == 0 ||
		len(ctx.FiveMinCandles) == 0 {
		return fmt.Errorf("missing candles in multi-timeframe context")
	}

	return nil
}

// Signal Tests
func testSignalGeneration() error {
	config := DefaultConfig()
	aggregator := NewSignalAggregator(config)

	// Create test context
	ctx := &MultiTimeframeContext{
		Symbol:              "BTCUSD",
		DailyCandles:        generateTestCandles(30, 100.0),
		EightHourCandles:    generateTestCandles(50, 100.0),
		FortyFiveMinCandles: generateTestCandles(60, 100.0),
		FifteenMinCandles:   generateTestCandles(80, 100.0),
		FiveMinCandles:      generateTestCandles(100, 100.0),
		LastUpdate:          time.Now(),
	}

	signal, err := aggregator.GenerateSignal(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate signal: %w", err)
	}

	if signal.Symbol != "BTCUSD" {
		return fmt.Errorf("signal symbol mismatch")
	}

	return nil
}

func testMultiTimeframeLogic() error {
	config := DefaultConfig()
	aggregator := NewSignalAggregator(config)

	// Test with trending data
	ctx := &MultiTimeframeContext{
		Symbol:              "BTCUSD",
		DailyCandles:        generateTrendingCandles(30, 100.0, 0.01),   // 1% uptrend
		EightHourCandles:    generateTrendingCandles(50, 100.0, 0.005),  // 0.5% uptrend
		FortyFiveMinCandles: generateTrendingCandles(60, 100.0, 0.003),  // 0.3% uptrend
		FifteenMinCandles:   generateTrendingCandles(80, 100.0, 0.002),  // 0.2% uptrend
		FiveMinCandles:      generateTrendingCandles(100, 100.0, 0.001), // 0.1% uptrend
		LastUpdate:          time.Now(),
	}

	signal, err := aggregator.GenerateSignal(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate signal: %w", err)
	}

	if signal.Confidence < 0 || signal.Confidence > 1 {
		return fmt.Errorf("signal confidence out of range: %f", signal.Confidence)
	}

	return nil
}

func testSignalConfidence() error {
	config := DefaultConfig()
	config.MinConfidence = 0.8 // High confidence requirement

	aggregator := NewSignalAggregator(config)

	// Test with weak signals
	ctx := &MultiTimeframeContext{
		Symbol:              "BTCUSD",
		DailyCandles:        generateTestCandles(30, 100.0),
		EightHourCandles:    generateTestCandles(50, 100.0),
		FortyFiveMinCandles: generateTestCandles(60, 100.0),
		FifteenMinCandles:   generateTestCandles(80, 100.0),
		FiveMinCandles:      generateTestCandles(100, 100.0),
		LastUpdate:          time.Now(),
	}

	signal, err := aggregator.GenerateSignal(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate signal: %w", err)
	}

	// Should be HOLD due to low confidence
	if signal.Signal != Hold {
		return fmt.Errorf("expected HOLD signal due to low confidence, got %s", signal.Signal.String())
	}

	return nil
}

// Helper functions for testing
func generateTestCandles(count int, basePrice float64) []Candle {
	candles := make([]Candle, count)
	currentPrice := basePrice

	for i := 0; i < count; i++ {
		// Generate random price movement
		change := (rand.Float64() - 0.5) * 0.02 // 2% max change
		newPrice := currentPrice * (1 + change)

		high := math.Max(currentPrice, newPrice) * 1.005
		low := math.Min(currentPrice, newPrice) * 0.995

		candles[i] = Candle{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Open:      currentPrice,
			High:      high,
			Low:       low,
			Close:     newPrice,
			Volume:    10000 + rand.Float64()*5000,
		}

		currentPrice = newPrice
	}

	return candles
}

func generateTrendingCandles(count int, basePrice float64, trendStrength float64) []Candle {
	candles := make([]Candle, count)
	currentPrice := basePrice

	for i := 0; i < count; i++ {
		// Add trend component
		trend := trendStrength * float64(i)
		change := (rand.Float64() - 0.5) * 0.01 // 1% max random change
		newPrice := currentPrice * (1 + trend + change)

		high := math.Max(currentPrice, newPrice) * 1.005
		low := math.Min(currentPrice, newPrice) * 0.995

		candles[i] = Candle{
			Timestamp: time.Now().Add(time.Duration(i) * time.Minute),
			Open:      currentPrice,
			High:      high,
			Low:       low,
			Close:     newPrice,
			Volume:    10000 + rand.Float64()*5000,
		}

		currentPrice = newPrice
	}

	return candles
}

// Helper function to convert bot candles to indicator candles
func convertCandlesToIndicator(candles []Candle) []indicator.Candle {
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

// TestCommand runs tests if called from command line
func TestCommand() {
	if len(os.Args) > 1 && os.Args[1] == "test" {
		RunAllTests()
		os.Exit(0)
	}
}
