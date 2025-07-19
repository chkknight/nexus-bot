# Trading Bot API Documentation

## Overview
A sophisticated multi-timeframe trading bot API that provides cryptocurrency price predictions using advanced technical analysis.

## Features
- **Multi-timeframe Analysis**: Analyzes 5 different timeframes (5m, 15m, 45m, 8h, 1d)
- **7 Technical Indicators**: RSI, MACD, Volume, Trend, Support/Resistance, Ichimoku Cloud, Reverse-MFI
- **Precise 5-Minute Predictions**: Predicts exact price direction 5 minutes from request time
- **Time-Specific Analysis**: Prioritizes 5-minute timeframe indicators for maximum accuracy
- **Real-time Target Tracking**: Shows prediction time and countdown to target
- **RESTful API**: Clean, well-documented REST endpoints
- **Swagger Documentation**: Interactive API documentation

## API Endpoints

### 🔮 Prediction Endpoint
```
GET /api/v1/predict
```
**Description**: Predict if price will be HIGHER/LOWER/NEUTRAL exactly 5 minutes from request time

**Enhanced Features**:
- **Time-Specific Analysis**: Focuses on 5-minute timeframe indicators for maximum accuracy
- **Prediction Target Time**: Shows exact time when prediction applies (request time + 5 minutes)
- **5-Minute Signal Analysis**: Detailed breakdown of 5-minute indicators
- **Real-Time Countdown**: Shows time remaining until prediction target

**Response Example**:
```json
{
  "symbol": "BTCUSD",
  "current_price": 50000.50,
  "prediction": "HIGHER",
  "confidence": 0.82,
  "reasoning": "5-minute analysis shows 4 buy vs 1 sell signals. Price expected to be above 50000.50 at 12:05:00",
  "timestamp": "2023-01-01T12:00:00Z",
  "prediction_time": "2023-01-01T12:05:00Z",
  "time_to_target": "5m0s",
  "five_minute_signal": "5-min indicators: 4 BUY, 1 SELL (80.0% bullish)",
  "indicators": [
    {
      "name": "RSI_5m",
      "signal": "BUY",
      "strength": 0.85,
      "timeframe": "5m"
    },
    {
      "name": "MACD_5m",
      "signal": "BUY",
      "strength": 0.78,
      "timeframe": "5m"
    }
  ]
}
```

### 📊 Status Endpoint
```
GET /api/v1/status
```
**Description**: Get detailed bot status and data availability

### 📈 Signals Endpoint
```
GET /api/v1/signals
```
**Description**: Get the latest trading signal with full indicator breakdown

### 🏥 Health Check
```
GET /api/v1/health
```
**Description**: Check API health and bot status

### 📚 API Information
```
GET /
```
**Description**: Get general API information and available endpoints

## Technical Analysis

### Indicators Used
1. **RSI (Relative Strength Index)**: 14-period, Overbought: 70, Oversold: 30
2. **MACD**: Fast: 12, Slow: 26, Signal: 9
3. **Volume Analysis**: 20-period with threshold detection
4. **Trend Analysis**: Short MA: 20, Long MA: 50
5. **Support/Resistance**: 20-period with 2% threshold
6. **Ichimoku Cloud**: Traditional Japanese settings (9, 26, 52, 26)

### Timeframes
- **Daily (1d)**: Long-term trend analysis
- **8 Hour (8h)**: Medium-term trend confirmation
- **45 Minute (45m)**: Intermediate trend analysis
- **15 Minute (15m)**: Short-term momentum
- **5 Minute (5m)**: Entry/exit timing

### Signal Aggregation
The bot uses a sophisticated multi-timeframe confluence system:
- **Daily**: 35% weight
- **8 Hour**: 25% weight
- **45 Minute**: 20% weight
- **15 Minute**: 15% weight
- **5 Minute**: 5% weight

## Usage Examples

### cURL Examples

#### Get Price Prediction
```bash
curl -X GET "http://localhost:8080/api/v1/predict" \
  -H "accept: application/json"
```

#### Check Bot Status
```bash
curl -X GET "http://localhost:8080/api/v1/status" \
  -H "accept: application/json"
```

#### Health Check
```bash
curl -X GET "http://localhost:8080/api/v1/health" \
  -H "accept: application/json"
```

### JavaScript Example
```javascript
async function getPrediction() {
  try {
    const response = await fetch('http://localhost:8080/api/v1/predict');
    const data = await response.json();
    
    console.log(`Prediction: ${data.prediction}`);
    console.log(`Confidence: ${(data.confidence * 100).toFixed(1)}%`);
    console.log(`Current Price: $${data.current_price.toFixed(2)}`);
    console.log(`Reasoning: ${data.reasoning}`);
    
    return data;
  } catch (error) {
    console.error('Error fetching prediction:', error);
  }
}

// Call the function
getPrediction();
```

### Python Example
```python
import requests
import json

def get_prediction():
    try:
        response = requests.get('http://localhost:8080/api/v1/predict')
        data = response.json()
        
        print(f"Prediction: {data['prediction']}")
        print(f"Confidence: {data['confidence']*100:.1f}%")
        print(f"Current Price: ${data['current_price']:.2f}")
        print(f"Reasoning: {data['reasoning']}")
        
        return data
    except Exception as e:
        print(f"Error: {e}")

# Call the function
prediction = get_prediction()
```

## Response Codes

- `200 OK`: Successful request
- `404 Not Found`: Resource not found
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Bot is initializing or not ready

## Rate Limiting

Currently, there are no rate limits implemented, but it's recommended to:
- Not exceed 1 request per second for predictions
- Use the health endpoint for monitoring
- Cache responses when appropriate

## Error Handling

All errors are returned in the following format:
```json
{
  "error": "Description of the error"
}
```

## Development

### Starting the Server
```bash
go run .
```

### Generating Swagger Documentation
```bash
swag init
```

### Running Tests
```bash
go test
```

## Configuration

The bot uses a JSON configuration file (`config.json`) for all indicator parameters. Default settings are optimized for cryptocurrency trading but can be adjusted based on your requirements.

## Support

For technical support or questions about the API, please refer to the inline documentation or contact the development team. 