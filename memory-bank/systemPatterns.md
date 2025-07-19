# System Patterns

## Architecture Overview
```
Market Data → Indicators → Signal Engine → Output
     ↓           ↓            ↓           ↓
  Data Feed → Calculations → Aggregation → Signals
```

## Package Structure (Updated)

### Core Packages
- **`pkg/bot/`**: Main trading bot logic, signal aggregation, configuration
- **`pkg/indicator/`**: **NEW** - Modular indicator implementations (RSI, MACD, Volume, Trend, Support/Resistance, Ichimoku, Reverse-MFI)

### Package Organization
- **Separation of Concerns**: Indicators separated from bot logic
- **Reusability**: Indicator package can be imported by other projects
- **Maintainability**: Each indicator in its own file (rsi.go, macd.go, etc.)
- **Type Safety**: Clean interfaces between packages with type conversion

## Key Design Patterns

### 1. Pipeline Pattern
- **Data Flow**: Market data flows through indicator calculations to signal generation
- **Benefits**: Clear separation of concerns, easy to test individual components
- **Implementation**: Channel-based pipeline in Go

### 2. Strategy Pattern
- **Indicators**: Each technical indicator as separate strategy
- **Signal Logic**: Configurable combination strategies
- **Benefits**: Easy to add new indicators and modify signal logic
- **Enhancement**: Now physically separated into individual files

### 3. Observer Pattern
- **Market Data**: Notify all indicators when new data arrives
- **Signals**: Notify subscribers when signals are generated
- **Benefits**: Loose coupling between components

### 4. Factory Pattern
- **Indicators**: Create indicator instances based on configuration
- **Data Sources**: Create appropriate data source connections
- **Benefits**: Flexible configuration and easy testing
- **Enhancement**: Package-level constructors (indicator.NewRSI, etc.)

### 5. Adapter Pattern **NEW**
- **Type Conversion**: Clean conversion between bot and indicator types
- **Interface Compliance**: Ensures compatibility across package boundaries
- **Data Transformation**: Candle and signal conversions

## Component Relationships

### Core Components
1. **DataProvider**: Handles market data input
2. **Indicator Package**: **NEW** - Modular technical indicators
3. **SignalAggregator**: Combines indicator outputs with type conversion
4. **ConfigManager**: Handles configuration and parameters

### Data Flow (Updated)
1. Market data arrives at DataProvider (bot.Candle)
2. SignalAggregator converts data to indicator.Candle
3. Each indicator calculates and outputs indicator.IndicatorSignal
4. SignalAggregator converts back to bot.IndicatorSignal
5. Final buy/sell decision generated

### Indicator Architecture
```
pkg/indicator/
├── types.go          # Shared types and interfaces
├── rsi.go           # RSI implementation
├── macd.go          # MACD implementation  
├── volume.go        # Volume analysis
├── trend.go         # Trend detection
├── support_resistance.go # S/R levels
├── ichimoku.go      # Ichimoku Cloud
└── reverse_mfi.go   # Reverse-MFI
```

## Concurrency Patterns
- **Goroutines**: Each indicator runs in separate goroutine
- **Channels**: Communication between components
- **Sync**: Coordination for signal generation
- **Context**: Graceful shutdown and timeouts

## Error Handling
- **Graceful Degradation**: Continue with available indicators if some fail
- **Retry Logic**: Handle temporary data source failures
- **Logging**: Comprehensive error logging for debugging
- **Package Isolation**: Indicator failures don't affect other components 