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

// MockPersonRepository implements PersonRepository for testing.
type MockPersonRepository struct {
	mock.Mock
}

func (m *MockPersonRepository) Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error) {
	args := m.Called(ctx, person, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func (m *MockPersonRepository) FindByEmail(ctx context.Context, email string, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, email, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func (m *MockPersonRepository) FindByEmailGlobal(ctx context.Context, email string) ([]domain.Person, error) {
	args := m.Called(ctx, email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.Person), args.Error(1)
}

func (m *MockPersonRepository) FindByID(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func (m *MockPersonRepository) FindByIDWithDetails(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, []domain.Address, []domain.PersonRole, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, nil, nil, args.Error(3)
	}
	return args.Get(0).(*domain.Person), args.Get(1).([]domain.Address), args.Get(2).([]domain.PersonRole), args.Error(3)
}

func (m *MockPersonRepository) Update(ctx context.Context, person domain.Person) (*domain.Person, error) {
	args := m.Called(ctx, person)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

func (m *MockPersonRepository) List(ctx context.Context, filter domain.PersonFilter) (*domain.PersonListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonListResult), args.Error(1)
}

func (m *MockPersonRepository) CheckDuplicate(ctx context.Context, documentType, documentNumber string, campusID uuid.UUID) (*domain.DuplicateCheckResult, error) {
	args := m.Called(ctx, documentType, documentNumber, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DuplicateCheckResult), args.Error(1)
}

func (m *MockPersonRepository) UpdateAddress(ctx context.Context, personID uuid.UUID, address domain.Address) (*domain.Address, error) {
	args := m.Called(ctx, personID, address)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Address), args.Error(1)
}

// MockPersonRoleRepository implements PersonRoleRepository for testing.
type MockPersonRoleRepository struct {
	mock.Mock
}

func (m *MockPersonRoleRepository) Create(ctx context.Context, role domain.PersonRole) (*domain.PersonRole, error) {
	args := m.Called(ctx, role)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonRole), args.Error(1)
}

func (m *MockPersonRoleRepository) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.PersonRole), args.Error(1)
}

func (m *MockPersonRoleRepository) ToggleActive(ctx context.Context, roleID uuid.UUID, isActive bool, userID *uuid.UUID) (*domain.PersonRole, error) {
	args := m.Called(ctx, roleID, isActive, userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.PersonRole), args.Error(1)
}

func newPersonTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "test@chesed.test",
		Roles:    []string{"COORDINATOR"},
		CampusID: uuid.New(),
	}
	ctx := auth.NewContext(context.Background(), claims)
	return ctx, claims
}

// MockVolunteerAgreementRepository implements VolunteerAgreementRepository for testing.
type MockVolunteerAgreementRepository struct {
	mock.Mock
}

func (m *MockVolunteerAgreementRepository) Create(ctx context.Context, agreement domain.VolunteerAgreement) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, agreement)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) FindByPersonID(ctx context.Context, personID uuid.UUID) ([]domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]domain.VolunteerAgreement), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) FindByPersonRoleID(ctx context.Context, personRoleID uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personRoleID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) FindPendingByPersonID(ctx context.Context, personID uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, personID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) HasAcceptedAgreement(ctx context.Context, personID uuid.UUID) (bool, error) {
	args := m.Called(ctx, personID)
	return args.Bool(0), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) AcceptDigital(ctx context.Context, id uuid.UUID, userID uuid.UUID, ip string, userAgent string) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, userID, ip, userAgent)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) Reject(ctx context.Context, id uuid.UUID, reason *string) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, reason)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func (m *MockVolunteerAgreementRepository) AcceptManualUpload(ctx context.Context, id uuid.UUID, documentPath string, uploadedBy uuid.UUID) (*domain.VolunteerAgreement, error) {
	args := m.Called(ctx, id, documentPath, uploadedBy)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.VolunteerAgreement), args.Error(1)
}

