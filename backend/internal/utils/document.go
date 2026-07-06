package utils

import "regexp"

var (
	ssnPattern         = regexp.MustCompile(`^\d{3}-?\d{2}-?\d{4}$`)
	genericIDCharacter = regexp.MustCompile(`^[A-Za-z0-9.\-/ ]+$`)
)

// ValidateDocumentFormat validates a document number against the format rules
// for its type. An empty number is treated as valid because document_number is
// an optional field; format is only enforced when a value is present.
//
// Rules by type:
//   - CPF: Brazilian check-digit algorithm (see ValidateCPF).
//   - SSN: US social security number, 9 digits with optional dashes.
//   - RG / EU_ID / PASSPORT: 3..30 chars, alphanumeric plus ". - / space".
//   - OTHER (and any unknown type): 1..30 chars, unrestricted characters.
func ValidateDocumentFormat(docType, number string) bool {
	if number == "" {
		return true
	}

	switch docType {
	case "CPF":
		return ValidateCPF(number)
	case "SSN":
		return ssnPattern.MatchString(number)
	case "RG", "EU_ID", "PASSPORT":
		return isBoundedGenericID(number, 3, 30)
	default:
		return len([]rune(number)) >= 1 && len([]rune(number)) <= 30
	}
}

// isBoundedGenericID reports whether number has a rune length within
// [minLen, maxLen] and contains only alphanumeric or ". - / space" characters.
func isBoundedGenericID(number string, minLen, maxLen int) bool {
	length := len([]rune(number))
	if length < minLen || length > maxLen {
		return false
	}
	return genericIDCharacter.MatchString(number)
}
