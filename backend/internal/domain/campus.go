package domain

import (
	"time"

	"github.com/google/uuid"
)

// Campus represents a physical location where the NGO operates.
type Campus struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Region    string    `json:"region"`
	City      *string   `json:"city,omitempty"`
	State     *string   `json:"state,omitempty"`
	Country   string    `json:"country"`
	Timezone  string    `json:"timezone"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
