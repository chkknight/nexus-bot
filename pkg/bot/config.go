package bot

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
)

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() Config {
	return Config{
		RSI: RSIConfig{
			Enabled:    true, // RSI enabled by default
			Period:     14,
			Overbought: 70.0,
			Oversold:   30.0,
		},
		MACD: MACDConfig{
			Enabled:      false, // MACD enabled by default
			FastPeriod:   12,
			SlowPeriod:   26,
			SignalPeriod: 9,
		},
		Volume: VolumeConfig{
			Enabled:         false, // Volume enabled by default
			Period:          20,
			VolumeThreshold: 15000.0,
		},
		Trend: TrendConfig{
			Enabled: true, // Trend enabled by default
			ShortMA: 12,
			LongMA:  26,
		},
		SupportResistance: SupportResistanceConfig{
			Enabled:   true, // Support/Resistance enabled by default
			Period:    20,
			Threshold: 0.02, // 2%
		},
		Ichimoku: IchimokuConfig{
			Enabled:      true, // Ichimoku enabled by default
			TenkanPeriod: 9,    // Conversion Line
			KijunPeriod:  26,   // Base Line
			SenkouPeriod: 52,   // Leading Span B
			Displacement: 26,   // Cloud displacement
		},
		MFI: MFIConfig{
			Enabled:    true, // Reverse-MFI enabled by default
			Period:     14,   // Standard MFI period
			Overbought: 80.0, // Overbought level
			Oversold:   20.0, // Oversold level
		},
		BollingerBands: BollingerBandsConfig{
			Enabled:       false, // BollingerBands disabled by default
			Period:        20,    // Standard BB period
			StandardDev:   2.0,   // Standard deviation multiplier
			OverboughtStd: 0.8,   // Overbought threshold
			OversoldStd:   0.2,   // Oversold threshold
		},
		Stochastic: StochasticConfig{
			Enabled:         true, // Stochastic enabled for 5-minute trading
			KPeriod:         9,    // Fast response for 5-minute
			DPeriod:         3,    // Quick smoothing
			SlowPeriod:      3,    // Fast slow K
			Overbought:      80.0, // Standard overbought
			Oversold:        20.0, // Standard oversold
			MomentumBoost:   1.2,  // Enhanced momentum detection
			DivergenceBoost: 1.3,  // Divergence boost
		},
		WilliamsR: WilliamsRConfig{
			Enabled:       true, // Williams %R enabled for 5-minute trading
			Period:        10,   // Fast response for 5-minute (shorter than default 14)
			Overbought:    -20,  // Standard overbought threshold
			Oversold:      -80,  // Standard oversold threshold
			FastResponse:  true, // Enhanced 5-minute response
			MomentumBoost: 1.3,  // Enhanced momentum detection
			ReversalBoost: 1.4,  // Enhanced reversal detection
		},
		PinBar: PinBarConfig{
			Enabled:           false,
			MinWickRatio:      1.5,
			MaxBodyRatio:      0.5,
			TrendConfirmation: true,
		},
		EMA: EMAConfig{
			Enabled:        true,
			FastPeriod:     12,
			SlowPeriod:     26,
			SignalPeriod:   9,
			TrendPeriod:    50,
			SlopeThreshold: 0.0001,
			CrossoverBoost: 1.3,
			TrendBoost:     1.2,
			VolumeConfirm:  false,
		},
		ElliottWave: ElliottWaveConfig{
			Enabled:            true,
			MinWaveLength:      5,
			FibonacciTolerance: 0.1,
			TrendStrength:      0.02,
			ImpulseBoost:       1.4,
			CorrectionBoost:    1.2,
			CompletionBoost:    1.5,
			MaxLookback:        100,
		},
		MinConfidence: 0.6, // 60% minimum confidence
		Symbol:        "BTCUSDT",
		Binance: BinanceConfig{
			APIKey:     "",
			SecretKey:  "",
			UseTestnet: false,
		},
		DataProvider: "sample", // Default to sample data
	}
}

