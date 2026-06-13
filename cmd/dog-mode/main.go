package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

)
const (
	SECC = 30
	MISS_TARGET = 3
	WEAK_TARGET = 6
	MISS_OR_WEAK_TARGET = 10
)
var (
	CheckInterval  = SECC * time.Second
	missedInARow = 0
	weakInARow = 0
	missOrWeak = 0 
)

func main() {
	fmt.Println("=== Tesla Dog Mode Beacon Monitor ===")
	fmt.Printf("Scans run every %d seconds\n", SECC)
	fmt.Println("Turns ON Dog Mode if a strong iBeacon signal is detected")
	fmt.Printf("Turns OFF Dog Mode after %d consecutive misses or %d consecutive weak signals\n", MISS_TARGET, WEAK_TARGET)
	fmt.Println("Press Ctrl+C to quit\n")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Println("\nShutting down...")
		os.Exit(0)
	}()

	ticker := time.NewTicker(CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			runCheck()
		case <-sigChan:
			return
		}
	}
}

func runCheck() {
	fmt.Printf("\n--- Check at %s ---\n", time.Now().Format("15:04:05"))
	// check if the car is occupied
	userPresent, err := isUserPresent()
	if err != nil {
		fmt.Printf("Failed to check body state: %v\n", err)
		return
	}

	if userPresent {
		fmt.Println("User is present in vehicle → Skipping beacon scan")
		resetCounters()
		return
	}

	fmt.Println("No user present → Scanning for beacon...")
	
	// scan for the beacon
	cmd := exec.Command("./ibeacons")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	cmd.Run()
	exitCode := cmd.ProcessState.ExitCode()

	switch exitCode {
	case 0: // SUCCESS: Beacon is close
		fmt.Println("Beacon is close enough")
		missedInARow = 0
		weakInARow = 0

		// Check if Dog Mode is already on
		isDog, err := isCurrentlyInDogMode()
		if err != nil {
			fmt.Printf("Failed to check Dog Mode status: %v\n", err)
			return
		}

		if !isDog {
			fmt.Println("Dog Mode is OFF → Turning it ON")
			turnDogModeOn()
		} else {
			fmt.Println("Dog Mode is already ON → No action")
		}

	case 1: // Weak beacon
		weakInARow++
		missOrWeak++
		fmt.Printf("Beacon too weak (%d/%d weak, %d/%d total)\n", weakInARow, WEAK_TARGET, missOrWeak, MISS_OR_WEAK_TARGET)

		if weakInARow >= WEAK_TARGET || missOrWeak >= MISS_OR_WEAK_TARGET {
			isDog, err := isCurrentlyInDogMode()
			if err != nil {
				fmt.Printf("Failed to check Dog Mode status: %v\n", err)
				return
			}
			if isDog {
				fmt.Println("Dog Mode is ON → Turning it OFF")
				turnDogModeOff()
			} else {
				fmt.Println("Dog Mode is already OFF → No action")
			}
			resetCounters()
		}

	case 2: // Not found
		missedInARow++
		missOrWeak++
		fmt.Printf("Beacon NOT found (%d/%d misses, %d/%d total)\n", missedInARow, MISS_TARGET, missOrWeak, MISS_OR_WEAK_TARGET)

		if missedInARow >= MISS_TARGET || missOrWeak >= MISS_OR_WEAK_TARGET {
			isDog, err := isCurrentlyInDogMode()
			if err != nil {
				fmt.Printf("Failed to check Dog Mode status: %v\n", err)
				return
			}
			if isDog {
				fmt.Println("Dog Mode is ON → Turning it OFF")
				turnDogModeOff()
			} else {
				fmt.Println("Dog Mode is already OFF → No action")
			}
			resetCounters()
		}

	default:
		fmt.Printf("Unknown exit code from ibeacon: %d\n", exitCode)
	}
}

func resetCounters() {
	missedInARow = 0
	weakInARow = 0
	missOrWeak = 0
}

func isUserPresent() (bool, error) {
	cmd := exec.Command("./tesla-control", "-ble", "body-controller-state")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return false, err
	}

	userPresence, ok := data["userPresence"].(string)
	if !ok {
		return false, nil
	}

	return userPresence == "VEHICLE_USER_PRESENCE_PRESENT", nil
}

// Check current climateKeeperMode using tesla-control
func isCurrentlyInDogMode() (bool, error) {
	cmd := exec.Command("./tesla-control", "-ble", "state", "climate")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(output, &data); err != nil {
		return false, err
	}

	climateState, ok := data["climateState"].(map[string]interface{})
	if !ok {
		return false, fmt.Errorf("invalid climate state format")
	}

	keeperMode, ok := climateState["climateKeeperMode"].(map[string]interface{})
	if !ok {
		return false, nil
	}

	_, isDog := keeperMode["Dog"]
	return isDog, nil
}

// Turn Dog Mode ON using tesla-control
func turnDogModeOn() {
	cmd := exec.Command("./tesla-control", "-ble", "climate-keeper", "dog")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to turn on Dog Mode: %v\n", err)
		fmt.Println(string(output))
		return
	}

	fmt.Println("   ✅ Dog Mode turned ON successfully!")
}

func turnDogModeOff() {
	cmd := exec.Command("./tesla-control", "-ble", "climate-keeper", "off")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("Failed to turn OFF Dog Mode: %v\n", err)
		fmt.Println(string(output))
		return
	}

	fmt.Println("   ✅ Dog Mode turned OFF successfully!")
}



// TODO: 
// look at ./tesla-control -ble state drive , this gives us the locationName --> 15033 Zinnia St. if we are home, this program doesn't need to scan for iBeacons
