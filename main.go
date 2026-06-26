package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"net/url"
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
	Location     *time.Location
	Name         string // IANA Name
	FriendlyName string // Custom Display Name
}

var cities []City

func init() {
	// Parse the embedded cities dataset
	if err := json.Unmarshal(citiesJSON, &cities); err != nil {
		log.Fatalf("Failed to parse embedded cities database: %v", err)
	}
}

func main() {
	// Prepare static file server
	subFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatalf("Failed to load embedded web assets: %v", err)
	}
	fileServer := http.FileServer(http.FS(subFS))

	// Main request handler
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		// Serve static assets directly
		if path == "/style.css" || path == "/app.js" || path == "/favicon.ico" {
			fileServer.ServeHTTP(w, r)
			return
		}

		// Parse query parameters
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

		// 2. If no query parameters but path contains segments, perform fuzzy lookup and redirect
		pathSegments := parsePathSegments(path)
		if len(pathSegments) > 0 {
			redirectWithResolvedZones(w, r, pathSegments)
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

	port := "8080"
	fmt.Printf("WorldTime server running on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// isCurl detects if the request is coming from curl or prefers plain text.
func isCurl(r *http.Request) bool {
	ua := r.UserAgent()
	if strings.HasPrefix(ua, "curl/") || strings.Contains(ua, "curl") {
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

// redirectWithResolvedZones fuzzy-matches path segments and redirects to root query format.
func redirectWithResolvedZones(w http.ResponseWriter, r *http.Request, segments []string) {
	params := url.Values{}

	for _, seg := range segments {
		// Unescape path segment
		unescaped, err := url.PathUnescape(seg)
		if err != nil {
			unescaped = seg
		}

		// 1. Try fuzzy matching city database first
		if city, ok := findCity(unescaped); ok {
			params.Add("tz", city.TZ)
			params.Add("friendlyName", city.Name)
			continue
		}

		// 2. Fall back to standard IANA / abbreviation mapping
		if loc, resolved, err := ResolveLocation(unescaped); err == nil {
			params.Add("tz", loc.String())
			params.Add("friendlyName", getFriendlyName(resolved))
			continue
		}
	}

	// If no zones were resolved, return a 400 Bad Request
	if len(params["tz"]) == 0 {
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintln(w, "Error: Could not resolve any timezones or cities from path.")
		return
	}

	// Redirect to root path with query parameters
	redirectURL := "/?" + params.Encode()
	http.Redirect(w, r, redirectURL, http.StatusFound)
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	renderPlaintextTimeline(w, zones)
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

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	renderPlaintextTimeline(w, zones)
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

// renderPlaintextTimeline prints the table to the writer.
func renderPlaintextTimeline(w http.ResponseWriter, zones []ZoneInfo) {
	// Base time is the current time in the first timezone
	now := time.Now().In(zones[0].Location)
	nowHour := now.Round(time.Hour)

	hoursWindow := 11
	offsetStart := -5

	fmt.Fprintf(w, "World Time Comparison — %s\n", now.Format("Monday, Jan 2, 2006"))
	fmt.Fprintln(w, strings.Repeat("—", 100))

	for _, z := range zones {
		nowInZone := time.Now().In(z.Location)
		dateStr := nowInZone.Format("Jan 02")

		// Get current offset string
		_, offsetSec := nowInZone.Zone()
		offsetHours := float64(offsetSec) / 3600.0
		var offsetStr string
		if offsetSec == 0 {
			offsetStr = "UTC"
		} else {
			sign := "+"
			if offsetHours < 0 {
				sign = "-"
				offsetHours = -offsetHours
			}
			if offsetHours == float64(int(offsetHours)) {
				offsetStr = fmt.Sprintf("UTC%s%d", sign, int(offsetHours))
			} else {
				offsetStr = fmt.Sprintf("UTC%s%.1f", sign, offsetHours)
			}
		}

		// Format left label column using custom FriendlyName
		labelText := fmt.Sprintf("%s (%s) %s", z.FriendlyName, offsetStr, dateStr)
		if len(labelText) < 32 {
			labelText = labelText + strings.Repeat(" ", 32-len(labelText))
		} else if len(labelText) > 32 {
			labelText = labelText[:29] + "..."
		}

		fmt.Fprintf(w, "%s │", labelText)

		for i := 0; i < hoursWindow; i++ {
			offset := offsetStart + i
			tTarget := nowHour.Add(time.Duration(offset) * time.Hour).In(z.Location)

			isCurrent := (offset == 0)
			cell := formatCell(tTarget, isCurrent, nowInZone)
			fmt.Fprintf(w, "%s│", cell)
		}
		fmt.Fprintln(w, "\n")
	}
}

// formatCell formats the time for the ASCII grid cell.
func formatCell(tCell time.Time, isCurrent bool, baseDate time.Time) string {
	timeStr := tCell.Format("15:04")

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

	if isCurrent {
		return fmt.Sprintf("[%s]", cellContent)
	}

	if dayDiff != 0 {
		return " " + cellContent
	}
	return " " + cellContent + " "
}
