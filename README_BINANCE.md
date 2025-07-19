# Binance Futures API Integration

This trading bot now supports real-time data from Binance Futures API instead of simulated data.

## Setup

### 1. Get Binance API Credentials

1. Go to [Binance](https://www.binance.com) and create an account
2. Go to API Management in your account settings
3. Create a new API key with the following permissions:
   - **Enable Reading** (required for market data)
   - **Enable Futures** (required for futures data)
   - **DO NOT** enable trading permissions for safety

### 2. Configure Environment Variables (Recommended)

The safest way to provide API credentials is through environment variables:

```bash
export BINANCE_API_KEY="your_api_key_here"
export BINANCE_SECRET_KEY="your_secret_key_here"
```

### 3. Configuration File

Use the provided `pkg/bot/binance_config.json` as a template:

```json
{
  "symbol": "BTCUSDT",
  "data_provider": "binance",
  "binance": {
    "api_key": "YOUR_BINANCE_API_KEY_HERE",
    "secret_key": "YOUR_BINANCE_SECRET_KEY_HERE",
    "use_testnet": false
  }
}
```

**Note**: If you set environment variables, you can leave the API keys in the config file as placeholders. The bot will automatically use environment variables if available.

## Supported Features

### Real-time Data
- **WebSocket Streams**: Real-time kline/candlestick data
- **Multiple Timeframes**: 5m, 15m, 1h, 8h, 1d
- **Automatic Reconnection**: Handles connection drops gracefully

### Historical Data
- **REST API**: Historical kline data for backtesting
- **Configurable Periods**: Customizable lookback periods
- **Rate Limiting**: Respects Binance API rate limits

### Symbol Support
- **Futures Pairs**: BTCUSDT, ETHUSDT, etc.
- **Automatic Conversion**: Converts BTCUSD to BTCUSDT format
- **Validation**: Ensures symbol exists on Binance

## Usage

### 1. Using Environment Variables (Recommended)

```bash
# Set environment variables
export BINANCE_API_KEY="your_api_key_here"
export BINANCE_SECRET_KEY="your_secret_key_here"

# Copy the Binance config template
cp pkg/bot/binance_config.json pkg/bot/config.json

# Edit the symbol if needed
# The bot will automatically use environment variables for API keys

# Run the bot
go run main.go
```

### 2. Using Configuration File

```bash
# Copy the Binance config template
cp pkg/bot/binance_config.json pkg/bot/config.json

# Edit config.json and replace:
# - "YOUR_BINANCE_API_KEY_HERE" with your actual API key
# - "YOUR_BINANCE_SECRET_KEY_HERE" with your actual secret key

# Run the bot
go run main.go
```

### 3. API Endpoints

The bot will automatically use real Binance data when making predictions:

```bash
# Get prediction based on real Binance data
curl http://localhost:8080/api/v1/predict

# Check bot status
curl http://localhost:8080/api/v1/status
```

## Configuration Options

### Binance-specific Settings

```json
{
  "binance": {
    "api_key": "your_api_key",
    "secret_key": "your_secret_key",
    "use_testnet": false  // Set to true for testnet (limited functionality)
  },
  "data_provider": "binance"  // Can be "binance" or "sample"
}
```

### Symbol Format

- **Internal Format**: BTCUSD (as used in config)
- **Binance Format**: BTCUSDT (automatically converted)
- **Supported Pairs**: Any futures pair available on Binance

## Data Flow

1. **Initialization**: Bot loads historical data for all timeframes
2. **Real-time Updates**: WebSocket streams provide live updates
3. **Signal Generation**: Technical indicators analyze real market data
4. **API Responses**: Predictions based on actual market conditions

## Error Handling

The bot handles various error scenarios:

- **API Rate Limits**: Automatic retry with exponential backoff
- **Connection Issues**: Automatic reconnection to WebSocket
- **Invalid Symbols**: Clear error messages for unsupported pairs
- **Missing Credentials**: Helpful error messages with setup instructions

## Security Best Practices

1. **Environment Variables**: Use environment variables for API keys
2. **Read-Only Keys**: Only enable reading permissions on your API keys
3. **IP Whitelist**: Restrict API key access to specific IP addresses
4. **Regular Rotation**: Rotate API keys periodically

## Troubleshooting

### Common Issues

1. **"API key required" error**
   - Ensure environment variables are set or config file has valid keys
   - Check that API key has futures permissions enabled

2. **"Symbol not found" error**
   - Verify the symbol exists on Binance Futures
   - Check symbol format (BTCUSDT, not BTCUSD)

3. **WebSocket connection failures**
   - Check internet connection
   - Verify Binance API is not under maintenance
   - Check for rate limiting

### Debug Mode

Enable debug logging to see detailed API interactions:

```bash
export DEBUG=1
go run main.go
```

## Migration from Sample Data

To switch from sample data to Binance data:

1. Update `data_provider` in config.json to `"binance"`
2. Add your Binance API credentials
3. Restart the bot

The bot will automatically detect the configuration change and use real data.

## API Reference

- **Binance Futures API**: https://binance-docs.github.io/apidocs/futures/en/
- **WebSocket Streams**: https://binance-docs.github.io/apidocs/futures/en/#websocket-market-streams
- **Rate Limits**: https://binance-docs.github.io/apidocs/futures/en/#limits

## Support

For issues related to:
- **Binance API**: Contact Binance support
- **Trading Bot**: Check the main README.md or create an issue
- **Configuration**: Refer to the example config files 