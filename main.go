package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"time"
)

//go:embed web/*
var webFS embed.FS

//go:embed cities.json
var citiesJSON []byte

type City struct {
	Name    string   `json:"name"`
	ASCII   string   `json:"ascii"`
	Alt     []string `json:"alt"`
	Country string   `json:"country"`
	Region  string   `json:"region"`
	Pop     int      `json:"pop"`
	TZ      string   `json:"tz"`
}

type ZoneInfo struct {
	Location     *time.Location `json:"-"`
	Name         string         `json:"tz"`           // IANA Name
	FriendlyName string         `json:"friendlyName"` // Custom Display Name
	SearchTerm   string         `json:"searchTerm"`   // Base name for clean URLs
}

var cities []City

var Version string = "dev"
var RepoURL string = "unknown"

func init() {
	// Parse the embedded cities dataset
	if err := json.Unmarshal(citiesJSON, &cities); err != nil {
		log.Fatalf("Failed to parse embedded cities database: %v", err)
	}
}

func main() {
	var serverFlag bool
	var focusFlag string
	var colorFlag string
	var compactFlag bool
	var doubleSpacedFlag bool
	var paddingFlag int

	flag.BoolVar(&serverFlag, "server", false, "Start HTTP server mode")
	flag.BoolVar(&serverFlag, "s", false, "Start HTTP server mode (shorthand)")
	flag.StringVar(&focusFlag, "focus", "", "Timezone or city name to focus the comparison grid around")
	flag.StringVar(&focusFlag, "f", "", "Timezone or city name to focus the comparison grid around (shorthand)")
	flag.StringVar(&colorFlag, "color", "auto", "ANSI color output: auto (default), always, never")
	flag.StringVar(&colorFlag, "c", "auto", "ANSI color output (shorthand)")
	flag.BoolVar(&compactFlag, "compact", false, "Enable compact mode (equivalent to --double-spaced=false --padding=0)")
	flag.BoolVar(&doubleSpacedFlag, "double-spaced", false, "Double space between rows")
	flag.BoolVar(&doubleSpacedFlag, "d", false, "Double space between rows (shorthand)")
	flag.IntVar(&paddingFlag, "padding", 0, "Padding size around hour numbers")
	flag.IntVar(&paddingFlag, "p", 0, "Padding size around hour numbers (shorthand)")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage of worldtime:\n")
		fmt.Fprintf(os.Stderr, "  worldtime [flags] [city1 city2 ...]\n\n")
		fmt.Fprintf(os.Stderr, "Flags:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  worldtime                          # Starts the web server\n")
		fmt.Fprintf(os.Stderr, "  worldtime Waterloo Bangalore       # Compares Waterloo and Bangalore\n")
		fmt.Fprintf(os.Stderr, "  worldtime -f Bangalore Waterloo Bangalore  # Focuses Bangalore\n")
	}

	flag.Parse()

	// If no flags/arguments are passed, or if the server flag is explicitly set, run as a server.
	// Otherwise, run in CLI mode.
	if len(os.Args) == 1 || serverFlag {
		runServer()
		return
	}

	doubleSpaced := doubleSpacedFlag
	padding := paddingFlag
	if compactFlag {
		doubleSpaced = false
		padding = 0
	}

	runCLI(focusFlag, colorFlag, doubleSpaced, padding, flag.Args())
}
