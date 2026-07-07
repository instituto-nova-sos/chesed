package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockPublicCampaignRepository implements PublicCampaignRepository.
type MockPublicCampaignRepository struct {
	mock.Mock
}

func (m *MockPublicCampaignRepository) List(ctx context.Context, filter domain.CampaignFilter) (*domain.CampaignListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.CampaignListResult), args.Error(1)
}

// MockPublicPersonRepository implements PublicPersonRepository.
type MockPublicPersonRepository struct {
	mock.Mock
}

func (m *MockPublicPersonRepository) Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error) {
	args := m.Called(ctx, person, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

// MockPublicPersonRoleRepository implements PublicPersonRoleRepository.
type MockPublicPersonRoleRepository struct {
	mock.Mock
}

func (m *MockPublicPersonRoleRepository) Create(ctx context.Context, role domain.PersonRole) (*domain.PersonRole, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonRole), args.Error(1)
}

// MockPublicAgreementRepository implements PublicAgreementRepository.
type MockPublicAgreementRepository struct {
	mock.Mock
}

func (m *MockPublicAgreementRepository) Create(ctx context.Context, agreement domain.VolunteerAgreement) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, agreement)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func newTestPublicService() (
	*PublicService,
	*MockPublicCampaignRepository,
	*MockPublicPersonRepository,
	*MockPublicPersonRoleRepository,
	*MockPublicAgreementRepository,
	*MockAuditRepository,
) {
	campaignRepo := new(MockPublicCampaignRepository)
	personRepo := new(MockPublicPersonRepository)
	roleRepo := new(MockPublicPersonRoleRepository)
	agreementRepo := new(MockPublicAgreementRepository)
	auditRepo := new(MockAuditRepository)
	svc := NewPublicService(campaignRepo, personRepo, roleRepo, agreementRepo, NewAuditService(auditRepo))
	return svc, campaignRepo, personRepo, roleRepo, agreementRepo, auditRepo
}

func TestPublicService_ListActiveCampaigns(t *testing.T) {
	svc, campaignRepo, _, _, _, _ := newTestPublicService()
	campusID := uuid.New()
	expected := &domain.CampaignListResult{
		Data: []domain.CampaignListItem{{ID: uuid.New(), Name: "Winter Drive", Status: "ACTIVE"}},
	}

	campaignRepo.On("List", mock.Anything, mock.MatchedBy(func(f domain.CampaignFilter) bool {
		return f.CampusID == campusID &&
			f.Status != nil && *f.Status == "ACTIVE" &&
			f.Page == 2 && f.PerPage == 10
	})).Return(expected, nil)

	result, err := svc.ListActiveCampaigns(context.Background(), campusID, 2, 10)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
	campaignRepo.AssertExpectations(t)
}

func TestPublicService_ListActiveCampaigns_RepoError(t *testing.T) {
	svc, campaignRepo, _, _, _, _ := newTestPublicService()
	campaignRepo.On("List", mock.Anything, mock.Anything).Return(nil, errors.New("db down"))

	result, err := svc.ListActiveCampaigns(context.Background(), uuid.New(), 1, 20)

	require.Error(t, err)
	assert.Nil(t, result)
}

func validVolunteerInput(campusID uuid.UUID) PublicVolunteerInput {
	email := "ada@example.com"
	phone := "+5511999999999"
	birth := "1990-01-15"
	referral := "instagram"
	return PublicVolunteerInput{
		FullName:       "Ada Lovelace",
		Email:          &email,
		Phone:          &phone,
		BirthDate:      &birth,
		ReferralSource: &referral,
		CampusID:       campusID.String(),
	}
}

func TestPublicService_RegisterVolunteer_Success(t *testing.T) {
	svc, _, personRepo, roleRepo, agreementRepo, auditRepo := newTestPublicService()
	campusID := uuid.New()
	input := validVolunteerInput(campusID)

	createdPerson := &domain.Person{ID: uuid.New(), FullName: "Ada Lovelace", CampusID: campusID, IsActive: true}
	personRepo.On("Create", mock.Anything, mock.MatchedBy(func(p domain.Person) bool {
		return p.FullName == "Ada Lovelace" && p.CampusID == campusID && p.IsActive
	}), (*domain.Address)(nil)).Return(createdPerson, nil)

	createdRole := &domain.PersonRole{ID: uuid.New(), PersonID: createdPerson.ID, RoleType: domain.RoleVolunteer, IsActive: true}
	roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
		return r.RoleType == domain.RoleVolunteer && r.IsActive
	})).Return(createdRole, nil)

	agreementRepo.On("Create", mock.Anything, mock.MatchedBy(func(a domain.VolunteerAgreement) bool {
		return a.Status == domain.AgreementPending && a.CampusID == campusID
	})).Return(&domain.VolunteerAgreement{ID: uuid.New(), Status: domain.AgreementPending, CampusID: campusID}, nil)

	auditRepo.On("Create", mock.Anything, mock.MatchedBy(func(e domain.AuditLog) bool {
		// Public path: no actor, campus scoped from validated campus id.
		return e.UserID == nil && e.CampusID != nil && *e.CampusID == campusID
	})).Return(nil)

	person, err := svc.RegisterVolunteer(context.Background(), input, "203.0.113.5", "wp-agent")

	require.NoError(t, err)
	require.NotNil(t, person)
	assert.Equal(t, "Ada Lovelace", person.FullName)
	require.NotNil(t, createdPerson)
	personRepo.AssertExpectations(t)
	roleRepo.AssertExpectations(t)
	agreementRepo.AssertExpectations(t)
	auditRepo.AssertExpectations(t)
}

func TestPublicService_RegisterVolunteer_Validation(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*PublicVolunteerInput)
	}{
		{name: "missing full name", mutate: func(i *PublicVolunteerInput) { i.FullName = "" }},
		{name: "missing campus id", mutate: func(i *PublicVolunteerInput) { i.CampusID = "" }},
		{name: "malformed campus id", mutate: func(i *PublicVolunteerInput) { i.CampusID = "not-a-uuid" }},
		{name: "invalid email", mutate: func(i *PublicVolunteerInput) { bad := "nope"; i.Email = &bad }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _, _, _, _, _ := newTestPublicService()
			input := validVolunteerInput(uuid.New())
			tt.mutate(&input)

			person, err := svc.RegisterVolunteer(context.Background(), input, "", "")

			require.Error(t, err)
			assert.Nil(t, person)
			assert.True(t, errors.Is(err, domain.ErrValidation),
				"expected ErrValidation, got %v", err)
		})
	}
}

func TestPublicService_RegisterVolunteer_PersonCreateFails(t *testing.T) {
	svc, _, personRepo, _, _, _ := newTestPublicService()
	input := validVolunteerInput(uuid.New())
	personRepo.On("Create", mock.Anything, mock.Anything, mock.Anything).
		Return(nil, errors.New("insert failed"))

	person, err := svc.RegisterVolunteer(context.Background(), input, "", "")

	require.Error(t, err)
	assert.Nil(t, person)
}
