package utils

import "strings"

// ValidateCPF validates a Brazilian CPF number using the check-digit algorithm.
// Accepts formatted (123.456.789-00) or unformatted (12345678900) input.
func ValidateCPF(cpf string) bool {
	cpf = strings.ReplaceAll(cpf, ".", "")
	cpf = strings.ReplaceAll(cpf, "-", "")
	cpf = strings.TrimSpace(cpf)

	if len(cpf) != 11 {
		return false
	}

	for _, c := range cpf {
		if c < '0' || c > '9' {
			return false
		}
	}

	if allSameDigit(cpf) {
		return false
	}

	d1 := calculateDigit(cpf[:9], 10)
	d2 := calculateDigit(cpf[:10], 11)

	return int(cpf[9]-'0') == d1 && int(cpf[10]-'0') == d2
}

func calculateDigit(digits string, startWeight int) int {
	sum := 0
	weight := startWeight
	for _, d := range digits {
		sum += int(d-'0') * weight
		weight--
	}
	remainder := sum % 11
	if remainder < 2 {
		return 0
	}
	return 11 - remainder
}

func allSameDigit(s string) bool {
	for i := 1; i < len(s); i++ {
		if s[i] != s[0] {
			return false
		}
	}
	return true
}