func newTestPersonService() (*PersonService, *MockPersonRepository, *MockPersonRoleRepository, *MockAuditRepository) {
	personRepo := new(MockPersonRepository)
	roleRepo := new(MockPersonRoleRepository)
	agreementRepo := new(MockVolunteerAgreementRepository)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	svc := NewPersonService(personRepo, roleRepo, agreementRepo, auditSvc)

	// Default: agreement creation succeeds (for tests that trigger it)
	agreementRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.VolunteerAgreement")).
		Return(&domain.VolunteerAgreement{ID: uuid.New()}, nil).Maybe()

	return svc, personRepo, roleRepo, auditRepo
}

func TestPersonService_CreatePerson(t *testing.T) {
	t.Run("success with address", func(t *testing.T) {
		svc, personRepo, _, auditRepo := newTestPersonService()
		ctx, _ := newPersonTestContext()

		input := CreatePersonInput{
			FullName:     "Maria Silva",
			DocumentType: "CPF",
			Address:      &AddressInput{City: strPtr("Sao Paulo")},
		}

		personRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.Person"), mock.AnythingOfType("*domain.Address")).
			Return(&domain.Person{ID: uuid.New(), FullName: "Maria Silva", IsActive: true}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.CreatePerson(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "Maria Silva", result.FullName)
		personRepo.AssertExpectations(t)
	})

	t.Run("success without address", func(t *testing.T) {
		svc, personRepo, _, auditRepo := newTestPersonService()
		ctx, _ := newPersonTestContext()

		input := CreatePersonInput{
			FullName:     "Joao Santos",
			DocumentType: "CPF",
		}

		personRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.Person"), mock.Anything).
			Return(&domain.Person{ID: uuid.New(), FullName: "Joao Santos", IsActive: true}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.CreatePerson(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "Joao Santos", result.FullName)
	})

	t.Run("validation error missing name", func(t *testing.T) {
		svc, _, _, _ := newTestPersonService()
		ctx, _ := newPersonTestContext()

		input := CreatePersonInput{DocumentType: "CPF"}

		_, err := svc.CreatePerson(ctx, input)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "FullName")
	})

	t.Run("validation error invalid document type", func(t *testing.T) {
		svc, _, _, _ := newTestPersonService()
		ctx, _ := newPersonTestContext()

		input := CreatePersonInput{FullName: "Test", DocumentType: "INVALID"}

		_, err := svc.CreatePerson(ctx, input)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "DocumentType")
	})

	t.Run("uses sync_id as person ID", func(t *testing.T) {
		svc, personRepo, _, auditRepo := newTestPersonService()
		ctx, _ := newPersonTestContext()

		syncID := uuid.New().String()
		input := CreatePersonInput{
			FullName:     "Offline Person",
			DocumentType: "CPF",
			SyncID:       &syncID,
		}

		personRepo.On("Create", mock.Anything, mock.MatchedBy(func(p domain.Person) bool {
			return p.ID.String() == syncID
		}), mock.Anything).
			Return(&domain.Person{ID: uuid.MustParse(syncID), FullName: "Offline Person"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.CreatePerson(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, syncID, result.ID.String())
	})
}

// TestPersonService_CreatePerson_DocumentFormat covers international document
// format validation (S09.2): CPF failures keep the specific ErrInvalidCPF, other
// document types surface the generic ErrInvalidDocumentFormat, and a valid RG is
// accepted.
func TestPersonService_CreatePerson_DocumentFormat(t *testing.T) {
	t.Run("invalid CPF returns ErrInvalidCPF", func(t *testing.T) {
		svc, _, _, _ := newTestPersonService()
		ctx, _ := newPersonTestContext()

		badCPF := "529.982.247-26"
		input := CreatePersonInput{
			FullName:       "Bad CPF",
			DocumentType:   "CPF",
			DocumentNumber: &badCPF,
		}

		_, err := svc.CreatePerson(ctx, input)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidCPF)
	})

	t.Run("invalid SSN returns ErrInvalidDocumentFormat", func(t *testing.T) {
		svc, _, _, _ := newTestPersonService()
		ctx, _ := newPersonTestContext()

		badSSN := "not-a-ssn"
		input := CreatePersonInput{
			FullName:       "Bad SSN",
			DocumentType:   "SSN",
			DocumentNumber: &badSSN,
		}

		_, err := svc.CreatePerson(ctx, input)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrInvalidDocumentFormat)
	})

	t.Run("valid RG document accepted", func(t *testing.T) {
		svc, personRepo, _, auditRepo := newTestPersonService()
		ctx, _ := newPersonTestContext()

		rg := "MG-12.345.678"
		input := CreatePersonInput{
			FullName:       "Valid RG",
			DocumentType:   "RG",
			DocumentNumber: &rg,
		}

		personRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.Person"), mock.Anything).
			Return(&domain.Person{ID: uuid.New(), FullName: "Valid RG", IsActive: true}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.CreatePerson(ctx, input)

		require.NoError(t, err)
		assert.Equal(t, "Valid RG", result.FullName)
	})
}

func TestPersonService_GetPerson(t *testing.T) {
	t.Run("success with details", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		person := &domain.Person{ID: personID, FullName: "Maria Silva", CampusID: claims.CampusID}
		addresses := []domain.Address{{ID: uuid.New(), PersonID: personID, City: strPtr("SP")}}
		roles := []domain.PersonRole{{ID: uuid.New(), PersonID: personID, RoleType: "VOLUNTEER"}}

		personRepo.On("FindByIDWithDetails", mock.Anything, personID, claims.CampusID).
			Return(person, addresses, roles, nil)

		detail, err := svc.GetPerson(ctx, personID)

		require.NoError(t, err)
		assert.Equal(t, "Maria Silva", detail.FullName)
		assert.Len(t, detail.Addresses, 1)
		assert.Len(t, detail.Roles, 1)
	})

	t.Run("not found", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		personRepo.On("FindByIDWithDetails", mock.Anything, personID, claims.CampusID).
			Return(nil, nil, nil, domain.ErrNotFound)

		_, err := svc.GetPerson(ctx, personID)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPersonService_UpdatePerson(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, personRepo, _, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		old := &domain.Person{ID: personID, FullName: "Old Name", CampusID: claims.CampusID, DocumentType: "CPF"}
		updated := &domain.Person{ID: personID, FullName: "New Name", CampusID: claims.CampusID, DocumentType: "CPF"}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).Return(old, nil)
		personRepo.On("Update", mock.Anything, mock.AnythingOfType("domain.Person")).Return(updated, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := UpdatePersonInput{FullName: "New Name", DocumentType: "CPF"}
		result, err := svc.UpdatePerson(ctx, personID, input)

		require.NoError(t, err)
		assert.Equal(t, "New Name", result.FullName)
	})

	t.Run("not found", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(nil, domain.ErrNotFound)

		input := UpdatePersonInput{FullName: "Name", DocumentType: "CPF"}
		_, err := svc.UpdatePerson(ctx, personID, input)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPersonService_ListPersons(t *testing.T) {
	t.Run("with search query", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		expected := &domain.PersonListResult{
			Data:       []domain.PersonListItem{{ID: uuid.New(), FullName: "Maria"}},
			Pagination: domain.Pagination{Page: 1, PerPage: 20, Total: 1, TotalPages: 1},
		}
		personRepo.On("List", mock.Anything, domain.PersonFilter{
			Query: "Maria", CampusID: claims.CampusID, Page: 1, PerPage: 20,
		}).Return(expected, nil)

		result, err := svc.ListPersons(ctx, "Maria", 1, 20, "")

		require.NoError(t, err)
		assert.Len(t, result.Data, 1)
		assert.Equal(t, 1, result.Pagination.Total)
	})

	t.Run("defaults page and perPage", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		expected := &domain.PersonListResult{
			Data:       []domain.PersonListItem{},
			Pagination: domain.Pagination{Page: 1, PerPage: 20, Total: 0, TotalPages: 0},
		}
		personRepo.On("List", mock.Anything, domain.PersonFilter{
			CampusID: claims.CampusID, Page: 1, PerPage: 20,
		}).Return(expected, nil)

		result, err := svc.ListPersons(ctx, "", 0, 0, "")

		require.NoError(t, err)
		assert.Empty(t, result.Data)
	})
}

