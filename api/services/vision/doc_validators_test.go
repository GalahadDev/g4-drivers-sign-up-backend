package vision

import (
	"fmt"
	"testing"
	"time"
)

func TestValidateDriverLicense(t *testing.T) {
	futureDate := time.Now().AddDate(1, 0, 0).Format("01/02/2006")
	pastDate := time.Now().AddDate(-1, 0, 0).Format("01/02/2006")

	tests := []struct {
		name         string
		text         string
		expectedName string
		wantValid    bool
		wantCode     string
	}{
		{"valid license with future expiry", "NEW YORK STATE DRIVER LICENSE\nJOHN DOE\nExpires " + futureDate, "John Doe", true, ""},
		{"expired license", "NEW YORK STATE DRIVER LICENSE\nJOHN DOE\nExpires " + pastDate, "John Doe", false, "EXPIRED"},
		{"expired 2-digit year", "DRIVER LICENSE\nJOHN DOE\nExpires 12/12/21", "John Doe", false, "EXPIRED"},
		{"no date skips expiry check", "NEW YORK DRIVER LICENSE\nJOHN DOE", "John Doe", true, ""},
		{"empty text is unreadable", "", "John Doe", false, "UNREADABLE"},
		{"wrong doc type", "CAR INSURANCE POLICY", "John Doe", false, "WRONG_DOC_TYPE"},
		{"name mismatch", "NEW YORK DRIVER LICENSE\nJANE SMITH\nExpires " + futureDate, "John Doe", false, "NAME_MISMATCH"},
		{"no expected name skips name check", "DRIVER LICENSE\nANY NAME", "", true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateDriverLicense(tc.text, tc.expectedName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tc.wantValid || result.ErrorCode != tc.wantCode {
				t.Errorf("got Valid=%v ErrorCode=%q, want Valid=%v ErrorCode=%q",
					result.Valid, result.ErrorCode, tc.wantValid, tc.wantCode)
			}
		})
	}
}

func TestValidateCarRegistration(t *testing.T) {
	futureDate := time.Now().AddDate(1, 0, 0).Format("01/02/2006")
	pastDate2d := fmt.Sprintf("%02d/%02d/%02d",
		int(time.Now().AddDate(-1, 0, 0).Month()),
		time.Now().AddDate(-1, 0, 0).Day(),
		time.Now().AddDate(-1, 0, 0).Year()%100,
	)

	tests := []struct {
		name         string
		text         string
		expectedName string
		wantValid    bool
		wantPlate    string
		wantCode     string
	}{
		{"valid with plate and future expiry", "REGISTRATION\nJOHN DOE\nABC1234\nExpires " + futureDate, "John Doe", true, "ABC1234", ""},
		{"expired 2-digit year (car reg format)", "NEW YORK STATE REGISTRATION DOCUMENT\nJOHN DOE\nExpires " + pastDate2d, "John Doe", false, "", "EXPIRED"},
		{"valid no plate found", "VEHICLE REGISTRATION\nJOHN DOE", "John Doe", true, "", ""},
		{"empty text", "", "", false, "", "UNREADABLE"},
		{"wrong doc type", "DRIVER LICENSE", "John Doe", false, "", "WRONG_DOC_TYPE"},
		{"name mismatch", "REGISTRATION\nJANE SMITH\nExpires " + futureDate, "John Doe", false, "", "NAME_MISMATCH"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateCarRegistration(tc.text, tc.expectedName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tc.wantValid || result.ErrorCode != tc.wantCode {
				t.Errorf("got Valid=%v ErrorCode=%q, want Valid=%v ErrorCode=%q",
					result.Valid, result.ErrorCode, tc.wantValid, tc.wantCode)
			}
			if result.ExtractedPlate != tc.wantPlate {
				t.Errorf("ExtractedPlate = %q, want %q", result.ExtractedPlate, tc.wantPlate)
			}
		})
	}
}

