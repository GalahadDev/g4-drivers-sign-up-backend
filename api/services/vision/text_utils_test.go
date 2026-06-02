package vision

import (
	"testing"
	"time"
)

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		name string
		a, b string
		want int
	}{
		{"identical", "john", "john", 0},
		{"one substitution (ocr 0 vs o)", "j0hn", "john", 1},
		{"two substitutions", "j0hn d0e", "john doe", 2},
		{"empty a", "", "abc", 3},
		{"empty b", "abc", "", 3},
		{"case insensitive", "JOHN", "john", 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := levenshtein(tc.a, tc.b)
			if got != tc.want {
				t.Errorf("levenshtein(%q, %q) = %d, want %d", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestNameFuzzyMatch(t *testing.T) {
	tests := []struct {
		name         string
		ocrText      string
		expectedName string
		want         bool
	}{
		{"exact match", "JOHN DOE\nLICENSE NY", "John Doe", true},
		{"ocr 0 for O", "J0HN D0E", "John Doe", true},
		{"last-name-first OCR order", "DOE JOHN\nDRIVER LICENSE", "John Doe", true},
		{"completely different name", "JANE SMITH", "John Doe", false},
		{"empty expected name always passes", "ANY TEXT", "", true},
		{"single-char words skipped", "A B C D E", "A", true},
		// Punctuation-tolerance: OCR often attaches commas/periods to tokens.
		{"ocr comma attached to last name", "RIVERA, CARLOS DRIVER LICENSE", "Carlos Rivera", true},
		{"ocr period attached to first name", "CARLOS. RIVERA NY LICENSE", "Carlos Rivera", true},
		// Order independence: first-last vs last-first in the form field.
		{"user typed last-first, ocr first-last", "CARLOS RIVERA", "Rivera Carlos", true},
		{"user typed first-last, ocr last-first", "RIVERA CARLOS", "Carlos Rivera", true},
		// Compound hyphenated last name.
		{"hyphenated last name in ocr", "CARLOS RIVERA-GOMEZ LICENSE", "Carlos Rivera Gomez", true},
		{"hyphenated last name typed", "CARLOS RIVERA GOMEZ LICENSE", "Carlos Rivera-Gomez", true},
		// Long name threshold (≤3 edits for 8+ char words).
		{"long name with 3 ocr errors", "ALEJANDRO RODRIGUEZ", "Alejandro Rodrigues", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := nameFuzzyMatch(tc.ocrText, tc.expectedName)
			if got != tc.want {
				t.Errorf("nameFuzzyMatch(%q, %q) = %v, want %v",
					tc.ocrText, tc.expectedName, got, tc.want)
			}
		})
	}
}

func TestExtractPlateNumber(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"standard NY ABC1234", "PLATE: ABC1234 NEW YORK", "ABC1234"},
		{"TLC plate T123456C", "VEHICLE REG T123456C", "T123456C"},
		{"no plate in text", "SOME RANDOM TEXT WITHOUT PLATE", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractPlateNumber(tc.text)
			if got != tc.want {
				t.Errorf("extractPlateNumber(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestExtractInspectionPlate(t *testing.T) {
	tests := []struct {
		name string
		text string
		want string
	}{
		{"NYS sticker Plate# label", "NEW YORK STATE SAFETY\n11-26\nPlate#:ABC1234", "ABC1234"},
		{"Plate# with space after colon", "SAFETY\nPlate#: XYZ9999", "XYZ9999"},
		{"fallback to generic regex", "VEHICLE INSPECTION ABC1234", "ABC1234"},
		{"no plate", "SOME RANDOM TEXT", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractInspectionPlate(tc.text)
			if got != tc.want {
				t.Errorf("extractInspectionPlate(%q) = %q, want %q", tc.text, got, tc.want)
			}
		})
	}
}

func TestNormalizePlate(t *testing.T) {
	if got := normalizePlate("abc 123"); got != "ABC123" {
		t.Errorf("normalizePlate = %q, want %q", got, "ABC123")
	}
}

func TestParseInspectionExpiry(t *testing.T) {
	now := time.Now()
	futureMonth := int(now.Month()) + 2
	futureYear := now.Year()
	if futureMonth > 12 {
		futureMonth -= 12
		futureYear++
	}
	pastMonth := int(now.Month()) - 3
	pastYear := now.Year()
	if pastMonth < 1 {
		pastMonth += 12
		pastYear--
	}

	tests := []struct {
		name        string
		text        string
		wantExpired bool
		wantFound   bool
	}{
		{
			"future sticker is valid",
			formatMMYY(futureMonth, futureYear),
			false, true,
		},
		{
			"past sticker is expired",
			formatMMYY(pastMonth, pastYear),
			true, true,
		},
		{"no date in text", "NEW YORK STATE SAFETY\nNo date here", false, false},
		{"invalid month 13 is ignored", "13-25 INSPECTION", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotExpired, gotFound := parseInspectionExpiry(tc.text)
			if gotFound != tc.wantFound {
				t.Errorf("found = %v, want %v (text: %q)", gotFound, tc.wantFound, tc.text)
			}
			if gotFound && gotExpired != tc.wantExpired {
				t.Errorf("expired = %v, want %v (text: %q)", gotExpired, tc.wantExpired, tc.text)
			}
		})
	}
}

func formatMMYY(month, year int) string {
	return time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC).Format("01-06")
}
