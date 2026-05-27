package utils

import "testing"

func TestValidateCPF(t *testing.T) {
	tests := []struct {
		name string
		cpf  string
		want bool
	}{
		{"valid formatted", "529.982.247-25", true},
		{"valid unformatted", "52998224725", true},
		{"valid another", "111.444.777-35", true},
		{"invalid check digit", "529.982.247-26", false},
		{"all zeros", "00000000000", false},
		{"all ones", "11111111111", false},
		{"all nines", "99999999999", false},
		{"too short", "1234567890", false},
		{"too long", "123456789012", false},
		{"empty", "", false},
		{"letters", "abc.def.ghi-jk", false},
		{"with spaces", " 529.982.247-25 ", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ValidateCPF(tt.cpf)
			if got != tt.want {
				t.Errorf("ValidateCPF(%q) = %v, want %v", tt.cpf, got, tt.want)
			}
		})
	}
}
