package main

import (
	"embed"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
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
	flag.BoolVar(&doubleSpacedFlag, "double-spaced", true, "Double space between rows")
	flag.BoolVar(&doubleSpacedFlag, "d", true, "Double space between rows (shorthand)")
	flag.IntVar(&paddingFlag, "padding", 2, "Padding size around hour numbers")
	flag.IntVar(&paddingFlag, "p", 2, "Padding size around hour numbers (shorthand)")

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

func runServer() {
	// Prepare static file server
	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to load embedded web assets: %v", err)
	}
	fileServer := http.FileServer(http.FS(subFS))

	// Register API endpoints before catch-all
	http.HandleFunc("/api/search", handleSearchAPI)
	http.HandleFunc("/api/resolve", handleResolveAPI)

	// Main request handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Serve static assets directly
		if path == "/style.css" || path == "/app.js" || path == "/favicon.ico" {
			fileServer.ServeHTTP(w, r)
			return
		}

		q := r.URL.Query()
		tzParams := q["tz"]
		friendlyParams := q["friendlyName"]

		// 1. If query parameters exist, parse and display (or serve SPA HTML)
		if len(tzParams) > 0 {
			if isCurl(r) {
				handleQueryCurl(w, r, tzParams, friendlyParams)
				return
			}
			// Serve SPA HTML for browsers
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// 2. If no query parameters but path contains segments, resolve and display directly for curl, or serve SPA
		pathSegments := parsePathSegments(path)
		if len(pathSegments) > 0 {
			if isCurl(r) {
				zones := resolvePathSegments(pathSegments)
				if len(zones) == 0 {
					w.WriteHeader(http.StatusBadRequest)
					fmt.Fprintln(w, "Error: Could not resolve any timezones or cities from path.")
					return
				}
				contentType, useColor := detectPlaintextContentTypeAndColorPreference(r)
				w.Header().Set("Content-Type", contentType)
				focusLoc := getFocusLocation(r, zones)

				doubleSpaced := true
				if dsParam := r.URL.Query().Get("doubleSpaced"); dsParam == "false" || dsParam == "0" {
					doubleSpaced = false
				}
				padding := 2
				if padParam := r.URL.Query().Get("padding"); padParam != "" {
					if p, err := strconv.Atoi(padParam); err == nil && p >= 0 {
						padding = p
					}
				}
				compact := (r.URL.Query().Get("compact") == "true" || r.URL.Query().Get("compact") == "1")
				if compact {
					doubleSpaced = false
					padding = 0
				}
				renderPlaintextTimeline(w, focusLoc, zones, useColor, doubleSpaced, padding)
				return
			}
			// Serve SPA HTML for browsers
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		// 3. Root path with no parameters
		if isCurl(r) {
			handleDefaultCurl(w, r)
			return
		}

		// Serve SPA HTML for browsers
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("WorldTime server running on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// isCurl detects if the request is coming from curl, wget, powershell, httpie, or prefers plain text.
func isCurl(r *http.Request) bool {
	ua := strings.ToLower(r.UserAgent())
	if strings.Contains(ua, "curl") || strings.Contains(ua, "wget") || strings.Contains(ua, "powershell") || strings.Contains(ua, "httpie") {
		accept := r.Header.Get("Accept")
		if strings.Contains(accept, "text/html") {
			return false
		}
		return true
	}

	accept := r.Header.Get("Accept")
	if strings.Contains(accept, "text/plain") && !strings.Contains(accept, "text/html") {
		return true
	}

	return false
}

// parsePathSegments splits the URL path and returns cleaned segments.
func parsePathSegments(path string) []string {
	parts := strings.Split(path, "/")
	var segments []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			segments = append(segments, p)
		}
	}
	return segments
}

// findCity matches a query against name, country, and admin divisions.
// Since cities are sorted by population, the first match is the most major city.
func findCity(query string) (City, bool) {
	q := strings.ToLower(query)
	q = strings.ReplaceAll(q, "+", " ")
	q = strings.ReplaceAll(q, ",", " ")
	tokens := strings.Fields(q)
	if len(tokens) == 0 {
		return City{}, false
	}

	// PASS 1: Exact City Name Match (matches name or ascii exactly across all cities)
	for _, c := range cities {
		cName := strings.ToLower(c.Name)
		cASCII := strings.ToLower(c.ASCII)
		if cName == q || cASCII == q {
			return c, true
		}
	}

	// PASS 2: Exact Alt Name Match (matches alt name exactly, e.g. airport codes YKF, ALO)
	for _, c := range cities {
		for _, alt := range c.Alt {
			if strings.ToLower(alt) == q {
				return c, true
			}
		}
	}

	// PASS 3: Token City Name Match (matches tokens against city name, region, or country - EXCLUDING alternate names)
	for _, c := range cities {
		allMatched := true
		cName := strings.ToLower(c.Name)
		cASCII := strings.ToLower(c.ASCII)
		cCountry := strings.ToLower(c.Country)
		cRegion := strings.ToLower(c.Region)

		for _, token := range tokens {
			matched := false

			// Check city name / ascii at word level
			nameWords := strings.Fields(strings.ReplaceAll(cName, "-", " "))
			nameMatched := false
			for _, w := range nameWords {
				if strings.HasPrefix(w, token) {
					nameMatched = true
					break
				}
			}
			if !nameMatched {
				asciiWords := strings.Fields(strings.ReplaceAll(cASCII, "-", " "))
				for _, w := range asciiWords {
					if strings.HasPrefix(w, token) {
						nameMatched = true
						break
					}
				}
			}
			if nameMatched {
				matched = true
			} else if cCountry == token {
				matched = true
			} else if token == "us" || token == "usa" || token == "america" || token == "united states" {
				if c.Country == "US" {
					matched = true
				}
			} else if token == "ca" || token == "canada" {
				if c.Country == "CA" {
					matched = true
				}
			} else if token == "gb" || token == "uk" || token == "united kingdom" || token == "england" || token == "london" {
				if c.Country == "GB" {
					matched = true
				}
			} else if func() bool {
				regionWords := strings.Fields(strings.ReplaceAll(cRegion, "-", " "))
				for _, w := range regionWords {
					if strings.HasPrefix(w, token) {
						return true
					}
				}
				return false
			}() {
				matched = true
			}

			if !matched {
				allMatched = false
				break
			}
		}

		if allMatched {
			return c, true
		}
	}

	// PASS 4: Token Alt Name Match (includes alternate names fallback checks)
	for _, c := range cities {
		allMatched := true
		cName := strings.ToLower(c.Name)
		cASCII := strings.ToLower(c.ASCII)
		cCountry := strings.ToLower(c.Country)
		cRegion := strings.ToLower(c.Region)

		for _, token := range tokens {
			matched := false

			// Check city name / ascii at word level
			nameWords := strings.Fields(strings.ReplaceAll(cName, "-", " "))
			nameMatched := false
			for _, w := range nameWords {
				if strings.HasPrefix(w, token) {
					nameMatched = true
					break
				}
			}
			if !nameMatched {
				asciiWords := strings.Fields(strings.ReplaceAll(cASCII, "-", " "))
				for _, w := range asciiWords {
					if strings.HasPrefix(w, token) {
						nameMatched = true
						break
					}
				}
			}
			if nameMatched {
				matched = true
			} else if cCountry == token {
				matched = true
			} else if token == "us" || token == "usa" || token == "america" || token == "united states" {
				if c.Country == "US" {
					matched = true
				}
			} else if token == "ca" || token == "canada" {
				if c.Country == "CA" {
					matched = true
				}
			} else if token == "gb" || token == "uk" || token == "united kingdom" || token == "england" || token == "london" {
				if c.Country == "GB" {
					matched = true
				}
			} else if func() bool {
				regionWords := strings.Fields(strings.ReplaceAll(cRegion, "-", " "))
				for _, w := range regionWords {
					if strings.HasPrefix(w, token) {
						return true
					}
				}
				return false
			}() {
				matched = true
			} else {
				// Check alternate names (including airport codes, local scripts, etc.)
				for _, alt := range c.Alt {
					altLower := strings.ToLower(alt)
					if altLower == token {
						matched = true
						break
					}
					// Word-level match
					altLower = strings.ReplaceAll(altLower, "-", " ")
					altLower = strings.ReplaceAll(altLower, ",", " ")
					altLower = strings.ReplaceAll(altLower, "(", " ")
					altLower = strings.ReplaceAll(altLower, ")", " ")
					words := strings.Fields(altLower)
					for _, w := range words {
						if w == token {
							matched = true
							break
						}
					}
					if matched {
						break
					}
				}
			}

			if !matched {
				allMatched = false
				break
			}
		}

		if allMatched {
			return c, true
		}
	}

	return City{}, false
}

// parseSegment handles extracting custom aliases via +as+ or ` as `
func parseSegment(seg string) (string, string) {
	s := strings.ReplaceAll(seg, "+as+", " as ")
	parts := strings.SplitN(s, " as ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(s), ""
}

// resolvePathSegments fuzzy-matches path segments and returns the resolved ZoneInfo.
func resolvePathSegments(segments []string) []ZoneInfo {
	var zones []ZoneInfo

	for _, seg := range segments {
		// Unescape path segment
		unescaped, err := url.PathUnescape(seg)
		if err != nil {
			unescaped = seg
		}
		searchTerm, alias := parseSegment(unescaped)

		// 1. Try fuzzy matching city database first
		if city, ok := findCity(searchTerm); ok {
			loc, err := time.LoadLocation(city.TZ)
			if err == nil {
				// Use the exact search term they provided, formatted nicely
				friendly := strings.Title(strings.ReplaceAll(searchTerm, "_", " "))
				if alias != "" {
					friendly = alias
				}
				zones = append(zones, ZoneInfo{
					Location:     loc,
					Name:         city.TZ,
					FriendlyName: friendly,
					SearchTerm:   searchTerm,
				})
			}
			continue
		}

		// 2. Fall back to standard IANA / abbreviation mapping
		if loc, resolved, err := ResolveLocation(searchTerm); err == nil {
			friendly := getFriendlyName(resolved)
			
			// If the user typed an abbreviation like EST or a clean name without slashes, use that
			if !strings.Contains(searchTerm, "/") {
				if strings.ToUpper(searchTerm) == searchTerm {
					friendly = searchTerm
				} else {
					friendly = strings.Title(strings.ReplaceAll(searchTerm, "_", " "))
				}
			}

			if alias != "" {
				friendly = alias
			}
			zones = append(zones, ZoneInfo{
				Location:     loc,
				Name:         resolved,
				FriendlyName: friendly,
				SearchTerm:   searchTerm,
			})
			continue
		}
	}
	return zones
}

func handleResolveAPI(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Query().Get("path")
	segments := parsePathSegments(path)
	zones := resolvePathSegments(segments)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(zones)
}

// handleQueryCurl renders plaintext table for queries with explicit tz & friendlyName parameters.
func handleQueryCurl(w http.ResponseWriter, r *http.Request, tzs []string, friendlyNames []string) {
	var zones []ZoneInfo
	for i, tzStr := range tzs {
		loc, err := time.LoadLocation(tzStr)
		if err != nil {
			// Try fuzzy resolver in case the query param tz contains something else
			var resolved string
			loc, resolved, err = ResolveLocation(tzStr)
			if err != nil {
				continue
			}
			tzStr = resolved
		}

		friendly := getFriendlyName(tzStr)
		if i < len(friendlyNames) && friendlyNames[i] != "" {
			friendly = friendlyNames[i]
		}

		zones = append(zones, ZoneInfo{
			Location:     loc,
			Name:         tzStr,
			FriendlyName: friendly,
		})
	}

	if len(zones) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Error: None of the requested timezones could be loaded.")
		return
	}

	contentType, useColor := detectPlaintextContentTypeAndColorPreference(r)
	w.Header().Set("Content-Type", contentType)
	focusLoc := getFocusLocation(r, zones)

	doubleSpaced := true
	if dsParam := r.URL.Query().Get("doubleSpaced"); dsParam == "false" || dsParam == "0" {
		doubleSpaced = false
	}
	padding := 2
	if padParam := r.URL.Query().Get("padding"); padParam != "" {
		if p, err := strconv.Atoi(padParam); err == nil && p >= 0 {
			padding = p
		}
	}
	compact := (r.URL.Query().Get("compact") == "true" || r.URL.Query().Get("compact") == "1")
	if compact {
		doubleSpaced = false
		padding = 0
	}

	renderPlaintextTimeline(w, focusLoc, zones, useColor, doubleSpaced, padding)
}

// handleDefaultCurl renders table for root requests with default zones.
func handleDefaultCurl(w http.ResponseWriter, r *http.Request) {
	defaultNames := []string{"Local", "UTC", "EST", "CEST", "JST"}
	var zones []ZoneInfo
	for _, name := range defaultNames {
		if name == "Local" {
			zones = append(zones, ZoneInfo{Location: time.Local, Name: "Local", FriendlyName: "Local"})
			continue
		}
		loc, resolved, err := ResolveLocation(name)
		if err == nil {
			zones = append(zones, ZoneInfo{
				Location:     loc,
				Name:         resolved,
				FriendlyName: getFriendlyName(resolved),
			})
		}
	}

	contentType, useColor := detectPlaintextContentTypeAndColorPreference(r)
	w.Header().Set("Content-Type", contentType)
	focusLoc := getFocusLocation(r, zones)

	doubleSpaced := true
	if dsParam := r.URL.Query().Get("doubleSpaced"); dsParam == "false" || dsParam == "0" {
		doubleSpaced = false
	}
	padding := 2
	if padParam := r.URL.Query().Get("padding"); padParam != "" {
		if p, err := strconv.Atoi(padParam); err == nil && p >= 0 {
			padding = p
		}
	}
	compact := (r.URL.Query().Get("compact") == "true" || r.URL.Query().Get("compact") == "1")
	if compact {
		doubleSpaced = false
		padding = 0
	}

	renderPlaintextTimeline(w, focusLoc, zones, useColor, doubleSpaced, padding)
}

// detectPlaintextContentTypeAndColorPreference inspects the Accept header to determine:
// 1. If colors should be enabled (true if Accept contains */*, text/x-ansi, text/ansi, application/x-ansi)
// 2. The Content-Type header to send back (custom ANSI mime type if matches, otherwise text/plain)
func detectPlaintextContentTypeAndColorPreference(r *http.Request) (string, bool) {
	accept := r.Header.Get("Accept")
	useColor := false
	contentType := "text/plain; charset=utf-8"

	if strings.Contains(accept, "text/x-ansi") {
		contentType = "text/x-ansi; charset=utf-8"
		useColor = true
	} else if strings.Contains(accept, "text/ansi") {
		contentType = "text/ansi; charset=utf-8"
		useColor = true
	} else if strings.Contains(accept, "application/x-ansi") {
		contentType = "application/x-ansi; charset=utf-8"
		useColor = true
	} else if strings.Contains(accept, "*/*") {
		useColor = true
	}

	return contentType, useColor
}

// getFriendlyName returns a simplified timezone name for display.
func getFriendlyName(name string) string {
	if name == "Local" {
		return "Local"
	}
	parts := strings.Split(name, "/")
	last := parts[len(parts)-1]
	return strings.ReplaceAll(last, "_", " ")
}

// getSubdomainFocus extracts a valid timezone/city from the request host's subdomain.
func getSubdomainFocus(host string) (string, bool) {
	h := host
	if ip := strings.Index(h, ":"); ip != -1 {
		h = h[:ip]
	}
	parts := strings.Split(h, ".")
	if len(parts) <= 2 {
		return "", false
	}
	sub := strings.ToLower(parts[0])
	tokens := strings.Split(sub, "-")
	for _, token := range tokens {
		if token == "" || token == "time" || token == "www" || token == "app" || token == "dev" || token == "worldtime" {
			continue
		}
		if city, ok := findCity(token); ok {
			return city.TZ, true
		}
		if _, _, err := ResolveLocation(token); err == nil {
			return token, true
		}
	}
	return "", false
}

// getFocusLocation resolves the location that should act as the timeline's anchor.
func getFocusLocation(r *http.Request, zones []ZoneInfo) *time.Location {
	if r != nil {
		// 1. Check subdomain tokens
		if focusTZ, ok := getSubdomainFocus(r.Host); ok {
			if loc, _, err := ResolveLocation(focusTZ); err == nil {
				return loc
			}
		}

		// 2. Check query parameter "focus"
		if focusParam := r.URL.Query().Get("focus"); focusParam != "" {
			if city, ok := findCity(focusParam); ok {
				if loc, _, err := ResolveLocation(city.TZ); err == nil {
					return loc
				}
			}
			if loc, _, err := ResolveLocation(focusParam); err == nil {
				return loc
			}
		}
	}

	// 3. Fall back to the first timezone in zones
	if len(zones) > 0 {
		return zones[0].Location
	}

	return time.Local
}

// renderPlaintextTimeline prints the table to the writer.
func renderPlaintextTimeline(w io.Writer, focusLoc *time.Location, zones []ZoneInfo, useColor bool, doubleSpaced bool, padding int) {

	// Base time is the start of the current day in the focused timezone
	now := time.Now().In(focusLoc)
	focusDayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, focusLoc)

	hoursWindow := 24
	offsetStart := 0

	focusFriendly := getFriendlyName(focusLoc.String())
	if (focusFriendly == "Local" || focusFriendly == "Local Time") && len(zones) > 0 {
		focusFriendly = zones[0].FriendlyName
	}
	fmt.Fprintf(w, "World Time Comparison (Focus: %s) — %s\n", focusFriendly, now.Format("Monday, Jan 2, 2006"))

	cellWidth := 2*padding + 3
	gridLineRepeat := 34 + 24*(cellWidth+1)
	fmt.Fprintln(w, strings.Repeat("—", gridLineRepeat))

	for _, z := range zones {
		nowInZone := time.Now().In(z.Location)

		// Calculate minute offset relative to the focused zone
		minuteDiff := (nowInZone.Minute() - now.Minute() + 60) % 60
		isHalfHourOffset := (minuteDiff == 30)

		// Calculate offset relative to focusLoc
		_, focusSec := now.Zone()
		_, zoneSec := nowInZone.Zone()
		diffSec := zoneSec - focusSec
		diffHours := float64(diffSec) / 3600.0

		var offsetStr string
		if diffHours == 0 {
			offsetStr = "+0"
		} else {
			sign := "+"
			val := diffHours
			if diffHours < 0 {
				sign = "-"
				val = -diffHours
			}
			if val == float64(int(val)) {
				offsetStr = fmt.Sprintf("%s%d", sign, int(val))
			} else {
				offsetStr = fmt.Sprintf("%s%.1f", sign, val)
			}
		}

		// Format left label column showing current time and name without date
		timeStr := nowInZone.Format("15:04")
		labelText := fmt.Sprintf("%s %s (%s)", timeStr, z.FriendlyName, offsetStr)
		labelPadding := 32
		if isHalfHourOffset {
			labelPadding = 33 - (cellWidth+1)/2
		}
		if len(labelText) < labelPadding {
			labelText = labelText + strings.Repeat(" ", labelPadding-len(labelText))
		} else if len(labelText) > labelPadding {
			labelText = labelText[:labelPadding-3] + "..."
		}

		if isHalfHourOffset {
			fmt.Fprintf(w, "%s ", labelText)
		} else {
			fmt.Fprintf(w, "%s │", labelText)
		}

		localHoursWindow := hoursWindow
		localOffsetStart := offsetStart
		if isHalfHourOffset {
			localHoursWindow = hoursWindow + 1
			localOffsetStart = offsetStart - 1
		}

		for i := 0; i < localHoursWindow; i++ {
			offset := localOffsetStart + i
			tTarget := focusDayStart.Add(time.Duration(offset) * time.Hour)
			if isHalfHourOffset {
				tTarget = tTarget.Add(30 * time.Minute)
			}
			tTargetInZone := tTarget.In(z.Location)

			isCurrent := (tTargetInZone.Hour() == nowInZone.Hour() && tTargetInZone.Day() == nowInZone.Day())
			cell := formatCell(tTargetInZone, isCurrent, nowInZone, useColor, padding)

			sep := "│"
			if useColor && isCurrent {
				sep = "\x1b[31m│\x1b[0m"
			}
			if isHalfHourOffset && i == localHoursWindow-1 {
				sep = ""
			}
			fmt.Fprintf(w, "%s%s", cell, sep)
		}
		if doubleSpaced {
			fmt.Fprintln(w, "\n")
		} else {
			fmt.Fprintln(w)
		}
	}
}

// formatCell formats the time for the ASCII grid cell, optionally colored with ANSI escape codes.
func formatCell(tCell time.Time, isCurrent bool, baseDate time.Time, useColor bool, padding int) string {
	// Format hour only (no minutes)
	timeStr := tCell.Format("15")

	// Calculate day difference using calendar dates
	dayDiff := 0
	y1, m1, d1 := tCell.Date()
	y2, m2, d2 := baseDate.Date()
	date1 := time.Date(y1, m1, d1, 0, 0, 0, 0, time.UTC)
	date2 := time.Date(y2, m2, d2, 0, 0, 0, 0, time.UTC)

	if date1.After(date2) {
		dayDiff = 1
	} else if date1.Before(date2) {
		dayDiff = -1
	}

	cellContent := timeStr
	if dayDiff > 0 {
		cellContent += "+"
	} else if dayDiff < 0 {
		cellContent += "-"
	}

	var formatted string
	cellWidth := 2*padding + 3

	if isCurrent && !useColor {
		bracketed := fmt.Sprintf("[%s]", cellContent)
		if len(bracketed) >= cellWidth {
			formatted = bracketed
		} else {
			left := padding
			if len(bracketed) == 5 && padding > 0 {
				left = padding - 1
			}
			right := cellWidth - len(bracketed) - left
			if right < 0 {
				right = 0
			}
			formatted = strings.Repeat(" ", left) + bracketed + strings.Repeat(" ", right)
		}
	} else {
		if len(cellContent) >= cellWidth {
			formatted = cellContent
		} else {
			left := padding
			right := cellWidth - len(cellContent) - left
			formatted = strings.Repeat(" ", left) + cellContent + strings.Repeat(" ", right)
		}
	}

	if useColor {
		color := getHourColor(tCell)
		return color + formatted + "\x1b[0m"
	}
	return formatted
}

// getHourColor returns the ANSI escape code corresponding to the hour's category:
// - Work hours (9 AM - 5 PM): Yellow (\x1b[33m)
// - Evening hours (5 PM - 10 PM): Light Blue / Cyan (\x1b[36m)
// - Night hours (10 PM - 6 AM): Dark Blue (\x1b[34m)
// - Morning hours (6 AM - 9 AM): Light Blue / Cyan (\x1b[36m)
func getHourColor(t time.Time) string {
	h := t.Hour()
	if h >= 9 && h < 17 {
		return "\x1b[33m" // Yellow (Work hours)
	}
	if h >= 17 && h < 22 {
		return "\x1b[36m" // Light Blue (Cyan) (Evening)
	}
	if h >= 6 && h < 9 {
		return "\x1b[36m" // Light Blue (Cyan) (Morning)
	}
	return "\x1b[34m" // Dark Blue (Night)
}

type SearchResult struct {
	Name       string `json:"name"`
	Zone       string `json:"zone"`
	Desc       string `json:"desc"`
	MatchedAlt string `json:"matchedAlt,omitempty"`
}

func handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := r.URL.Query().Get("q")
	results := searchCities(q)
	if results == nil {
		results = []SearchResult{}
	}
	json.NewEncoder(w).Encode(results)
}

