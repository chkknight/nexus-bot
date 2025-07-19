# Run the trading bot
run:
    go run .

# Build the trading bot
build:
    go build -o trading-bot .

# Clean built binaries
clean:
    rm -f trading-bot

# Test the trading bot
test:
    go test ./...

# Generate Swagger documentation
swagger:
    ~/go/bin/swag init

# Show available commands
help:
    @just --list 