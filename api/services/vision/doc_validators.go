package vision

// DocumentValidationRequest carries all context needed to validate one document.
type DocumentValidationRequest struct {
	// DocType is one of: "driverLicense", "tlcLicense", "carRegistration",
	// "vehicleInspection", "tlcDiamond", "insuranceFiles"
	DocType       string
	Content       []byte
	MimeType      string // "image/jpeg" | "image/png" | "application/pdf"
	ExpectedName  string // fullName from Q3
	ExpectedPlate string // plate extracted from carRegistration; empty until available
}

// DocumentValidationResult is the outcome of a single document check.
// A validation failure is NOT a system error — it is a valid result with Valid=false.
type DocumentValidationResult struct {
	Valid          bool
	ExtractedPlate string // populated for carRegistration, vehicleInspection, insuranceFiles
	// ErrorCode is one of: "WRONG_DOC_TYPE", "NAME_MISMATCH", "PLATE_MISMATCH", "EXPIRED", "UNREADABLE"
	ErrorCode    string
	ErrorMessage string // user-facing message
}

func validateDriverLicense(text, expectedName string) (DocumentValidationResult, error) {
	if text == "" {
		return DocumentValidationResult{
			ErrorCode:    "UNREADABLE",
			ErrorMessage: "We couldn't read your Driver License. Please upload a clearer image.",
		}, nil
	}
	if !containsAny(text, "DRIVER LICENSE", "DRIVER'S LICENSE", "LICENSE") {
		return DocumentValidationResult{
			ErrorCode:    "WRONG_DOC_TYPE",
			ErrorMessage: "This doesn't appear to be a Driver License. Please upload the correct document.",
		}, nil
	}
	if expired, found := parseDocumentExpiry(text); found && expired {
		return DocumentValidationResult{
			ErrorCode:    "EXPIRED",
			ErrorMessage: "Your Driver License has expired. Please upload a current license.",
		}, nil
	}
	if !nameFuzzyMatch(text, expectedName) {
		return DocumentValidationResult{
			ErrorCode:    "NAME_MISMATCH",
			ErrorMessage: "The name on your Driver License doesn't match your registration name. Please re-upload.",
		}, nil
	}
	return DocumentValidationResult{Valid: true}, nil
}

func validateTLCLicense(text, expectedName string) (DocumentValidationResult, error) {
	if text == "" {
		return DocumentValidationResult{
			ErrorCode:    "UNREADABLE",
			ErrorMessage: "We couldn't read your TLC License. Please upload a clearer image.",
		}, nil
	}
	if !containsAny(text, "TLC", "TAXI", "LIMOUSINE COMMISSION") {
		return DocumentValidationResult{
			ErrorCode:    "WRONG_DOC_TYPE",
			ErrorMessage: "This doesn't appear to be a TLC License. Please upload the correct document.",
		}, nil
	}
	if expired, found := parseDocumentExpiry(text); found && expired {
		return DocumentValidationResult{
			ErrorCode:    "EXPIRED",
			ErrorMessage: "Your TLC License has expired. Please upload a current license.",
		}, nil
	}
	if !nameFuzzyMatch(text, expectedName) {
		return DocumentValidationResult{
			ErrorCode:    "NAME_MISMATCH",
			ErrorMessage: "The name on your TLC License doesn't match your registration name. Please re-upload.",
		}, nil
	}
	return DocumentValidationResult{Valid: true}, nil
}

func validateCarRegistration(text, expectedName string) (DocumentValidationResult, error) {
	if text == "" {
		return DocumentValidationResult{
			ErrorCode:    "UNREADABLE",
			ErrorMessage: "We couldn't read your Car Registration. Please upload a clearer image.",
		}, nil
	}
	if !containsAny(text, "REGISTRATION", "REGISTERED OWNER") {
		return DocumentValidationResult{
			ErrorCode:    "WRONG_DOC_TYPE",
			ErrorMessage: "This doesn't appear to be a Car Registration. Please upload the correct document.",
		}, nil
	}
	if expired, found := parseDocumentExpiry(text); found && expired {
		return DocumentValidationResult{
			ErrorCode:    "EXPIRED",
			ErrorMessage: "Your Car Registration has expired. Please upload a current registration.",
		}, nil
	}
	if !nameFuzzyMatch(text, expectedName) {
		return DocumentValidationResult{
			ErrorCode:    "NAME_MISMATCH",
			ErrorMessage: "The name on your Car Registration doesn't match your registration name. Please re-upload.",
		}, nil
	}
	return DocumentValidationResult{Valid: true, ExtractedPlate: extractPlateNumber(text)}, nil
}

