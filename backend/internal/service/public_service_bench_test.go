package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"

	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// BenchmarkListActiveCampaigns measures the public campaign-list read path
// (RNF-08, S12.5) — the highest-fan-out, cache-cold read behind
// GET /public/campaigns and the primary throughput driver of the k6 load test.
// It exercises the real PublicService.ListActiveCampaigns against a mock
// campaign repo returning a fixed lean projection, so the measurement is the
// service-layer cost independent of the network and DB.
func BenchmarkListActiveCampaigns(b *testing.B) {
	campusID := uuid.New()
	ctx := context.Background()

	result := fixedCampaignListResult(campusID, 20)

	campaignRepo := new(MockPublicCampaignRepository)
	campaignRepo.On("List", mock.Anything, mock.AnythingOfType("domain.CampaignFilter")).
		Return(result, nil)
	svc := NewPublicService(
		campaignRepo,
		new(MockPublicPersonRepository),
		new(MockPublicPersonRoleRepository),
		new(MockPublicAgreementRepository),
		NewAuditService(new(MockAuditRepository)),
	)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		if _, err := svc.ListActiveCampaigns(ctx, campusID, 1, 20); err != nil {
			b.Fatalf("ListActiveCampaigns: %v", err)
		}
	}
}

// fixedCampaignListResult builds a deterministic page of active campaigns in the
// lean, no-PII projection the public endpoint returns.
func fixedCampaignListResult(_ uuid.UUID, n int) *domain.CampaignListResult {
	items := make([]domain.CampaignListItem, 0, n)
	start := time.Date(2026, 7, 10, 0, 0, 0, 0, time.UTC)
	for range n {
		items = append(items, domain.CampaignListItem{
			ID:           uuid.New(),
			Name:         "Social Action",
			CampaignType: "SOCIAL_ACTION",
			Status:       "ACTIVE",
			StartDate:    start,
		})
	}
	return &domain.CampaignListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       1,
			PerPage:    n,
			Total:      n,
			TotalPages: 1,
		},
	}
}
