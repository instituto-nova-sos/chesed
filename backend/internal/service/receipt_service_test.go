package service

import (
	"bytes"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func ptrStr(s string) *string   { return &s }
func ptrF64(f float64) *float64 { return &f }

func TestReceiptRenderer_Render(t *testing.T) {
	renderer := NewReceiptRenderer()
	date := time.Date(2026, 3, 15, 0, 0, 0, 0, time.UTC)
	amount := 1250.5

	tests := []struct {
		name string
		data domain.ReceiptData
	}{
		{
			name: "fully populated financial receipt",
			data: domain.ReceiptData{
				Donation: domain.Donation{
					ID:            uuid.New(),
					DonationType:  "FINANCIAL",
					Amount:        &amount,
					Currency:      "BRL",
					DonationDate:  date,
					ReceiptNumber: ptrStr("REC-2026-1A2B3C4D"),
				},
				DonorName:     ptrStr("Maria Silva"),
				DonorDocument: ptrStr("529.982.247-25"),
				IssuerName:    ptrStr("Instituto Nova SOS"),
				IssuerCNPJ:    ptrStr("12.345.678/0001-90"),
				IssuerAddress: ptrStr("Rua das Flores, 100"),
				IssuerCity:    ptrStr("São Paulo"),
				IssuerState:   ptrStr("SP"),
				IssuerZip:     ptrStr("01234-567"),
				CampusName:    "Campus Central",
			},
		},
		{
			name: "missing issuer and donor render placeholders without panic",
			data: domain.ReceiptData{
				Donation: domain.Donation{
					ID:            uuid.New(),
					DonationType:  "FINANCIAL",
					Amount:        &amount,
					Currency:      "USD",
					DonationDate:  date,
					ReceiptNumber: ptrStr("REC-2026-DEADBEEF"),
				},
				CampusName: "Campus Central",
			},
		},
		{
			name: "goods donation with nil amount",
			data: domain.ReceiptData{
				Donation: domain.Donation{
					ID:              uuid.New(),
					DonationType:    "GOODS",
					Currency:        "BRL",
					ItemDescription: ptrStr("10 cestas básicas"),
					DonationDate:    date,
					ReceiptNumber:   ptrStr("REC-2026-0000AAAA"),
				},
				DonorName:  ptrStr("Anônimo"),
				CampusName: "Campus Central",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			pdf, err := renderer.Render(tc.data)
			require.NoError(t, err)
			require.NotEmpty(t, pdf)
			assert.True(t, bytes.HasPrefix(pdf, []byte("%PDF")), "output must be a PDF")
		})
	}
}

func TestFormatMoney(t *testing.T) {
	tests := []struct {
		name     string
		amount   *float64
		currency string
		want     string
	}{
		{"brl", ptrF64(1250.5), "BRL", "R$ 1.250,50"},
		{"usd", ptrF64(99), "USD", "US$ 99,00"},
		{"eur", ptrF64(1000000), "EUR", "€ 1.000.000,00"},
		{"nil amount", nil, "BRL", "—"},
		{"unknown currency uses code", ptrF64(10), "GBP", "GBP 10,00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			assert.Equal(t, tc.want, formatMoney(tc.amount, tc.currency))
		})
	}
}
