package utils

import "testing"

func TestValidateDocumentFormat(t *testing.T) {
	tests := []struct {
		name    string
		docType string
		number  string
		want    bool
	}{
		{"empty number is allowed", "CPF", "", true},
		{"empty number allowed for passport", "PASSPORT", "", true},
		{"valid CPF formatted", "CPF", "529.982.247-25", true},
		{"valid CPF unformatted", "CPF", "52998224725", true},
		{"invalid CPF check digit", "CPF", "529.982.247-26", false},
		{"invalid CPF letters", "CPF", "abcdefghijk", false},
		{"valid SSN formatted", "SSN", "123-45-6789", true},
		{"valid SSN unformatted", "SSN", "123456789", true},
		{"invalid SSN too short", "SSN", "123-45-678", false},
		{"invalid SSN letters", "SSN", "abc-de-fghi", false},
		{"valid RG alphanumeric", "RG", "12.345.678-9", true},
		{"valid RG plain", "RG", "MG1234567", true},
		{"invalid RG too short", "RG", "AB", false},
		{"invalid RG special chars", "RG", "12@345", false},
		{"valid EU_ID", "EU_ID", "DE-123456789", true},
		{"invalid EU_ID too long", "EU_ID", "123456789012345678901234567890X", false},
		{"valid PASSPORT", "PASSPORT", "AB1234567", true},
		{"invalid PASSPORT too short", "PASSPORT", "A1", false},
		{"valid OTHER short", "OTHER", "X", true},
		{"valid OTHER max", "OTHER", "123456789012345678901234567890", true},
		{"invalid OTHER too long", "OTHER", "1234567890123456789012345678901", false},
		{"unknown type falls back to OTHER-like", "UNKNOWN", "abc", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateDocumentFormat(tt.docType, tt.number)
			if got != tt.want {
				t.Errorf("ValidateDocumentFormat(%q, %q) = %v, want %v", tt.docType, tt.number, got, tt.want)
			}
		})
	}
}
