package domain

import (
	"time"

	"github.com/google/uuid"
)

// Donation represents a contribution (financial, goods, or services) recorded
// against a campus, optionally linked to a donor and a campaign.
type Donation struct {
	ID              uuid.UUID  `json:"id"`
	DonorPersonID   *uuid.UUID `json:"donor_person_id,omitempty"`
	CampaignID      *uuid.UUID `json:"campaign_id,omitempty"`
	CampusID        uuid.UUID  `json:"campus_id"`
	DonationType    string     `json:"donation_type"`
	Amount          *float64   `json:"amount,omitempty"`
	Currency        string     `json:"currency"`
	ItemDescription *string    `json:"item_description,omitempty"`
	DonationDate    time.Time  `json:"donation_date"`
	ReceiptNumber   *string    `json:"receipt_number,omitempty"`
	ReceiptIssuedAt *time.Time `json:"receipt_issued_at,omitempty"`
	Notes           *string    `json:"notes,omitempty"`
	RegisteredBy    uuid.UUID  `json:"registered_by"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// DonationDetail is the full donation representation with resolved names for
// the linked donor and campaign.
type DonationDetail struct {
	Donation
	DonorName    *string `json:"donor_name,omitempty"`
	CampaignName *string `json:"campaign_name,omitempty"`
}

// DonationListItem is a summary representation for list responses.
type DonationListItem struct {
	ID           uuid.UUID `json:"id"`
	DonationType string    `json:"donation_type"`
	Amount       *float64  `json:"amount,omitempty"`
	Currency     string    `json:"currency"`
	DonationDate time.Time `json:"donation_date"`
	DonorName    *string   `json:"donor_name,omitempty"`
	CampaignName *string   `json:"campaign_name,omitempty"`
}

// DonationListResult contains paginated donation list data.
type DonationListResult struct {
	Data       []DonationListItem `json:"data"`
	Pagination Pagination         `json:"pagination"`
}

// DonationFilter defines search/filter criteria for listing donations.
type DonationFilter struct {
	CampusID     uuid.UUID
	DonationType *string
	CampaignID   *uuid.UUID
	Page         int
	PerPage      int
}

// DonationTypes enumerates the valid donation categories.
var DonationTypes = []string{"FINANCIAL", "GOODS", "SERVICES"}

// DonationCurrencies enumerates the supported ISO 4217 currency codes.
var DonationCurrencies = []string{"BRL", "USD", "EUR"}
