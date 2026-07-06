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

// MockDonationRepository implements DonationRepository for testing.
type MockDonationRepository struct {
	mock.Mock
}

func (m *MockDonationRepository) Create(ctx context.Context, d domain.Donation) (*domain.Donation, error) {
	args := m.Called(ctx, d)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Donation), args.Error(1)
}

func (m *MockDonationRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.DonationDetail, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DonationDetail), args.Error(1)
}

func (m *MockDonationRepository) List(ctx context.Context, filter domain.DonationFilter) (*domain.DonationListResult, error) {
	args := m.Called(ctx, filter)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.DonationListResult), args.Error(1)
}

func (m *MockDonationRepository) Update(ctx context.Context, d domain.Donation) (*domain.Donation, error) {
	args := m.Called(ctx, d)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Donation), args.Error(1)
}

func (m *MockDonationRepository) FindReceiptData(ctx context.Context, id, campusID uuid.UUID) (*domain.ReceiptData, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.ReceiptData), args.Error(1)
}

func (m *MockDonationRepository) MarkReceiptIssued(ctx context.Context, id, campusID uuid.UUID, number string, issuedAt time.Time) error {
	args := m.Called(ctx, id, campusID, number, issuedAt)
	return args.Error(0)
}

// stubReceiptRenderer is a minimal ReceiptRenderer for service tests.
type stubReceiptRenderer struct {
	pdf []byte
	err error
}

func (s stubReceiptRenderer) Render(domain.ReceiptData) ([]byte, error) {
	return s.pdf, s.err
}

// MockDonationPersonRepository implements DonationPersonRepository for testing.
type MockDonationPersonRepository struct {
	mock.Mock
}

func (m *MockDonationPersonRepository) FindByID(ctx context.Context, id, campusID uuid.UUID) (*domain.Person, error) {
	args := m.Called(ctx, id, campusID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*domain.Person), args.Error(1)
}

// newTestDonationService wires the service with fresh mocks. MockCampaignRepository
// (declared in campaign_service_test.go) satisfies CampaignRefRepository.
// Storage/renderer are stubbed and only exercised by the receipt tests.
func newTestDonationService() (*DonationService, *MockDonationRepository, *MockDonationPersonRepository, *MockCampaignRepository, *MockAuditRepository) {
	svc, repo, personRepo, campaignRepo, _, auditRepo := newTestDonationServiceWithStorage()
	return svc, repo, personRepo, campaignRepo, auditRepo
}

// newTestDonationServiceWithStorage additionally exposes the storage mock for
// the receipt-issuance tests.
func newTestDonationServiceWithStorage() (*DonationService, *MockDonationRepository, *MockDonationPersonRepository, *MockCampaignRepository, *MockObjectStorage, *MockAuditRepository) {
	repo := new(MockDonationRepository)
	personRepo := new(MockDonationPersonRepository)
	campaignRepo := new(MockCampaignRepository)
	store := new(MockObjectStorage)
	auditRepo := new(MockAuditRepository)
	auditSvc := NewAuditService(auditRepo)
	renderer := stubReceiptRenderer{pdf: []byte("%PDF-1.7 stub")}
	svc := NewDonationService(repo, personRepo, campaignRepo, store, renderer, auditSvc)
	return svc, repo, personRepo, campaignRepo, store, auditRepo
}

func donationTestContext() (context.Context, auth.AuthClaims) {
	claims := auth.AuthClaims{
		Subject:  uuid.New().String(),
		Email:    "secretary@chesed.test",
		Roles:    []string{"SECRETARY"},
		CampusID: uuid.New(),
	}
	return auth.NewContext(context.Background(), claims), claims
}

func ptrString(s string) *string  { return &s }
func ptrFloat(f float64) *float64 { return &f }

func validGoodsInput() CreateDonationInput {
	return CreateDonationInput{
		DonationType:    "GOODS",
		ItemDescription: ptrString("10 blankets"),
	}
}

