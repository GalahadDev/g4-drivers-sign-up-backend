package vision

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// plateRegex matches NY-style license plates:
//   - Standard:  ABC1234, AB12, ABC12A
//   - TLC:       T123456C (T + 5-6 digits + optional letter)
//   - Numeric-first: 4L22B
var plateRegex = regexp.MustCompile(`\b([A-Z]{1,3}[0-9]{1,4}[A-Z]{0,2}|T[0-9]{5,6}[A-Z]?|[0-9]{1,2}[A-Z]{1,3}[0-9]{0,4})\b`)

// inspectionPlateRegex matches the "Plate#:XXXX" label printed on NYS inspection stickers.
var inspectionPlateRegex = regexp.MustCompile(`(?i)PLATE#\s*:?\s*([A-Z0-9]+)`)

// inspectionDateRegex matches the MM-YY expiry printed large on NYS inspection stickers.
var inspectionDateRegex = regexp.MustCompile(`\b(\d{2})-(\d{2})\b`)

// insuranceExpiryRegex matches MM/DD/YYYY dates on NY FH-1 insurance certificates.
var insuranceExpiryRegex = regexp.MustCompile(`\b(\d{2})/(\d{2})/(\d{4})\b`)

// docExpiryRegex matches MM/DD/YYYY and MM/DD/YY dates found on NY driver documents.
var docExpiryRegex = regexp.MustCompile(`\b(\d{2})/(\d{2})/(\d{2,4})\b`)

// tlcDiamondPlateRegex matches "Plate #: T123456C" on TLC diamond stickers.
// The space between "Plate" and "#" is specific to this format.
var tlcDiamondPlateRegex = regexp.MustCompile(`(?i)PLATE\s*#\s*:?\s*([A-Z0-9]+)`)

// levenshtein returns the edit distance between a and b (case-insensitive).
func levenshtein(a, b string) int {
	ra, rb := []rune(strings.ToLower(a)), []rune(strings.ToLower(b))
	la, lb := len(ra), len(rb)
	dp := make([][]int, la+1)
	for i := range dp {
		dp[i] = make([]int, lb+1)
		dp[i][0] = i
	}
	for j := 0; j <= lb; j++ {
		dp[0][j] = j
	}
	for i := 1; i <= la; i++ {
		for j := 1; j <= lb; j++ {
			if ra[i-1] == rb[j-1] {
				dp[i][j] = dp[i-1][j-1]
			} else {
				dp[i][j] = 1 + minInt3(dp[i-1][j], dp[i][j-1], dp[i-1][j-1])
			}
		}
	}
	return dp[la][lb]
}

func minInt3(a, b, c int) int {
	if a <= b && a <= c {
		return a
	}
	if b <= c {
		return b
	}
	return c
}

// normalizeNameText uppercases the string, replaces hyphens with spaces
// (for compound names like RIVERA-GOMEZ), and strips all non-letter, non-space
// characters so OCR punctuation like "RIVERA," doesn't break word matching.
func normalizeNameText(s string) string {
	upper := strings.ToUpper(s)
	var b strings.Builder
	b.Grow(len(upper))
	for _, r := range upper {
		switch {
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r == '-' || r == '\'':
			b.WriteRune(' ') // compound names → separate tokens
		case r == ' ' || r == '\t' || r == '\n' || r == '\r':
			b.WriteRune(' ')
		// drop all other chars (commas, periods, digits in name context, etc.)
		}
	}
	return b.String()
}

