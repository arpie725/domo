package main

import (
	"fmt"
	"os"
	"time"

	"tinygo.org/x/bluetooth"
)

const (
	TargetName    = "BCPro_207369"
	RSSITreshold  = -78
	ScanDuration  = 8 * time.Second
)

func main() {
	adapter := bluetooth.DefaultAdapter
	if err := adapter.Enable(); err != nil {
		fmt.Printf("Could not enable Bluetooth: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Starting beacon scan...")

	found := false
	bestRSSI := -999
	startTime := time.Now()

	// Run scan with timeout control
	done := make(chan bool, 1)

	go func() {
		adapter.Scan(func(a *bluetooth.Adapter, result bluetooth.ScanResult) {
			if result.LocalName() == TargetName {
				rssi := int(result.RSSI)
				fmt.Printf("Beacon found | RSSI: %d dBm\n", rssi)

				if rssi > bestRSSI {
					bestRSSI = rssi
				}
				if rssi > RSSITreshold {
					found = true
				}
			}
		})
		done <- true
	}()

	// Wait for scan duration
	select {
	case <-time.After(ScanDuration):
		fmt.Println("Scan timeout reached")
	case <-done:
		fmt.Println("Scan finished naturally")
	}

	elapsed := time.Since(startTime)

	// Final result
	if found {
		fmt.Printf("SUCCESS: Beacon is close enough (best RSSI: %d) | Time: %.1fs\n", bestRSSI, elapsed.Seconds())
		os.Exit(0)
	} else if bestRSSI > -999 {
		fmt.Printf("WEAK: Beacon found but too far (best RSSI: %d) | Time: %.1fs\n", bestRSSI, elapsed.Seconds())
		os.Exit(1)
	} else {
		fmt.Printf("NOT FOUND: Beacon not detected | Time: %.1fs\n", elapsed.Seconds())
		os.Exit(2)
	}
}