func validFinancialInput() CreateDonationInput {
	return CreateDonationInput{
		DonationType: "FINANCIAL",
		Amount:       ptrFloat(250.00),
	}
}

func TestDonationService_CreateDonation(t *testing.T) {
	t.Run("goods happy path stamps campus and identity and audits", func(t *testing.T) {
		svc, repo, _, _, auditRepo := newTestDonationService()
		ctx, claims := donationTestContext()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Donation")).
			Return(&domain.Donation{ID: uuid.New(), DonationType: "GOODS", Currency: "BRL"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		created, err := svc.CreateDonation(ctx, validGoodsInput())
		require.NoError(t, err)
		assert.Equal(t, "GOODS", created.DonationType)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Donation)
		assert.Equal(t, claims.CampusID, stored.CampusID)
		assert.NotEqual(t, uuid.Nil, stored.ID)
		assert.NotEqual(t, uuid.Nil, stored.RegisteredBy)
		assert.Equal(t, "BRL", stored.Currency)
		assert.False(t, stored.DonationDate.IsZero(), "donation_date must default to today")
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("financial with amount is accepted", func(t *testing.T) {
		svc, repo, _, _, auditRepo := newTestDonationService()
		ctx, _ := donationTestContext()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Donation")).
			Return(&domain.Donation{ID: uuid.New(), DonationType: "FINANCIAL"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		_, err := svc.CreateDonation(ctx, validFinancialInput())
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Donation)
		require.NotNil(t, stored.Amount)
		assert.Equal(t, 250.00, *stored.Amount)
	})

	t.Run("missing campus context returns forbidden", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()

		_, err := svc.CreateDonation(context.Background(), validGoodsInput())
		require.ErrorIs(t, err, domain.ErrForbidden)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("invalid donation type fails validation", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		input := validGoodsInput()
		input.DonationType = "CRYPTO"
		_, err := svc.CreateDonation(ctx, input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("financial without amount is a validation error", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		input := CreateDonationInput{DonationType: "FINANCIAL"}
		_, err := svc.CreateDonation(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("goods without item description is a validation error", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		input := CreateDonationInput{DonationType: "GOODS"}
		_, err := svc.CreateDonation(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})
}

// TestDonationService_Currency covers multi-currency support (S09.3): the
// supported set is BRL/USD/EUR, an omitted currency defaults to BRL, and any
// other ISO code is rejected at validation before reaching the repository.
func TestDonationService_Currency(t *testing.T) {
	supported := []string{"BRL", "USD", "EUR"}
	for _, currency := range supported {
		t.Run("supported currency "+currency+" is accepted on create", func(t *testing.T) {
			svc, repo, _, _, auditRepo := newTestDonationService()
			ctx, _ := donationTestContext()

			input := validGoodsInput()
			input.Currency = ptrString(currency)

			repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Donation")).
				Return(&domain.Donation{ID: uuid.New(), DonationType: "GOODS", Currency: currency}, nil)
			auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

			_, err := svc.CreateDonation(ctx, input)
			require.NoError(t, err)

			stored := repo.Calls[0].Arguments.Get(1).(domain.Donation)
			assert.Equal(t, currency, stored.Currency)
		})
	}

	t.Run("unsupported currency is rejected on create", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		input := validGoodsInput()
		input.Currency = ptrString("GBP")

		_, err := svc.CreateDonation(ctx, input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("omitted currency defaults to BRL on create", func(t *testing.T) {
		svc, repo, _, _, auditRepo := newTestDonationService()
		ctx, _ := donationTestContext()

		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Donation")).
			Return(&domain.Donation{ID: uuid.New(), DonationType: "GOODS", Currency: "BRL"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		_, err := svc.CreateDonation(ctx, validGoodsInput())
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Donation)
		assert.Equal(t, "BRL", stored.Currency)
	})

	t.Run("unsupported currency is rejected on update", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, claims := donationTestContext()

		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.DonationDetail{Donation: domain.Donation{ID: id, DonationType: "GOODS", CampusID: claims.CampusID}}, nil).Maybe()

		input := UpdateDonationInput{
			DonationType:    "GOODS",
			ItemDescription: ptrString("10 blankets"),
			Currency:        ptrString("GBP"),
		}

		_, err := svc.UpdateDonation(ctx, id, input)
		require.Error(t, err)
		repo.AssertNotCalled(t, "Update")
	})
}

// TestDonationService_CreateDonation_References covers the optional donor and
// campaign linkage resolution (S09.2), which is campus-scoped and rejects
// foreign references generically (threat model T3).
func TestDonationService_CreateDonation_References(t *testing.T) {
	t.Run("donor in campus is accepted", func(t *testing.T) {
		svc, repo, personRepo, _, auditRepo := newTestDonationService()
		ctx, claims := donationTestContext()

		donor := uuid.New()
		donorID := donor.String()
		personRepo.On("FindByID", mock.Anything, donor, claims.CampusID).
			Return(&domain.Person{ID: donor}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Donation")).
			Return(&domain.Donation{ID: uuid.New(), DonationType: "GOODS"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := validGoodsInput()
		input.DonorPersonID = &donorID
		_, err := svc.CreateDonation(ctx, input)
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Donation)
		require.NotNil(t, stored.DonorPersonID)
		assert.Equal(t, donor, *stored.DonorPersonID)
	})

	t.Run("donor outside campus is a generic validation error", func(t *testing.T) {
		svc, repo, personRepo, _, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		donorID := uuid.New().String()
		personRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		input := validGoodsInput()
		input.DonorPersonID = &donorID
		_, err := svc.CreateDonation(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})

	t.Run("campaign in campus is linked", func(t *testing.T) {
		svc, repo, _, campaignRepo, auditRepo := newTestDonationService()
		ctx, claims := donationTestContext()

		campaign := uuid.New()
		campaignID := campaign.String()
		campaignRepo.On("FindByID", mock.Anything, campaign, claims.CampusID).
			Return(&domain.Campaign{ID: campaign, CampusID: claims.CampusID}, nil)
		repo.On("Create", mock.Anything, mock.AnythingOfType("domain.Donation")).
			Return(&domain.Donation{ID: uuid.New(), DonationType: "GOODS"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		input := validGoodsInput()
		input.CampaignID = &campaignID
		_, err := svc.CreateDonation(ctx, input)
		require.NoError(t, err)

		stored := repo.Calls[0].Arguments.Get(1).(domain.Donation)
		require.NotNil(t, stored.CampaignID)
		assert.Equal(t, campaign, *stored.CampaignID)
	})

	t.Run("campaign outside campus is a generic validation error", func(t *testing.T) {
		svc, repo, _, campaignRepo, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		campaignID := uuid.New().String()
		campaignRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		input := validGoodsInput()
		input.CampaignID = &campaignID
		_, err := svc.CreateDonation(ctx, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Create")
	})
}

func TestDonationService_GetDonation(t *testing.T) {
	t.Run("returns detail scoped to campus", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, claims := donationTestContext()

		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.DonationDetail{Donation: domain.Donation{ID: id, DonationType: "GOODS"}, DonorName: ptrString("Maria")}, nil)

		detail, err := svc.GetDonation(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, id, detail.ID)
		require.NotNil(t, detail.DonorName)
		assert.Equal(t, "Maria", *detail.DonorName)
	})

	t.Run("missing campus returns forbidden", func(t *testing.T) {
		svc, _, _, _, _ := newTestDonationService()

		_, err := svc.GetDonation(context.Background(), uuid.New())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestDonationService_ListDonations(t *testing.T) {
	t.Run("applies campus from context", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, claims := donationTestContext()

		repo.On("List", mock.Anything, mock.MatchedBy(func(f domain.DonationFilter) bool {
			return f.CampusID == claims.CampusID
		})).Return(&domain.DonationListResult{Data: []domain.DonationListItem{}}, nil)

		_, err := svc.ListDonations(ctx, domain.DonationFilter{Page: 1, PerPage: 20})
		require.NoError(t, err)
		repo.AssertExpectations(t)
	})

	t.Run("missing campus returns forbidden", func(t *testing.T) {
		svc, _, _, _, _ := newTestDonationService()

		_, err := svc.ListDonations(context.Background(), domain.DonationFilter{})
		require.ErrorIs(t, err, domain.ErrForbidden)
	})
}

func TestDonationService_UpdateDonation(t *testing.T) {
	validUpdate := func() UpdateDonationInput {
		return UpdateDonationInput{
			DonationType:    "GOODS",
			ItemDescription: ptrString("20 blankets"),
		}
	}

	t.Run("success updates fields and audits old values", func(t *testing.T) {
		svc, repo, _, _, auditRepo := newTestDonationService()
		ctx, claims := donationTestContext()

		id := uuid.New()
		registeredBy := uuid.New()
		existing := &domain.DonationDetail{Donation: domain.Donation{
			ID: id, CampusID: claims.CampusID, DonationType: "FINANCIAL",
			Amount: ptrFloat(100), Currency: "BRL", RegisteredBy: registeredBy,
		}}
		repo.On("FindByID", mock.Anything, id, claims.CampusID).Return(existing, nil)
		repo.On("Update", mock.Anything, mock.AnythingOfType("domain.Donation")).
			Return(&domain.Donation{ID: id, DonationType: "GOODS"}, nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)

		updated, err := svc.UpdateDonation(ctx, id, validUpdate())
		require.NoError(t, err)
		assert.Equal(t, "GOODS", updated.DonationType)

		// Immutable fields carried forward from the existing record.
		stored := repo.Calls[1].Arguments.Get(1).(domain.Donation)
		assert.Equal(t, claims.CampusID, stored.CampusID)
		assert.Equal(t, registeredBy, stored.RegisteredBy)
		auditRepo.AssertCalled(t, "Create", mock.Anything, mock.AnythingOfType("domain.AuditLog"))
	})

	t.Run("invalid business rules fail validation", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, claims := donationTestContext()

		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.DonationDetail{Donation: domain.Donation{ID: id, CampusID: claims.CampusID}}, nil)

		input := UpdateDonationInput{DonationType: "FINANCIAL"} // no amount
		_, err := svc.UpdateDonation(ctx, id, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Update")
	})

	t.Run("not found propagates", func(t *testing.T) {
		svc, repo, _, _, _ := newTestDonationService()
		ctx, _ := donationTestContext()

		repo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		_, err := svc.UpdateDonation(ctx, uuid.New(), validUpdate())
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("campaign outside campus is a generic validation error", func(t *testing.T) {
		svc, repo, _, campaignRepo, _ := newTestDonationService()
		ctx, claims := donationTestContext()

		id := uuid.New()
		repo.On("FindByID", mock.Anything, id, claims.CampusID).
			Return(&domain.DonationDetail{Donation: domain.Donation{ID: id, CampusID: claims.CampusID}}, nil)
		campaignRepo.On("FindByID", mock.Anything, mock.Anything, mock.Anything).
			Return(nil, domain.ErrNotFound)

		input := validUpdate()
		foreign := uuid.New().String()
		input.CampaignID = &foreign
		_, err := svc.UpdateDonation(ctx, id, input)
		require.ErrorIs(t, err, domain.ErrValidation)
		repo.AssertNotCalled(t, "Update")
	})
}

func TestDonationService_GenerateReceipt(t *testing.T) {
	t.Run("first issuance renders, stores, stamps, audits and presigns", func(t *testing.T) {
		svc, repo, _, _, store, auditRepo := newTestDonationServiceWithStorage()
		ctx, claims := donationTestContext()

		id := uuid.New()
		repo.On("FindReceiptData", mock.Anything, id, claims.CampusID).
			Return(&domain.ReceiptData{
				Donation:   domain.Donation{ID: id, CampusID: claims.CampusID, DonationDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)},
				CampusName: "Campus Central",
			}, nil)
		store.On("Put", mock.Anything, mock.MatchedBy(func(k string) bool {
			return k == "receipts/"+claims.CampusID.String()+"/"+id.String()+".pdf"
		}), "application/pdf", mock.AnythingOfType("int64"), mock.Anything).Return(nil)
		repo.On("MarkReceiptIssued", mock.Anything, id, claims.CampusID, mock.MatchedBy(func(n string) bool {
			return len(n) > 0 && n[:9] == "REC-2026-"
		}), mock.AnythingOfType("time.Time")).Return(nil)
		auditRepo.On("Create", mock.Anything, mock.AnythingOfType("domain.AuditLog")).Return(nil)
		store.On("PresignGet", mock.Anything, mock.Anything, mock.Anything).Return("https://signed", nil)

		dl, err := svc.GenerateReceipt(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "https://signed", dl.URL)
		store.AssertCalled(t, "Put", mock.Anything, mock.Anything, "application/pdf", mock.Anything, mock.Anything)
		repo.AssertCalled(t, "MarkReceiptIssued", mock.Anything, id, claims.CampusID, mock.Anything, mock.Anything)
	})

	t.Run("already issued skips generation and just presigns", func(t *testing.T) {
		svc, repo, _, _, store, _ := newTestDonationServiceWithStorage()
		ctx, claims := donationTestContext()

		id := uuid.New()
		existing := "REC-2026-DEADBEEF"
		repo.On("FindReceiptData", mock.Anything, id, claims.CampusID).
			Return(&domain.ReceiptData{
				Donation:   domain.Donation{ID: id, CampusID: claims.CampusID, ReceiptNumber: &existing},
				CampusName: "Campus Central",
			}, nil)
		store.On("PresignGet", mock.Anything, mock.Anything, mock.Anything).Return("https://signed", nil)

		dl, err := svc.GenerateReceipt(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "https://signed", dl.URL)
		store.AssertNotCalled(t, "Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
		repo.AssertNotCalled(t, "MarkReceiptIssued", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything)
	})

	t.Run("missing campus is forbidden", func(t *testing.T) {
		svc, _, _, _, _, _ := newTestDonationServiceWithStorage()
		_, err := svc.GenerateReceipt(context.Background(), uuid.New())
		require.ErrorIs(t, err, domain.ErrForbidden)
	})

	t.Run("not found propagates", func(t *testing.T) {
		svc, repo, _, _, _, _ := newTestDonationServiceWithStorage()
		ctx, claims := donationTestContext()
		id := uuid.New()
		repo.On("FindReceiptData", mock.Anything, id, claims.CampusID).Return(nil, domain.ErrNotFound)
		_, err := svc.GenerateReceipt(ctx, id)
		require.ErrorIs(t, err, domain.ErrNotFound)
	})

	t.Run("duplicate on stamp is treated as already-issued and re-presigns", func(t *testing.T) {
		// A concurrent first-call won the race and stamped the row; MarkReceiptIssued
		// returns ErrDuplicate. The losing call must fall through to presign the
		// (already-written) object rather than surfacing a spurious 409.
		svc, repo, _, _, store, _ := newTestDonationServiceWithStorage()
		ctx, claims := donationTestContext()
		id := uuid.New()
		repo.On("FindReceiptData", mock.Anything, id, claims.CampusID).
			Return(&domain.ReceiptData{Donation: domain.Donation{ID: id, CampusID: claims.CampusID, DonationDate: time.Now()}}, nil)
		store.On("Put", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)
		repo.On("MarkReceiptIssued", mock.Anything, id, claims.CampusID, mock.Anything, mock.Anything).Return(domain.ErrDuplicate)
		store.On("PresignGet", mock.Anything, mock.Anything, mock.Anything).Return("https://signed", nil)

		dl, err := svc.GenerateReceipt(ctx, id)
		require.NoError(t, err)
		assert.Equal(t, "https://signed", dl.URL)
	})
}