func validateVehicleInspection(text, expectedPlate string) (DocumentValidationResult, error) {
	if text == "" {
		return DocumentValidationResult{
			ErrorCode:    "UNREADABLE",
			ErrorMessage: "We couldn't read your Vehicle Inspection sticker. Please upload a clearer image.",
		}, nil
	}
	if !containsAny(text, "INSPECTION", "SAFETY", "VEHICLE INSPECTION") {
		return DocumentValidationResult{
			ErrorCode:    "WRONG_DOC_TYPE",
			ErrorMessage: "This doesn't appear to be a Vehicle Inspection document. Please upload the correct document.",
		}, nil
	}
	// Hard check: reject expired stickers.
	if expired, found := parseInspectionExpiry(text); found && expired {
		return DocumentValidationResult{
			ErrorCode:    "EXPIRED",
			ErrorMessage: "Your Vehicle Inspection sticker has expired. Please get a new inspection before applying.",
		}, nil
	}
	plate := extractInspectionPlate(text)
	if expectedPlate != "" && plate != "" && normalizePlate(plate) != normalizePlate(expectedPlate) {
		return DocumentValidationResult{
			ErrorCode:    "PLATE_MISMATCH",
			ErrorMessage: "The license plate on your Vehicle Inspection doesn't match your Car Registration. Please re-upload.",
		}, nil
	}
	return DocumentValidationResult{Valid: true, ExtractedPlate: plate}, nil
}

func validateTLCDiamond(text, expectedPlate string) (DocumentValidationResult, error) {
	if text == "" {
		return DocumentValidationResult{
			ErrorCode:    "UNREADABLE",
			ErrorMessage: "We couldn't read your TLC Diamond. Please upload a clearer image.",
		}, nil
	}
	if !containsAny(text, "TLC", "TAXI", "LIMOUSINE COMMISSION") {
		return DocumentValidationResult{
			ErrorCode:    "WRONG_DOC_TYPE",
			ErrorMessage: "This doesn't appear to be a TLC Diamond. Please upload the correct document.",
		}, nil
	}
	if expired, found := parseInsuranceExpiry(text); found && expired {
		return DocumentValidationResult{
			ErrorCode:    "EXPIRED",
			ErrorMessage: "Your TLC Diamond has expired. Please upload a current one.",
		}, nil
	}
	plate := extractTLCDiamondPlate(text)
	if expectedPlate != "" && plate != "" && normalizePlate(plate) != normalizePlate(expectedPlate) {
		return DocumentValidationResult{
			ErrorCode:    "PLATE_MISMATCH",
			ErrorMessage: "The license plate on your TLC Diamond doesn't match your Car Registration. Please re-upload.",
		}, nil
	}
	return DocumentValidationResult{Valid: true, ExtractedPlate: plate}, nil
}

func validateInsurance(text string) (DocumentValidationResult, error) {
	if text == "" {
		return DocumentValidationResult{
			ErrorCode:    "UNREADABLE",
			ErrorMessage: "We couldn't read your Car Insurance document. Please upload a clearer image.",
		}, nil
	}
	if !containsAny(text, "INSURANCE", "POLICY", "LIABILITY") {
		return DocumentValidationResult{
			ErrorCode:    "WRONG_DOC_TYPE",
			ErrorMessage: "This doesn't appear to be a Car Insurance document. Please upload the correct document.",
		}, nil
	}
	if expired, found := parseInsuranceExpiry(text); found && expired {
		return DocumentValidationResult{
			ErrorCode:    "EXPIRED",
			ErrorMessage: "Your Car Insurance has expired. Please upload a current policy.",
		}, nil
	}
	return DocumentValidationResult{Valid: true}, nil
}
