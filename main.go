package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/go-vgo/robotgo"
	hook "github.com/robotn/gohook"
)

type Config struct {
	Hotkey  int    `yaml:"hotkey"`
	Enabled bool   `yaml:"enabled"`
	DelayMS int    `yaml:"delay_ms"`
	Check   string `yaml:"check"`
	Target  string `yaml:"target"`
}

var cfg Config
var running int32
var mu sync.Mutex

var clickSemaphore = make(chan struct{}, 10)

func loadConfig() {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	configPath := filepath.Join(dir, "config.yml")

	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultCfg := Config{
			Hotkey:  164,
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

func clickMouse(target string) {
	mu.Lock()
	defer mu.Unlock()

	if target == "left" {
		robotgo.Click("left", false)
	} else if target == "right" {
		robotgo.Click("right", false)
	}
}

func handleClickEvent() {
	clickSemaphore <- struct{}{}
	defer func() { <-clickSemaphore }()

	time.Sleep(time.Duration(cfg.DelayMS) * time.Millisecond)

	if atomic.LoadInt32(&running) == 1 && cfg.Enabled {
		clickMouse(cfg.Target)
	}
}

func main() {
	if len(os.Args) > 1 && os.Args[1] == "-debug-keys" {
		debugKeys()
		return
	}

	loadConfig()

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	fmt.Printf("Press key with code %d to toggle feature ON/OFF\n", cfg.Hotkey)
	fmt.Printf("Configuration: Check %s mouse button, Target %s mouse button\n", cfg.Check, cfg.Target)
	fmt.Println("To find key codes, run: go run main.go -debug-keys")

	evChan := hook.Start()
	defer hook.End()

	eventQueue := make(chan hook.Event, 100)

	go func() {
		for ev := range evChan {
			select {
			case eventQueue <- ev:
			default:
				fmt.Println("Event queue full, dropping event")
			}
		}
	}()

	go func() {
		for ev := range eventQueue {
			if ev.Kind == hook.KeyDown && ev.Rawcode == uint16(cfg.Hotkey) {
				newState := atomic.LoadInt32(&running) == 0
				if newState {
					atomic.StoreInt32(&running, 1)
				} else {
					atomic.StoreInt32(&running, 0)
				}
				fmt.Println("Feature", map[bool]string{true: "ON", false: "OFF"}[newState])
				continue
			}

			if atomic.LoadInt32(&running) == 1 && cfg.Enabled {
				if (cfg.Check == "left" && ev.Kind == hook.MouseDown && ev.Button == 1) ||
					(cfg.Check == "right" && ev.Kind == hook.MouseDown && ev.Button == 2) {
					go handleClickEvent()
				}
			}
		}
	}()

	fmt.Println("Program is running. Press Ctrl+C to exit.")
	<-c
	fmt.Println("\nExiting...")
}