func searchCities(query string) []SearchResult {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return nil
	}

	var results []SearchResult
	seen := make(map[string]bool)

	// 1. Try abbreviation match first
	if ianaZone, exists := AbbreviationToIANA[q]; exists {
		res := SearchResult{
			Name: strings.ToUpper(q),
			Zone: ianaZone,
			Desc: "Timezone Abbreviation",
		}
		results = append(results, res)
		seen[ianaZone+"#"+res.Name] = true
	}

	qClean := strings.ReplaceAll(q, "+", " ")
	qClean = strings.ReplaceAll(q, ",", " ")
	tokens := strings.Fields(qClean)
	if len(tokens) == 0 {
		return results
	}

	var exactMatches []SearchResult
	var prefixMatches []SearchResult
	var fuzzyMatches []SearchResult

	for _, c := range cities {
		cName := strings.ToLower(c.Name)
		cASCII := strings.ToLower(c.ASCII)

		// Format description
		desc := c.Country
		if c.Region != "" {
			desc = c.Region + ", " + c.Country
		}

		// Pass 1: Exact matches
		isExact := cName == qClean || cASCII == qClean
		var matchedAlt string
		if !isExact {
			for _, alt := range c.Alt {
				if strings.ToLower(alt) == qClean {
					isExact = true
					matchedAlt = alt
					break
				}
			}
		}
		if isExact {
			key := c.TZ + "#" + c.Name
			if !seen[key] {
				exactMatches = append(exactMatches, SearchResult{
					Name:       c.Name,
					Zone:       c.TZ,
					Desc:       desc,
					MatchedAlt: matchedAlt,
				})
				seen[key] = true
			}
			continue
		}

		// Pass 2: Prefix matches
		isPrefix := strings.HasPrefix(cName, qClean) || strings.HasPrefix(cASCII, qClean)
		matchedAlt = ""
		if !isPrefix {
			for _, alt := range c.Alt {
				if strings.HasPrefix(strings.ToLower(alt), qClean) {
					isPrefix = true
					matchedAlt = alt
					break
				}
			}
		}
		if isPrefix {
			key := c.TZ + "#" + c.Name
			if !seen[key] {
				prefixMatches = append(prefixMatches, SearchResult{
					Name:       c.Name,
					Zone:       c.TZ,
					Desc:       desc,
					MatchedAlt: matchedAlt,
				})
				seen[key] = true
			}
			continue
		}

		// Pass 3: Token matching (fuzzy)
		allMatched := true
		cCountry := strings.ToLower(c.Country)
		cRegion := strings.ToLower(c.Region)

		for _, token := range tokens {
			matched := false
			// Check city name / ascii at word level
			nameWords := strings.Fields(strings.ReplaceAll(cName, "-", " "))
			for _, w := range nameWords {
				if strings.HasPrefix(w, token) {
					matched = true
					break
				}
			}
			if !matched {
				asciiWords := strings.Fields(strings.ReplaceAll(cASCII, "-", " "))
				for _, w := range asciiWords {
					if strings.HasPrefix(w, token) {
						matched = true
						break
					}
				}
			}
			if matched {
				// fine
			} else if cCountry == token {
				matched = true
			} else if token == "us" || token == "usa" || token == "america" {
				if c.Country == "US" { matched = true }
			} else if token == "ca" || token == "canada" {
				if c.Country == "CA" { matched = true }
			} else if token == "gb" || token == "uk" || token == "united kingdom" {
				if c.Country == "GB" { matched = true }
			} else {
				regionWords := strings.Fields(strings.ReplaceAll(cRegion, "-", " "))
				for _, w := range regionWords {
					if strings.HasPrefix(w, token) {
						matched = true
						break
					}
				}
				if !matched {
					// Fall back to checking alternate names
					for _, alt := range c.Alt {
						altLower := strings.ToLower(alt)
						if altLower == token {
							matched = true
							matchedAlt = alt
							break
						}
						altWords := strings.Fields(strings.ReplaceAll(strings.ReplaceAll(altLower, "-", " "), ",", " "))
						for _, w := range altWords {
							if strings.HasPrefix(w, token) {
								matched = true
								matchedAlt = alt
								break
							}
						}
						if matched {
							break
						}
					}
				}
			}
			if !matched {
				allMatched = false
				break
			}
		}

		if allMatched {
			key := c.TZ + "#" + c.Name
			if !seen[key] {
				fuzzyMatches = append(fuzzyMatches, SearchResult{
					Name:       c.Name,
					Zone:       c.TZ,
					Desc:       desc,
					MatchedAlt: matchedAlt,
				})
				seen[key] = true
			}
		}
	}

	results = append(results, exactMatches...)
	results = append(results, prefixMatches...)
	results = append(results, fuzzyMatches...)

	if len(results) > 10 {
		results = results[:10]
	}
	return results
}
