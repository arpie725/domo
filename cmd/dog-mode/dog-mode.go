package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/teslamotors/vehicle-command/pkg/cli"
	"github.com/teslamotors/vehicle-command/pkg/connector/ble"
	"github.com/teslamotors/vehicle-command/pkg/vehicle"
)

func main() {
	override := flag.Bool("override", false, "Force override if needed")

	flag.Parse()

	// Initialize config
	config, err := cli.NewConfig(cli.FlagVIN | cli.FlagPrivateKey | cli.FlagBLE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create config: %v\n", err)
		os.Exit(1)
	}

	// Read TESLA_VIN from environment variable
	config.ReadFromEnvironment()

	if config.VIN == "" {
		fmt.Println("Error: TESLA_VIN environment variable is not set.")
		fmt.Println("Please set it with: export TESLA_VIN=your_vin_here")
		os.Exit(1)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("Connecting to vehicle %s via BLE...\n", config.VIN)

	_, car, err := config.Connect(ctx)
	if err != nil {
		if ble.IsAdapterError(err) {
			fmt.Fprintf(os.Stderr, "BLE Adapter error: %s\n", ble.AdapterErrorHelpMessage(err))
		} else {
			fmt.Fprintf(os.Stderr, "Connection failed: %v\n", err)
		}
		os.Exit(1)
	}

	if car == nil {
		fmt.Println("Error: Could not get vehicle connection")
		os.Exit(1)
	}
	defer car.Disconnect()

	fmt.Println("Connected successfully. Turning Dog Mode ON...")

	err = car.SetClimateKeeperMode(ctx, vehicle.ClimateKeeperModeDog, *override)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to turn on Dog Mode: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✅ Dog Mode turned ON successfully!")
}