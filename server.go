package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

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
		rawPath := r.URL.Path
		if r.URL.RawPath != "" {
			rawPath = r.URL.RawPath
		}
		pathSegments := parsePathSegments(rawPath)
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

				doubleSpaced := false
				if dsParam := r.URL.Query().Get("doubleSpaced"); dsParam == "true" || dsParam == "1" {
					doubleSpaced = true
				}
				padding := 0
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

	// Inject custom headers for Version and Repository
	injectHeaders := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Server", fmt.Sprintf("worldtime/%s", Version))
			w.Header().Set("Link", fmt.Sprintf("<%s>; rel=\"repository\"", RepoURL))
			next.ServeHTTP(w, r)
		})
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	fmt.Printf("WorldTime server running on http://localhost:%s\n", port)
	if err := http.ListenAndServe(":"+port, injectHeaders(http.DefaultServeMux)); err != nil {
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

	doubleSpaced := false
	if dsParam := r.URL.Query().Get("doubleSpaced"); dsParam == "true" || dsParam == "1" {
		doubleSpaced = true
	}
	padding := 0
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

	doubleSpaced := false
	if dsParam := r.URL.Query().Get("doubleSpaced"); dsParam == "true" || dsParam == "1" {
		doubleSpaced = true
	}
	padding := 0
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

func handleSearchAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := r.URL.Query().Get("q")
	results := searchCities(q)
	if results == nil {
		results = []SearchResult{}
	}
	json.NewEncoder(w).Encode(results)
}