// nameFuzzyMatch returns true if every word in expectedName appears somewhere
// in ocrText. Matching is order-independent and tolerates OCR noise:
//   - punctuation stripped before comparison
//   - hyphens in compound names split into separate tokens
//   - Levenshtein threshold scales with word length (≤2 for short, ≤3 for long)
func nameFuzzyMatch(ocrText, expectedName string) bool {
	if expectedName == "" {
		return true
	}
	ocrWords := strings.Fields(normalizeNameText(ocrText))
	expectedWords := strings.Fields(normalizeNameText(expectedName))
	for _, expected := range expectedWords {
		if len(expected) < 2 {
			continue
		}
		// Allow one extra edit for longer names (8+ chars) where OCR makes more mistakes.
		threshold := 2
		if len(expected) >= 8 {
			threshold = 3
		}
		found := false
		for _, ocr := range ocrWords {
			if levenshtein(expected, ocr) <= threshold {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// extractPlateNumber returns the first license-plate-like token from text.
// Returns empty string if none found.
func extractPlateNumber(text string) string {
	return plateRegex.FindString(strings.ToUpper(text))
}

// extractInspectionPlate first tries the "Plate#:XXXX" label on NYS stickers,
// then falls back to the generic plate regex.
func extractInspectionPlate(text string) string {
	if m := inspectionPlateRegex.FindStringSubmatch(strings.ToUpper(text)); len(m) > 1 {
		return m[1]
	}
	return extractPlateNumber(text)
}

// parseInspectionExpiry reads the MM-YY expiry from an NYS inspection sticker text.
// Returns expired=true if the sticker has passed its expiry month.
// Returns found=false if no MM-YY date could be parsed.
func parseInspectionExpiry(text string) (expired bool, found bool) {
	matches := inspectionDateRegex.FindStringSubmatch(text)
	if matches == nil {
		return false, false
	}
	month, err1 := strconv.Atoi(matches[1])
	year2d, err2 := strconv.Atoi(matches[2])
	if err1 != nil || err2 != nil || month < 1 || month > 12 {
		return false, false
	}
	fullYear := 2000 + year2d
	// Sticker is valid through the end of the stated month.
	expiry := time.Date(fullYear, time.Month(month+1), 1, 0, 0, 0, 0, time.UTC)
	return time.Now().UTC().After(expiry), true
}

// extractTLCDiamondPlate reads the "Plate #: XXXX" field from a TLC diamond sticker.
func extractTLCDiamondPlate(text string) string {
	if m := tlcDiamondPlateRegex.FindStringSubmatch(strings.ToUpper(text)); len(m) > 1 {
		return m[1]
	}
	return ""
}

// parseDocumentExpiry finds all MM/DD/YY or MM/DD/YYYY dates in the text and checks
// whether the latest one has passed. Handles driver licenses, TLC licenses, and
// car registrations which may use 2-digit years (e.g. 02/19/20 → 2020).
func parseDocumentExpiry(text string) (expired bool, found bool) {
	matches := docExpiryRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return false, false
	}
	var latest time.Time
	for _, m := range matches {
		month, err1 := strconv.Atoi(m[1])
		day, err2 := strconv.Atoi(m[2])
		year, err3 := strconv.Atoi(m[3])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			continue
		}
		if year < 100 {
			year += 2000
		}
		t := time.Date(year, time.Month(month), day, 23, 59, 59, 0, time.UTC)
		if t.After(latest) {
			latest = t
			found = true
		}
	}
	if !found {
		return false, false
	}
	return time.Now().UTC().After(latest), true
}

// parseInsuranceExpiry finds all MM/DD/YYYY dates in the text and checks whether
// the latest one (the expiration date) has passed. The FH-1 form prints both
// effective and expiration dates; taking the max avoids misreading the earlier one.
func parseInsuranceExpiry(text string) (expired bool, found bool) {
	matches := insuranceExpiryRegex.FindAllStringSubmatch(text, -1)
	if len(matches) == 0 {
		return false, false
	}
	var latest time.Time
	for _, m := range matches {
		month, err1 := strconv.Atoi(m[1])
		day, err2 := strconv.Atoi(m[2])
		year, err3 := strconv.Atoi(m[3])
		if err1 != nil || err2 != nil || err3 != nil {
			continue
		}
		if month < 1 || month > 12 || day < 1 || day > 31 {
			continue
		}
		t := time.Date(year, time.Month(month), day, 23, 59, 59, 0, time.UTC)
		if t.After(latest) {
			latest = t
			found = true
		}
	}
	if !found {
		return false, false
	}
	return time.Now().UTC().After(latest), true
}

// normalizePlate strips spaces and uppercases for comparison.
func normalizePlate(plate string) string {
	return strings.ToUpper(strings.ReplaceAll(plate, " ", ""))
}

// containsAny returns true if text (uppercased) contains any of the given substrings.
func containsAny(text string, substrs ...string) bool {
	upper := strings.ToUpper(text)
	for _, s := range substrs {
		if strings.Contains(upper, strings.ToUpper(s)) {
			return true
		}
	}
	return false
}