func TestPersonService_CheckDuplicate(t *testing.T) {
	t.Run("has duplicates", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		expected := &domain.DuplicateCheckResult{
			HasDuplicates: true,
			Matches:       []domain.DuplicateMatch{{ID: uuid.New(), FullName: "Existing", MatchType: "exact_document"}},
		}
		personRepo.On("CheckDuplicate", mock.Anything, "CPF", "123.456.789-00", claims.CampusID).
			Return(expected, nil)

		result, err := svc.CheckDuplicate(ctx, "CPF", "123.456.789-00")

		require.NoError(t, err)
		assert.True(t, result.HasDuplicates)
		assert.Len(t, result.Matches, 1)
	})

	t.Run("no duplicates", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		expected := &domain.DuplicateCheckResult{HasDuplicates: false, Matches: []domain.DuplicateMatch{}}
		personRepo.On("CheckDuplicate", mock.Anything, "CPF", "999.999.999-99", claims.CampusID).
			Return(expected, nil)

		result, err := svc.CheckDuplicate(ctx, "CPF", "999.999.999-99")

		require.NoError(t, err)
		assert.False(t, result.HasDuplicates)
	})
}

func TestPersonService_AddRole(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		person := &domain.Person{ID: personID, CampusID: claims.CampusID}
		createdRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "VOLUNTEER", IsActive: true}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).Return(person, nil)
		roleRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.PersonRole")).Return(createdRole, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := AddRoleInput{RoleType: "VOLUNTEER"}
		result, err := svc.AddRole(ctx, personID, input)

		require.NoError(t, err)
		assert.Equal(t, "VOLUNTEER", result.RoleType)
	})

	t.Run("duplicate role", func(t *testing.T) {
		svc, personRepo, roleRepo, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID, CampusID: claims.CampusID}, nil)
		roleRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.PersonRole")).
			Return(nil, domain.ErrDuplicate)

		input := AddRoleInput{RoleType: "VOLUNTEER"}
		_, err := svc.AddRole(ctx, personID, input)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrDuplicate)
	})

	t.Run("person not found", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(nil, domain.ErrNotFound)

		input := AddRoleInput{RoleType: "VOLUNTEER"}
		_, err := svc.AddRole(ctx, personID, input)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPersonService_ToggleRole(t *testing.T) {
	t.Run("deactivate", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		roleID := uuid.New()
		now := time.Now()
		deactivated := &domain.PersonRole{ID: roleID, PersonID: personID, RoleType: "VOLUNTEER", IsActive: false, DeactivatedAt: &now}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID, CampusID: claims.CampusID}, nil)
		roleRepo.On("ToggleActive", mock.Anything, roleID, false, mock.Anything).Return(deactivated, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.ToggleRole(ctx, personID, roleID, false)

		require.NoError(t, err)
		assert.False(t, result.IsActive)
	})

	t.Run("activate", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		roleID := uuid.New()
		activated := &domain.PersonRole{ID: roleID, PersonID: personID, RoleType: "VOLUNTEER", IsActive: true}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID, CampusID: claims.CampusID}, nil)
		roleRepo.On("ToggleActive", mock.Anything, roleID, true, mock.Anything).Return(activated, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		result, err := svc.ToggleRole(ctx, personID, roleID, true)

		require.NoError(t, err)
		assert.True(t, result.IsActive)
	})
}

