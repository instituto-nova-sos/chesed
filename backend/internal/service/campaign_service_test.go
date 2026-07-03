package service

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/auth"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockCampaignRepository implements CampaignRepository for testing.
type MockCampaignRepository struct {
	mock.Mock
}

func (m *MockCampaignRepository) Create(ctx context.Context, c domain.Campaign) (*domain.Campaign, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *MockCampaignRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Campaign, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *MockCampaignRepository) List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignListResult), args.Error(1)
}

func (m *MockCampaignRepository) Update(ctx context.Context, c domain.Campaign) (*domain.Campaign, error) {
	args := m.Called(ctx, c)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Campaign), args.Error(1)
}

func (m *MockCampaignRepository) AddTeamMember(ctx context.Context, campaignID, personID uuid.UUID, role string, assignedBy *uuid.UUID) (*domain.CampaignTeamMember, error) {
	args := m.Called(ctx, campaignID, personID, role, assignedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignTeamMember), args.Error(1)
}

func (m *MockCampaignRepository) RemoveTeamMember(ctx context.Context, campaignID, personID uuid.UUID) error {
	args := m.Called(ctx, campaignID, personID)
	return args.Error(0)
}

func (m *MockCampaignRepository) ListTeam(ctx context.Context, campaignID uuid.UUID) ([]domain.CampaignTeamMember, error) {
	args := m.Called(ctx, campaignID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.CampaignTeamMember), args.Error(1)
}

// MockCampaignPersonRepository implements CampaignPersonRepository for testing.
type MockCampaignPersonRepository struct {
	mock.Mock
}

