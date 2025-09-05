package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

type Config struct {
	Hotkey  string `yaml:"hotkey"`
	Enabled bool   `yaml:"enabled"`
	DelayMS int    `yaml:"delay_ms"`
	Check   string `yaml:"check"`  // left / right
	Target  string `yaml:"target"` // left / right
}

var cfg Config
var running = false

// Cross-platform key code mapping
var keyMap = map[string]uint16{
	// Modifier keys
	"alt":     56, // Windows
	"option":  58, // macOS
	"ctrl":    29,
	"control": 29,
	"shift":   42,
	"cmd":     3675, // macOS Command
	"command": 3675,
	"win":     3676, // Windows Key
	"super":   3676,

	// Alphabet keys
	"a": 30, "b": 48, "c": 46, "d": 32, "e": 18, "f": 33, "g": 34, "h": 35,
	"i": 23, "j": 36, "k": 37, "l": 38, "m": 50, "n": 49, "o": 24, "p": 25,
	"q": 16, "r": 19, "s": 31, "t": 20, "u": 22, "v": 47, "w": 17, "x": 45,
	"y": 21, "z": 44,

	// Number keys
	"0": 11, "1": 2, "2": 3, "3": 4, "4": 5, "5": 6, "6": 7, "7": 8, "8": 9, "9": 10,

	// Function keys
	"f1": 59, "f2": 60, "f3": 61, "f4": 62, "f5": 63, "f6": 64, "f7": 65, "f8": 66,
	"f9": 67, "f10": 68, "f11": 87, "f12": 88,

	// Other keys
	"space":     57,
	"enter":     28,
	"return":    28,
	"esc":       1,
	"escape":    1,
	"tab":       15,
	"caps":      58,
	"backspace": 14,
	"delete":    83,
	"insert":    82,
	"home":      71,
	"end":       79,
	"pageup":    73,
	"pagedown":  81,
}

func init() {
	// Adjust key codes based on operating system
	if runtime.GOOS == "darwin" { // macOS
		keyMap["alt"] = 58
		keyMap["ctrl"] = 59
		keyMap["control"] = 59
		keyMap["cmd"] = 55
		keyMap["command"] = 55
	} else if runtime.GOOS == "linux" {
		// Linux key code adjustments
		keyMap["alt"] = 64
		keyMap["ctrl"] = 37
		keyMap["control"] = 37
		keyMap["super"] = 133
	}
}

// Load configuration from config.yml or create default if not exist
func loadConfig() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	configPath := filepath.Join(dir, "config.yml")

	// If config.yml does not exist, create a default one
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := Config{
			Hotkey:  "alt",
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
	fmt.Printf("Config loaded: %+v\n", cfg)
}

// Get key code from configuration string
func getKeyCode(keyName string) (uint16, error) {
	keyName = strings.ToLower(keyName)
	if code, exists := keyMap[keyName]; exists {
		return code, nil
	}
	return 0, fmt.Errorf("unknown key: %s", keyName)
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
	loadConfig()

	// Get hotkey code from configuration
	hotkeyCode, err := getKeyCode(cfg.Hotkey)
	if err != nil {
		log.Fatalf("Invalid hotkey '%s': %v", cfg.Hotkey, err)
	}

	// Capture interrupt signal for safe exit
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Press %s to toggle feature ON/OFF\n", strings.ToUpper(cfg.Hotkey))
	fmt.Printf("Configuration: Check %s mouse button, Target %s mouse button\n", cfg.Check, cfg.Target)

	// Use gohook to listen for events
	evChan := hook.Start()
	defer hook.End()

	go func() {
		for ev := range evChan {
			// Listen for hotkey press to toggle state
			if ev.Kind == hook.KeyDown && ev.Rawcode == hotkeyCode {
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