func TestPersonService_GetHistory(t *testing.T) {
	t.Run("returns empty slice", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(&domain.Person{ID: personID, CampusID: claims.CampusID}, nil)

		history, err := svc.GetHistory(ctx, personID)

		require.NoError(t, err)
		assert.Empty(t, history)
	})

	t.Run("person not found", func(t *testing.T) {
		svc, personRepo, _, _ := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).
			Return(nil, domain.ErrNotFound)

		_, err := svc.GetHistory(ctx, personID)

		require.Error(t, err)
		assert.ErrorIs(t, err, domain.ErrNotFound)
	})
}

func TestPersonService_AddRole_Hierarchy(t *testing.T) {
	t.Run("adding PROFESSIONAL auto-creates VOLUNTEER", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		person := &domain.Person{ID: personID, CampusID: claims.CampusID}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).Return(person, nil)
		// No existing roles
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{}, nil)
		// Auto-create VOLUNTEER
		volunteerRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "VOLUNTEER", IsActive: true}
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
			return r.RoleType == "VOLUNTEER"
		})).Return(volunteerRole, nil).Once()
		// Then create PROFESSIONAL
		professionalRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "PROFESSIONAL", IsActive: true}
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
			return r.RoleType == "PROFESSIONAL"
		})).Return(professionalRole, nil).Once()
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := AddRoleInput{RoleType: "PROFESSIONAL"}
		result, err := svc.AddRole(ctx, personID, input)

		require.NoError(t, err)
		assert.Equal(t, "PROFESSIONAL", result.RoleType)
		// Verify VOLUNTEER was also created
		roleRepo.AssertCalled(t, "FindByPersonID", mock.Anything, personID)
	})

	t.Run("adding ADMIN auto-creates VOLUNTEER if missing", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		person := &domain.Person{ID: personID, CampusID: claims.CampusID}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).Return(person, nil)
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return([]domain.PersonRole{}, nil)
		volunteerRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "VOLUNTEER", IsActive: true}
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
			return r.RoleType == "VOLUNTEER"
		})).Return(volunteerRole, nil).Once()
		adminRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "ADMIN", IsActive: true}
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
			return r.RoleType == "ADMIN"
		})).Return(adminRole, nil).Once()
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := AddRoleInput{RoleType: "ADMIN"}
		result, err := svc.AddRole(ctx, personID, input)

		require.NoError(t, err)
		assert.Equal(t, "ADMIN", result.RoleType)
	})

	t.Run("adding PROFESSIONAL skips VOLUNTEER if already exists", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		person := &domain.Person{ID: personID, CampusID: claims.CampusID}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).Return(person, nil)
		// VOLUNTEER already exists
		existingRoles := []domain.PersonRole{{ID: uuid.New(), PersonID: personID, RoleType: "VOLUNTEER", IsActive: true}}
		roleRepo.On("FindByPersonID", mock.Anything, personID).Return(existingRoles, nil)
		// Only PROFESSIONAL is created
		professionalRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "PROFESSIONAL", IsActive: true}
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
			return r.RoleType == "PROFESSIONAL"
		})).Return(professionalRole, nil).Once()
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := AddRoleInput{RoleType: "PROFESSIONAL"}
		result, err := svc.AddRole(ctx, personID, input)

		require.NoError(t, err)
		assert.Equal(t, "PROFESSIONAL", result.RoleType)
	})

	t.Run("adding ASSISTED does NOT auto-create VOLUNTEER", func(t *testing.T) {
		svc, personRepo, roleRepo, auditRepo := newTestPersonService()
		ctx, claims := newPersonTestContext()

		personID := uuid.New()
		person := &domain.Person{ID: personID, CampusID: claims.CampusID}

		personRepo.On("FindByID", mock.Anything, personID, claims.CampusID).Return(person, nil)
		assistedRole := &domain.PersonRole{ID: uuid.New(), PersonID: personID, RoleType: "ASSISTED", IsActive: true}
		roleRepo.On("Create", mock.Anything, mock.MatchedBy(func(r domain.PersonRole) bool {
			return r.RoleType == "ASSISTED"
		})).Return(assistedRole, nil).Once()
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := AddRoleInput{RoleType: "ASSISTED"}
		result, err := svc.AddRole(ctx, personID, input)

		require.NoError(t, err)
		assert.Equal(t, "ASSISTED", result.RoleType)
		// FindByPersonID should NOT be called (no hierarchy check)
		roleRepo.AssertNotCalled(t, "FindByPersonID", mock.Anything, personID)
	})
}

func strPtr(s string) *string {
	return &s
}
