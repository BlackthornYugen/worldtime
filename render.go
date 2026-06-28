package main

import (
	"fmt"
	"io"
	"strings"
	"time"
)

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
			fmt.Fprintln(w, "")
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
