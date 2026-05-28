package domain

import (
	"time"

	"github.com/google/uuid"
)

// Address represents a person's physical address.
type Address struct {
	ID           uuid.UUID `json:"id"`
	PersonID     uuid.UUID `json:"person_id"`
	Street       *string   `json:"street,omitempty"`
	Number       *string   `json:"number,omitempty"`
	Complement   *string   `json:"complement,omitempty"`
	Neighborhood *string   `json:"neighborhood,omitempty"`
	City         *string   `json:"city,omitempty"`
	State        *string   `json:"state,omitempty"`
	ZipCode      *string   `json:"zip_code,omitempty"`
	Country      string    `json:"country"`
	IsPrimary    bool      `json:"is_primary"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
