package domain

import "time"

// HistoryEntry represents a timeline item in a person's service history.
type HistoryEntry struct {
	Type        string  `json:"type"`
	ID          string  `json:"id"`
	Date        time.Time `json:"date"`
	Summary     string  `json:"summary"`
	ServiceType *string `json:"service_type,omitempty"`
	Professional *string `json:"professional,omitempty"`
	Status      *string `json:"status,omitempty"`
	Campaign    *string `json:"campaign,omitempty"`
}