func (m *MockCampaignPersonRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func newTestCampaignService() (*CampaignService, *MockCampaignRepository, *MockCampaignPersonRepository, *MockAuditRepository) {
	repo := new(MockCampaignRepository)
	personRepo := new(MockCampaignPersonRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	return NewCampaignService(repo, personRepo, auditSvc), repo, personRepo, auditRepo
}

func campaignTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "coordinator@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func validCreateCampaignInput() CreateCampaignInput {
	return CreateCampaignInput{
		Name:         "March Social Action",
		CampaignType: "SOCIAL_ACTION",
		StartDate:    "2026-07-10",
	}
}

func TestCampaignService_CreateCampaign(t *testing.T) {
	t.Run("success with minimal input", func(t *testing.T) {
		svc, repo, _, auditRepo := newTestCampaignService()
		ctx, claims := campaignTestContext()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Campaign")).
			Return(&domain.Campaign{ID: uuid.New(), Name: "March Social Action", Status: "PLANNED"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		created, err := svc.CreateCampaign(ctx, validCreateCampaignInput())
		require.NoError(t, err)
		assert.Equal(t, "PLANNED", created.Status)

		// The campaign must be stamped with the caller's campus and PLANNED status.
		stored := repo.Calls[0].Arguments.Get(1).(domain.Campaign)
		assert.Equal(t, claims.CampusID, stored.CampusID)
		assert.Equal(t, "PLANNED", stored.Status)
		assert.NotEqual(t, uuid.Nil, stored.ID)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("missing campus context returns forbidden", func(t *testing.T) {
		svc, _, _, _ := newTestCampaignService()

		_, err := svc.CreateCampaign(context.Background(), validCreateCampaignInput())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("invalid campaign type fails validation", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		input := validCreateCampaignInput()
		input.CampaignType = "PARTY"
		_, err := svc.CreateCampaign(ctx, input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("end date before start date is rejected", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		end := "2026-07-01"
		input := validCreateCampaignInput()
		input.EndDate = &end
		_, err := svc.CreateCampaign(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("coordinator outside campus is rejected as validation error", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		coordinatorID := uuid.New().String()
		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		input := validCreateCampaignInput()
		input.CoordinatorID = &coordinatorID
		_, err := svc.CreateCampaign(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("coordinator in campus is accepted", func(t *testing.T) {
		svc, repo, personRepo, auditRepo := newTestCampaignService()
		ctx, claims := campaignTestContext()

		coordinator := uuid.New()
		coordinatorID := coordinator.String()
		personRepo.On("FindByID", mock.Anything, coordinator, claims.CampusID).
			Return(&domain.Person{ID: coordinator}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Campaign")).
			Return(&domain.Campaign{ID: uuid.New(), Status: "PLANNED"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := validCreateCampaignInput()
		input.CoordinatorID = &coordinatorID
		_, err := svc.CreateCampaign(ctx, input)
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Campaign)
		require.NotNil(t, stored.CoordinatorID)
		assert.Equal(t, coordinator, *stored.CoordinatorID)
	})
}

func TestCampaignService_GetCampaign(t *testing.T) {
	t.Run("returns detail with team scoped to campus", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, claims := campaignTestContext()

		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.Campaign{ID: id, Name: "March"}, nil)
		repo.On("ListTeam", mock.Anything, id).
			Return([]domain.CampaignTeamMember{{PersonID: uuid.New(), PersonName: "Maria", RoleInCampaign: "VOLUNTEER"}}, nil)

		detail, err := svc.GetCampaign(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, detail.ID)
		require.Len(t, detail.Team, 1)
		assert.Equal(t, "Maria", detail.Team[0].PersonName)
	})

	t.Run("not found propagates", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.GetCampaign(ctx, uuid.New())
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("missing campus returns forbidden", func(t *testing.T) {
		svc, _, _, _ := newTestCampaignService()

		_, err := svc.GetCampaign(context.Background(), uuid.New())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestCampaignService_ListCampaigns(t *testing.T) {
	t.Run("applies campus from context", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, claims := campaignTestContext()

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.CampaignFilter) bool {
			return f.CampusID == claims.CampusID
		})).Return(&domain.CampaignListResult{Data: []domain.CampaignListItem{}}, nil)

		_, err := svc.ListCampaigns(ctx, domain.CampaignFilter{Page: 1, PerPage: 20})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("missing campus returns forbidden", func(t *testing.T) {
		svc, _, _, _ := newTestCampaignService()

		_, err := svc.ListCampaigns(context.Background(), domain.CampaignFilter{})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestCampaignService_UpdateCampaign(t *testing.T) {
	validUpdate := func() UpdateCampaignInput {
		return UpdateCampaignInput{
			Name:         "Updated Action",
			CampaignType: "HEALTH",
			StartDate:    "2026-07-10",
			Status:       "ACTIVE",
		}
	}

	t.Run("success updates fields and audits old values", func(t *testing.T) {
		svc, repo, _, auditRepo := newTestCampaignService()
		ctx, claims := campaignTestContext()

		id := uuid.New()
		existing := &domain.Campaign{ID: id, CampusID: claims.CampusID, Name: "Old Name", Status: "PLANNED", CampaignType: "OTHER", StartDate: time.Now()}
		repo.On("FindByID", mock.Anything, id, claims.CampusID).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("domain.Campaign")).
			Return(&domain.Campaign{ID: id, Name: "Updated Action", Status: "ACTIVE"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		updated, err := svc.UpdateCampaign(ctx, id, validUpdate())
		require.NoError(t, err)
		assert.Equal(t, "ACTIVE", updated.Status)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("invalid status fails validation", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		input := validUpdate()
		input.Status = "ARCHIVED"
		_, err := svc.UpdateCampaign(ctx, uuid.New(), input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Update")
	})

	t.Run("not found propagates", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.UpdateCampaign(ctx, uuid.New(), validUpdate())
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestCampaignService_AddTeamMember(t *testing.T) {
	t.Run("success adds member and audits", func(t *testing.T) {
		svc, repo, personRepo, auditRepo := newTestCampaignService()
		ctx, claims := campaignTestContext()

		campaignID := uuid.New()
		personID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, claims.CampusID).
			Return(&domain.Campaign{ID: campaignID, CampusID: claims.CampusID}, nil)
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID, FullName: "Maria Silva"}, nil)
		repo.On("AddTeamMember", mock.Anything, campaignID, personID, "VOLUNTEER", mock.Anything).
			Return(&domain.CampaignTeamMember{PersonID: personID, PersonName: "Maria Silva", RoleInCampaign: "VOLUNTEER"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		member, err := svc.AddTeamMember(ctx, campaignID, AddTeamMemberInput{PersonID: personID.String(), RoleInCampaign: "VOLUNTEER"})
		require.NoError(t, err)
		assert.Equal(t, "Maria Silva", member.PersonName)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("campaign not found propagates", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.AddTeamMember(ctx, uuid.New(), AddTeamMemberInput{PersonID: uuid.New().String(), RoleInCampaign: "VOLUNTEER"})
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("person outside campus is a generic validation error", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestCampaignService()
		ctx, claims := campaignTestContext()

		campaignID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, claims.CampusID).
			Return(&domain.Campaign{ID: campaignID}, nil)
		personRepo.On("FindByID", mock.Anything, mock.Anything, claims.CampusID).
			Return(nil, domain.ErrNotFound)

		_, err := svc.AddTeamMember(ctx, campaignID, AddTeamMemberInput{PersonID: uuid.New().String(), RoleInCampaign: "VOLUNTEER"})
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "AddTeamMember")
	})

	t.Run("invalid role fails validation", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, _ := campaignTestContext()

		_, err := svc.AddTeamMember(ctx, uuid.New(), AddTeamMemberInput{PersonID: uuid.New().String(), RoleInCampaign: "MASCOT"})
		require.Error(t, err)
		repo.AssertNotCalled(t, "AddTeamMember")
	})

	t.Run("duplicate assignment propagates", func(t *testing.T) {
		svc, repo, personRepo, _ := newTestCampaignService()
		ctx, claims := campaignTestContext()

		campaignID := uuid.New()
		personID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, claims.CampusID).
			Return(&domain.Campaign{ID: campaignID}, nil)
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID}, nil)
		repo.On("AddTeamMember", mock.Anything, campaignID, personID, "VOLUNTEER", mock.Anything).
			Return(nil, domain.ErrDuplicate)

		_, err := svc.AddTeamMember(ctx, campaignID, AddTeamMemberInput{PersonID: personID.String(), RoleInCampaign: "VOLUNTEER"})
		require.ErrorIs(t, err, domain.ErrDuplicate)
	})
}

func TestCampaignService_RemoveTeamMember(t *testing.T) {
	t.Run("success removes member and audits", func(t *testing.T) {
		svc, repo, _, auditRepo := newTestCampaignService()
		ctx, claims := campaignTestContext()

		campaignID := uuid.New()
		personID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, claims.CampusID).
			Return(&domain.Campaign{ID: campaignID}, nil)
		repo.On("RemoveTeamMember", mock.Anything, campaignID, personID).Return(nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		err := svc.RemoveTeamMember(ctx, campaignID, personID)
		require.NoError(t, err)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("missing assignment propagates not found", func(t *testing.T) {
		svc, repo, _, _ := newTestCampaignService()
		ctx, claims := campaignTestContext()

		campaignID := uuid.New()
		repo.On("FindByID", mock.Anything, campaignID, claims.CampusID).
			Return(&domain.Campaign{ID: campaignID}, nil)
		repo.On("RemoveTeamMember", mock.Anything, campaignID, mock.Anything).
			Return(domain.ErrNotFound)

		err := svc.RemoveTeamMember(ctx, campaignID, uuid.New())
		require.ErrorIs(t, err, domain.ErrNotFound)
	})
}
