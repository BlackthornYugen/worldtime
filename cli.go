package main

import (
	"log"
	"os"
	"time"
)

func runCLI(focusFlag string, colorFlag string, doubleSpaced bool, padding int, args []string) {
	var zones []ZoneInfo
	var resolvedLocs []*time.Location

	if len(args) == 0 {
		// Use default cities
		defaultNames := []string{"Local", "UTC", "EST", "CEST", "JST"}
		for _, name := range defaultNames {
			if name == "Local" {
				zones = append(zones, ZoneInfo{Location: time.Local, Name: "Local", FriendlyName: "Local"})
				resolvedLocs = append(resolvedLocs, time.Local)
				continue
			}
			loc, resolved, err := ResolveLocation(name)
			if err == nil {
				zones = append(zones, ZoneInfo{
					Location:     loc,
					Name:         resolved,
					FriendlyName: getFriendlyName(resolved),
				})
				resolvedLocs = append(resolvedLocs, loc)
			}
		}
	} else {
		for _, arg := range args {
			// Try fuzzy matching city database first
			if city, ok := findCity(arg); ok {
				loc, err := time.LoadLocation(city.TZ)
				if err == nil {
					zones = append(zones, ZoneInfo{
						Location:     loc,
						Name:         city.TZ,
						FriendlyName: city.Name,
					})
					resolvedLocs = append(resolvedLocs, loc)
					continue
				}
			}
			// Fall back to IANA / abbreviation resolver
			loc, resolved, err := ResolveLocation(arg)
			if err == nil {
				zones = append(zones, ZoneInfo{
					Location:     loc,
					Name:         resolved,
					FriendlyName: getFriendlyName(resolved),
				})
				resolvedLocs = append(resolvedLocs, loc)
			} else {
				log.Fatalf("Error: Could not resolve timezone or city for %q", arg)
			}
		}
	}

	if len(zones) == 0 {
		log.Fatalf("Error: No valid timezones to compare.")
	}

	// Resolve focus location
	var focusLoc *time.Location
	if focusFlag != "" {
		if city, ok := findCity(focusFlag); ok {
			loc, err := time.LoadLocation(city.TZ)
			if err == nil {
				focusLoc = loc
			}
		}
		if focusLoc == nil {
			loc, _, err := ResolveLocation(focusFlag)
			if err == nil {
				focusLoc = loc
			}
		}
		if focusLoc == nil {
			log.Fatalf("Error: Could not resolve focus timezone or city %q", focusFlag)
		}
	} else {
		focusLoc = resolvedLocs[0]
	}

	// Resolve color preference
	useColor := false
	if colorFlag == "always" {
		useColor = true
	} else if colorFlag == "never" {
		useColor = false
	} else {
		// auto: check if stdout is a character device (terminal)
		fileInfo, err := os.Stdout.Stat()
		if err == nil {
			useColor = (fileInfo.Mode() & os.ModeCharDevice) != 0
		}
	}

	renderPlaintextTimeline(os.Stdout, focusLoc, zones, useColor, doubleSpaced, padding)
}