func TestValidateVehicleInspection(t *testing.T) {
	now := time.Now()

	futureMonth := int(now.Month()) + 2
	futureYear := now.Year()
	if futureMonth > 12 {
		futureMonth -= 12
		futureYear++
	}
	futureDate := fmt.Sprintf("%02d-%02d", futureMonth, futureYear%100)

	pastMonth := int(now.Month()) - 3
	pastYear := now.Year()
	if pastMonth < 1 {
		pastMonth += 12
		pastYear--
	}
	pastDate := fmt.Sprintf("%02d-%02d", pastMonth, pastYear%100)

	tests := []struct {
		name          string
		text          string
		expectedPlate string
		wantValid     bool
		wantCode      string
	}{
		{
			"valid inspection matching plate",
			"NEW YORK STATE SAFETY\n" + futureDate + "\nPlate#:ABC1234",
			"ABC1234", true, "",
		},
		{
			"valid inspection no expected plate",
			"SAFETY INSPECTION\n" + futureDate,
			"", true, "",
		},
		{
			"plate mismatch",
			"VEHICLE INSPECTION\n" + futureDate + "\nPlate#:XYZ9999",
			"ABC1234", false, "PLATE_MISMATCH",
		},
		{
			"expired sticker",
			"NEW YORK STATE SAFETY\n" + pastDate + "\nPlate#:ABC1234",
			"ABC1234", false, "EXPIRED",
		},
		{"empty text", "", "", false, "UNREADABLE"},
		{"wrong doc type", "CAR INSURANCE", "ABC1234", false, "WRONG_DOC_TYPE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateVehicleInspection(tc.text, tc.expectedPlate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tc.wantValid || result.ErrorCode != tc.wantCode {
				t.Errorf("got Valid=%v ErrorCode=%q, want Valid=%v ErrorCode=%q",
					result.Valid, result.ErrorCode, tc.wantValid, tc.wantCode)
			}
		})
	}
}

func TestValidateTLCDiamond(t *testing.T) {
	futureDate := time.Now().AddDate(1, 0, 0).Format("01/02/2006")
	pastDate := time.Now().AddDate(-1, 0, 0).Format("01/02/2006")

	tests := []struct {
		name          string
		text          string
		expectedPlate string
		wantValid     bool
		wantCode      string
	}{
		{
			"valid diamond with matching plate",
			"TLC Lic #\nTaxi & Limousine Commission\nPlate #: T123456C\nLic Exp Date " + futureDate,
			"T123456C", true, "",
		},
		{
			"valid diamond no expected plate",
			"TLC\nTaxi & Limousine Commission\nLic Exp Date " + futureDate,
			"", true, "",
		},
		{
			"plate mismatch",
			"TLC\nTaxi & Limousine Commission\nPlate #: T123456C\nLic Exp Date " + futureDate,
			"ABC1234", false, "PLATE_MISMATCH",
		},
		{
			"expired TLC diamond",
			"TLC\nTaxi & Limousine Commission\nPlate #: T123456C\nLic Exp Date " + pastDate,
			"T123456C", false, "EXPIRED",
		},
		{"wrong doc", "DRIVER LICENSE", "", false, "WRONG_DOC_TYPE"},
		{"empty", "", "", false, "UNREADABLE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateTLCDiamond(tc.text, tc.expectedPlate)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tc.wantValid || result.ErrorCode != tc.wantCode {
				t.Errorf("got Valid=%v ErrorCode=%q, want Valid=%v ErrorCode=%q",
					result.Valid, result.ErrorCode, tc.wantValid, tc.wantCode)
			}
		})
	}
}

func TestValidateInsurance(t *testing.T) {
	futureDate := time.Now().AddDate(1, 0, 0).Format("01/02/2006")
	pastDate := time.Now().AddDate(-1, 0, 0).Format("01/02/2006")

	tests := []struct {
		name      string
		text      string
		wantValid bool
		wantCode  string
	}{
		{"valid FH-1 with future expiry", "INSURANCE CERTIFICATE\nEffective Date 03/01/2023\nExpiration Date " + futureDate, true, ""},
		{"valid liability doc no date", "LIABILITY INSURANCE POLICY", true, ""},
		{"expired insurance", "INSURANCE POLICY\nExpiration Date " + pastDate, false, "EXPIRED"},
		{"empty text", "", false, "UNREADABLE"},
		{"wrong doc type", "DRIVER LICENSE", false, "WRONG_DOC_TYPE"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result, err := validateInsurance(tc.text)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if result.Valid != tc.wantValid || result.ErrorCode != tc.wantCode {
				t.Errorf("got Valid=%v ErrorCode=%q, want Valid=%v ErrorCode=%q",
					result.Valid, result.ErrorCode, tc.wantValid, tc.wantCode)
			}
		})
	}
}
