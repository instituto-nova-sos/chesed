package service

import (
	"fmt"
	"strings"

	"github.com/go-pdf/fpdf"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// PDFReceiptRenderer renders a donation receipt as a self-contained PDF. It has
// no dependencies beyond the fpdf drawing library so it can be unit-tested in
// isolation and injected into DonationService (it satisfies ReceiptRenderer).
type PDFReceiptRenderer struct{}

// NewReceiptRenderer creates a PDFReceiptRenderer.
func NewReceiptRenderer() *PDFReceiptRenderer { return &PDFReceiptRenderer{} }

// Render produces the receipt PDF bytes for a fully resolved donation. Missing
// issuer/donor fields render as an em dash rather than failing.
func (PDFReceiptRenderer) Render(d domain.ReceiptData) ([]byte, error) {
	pdf := fpdf.New("P", "mm", "A4", "")
	pdf.SetMargins(20, 20, 20)
	pdf.AddPage()

	renderReceiptHeader(pdf, d)
	renderReceiptBody(pdf, d)
	renderReceiptFooter(pdf, d)

	var buf strings.Builder
	if err := pdf.Output(&buf); err != nil {
		return nil, fmt.Errorf("receiptRenderer.Render: %w", err)
	}
	return []byte(buf.String()), nil
}

// renderReceiptHeader draws the issuing organization block.
func renderReceiptHeader(pdf *fpdf.Fpdf, d domain.ReceiptData) {
	pdf.SetFont("Helvetica", "B", 16)
	pdf.CellFormat(0, 10, "RECIBO DE DOACAO", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	pdf.SetFont("Helvetica", "B", 11)
	pdf.CellFormat(0, 6, orDash(d.IssuerName, d.CampusName), "", 1, "L", false, 0, "")
	pdf.SetFont("Helvetica", "", 9)
	pdf.CellFormat(0, 5, "CNPJ: "+orDashPtr(d.IssuerCNPJ), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, orDashPtr(d.IssuerAddress), "", 1, "L", false, 0, "")
	pdf.CellFormat(0, 5, joinCityState(d.IssuerCity, d.IssuerState, d.IssuerZip), "", 1, "L", false, 0, "")
	pdf.Ln(6)
	pdf.SetDrawColor(200, 200, 200)
	y := pdf.GetY()
	pdf.Line(20, y, 190, y)
	pdf.Ln(6)
}

// renderReceiptBody draws the donor + amount block.
func renderReceiptBody(pdf *fpdf.Fpdf, d domain.ReceiptData) {
	line := func(label, value string) {
		pdf.SetFont("Helvetica", "B", 10)
		pdf.CellFormat(45, 7, label, "", 0, "L", false, 0, "")
		pdf.SetFont("Helvetica", "", 10)
		pdf.CellFormat(0, 7, value, "", 1, "L", false, 0, "")
	}
	line("Doador:", orDashPtr(d.DonorName))
	line("Documento:", orDashPtr(d.DonorDocument))
	line("Valor:", formatMoney(d.Donation.Amount, d.Donation.Currency))
	if d.Donation.ItemDescription != nil && *d.Donation.ItemDescription != "" {
		line("Item:", *d.Donation.ItemDescription)
	}
	line("Data:", d.Donation.DonationDate.Format("02/01/2006"))
	line("Recibo No:", orDashPtr(d.Donation.ReceiptNumber))
}

// renderReceiptFooter draws the closing note.
func renderReceiptFooter(pdf *fpdf.Fpdf, d domain.ReceiptData) {
	pdf.Ln(12)
	pdf.SetFont("Helvetica", "I", 8)
	pdf.MultiCell(0, 4,
		"Este recibo comprova o recebimento da doacao acima descrita pela organizacao emitente.",
		"", "L", false)
}

// formatMoney renders an amount with the currency symbol using pt-BR grouping
// (thousands '.', decimals ','). A nil amount renders as an em dash.
func formatMoney(amount *float64, currency string) string {
	if amount == nil {
		return "—"
	}
	symbol := currencySymbol(currency)
	return fmt.Sprintf("%s %s", symbol, ptBRDecimal(*amount))
}

// currencySymbol maps a currency code to its display prefix, falling back to
// the code itself for unknown currencies.
func currencySymbol(currency string) string {
	switch currency {
	case "BRL":
		return "R$"
	case "USD":
		return "US$"
	case "EUR":
		return "€"
	default:
		return currency
	}
}

// ptBRDecimal formats a float with two decimals, '.' thousands separators and
// ',' as the decimal separator (Brazilian convention).
func ptBRDecimal(v float64) string {
	s := fmt.Sprintf("%.2f", v)
	intPart, frac, _ := strings.Cut(s, ".")
	neg := strings.HasPrefix(intPart, "-")
	intPart = strings.TrimPrefix(intPart, "-")

	var grouped strings.Builder
	for i, digit := range intPart {
		if i > 0 && (len(intPart)-i)%3 == 0 {
			grouped.WriteByte('.')
		}
		grouped.WriteRune(digit)
	}
	out := grouped.String() + "," + frac
	if neg {
		out = "-" + out
	}
	return out
}

func orDash(p *string, fallback string) string {
	if p != nil && *p != "" {
		return *p
	}
	if fallback != "" {
		return fallback
	}
	return "—"
}

func orDashPtr(p *string) string {
	if p != nil && *p != "" {
		return *p
	}
	return "—"
}

// joinCityState assembles "City - ST, ZIP", collapsing missing parts.
func joinCityState(city, state, zip *string) string {
	parts := []string{}
	if city != nil && *city != "" {
		parts = append(parts, *city)
	}
	if state != nil && *state != "" {
		parts = append(parts, *state)
	}
	loc := strings.Join(parts, " - ")
	if zip != nil && *zip != "" {
		if loc != "" {
			loc += ", "
		}
		loc += *zip
	}
	if loc == "" {
		return "—"
	}
	return loc
}
