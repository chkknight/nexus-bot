// @title Trading Bot API
// @version 1.0
// @description Multi-timeframe trading bot API for cryptocurrency price prediction
// @termsOfService http://swagger.io/terms/

// @contact.name API Support
// @contact.url http://www.swagger.io/support
// @contact.email support@swagger.io

// @license.name MIT
// @license.url https://opensource.org/licenses/MIT

// @host localhost:8080
// @BasePath /api/v1
// @schemes http

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	_ "trading-bot/docs" // Import generated docs
	"trading-bot/internal"
	"trading-bot/pkg/bot"
)

func main() {
	// Check for test command
	// TestCommand()

	fmt.Println("🚀 Multi-Timeframe Trading Bot with API")
	fmt.Println("==========================================")

	// Load configuration
	configManager := bot.NewConfigManager("config.json")
	if err := configManager.Load(); err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	config := configManager.GetConfig()

	// Display configuration summary
	fmt.Print(configManager.GetSummary())

	// Create trading bot
	bot := bot.NewTradingBot(config)

	// Setup graceful shutdown
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)

	// Start the bot
	fmt.Printf("🎯 Starting trading bot for %s...\n", config.Symbol)
	if err := bot.Start(); err != nil {
		log.Fatalf("Failed to start trading bot: %v", err)
	}

	// Create and start API server
	apiServer := internal.NewAPIServer(config, bot, "8080")

	// Start API server in a goroutine
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		if err := apiServer.StartWithContext(ctx); err != nil {
			log.Printf("API server error: %v", err)
		}
	}()

	// Display status periodically
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				status := bot.GetStatus()
				fmt.Printf("\n📈 Status Update:\n")
				fmt.Printf("   Running: %t\n", status.Running)
				fmt.Printf("   Symbol: %s\n", status.Symbol)
				fmt.Printf("   Data Available:\n")
				for tf, count := range status.DataSummary {
					ready := "❌"
					if status.ReadyStatus[tf] {
						ready = "✅"
					}
					fmt.Printf("     %s: %d candles %s\n", tf.String(), count, ready)
				}
				if status.LastSignal != nil {
					fmt.Printf("   Last Signal: %s (%.1f%% confidence)\n",
						status.LastSignal.Signal.String(),
						status.LastSignal.Confidence*100)
				}
				fmt.Printf("   Last Update: %s\n", status.LastUpdate.Format(time.RFC3339))
				fmt.Printf("   🌐 API: http://localhost:8080/api/v1/predict\n")
				fmt.Println()
			case <-ctx.Done():
				return
			}
		}
	}()

	// Wait for shutdown signal
	fmt.Println("✅ Trading bot and API server are running.")
	fmt.Println("📡 Prediction API: http://localhost:8080/api/v1/predict")
	fmt.Println("📊 Status API: http://localhost:8080/api/v1/status")
	fmt.Println("📚 Swagger docs: http://localhost:8080/swagger/index.html")
	fmt.Println("🔍 All endpoints: http://localhost:8080/")
	fmt.Println("Press Ctrl+C to stop.")
	<-signalChan

	// Graceful shutdown
	fmt.Println("\n🛑 Shutting down trading bot and API server...")

	// Cancel context to stop API server
	cancel()

	// Stop trading bot
	if err := bot.Stop(); err != nil {
		log.Printf("Error during bot shutdown: %v", err)
	}

	fmt.Println("👋 Trading bot and API server stopped. Goodbye!")
}