// LoadConfig loads configuration from a JSON file
func LoadConfig(filename string) (Config, error) {
	// Start with defaults
	config := DefaultConfig()

	// Check if file exists
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// File doesn't exist, create it with defaults
		if err := SaveConfig(config, filename); err != nil {
			return config, fmt.Errorf("failed to create default config file: %w", err)
		}
		return config, nil
	}

	// Read file
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return config, fmt.Errorf("failed to read config file: %w", err)
	}

	// Parse JSON
	if err := json.Unmarshal(data, &config); err != nil {
		return config, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Load API keys from environment variables if not set in config
	config = loadAPIKeysFromEnv(config)

	// Validate configuration
	if err := ValidateConfig(config); err != nil {
		return config, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// loadAPIKeysFromEnv loads API keys from environment variables if not set in config
func loadAPIKeysFromEnv(config Config) Config {
	// Load Binance API keys from environment variables if not set
	if config.Binance.APIKey == "" || strings.Contains(config.Binance.APIKey, "YOUR_") {
		if envAPIKey := os.Getenv("BINANCE_API_KEY"); envAPIKey != "" {
			config.Binance.APIKey = envAPIKey
			fmt.Println("📊 Loaded Binance API Key from environment variable")
		}
	}

	if config.Binance.SecretKey == "" || strings.Contains(config.Binance.SecretKey, "YOUR_") {
		if envSecretKey := os.Getenv("BINANCE_SECRET_KEY"); envSecretKey != "" {
			config.Binance.SecretKey = envSecretKey
			fmt.Println("🔐 Loaded Binance Secret Key from environment variable")
		}
	}

	return config
}

// SaveConfig saves configuration to a JSON file
func SaveConfig(config Config, filename string) error {
	// Convert to JSON with indentation
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write to file
	if err := ioutil.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ValidateConfig validates the configuration parameters
func ValidateConfig(config Config) error {
	// Validate RSI
	if config.RSI.Period < 1 || config.RSI.Period > 100 {
		return fmt.Errorf("RSI period must be between 1 and 100")
	}
	if config.RSI.Overbought <= config.RSI.Oversold {
		return fmt.Errorf("RSI overbought level must be greater than oversold level")
	}
	if config.RSI.Overbought < 50 || config.RSI.Overbought > 100 {
		return fmt.Errorf("RSI overbought level must be between 50 and 100")
	}
	if config.RSI.Oversold < 0 || config.RSI.Oversold > 50 {
		return fmt.Errorf("RSI oversold level must be between 0 and 50")
	}

	// Validate MACD
	if config.MACD.FastPeriod < 1 || config.MACD.FastPeriod > 50 {
		return fmt.Errorf("MACD fast period must be between 1 and 50")
	}
	if config.MACD.SlowPeriod < 1 || config.MACD.SlowPeriod > 100 {
		return fmt.Errorf("MACD slow period must be between 1 and 100")
	}
	if config.MACD.SignalPeriod < 1 || config.MACD.SignalPeriod > 50 {
		return fmt.Errorf("MACD signal period must be between 1 and 50")
	}
	if config.MACD.FastPeriod >= config.MACD.SlowPeriod {
		return fmt.Errorf("MACD fast period must be less than slow period")
	}

	// Validate Volume
	if config.Volume.Period < 1 || config.Volume.Period > 100 {
		return fmt.Errorf("Volume period must be between 1 and 100")
	}
	if config.Volume.VolumeThreshold < 0 {
		return fmt.Errorf("Volume threshold must be positive")
	}

	// Validate Trend
	if config.Trend.ShortMA < 1 || config.Trend.ShortMA > 100 {
		return fmt.Errorf("Trend short MA must be between 1 and 100")
	}
	if config.Trend.LongMA < 1 || config.Trend.LongMA > 200 {
		return fmt.Errorf("Trend long MA must be between 1 and 200")
	}
	if config.Trend.ShortMA >= config.Trend.LongMA {
		return fmt.Errorf("Trend short MA must be less than long MA")
	}

	// Validate Support/Resistance
	if config.SupportResistance.Period < 1 || config.SupportResistance.Period > 100 {
		return fmt.Errorf("Support/Resistance period must be between 1 and 100")
	}
	if config.SupportResistance.Threshold < 0 || config.SupportResistance.Threshold > 1 {
		return fmt.Errorf("Support/Resistance threshold must be between 0 and 1")
	}

	// Validate Ichimoku
	if config.Ichimoku.TenkanPeriod < 1 || config.Ichimoku.TenkanPeriod > 50 {
		return fmt.Errorf("Ichimoku Tenkan period must be between 1 and 50")
	}
	if config.Ichimoku.KijunPeriod < 1 || config.Ichimoku.KijunPeriod > 100 {
		return fmt.Errorf("Ichimoku Kijun period must be between 1 and 100")
	}
	if config.Ichimoku.SenkouPeriod < 1 || config.Ichimoku.SenkouPeriod > 200 {
		return fmt.Errorf("Ichimoku Senkou period must be between 1 and 200")
	}
	if config.Ichimoku.Displacement < 1 || config.Ichimoku.Displacement > 100 {
		return fmt.Errorf("Ichimoku displacement must be between 1 and 100")
	}
	if config.Ichimoku.TenkanPeriod >= config.Ichimoku.KijunPeriod {
		return fmt.Errorf("Ichimoku Tenkan period must be less than Kijun period")
	}
	if config.Ichimoku.KijunPeriod >= config.Ichimoku.SenkouPeriod {
		return fmt.Errorf("Ichimoku Kijun period must be less than Senkou period")
	}

	// Validate MFI
	if config.MFI.Period < 1 || config.MFI.Period > 100 {
		return fmt.Errorf("MFI period must be between 1 and 100")
	}
	if config.MFI.Overbought <= config.MFI.Oversold {
		return fmt.Errorf("MFI overbought level must be greater than oversold level")
	}
	if config.MFI.Overbought < 50 || config.MFI.Overbought > 100 {
		return fmt.Errorf("MFI overbought level must be between 50 and 100")
	}
	if config.MFI.Oversold < 0 || config.MFI.Oversold > 50 {
		return fmt.Errorf("MFI oversold level must be between 0 and 50")
	}

	// Validate general settings
	if config.MinConfidence < 0 || config.MinConfidence > 1 {
		return fmt.Errorf("Minimum confidence must be between 0 and 1")
	}
	if config.Symbol == "" {
		return fmt.Errorf("Symbol cannot be empty")
	}

	// Validate Binance settings if using Binance data provider
	if config.DataProvider == "binance" {
		// API keys are optional for public data (klines)
		// Only warn if they're not set
		if config.Binance.APIKey == "" || strings.Contains(config.Binance.APIKey, "YOUR_") {
			fmt.Println("⚠️  Note: Using Binance public API (no API key). For advanced features, set BINANCE_API_KEY environment variable.")
		}
	}

	return nil
}

// GetConfigSummary returns a human-readable summary of the configuration
func GetConfigSummary(config Config) string {
	summary := fmt.Sprintf("📊 Trading Bot Configuration for %s\n", config.Symbol)
	summary += fmt.Sprintf("══════════════════════════════════════\n")

	// Show enabled/disabled status with emojis
	enabledCount := 0

	summary += fmt.Sprintf("📈 Technical Indicators Status:\n")
	if config.RSI.Enabled {
		summary += fmt.Sprintf("  ✅ RSI: Period %d, Overbought %.1f, Oversold %.1f\n",
			config.RSI.Period, config.RSI.Overbought, config.RSI.Oversold)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ RSI: DISABLED\n")
	}

	if config.MACD.Enabled {
		summary += fmt.Sprintf("  ✅ MACD: Fast %d, Slow %d, Signal %d\n",
			config.MACD.FastPeriod, config.MACD.SlowPeriod, config.MACD.SignalPeriod)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ MACD: DISABLED\n")
	}

	if config.Volume.Enabled {
		summary += fmt.Sprintf("  ✅ Volume: Period %d, Threshold %.0f\n",
			config.Volume.Period, config.Volume.VolumeThreshold)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Volume: DISABLED\n")
	}

	if config.Trend.Enabled {
		summary += fmt.Sprintf("  ✅ Trend: Short MA %d, Long MA %d\n",
			config.Trend.ShortMA, config.Trend.LongMA)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Trend: DISABLED\n")
	}

	if config.SupportResistance.Enabled {
		summary += fmt.Sprintf("  ✅ Support/Resistance: Period %d, Threshold %.1f%%\n",
			config.SupportResistance.Period, config.SupportResistance.Threshold*100)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Support/Resistance: DISABLED\n")
	}

	if config.Ichimoku.Enabled {
		summary += fmt.Sprintf("  ✅ Ichimoku: Tenkan %d, Kijun %d, Senkou %d, Disp %d\n",
			config.Ichimoku.TenkanPeriod, config.Ichimoku.KijunPeriod,
			config.Ichimoku.SenkouPeriod, config.Ichimoku.Displacement)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Ichimoku: DISABLED\n")
	}

	if config.MFI.Enabled {
		summary += fmt.Sprintf("  ✅ Reverse-MFI: Period %d, Overbought %.1f, Oversold %.1f\n",
			config.MFI.Period, config.MFI.Overbought, config.MFI.Oversold)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Reverse-MFI: DISABLED\n")
	}

	if config.BollingerBands.Enabled {
		summary += fmt.Sprintf("  ✅ Bollinger Bands: Period %d, StdDev %.1f, Upper %.2f, Lower %.2f\n",
			config.BollingerBands.Period, config.BollingerBands.StandardDev,
			config.BollingerBands.OverboughtStd, config.BollingerBands.OversoldStd)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Bollinger Bands: DISABLED\n")
	}

	if config.Stochastic.Enabled {
		summary += fmt.Sprintf("  ✅ Stochastic: K-Period %d, D-Period %d, Slow %d, OB %.0f, OS %.0f\n",
			config.Stochastic.KPeriod, config.Stochastic.DPeriod, config.Stochastic.SlowPeriod,
			config.Stochastic.Overbought, config.Stochastic.Oversold)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Stochastic: DISABLED\n")
	}

	if config.WilliamsR.Enabled {
		summary += fmt.Sprintf("  ✅ Williams %%R: Period %d, OB %.0f, OS %.0f, FastResp %v\n",
			config.WilliamsR.Period, config.WilliamsR.Overbought, config.WilliamsR.Oversold,
			config.WilliamsR.FastResponse)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Williams %%R: DISABLED\n")
	}

	if config.PinBar.Enabled {
		summary += fmt.Sprintf("  ✅ Pin Bar: WickRatio %.1f, BodyRatio %.2f, TrendConf %v\n",
			config.PinBar.MinWickRatio, config.PinBar.MaxBodyRatio, config.PinBar.TrendConfirmation)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Pin Bar: DISABLED\n")
	}

	if config.EMA.Enabled {
		summary += fmt.Sprintf("  ✅ EMA: Fast %d, Slow %d, Signal %d, Trend %d\n",
			config.EMA.FastPeriod, config.EMA.SlowPeriod, config.EMA.SignalPeriod, config.EMA.TrendPeriod)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ EMA: DISABLED\n")
	}

	if config.ElliottWave.Enabled {
		summary += fmt.Sprintf("  ✅ Elliott Wave: MinWave %d, FibTol %.2f, ImpulseBoost %.1f\n",
			config.ElliottWave.MinWaveLength, config.ElliottWave.FibonacciTolerance, config.ElliottWave.ImpulseBoost)
		enabledCount++
	} else {
		summary += fmt.Sprintf("  ❌ Elliott Wave: DISABLED\n")
	}

	summary += fmt.Sprintf("══════════════════════════════════════\n")
	summary += fmt.Sprintf("🎯 Active Indicators: %d/13\n", enabledCount)
	summary += fmt.Sprintf("📊 Min Confidence: %.1f%%\n", config.MinConfidence*100)
	summary += fmt.Sprintf("══════════════════════════════════════\n")

	return summary
}

// ConfigManager manages configuration loading and saving
type ConfigManager struct {
	filename string
	config   Config
}

// NewConfigManager creates a new configuration manager
func NewConfigManager(filename string) *ConfigManager {
	return &ConfigManager{
		filename: filename,
		config:   DefaultConfig(),
	}
}

// Load loads the configuration from file
func (cm *ConfigManager) Load() error {
	config, err := LoadConfig(cm.filename)
	if err != nil {
		return err
	}
	cm.config = config
	return nil
}

// Save saves the current configuration to file
func (cm *ConfigManager) Save() error {
	return SaveConfig(cm.config, cm.filename)
}

// GetConfig returns the current configuration
func (cm *ConfigManager) GetConfig() Config {
	return cm.config
}

// UpdateConfig updates the configuration
func (cm *ConfigManager) UpdateConfig(config Config) error {
	if err := ValidateConfig(config); err != nil {
		return err
	}
	cm.config = config
	return nil
}

// UpdateSymbol updates the trading symbol
func (cm *ConfigManager) UpdateSymbol(symbol string) error {
	if symbol == "" {
		return fmt.Errorf("symbol cannot be empty")
	}
	cm.config.Symbol = symbol
	return nil
}

// UpdateMinConfidence updates the minimum confidence threshold
func (cm *ConfigManager) UpdateMinConfidence(confidence float64) error {
	if confidence < 0 || confidence > 1 {
		return fmt.Errorf("confidence must be between 0 and 1")
	}
	cm.config.MinConfidence = confidence
	return nil
}

// GetSummary returns a configuration summary
func (cm *ConfigManager) GetSummary() string {
	return GetConfigSummary(cm.config)
}

// EnableIndicator enables a specific indicator by name
func (cm *ConfigManager) EnableIndicator(indicatorName string) error {
	switch indicatorName {
	case "rsi":
		cm.config.RSI.Enabled = true
	case "macd":
		cm.config.MACD.Enabled = true
	case "volume":
		cm.config.Volume.Enabled = true
	case "trend":
		cm.config.Trend.Enabled = true
	case "support_resistance", "sr":
		cm.config.SupportResistance.Enabled = true
	case "ichimoku":
		cm.config.Ichimoku.Enabled = true
	case "mfi", "reverse_mfi":
		cm.config.MFI.Enabled = true
	case "bollinger_bands", "bb":
		cm.config.BollingerBands.Enabled = true
	case "all":
		cm.config.RSI.Enabled = true
		cm.config.MACD.Enabled = true
		cm.config.Volume.Enabled = true
		cm.config.Trend.Enabled = true
		cm.config.SupportResistance.Enabled = true
		cm.config.Ichimoku.Enabled = true
		cm.config.MFI.Enabled = true
		cm.config.BollingerBands.Enabled = true
	default:
		return fmt.Errorf("unknown indicator: %s. Available: rsi, macd, volume, trend, support_resistance, ichimoku, mfi, bollinger_bands, all", indicatorName)
	}
	return nil
}

// DisableIndicator disables a specific indicator by name
func (cm *ConfigManager) DisableIndicator(indicatorName string) error {
	switch indicatorName {
	case "rsi":
		cm.config.RSI.Enabled = false
	case "macd":
		cm.config.MACD.Enabled = false
	case "volume":
		cm.config.Volume.Enabled = false
	case "trend":
		cm.config.Trend.Enabled = false
	case "support_resistance", "sr":
		cm.config.SupportResistance.Enabled = false
	case "ichimoku":
		cm.config.Ichimoku.Enabled = false
	case "mfi", "reverse_mfi":
		cm.config.MFI.Enabled = false
	case "bollinger_bands", "bb":
		cm.config.BollingerBands.Enabled = false
	case "all":
		cm.config.RSI.Enabled = false
		cm.config.MACD.Enabled = false
		cm.config.Volume.Enabled = false
		cm.config.Trend.Enabled = false
		cm.config.SupportResistance.Enabled = false
		cm.config.Ichimoku.Enabled = false
		cm.config.MFI.Enabled = false
		cm.config.BollingerBands.Enabled = false
	default:
		return fmt.Errorf("unknown indicator: %s. Available: rsi, macd, volume, trend, support_resistance, ichimoku, mfi, bollinger_bands, all", indicatorName)
	}
	return nil
}

// ToggleIndicator toggles a specific indicator on/off
func (cm *ConfigManager) ToggleIndicator(indicatorName string) error {
	switch indicatorName {
	case "rsi":
		cm.config.RSI.Enabled = !cm.config.RSI.Enabled
	case "macd":
		cm.config.MACD.Enabled = !cm.config.MACD.Enabled
	case "volume":
		cm.config.Volume.Enabled = !cm.config.Volume.Enabled
	case "trend":
		cm.config.Trend.Enabled = !cm.config.Trend.Enabled
	case "support_resistance", "sr":
		cm.config.SupportResistance.Enabled = !cm.config.SupportResistance.Enabled
	case "ichimoku":
		cm.config.Ichimoku.Enabled = !cm.config.Ichimoku.Enabled
	case "mfi", "reverse_mfi":
		cm.config.MFI.Enabled = !cm.config.MFI.Enabled
	case "bollinger_bands", "bb":
		cm.config.BollingerBands.Enabled = !cm.config.BollingerBands.Enabled
	default:
		return fmt.Errorf("unknown indicator: %s. Available: rsi, macd, volume, trend, support_resistance, ichimoku, mfi, bollinger_bands", indicatorName)
	}
	return nil
}

// GetEnabledIndicators returns a list of currently enabled indicators
func (cm *ConfigManager) GetEnabledIndicators() []string {
	var enabled []string
	if cm.config.RSI.Enabled {
		enabled = append(enabled, "RSI")
	}
	if cm.config.MACD.Enabled {
		enabled = append(enabled, "MACD")
	}
	if cm.config.Volume.Enabled {
		enabled = append(enabled, "Volume")
	}
	if cm.config.Trend.Enabled {
		enabled = append(enabled, "Trend")
	}
	if cm.config.SupportResistance.Enabled {
		enabled = append(enabled, "Support/Resistance")
	}
	if cm.config.Ichimoku.Enabled {
		enabled = append(enabled, "Ichimoku")
	}
	if cm.config.MFI.Enabled {
		enabled = append(enabled, "Reverse-MFI")
	}
	if cm.config.BollingerBands.Enabled {
		enabled = append(enabled, "Bollinger Bands")
	}
	return enabled
}
