package main

import (
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"blueowl-ble/internal/ble"
	"blueowl-ble/internal/hardware"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	slog.Info("BlueOwl Cam System Starting...")

	hw := hardware.NewController()
	if err := hw.Init(); err != nil {
		slog.Error("Failed to initialize hardware", "err", err)
		os.Exit(1)
	}
	defer hw.Close()

	// Initialize BLE Server
	btServer := ble.NewServer(hw)
	if err := btServer.Start(); err != nil {
		slog.Error("Failed to start BLE server", "err", err)
		os.Exit(1)
	}

	slog.Info("BlueOWL Controller Ready. Press Ctrl+C to exit.")

	// Block forever until Ctrl+C (SIGINT)
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Shutting down...")
}
