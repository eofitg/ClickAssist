package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

type Config struct {
	Hotkey  int    `yaml:"hotkey"` // Changed to int for key code
	Enabled bool   `yaml:"enabled"`
	DelayMS int    `yaml:"delay_ms"`
	Check   string `yaml:"check"`  // left / right
	Target  string `yaml:"target"` // left / right
}

var cfg Config
var running = false

// Load configuration from config.yml or create default if not exist
func loadConfig() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	configPath := filepath.Join(dir, "config.yml")

	// If config.yml does not exist, create a default one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := Config{
			Hotkey:  164, // Default to ALT key code for Windows
			Enabled: true,
			DelayMS: 150,
			Check:   "left",
			Target:  "right",
		}
		data, _ := yaml.Marshal(&defaultCfg)
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			log.Fatalf("Failed to create default config.yml: %v", err)
		}
		fmt.Println("No config.yml found. Created default config.yml")
		fmt.Println("To find your key code, run: go run main.go -debug-keys")
		cfg = defaultCfg
		return
	}

	// Otherwise load from existing file
	data, err := os.ReadFile(configPath)
	if err != nil {
		log.Fatalf("Failed to read config.yml: %v", err)
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		log.Fatalf("Failed to parse config.yml: %v", err)
	}
	fmt.Printf("Config loaded: Hotkey=%d, Enabled=%t, DelayMS=%d, Check=%s, Target=%s\n",
		cfg.Hotkey, cfg.Enabled, cfg.DelayMS, cfg.Check, cfg.Target)
}

// Debug function to show key codes
func debugKeys() {
	fmt.Println("Press any key to see its key code. Press ESC to exit.")

	evChan := hook.Start()
	defer hook.End()

	for ev := range evChan {
		if ev.Kind == hook.KeyDown {
			fmt.Printf("Key pressed: Code=%d, Char='%c'\n", ev.Rawcode, ev.Keychar)
			if ev.Rawcode == 1 { // ESC key
				break
			}
		}
	}
}

// Simulate mouse click
func clickMouse(target string) {
	if target == "left" {
		robotgo.Click("left", false)
	} else if target == "right" {
		robotgo.Click("right", false)
	}
}

func main() {
	// Check for debug flag
	if len(os.Args) > 1 && os.Args[1] == "-debug-keys" {
		debugKeys()
		return
	}

	loadConfig()

	// Capture interrupt signal for safe exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Press key with code %d to toggle feature ON/OFF\n", cfg.Hotkey)
	fmt.Printf("Configuration: Check %s mouse button, Target %s mouse button\n", cfg.Check, cfg.Target)
	fmt.Println("To find key codes, run: go run main.go -debug-keys")

	// Use gohook to listen for events
	evChan := hook.Start()
	defer hook.End()

	go func() {
		for ev := range evChan {
			// Listen for hotkey press to toggle state
			if ev.Kind == hook.KeyDown && ev.Rawcode == uint16(cfg.Hotkey) {
				running = !running
				fmt.Println("Feature", map[bool]string{true: "ON", false: "OFF"}[running])
			}

			// Listen for mouse click events
			if running && cfg.Enabled {
				if cfg.Check == "left" && ev.Kind == hook.MouseDown && ev.Button == 1 {
					go func() {
						time.Sleep(time.Duration(cfg.DelayMS) * time.Millisecond)
						clickMouse(cfg.Target)
					}()
				} else if cfg.Check == "right" && ev.Kind == hook.MouseDown && ev.Button == 2 {
					go func() {
						time.Sleep(time.Duration(cfg.DelayMS) * time.Millisecond)
						clickMouse(cfg.Target)
					}()
				}
			}
		}
	}()

	fmt.Println("Program is running. Press Ctrl+C to exit.")
	<-c
	fmt.Println("\nExiting...")
}
