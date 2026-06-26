package main

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

// AbbreviationToIANA maps common timezone abbreviations to representative standard IANA locations.
// This ensures that daylight saving time transitions are automatically resolved.
var AbbreviationToIANA = map[string]string{
	"utc":   "UTC",
	"gmt":   "Etc/GMT", // or Europe/London
	"est":   "America/New_York",
	"edt":   "America/New_York",
	"et":    "America/New_York",
	"cst":   "America/Chicago",
	"cdt":   "America/Chicago",
	"ct":    "America/Chicago",
	"mst":   "America/Denver",
	"mdt":   "America/Denver",
	"mt":    "America/Denver",
	"pst":   "America/Los_Angeles",
	"pdt":   "America/Los_Angeles",
	"pt":    "America/Los_Angeles",
	"akst":  "America/Anchorage",
	"akdt":  "America/Anchorage",
	"hst":   "Pacific/Honolulu",
	"ast":   "America/Halifax",
	"adt":   "America/Halifax",
	"bst":   "Europe/London",
	"cet":   "Europe/Paris",
	"cest":  "Europe/Paris",
	"eet":   "Europe/Athens",
	"eest":  "Europe/Athens",
	"msk":   "Europe/Moscow",
	"ist":   "Asia/Kolkata",
	"jst":   "Asia/Tokyo",
	"kst":   "Asia/Seoul",
	"aest":  "Australia/Sydney",
	"aedt":  "Australia/Sydney",
	"aet":   "Australia/Sydney",
	"awst":  "Australia/Perth",
	"acst":  "Australia/Adelaide",
	"acdt":  "Australia/Adelaide",
	"nzst":  "Pacific/Auckland",
	"nzdt":  "Pacific/Auckland",
	"nzt":   "Pacific/Auckland",
	"sgt":   "Asia/Singapore",
	"hkt":   "Asia/Hong_Kong",
	"wet":   "Europe/Lisbon",
	"west":  "Europe/Lisbon",
	"cat":   "Africa/Maputo",
	"eat":   "Africa/Nairobi",
	"wat":   "Africa/Lagos",
}

// NormalizeIANAName normalizes casing for IANA timezone names (e.g., "america/new_york" -> "America/New_York").
func NormalizeIANAName(name string) string {
	parts := strings.Split(name, "/")
	for idx, part := range parts {
		// Capitalize each word segment inside the part (split by underscore)
		subParts := strings.Split(part, "_")
		for sIdx, subPart := range subParts {
			if len(subPart) > 0 {
				subParts[sIdx] = strings.ToUpper(subPart[:1]) + strings.ToLower(subPart[1:])
			}
		}
		parts[idx] = strings.Join(subParts, "_")
	}
	res := strings.Join(parts, "/")

	// Handle special cases
	// GMT offsets are often capitalized like GMT+5, Etc/GMT-8
	if strings.HasPrefix(strings.ToLower(res), "etc/gmt") {
		res = "Etc/GMT" + res[7:]
	} else if strings.HasPrefix(strings.ToLower(res), "gmt") {
		res = "GMT" + res[3:]
	}
	return res
}

// ResolveLocation resolves a string (abbreviation or IANA name) to a time.Location object.
func ResolveLocation(name string) (*time.Location, string, error) {
	nameLower := strings.ToLower(name)

	// 1. Check abbreviation map
	if iana, ok := AbbreviationToIANA[nameLower]; ok {
		loc, err := time.LoadLocation(iana)
		if err == nil {
			// Return loaded location and the friendly name requested (capitalized)
			return loc, strings.ToUpper(name), nil
		}
	}

	// 2. Try loading as-is
	loc, err := time.LoadLocation(name)
	if err == nil {
		return loc, name, nil
	}

	// 3. Normalize casing and try loading
	normalized := NormalizeIANAName(name)
	loc, err = time.LoadLocation(normalized)
	if err == nil {
		return loc, normalized, nil
	}

	return nil, "", fmt.Errorf("unknown timezone: %s", name)
}
