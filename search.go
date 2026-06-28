package main

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

type SearchResult struct {
	Name       string `json:"name"`
	Zone       string `json:"zone"`
	Desc       string `json:"desc"`
	MatchedAlt string `json:"matchedAlt,omitempty"`
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
	parts := strings.SplitN(seg, " as ", 2)
	if len(parts) == 2 {
		return strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1])
	}
	return strings.TrimSpace(seg), ""
}

// resolvePathSegments fuzzy-matches path segments and returns the resolved ZoneInfo.
func resolvePathSegments(segments []string) []ZoneInfo {
	var zones []ZoneInfo

	for _, seg := range segments {
		// Treat + as space and +as+ as alias separator, before URL decoding
		// so that actual %2B can still decode to a literal + if desired.
		seg = strings.ReplaceAll(seg, "+as+", " as ")
		seg = strings.ReplaceAll(seg, "+", " ")

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
