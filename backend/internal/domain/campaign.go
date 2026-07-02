package domain

import (
	"time"

	"github.com/google/uuid"
)

// Campaign represents a social action event scoped to a campus.
type Campaign struct {
	ID              uuid.UUID  `json:"id"`
	Name            string     `json:"name"`
	Description     *string    `json:"description,omitempty"`
	CampaignType    string     `json:"campaign_type"`
	StartDate       time.Time  `json:"start_date"`
	EndDate         *time.Time `json:"end_date,omitempty"`
	LocationName    *string    `json:"location_name,omitempty"`
	LocationAddress *string    `json:"location_address,omitempty"`
	CampusID        uuid.UUID  `json:"campus_id"`
	CoordinatorID   *uuid.UUID `json:"coordinator_id,omitempty"`
	Status          string     `json:"status"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	CreatedBy       *uuid.UUID `json:"-"`
}

// CampaignTeamMember is a person assigned to a campaign with a role.
type CampaignTeamMember struct {
	PersonID       uuid.UUID `json:"person_id"`
	PersonName     string    `json:"person_name"`
	RoleInCampaign string    `json:"role_in_campaign"`
	AssignedAt     time.Time `json:"assigned_at"`
}

// CampaignDetail is the full campaign representation including its team.
type CampaignDetail struct {
	Campaign
	Team []CampaignTeamMember `json:"team"`
}

// CampaignListItem is a summary representation for list responses.
type CampaignListItem struct {
	ID           uuid.UUID  `json:"id"`
	Name         string     `json:"name"`
	CampaignType string     `json:"campaign_type"`
	Status       string     `json:"status"`
	StartDate    time.Time  `json:"start_date"`
	EndDate      *time.Time `json:"end_date,omitempty"`
	LocationName *string    `json:"location_name,omitempty"`
}

// CampaignListResult contains paginated campaign list data.
type CampaignListResult struct {
	Data       []CampaignListItem `json:"data"`
	Pagination Pagination         `json:"pagination"`
}

// CampaignFilter defines search/filter criteria for listing campaigns.
type CampaignFilter struct {
	CampusID uuid.UUID
	Status   *string
	Type     *string
	Page     int
	PerPage  int
}

// CampaignStatuses enumerates the valid campaign lifecycle states.
var CampaignStatuses = []string{"PLANNED", "ACTIVE", "COMPLETED", "CANCELLED"}

// CampaignTypes enumerates the valid campaign categories.
var CampaignTypes = []string{"SOCIAL_ACTION", "EDUCATIONAL", "HEALTH", "COMMUNITY", "OTHER"}
