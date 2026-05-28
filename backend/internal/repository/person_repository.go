package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/instituto-nova-sos/chesed/internal/domain"
)

// PersonRepository handles person persistence.
type PersonRepository struct {
	pool *pgxpool.Pool
}

// NewPersonRepository creates a new PersonRepository.
func NewPersonRepository(pool *pgxpool.Pool) *PersonRepository {
	return &PersonRepository{pool: pool}
}

// Create inserts a person and optional address in a single transaction.
func (r *PersonRepository) Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("personRepository.Create: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	personQuery := `
		INSERT INTO person (id, full_name, birth_date, document_type, document_number,
		                     gender, email, phone, photo_url, referral_source,
		                     nationality, campus_id, is_active, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		RETURNING created_at, updated_at`

	err = tx.QueryRow(ctx, personQuery,
		person.ID, person.FullName, person.BirthDate, person.DocumentType,
		person.DocumentNumber, person.Gender, person.Email, person.Phone,
		person.PhotoURL, person.ReferralSource, person.Nationality, person.CampusID,
		person.IsActive, person.CreatedBy,
	).Scan(&person.CreatedAt, &person.UpdatedAt)
	if err != nil {
		if dupErr := classifyUniqueViolation(err); dupErr != nil {
			return nil, dupErr
		}
		return nil, fmt.Errorf("personRepository.Create: insert person: %w", err)
	}

	if address != nil {
		addressQuery := `
			INSERT INTO address (id, person_id, street, number, complement,
			                     neighborhood, city, state, zip_code, country, is_primary)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			RETURNING created_at, updated_at`

		err = tx.QueryRow(ctx, addressQuery,
			address.ID, person.ID, address.Street, address.Number,
			address.Complement, address.Neighborhood, address.City,
			address.State, address.ZipCode, address.Country, address.IsPrimary,
		).Scan(&address.CreatedAt, &address.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("personRepository.Create: insert address: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("personRepository.Create: commit: %w", err)
	}

	return &person, nil
}

// FindByID returns a person by ID, scoped to campus.
func (r *PersonRepository) FindByID(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, error) {
	query := `
		SELECT id, full_name, birth_date, document_type, document_number,
		       gender, email, phone, photo_url, referral_source,
		       nationality, campus_id, is_active, created_at, updated_at, created_by
		FROM person
		WHERE id = $1 AND campus_id = $2 AND is_active = TRUE`

	var p domain.Person
	err := r.pool.QueryRow(ctx, query, id, campusID).Scan(
		&p.ID, &p.FullName, &p.BirthDate, &p.DocumentType, &p.DocumentNumber,
		&p.Gender, &p.Email, &p.Phone, &p.PhotoURL, &p.ReferralSource,
		&p.Nationality, &p.CampusID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("personRepository.FindByID: %w", err)
	}

	return &p, nil
}

// FindByEmail returns an active person by email, scoped to campus.
func (r *PersonRepository) FindByEmail(ctx context.Context, email string, campusID uuid.UUID) (*domain.Person, error) {
	query := `
		SELECT id, full_name, birth_date, document_type, document_number,
		       gender, email, phone, photo_url, referral_source,
		       nationality, campus_id, is_active, created_at, updated_at, created_by
		FROM person
		WHERE email = $1 AND campus_id = $2 AND is_active = TRUE`

	var p domain.Person
	err := r.pool.QueryRow(ctx, query, email, campusID).Scan(
		&p.ID, &p.FullName, &p.BirthDate, &p.DocumentType, &p.DocumentNumber,
		&p.Gender, &p.Email, &p.Phone, &p.PhotoURL, &p.ReferralSource,
		&p.Nationality, &p.CampusID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("personRepository.FindByEmail: %w", err)
	}

	return &p, nil
}

// FindByEmailGlobal returns all active persons matching an email, across all campuses.
// Used during onboarding when the user's campus is not yet known.
func (r *PersonRepository) FindByEmailGlobal(ctx context.Context, email string) ([]domain.Person, error) {
	query := `
		SELECT id, full_name, birth_date, document_type, document_number,
		       gender, email, phone, photo_url, referral_source,
		       nationality, campus_id, is_active, created_at, updated_at, created_by
		FROM person
		WHERE lower(email) = lower($1) AND is_active = TRUE`

	rows, err := r.pool.Query(ctx, query, email)
	if err != nil {
		return nil, fmt.Errorf("personRepository.FindByEmailGlobal: %w", err)
	}
	defer rows.Close()

	var persons []domain.Person
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(
			&p.ID, &p.FullName, &p.BirthDate, &p.DocumentType, &p.DocumentNumber,
			&p.Gender, &p.Email, &p.Phone, &p.PhotoURL, &p.ReferralSource,
			&p.Nationality, &p.CampusID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("personRepository.FindByEmailGlobal: scan: %w", err)
		}
		persons = append(persons, p)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personRepository.FindByEmailGlobal: rows: %w", err)
	}

	return persons, nil
}

// FindByIDWithDetails returns a person with addresses and roles.
func (r *PersonRepository) FindByIDWithDetails(ctx context.Context, id uuid.UUID, campusID uuid.UUID) (*domain.Person, []domain.Address, []domain.PersonRole, error) {
	person, err := r.FindByID(ctx, id, campusID)
	if err != nil {
		return nil, nil, nil, err
	}

	addresses, err := r.findAddresses(ctx, id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("personRepository.FindByIDWithDetails: addresses: %w", err)
	}

	roles, err := r.findRoles(ctx, id)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("personRepository.FindByIDWithDetails: roles: %w", err)
	}

	return person, addresses, roles, nil
}

// Update updates a person record.
func (r *PersonRepository) Update(ctx context.Context, person domain.Person) (*domain.Person, error) {
	query := `
		UPDATE person
		SET full_name = $1, birth_date = $2, document_type = $3, document_number = $4,
		    gender = $5, email = $6, phone = $7, photo_url = $8, referral_source = $9,
		    nationality = $10, updated_at = NOW()
		WHERE id = $11 AND campus_id = $12 AND is_active = TRUE
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		person.FullName, person.BirthDate, person.DocumentType, person.DocumentNumber,
		person.Gender, person.Email, person.Phone, person.PhotoURL, person.ReferralSource,
		person.Nationality, person.ID, person.CampusID,
	).Scan(&person.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("personRepository.Update: %w", err)
	}

	return &person, nil
}

// List returns a paginated list of persons matching the filter.
func (r *PersonRepository) List(ctx context.Context, filter domain.PersonFilter) (*domain.PersonListResult, error) {
	offset := (filter.Page - 1) * filter.PerPage

	// Build dynamic query with optional agreement filter
	var sb strings.Builder
	args := []any{filter.CampusID}
	argIdx := 2

	sb.WriteString(`
		SELECT p.id, p.full_name, p.document_number, p.phone, p.is_active,
		       COUNT(*) OVER() AS total_count,
		       COALESCE(
		           (SELECT array_agg(pr2.role_type) FROM person_role pr2 WHERE pr2.person_id = p.id AND pr2.is_active = TRUE),
		           '{}'
		       ) AS roles
		FROM person p`)

	// Join for agreement filtering
	if filter.AgreementStatus != "" {
		sb.WriteString(`
		LEFT JOIN person_role pr_vol ON pr_vol.person_id = p.id AND pr_vol.role_type = 'VOLUNTEER' AND pr_vol.is_active = TRUE
		LEFT JOIN volunteer_agreement va ON va.person_role_id = pr_vol.id`)
	}

	sb.WriteString(`
		WHERE p.campus_id = $1 AND p.is_active = TRUE`)

	if filter.Query != "" {
		fmt.Fprintf(&sb, `
		  AND p.search_vector @@ plainto_tsquery('portuguese', $%d)`, argIdx)
		args = append(args, filter.Query)
		argIdx++
	}

	// Agreement filter conditions
	switch filter.AgreementStatus {
	case "with_agreement":
		sb.WriteString(`
		  AND pr_vol.id IS NOT NULL AND va.status = 'ACCEPTED'`)
	case "without_agreement":
		sb.WriteString(`
		  AND pr_vol.id IS NOT NULL AND (va.id IS NULL OR va.status = 'PENDING')`)
	case "rejected":
		sb.WriteString(`
		  AND pr_vol.id IS NOT NULL AND va.status = 'REJECTED'`)
	}

	if filter.Query != "" {
		sb.WriteString(`
		ORDER BY ts_rank(p.search_vector, plainto_tsquery('portuguese', $2)) DESC, p.full_name`)
	} else {
		sb.WriteString(`
		ORDER BY p.full_name`)
	}

	fmt.Fprintf(&sb, `
		LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, filter.PerPage, offset)

	rows, err := r.pool.Query(ctx, sb.String(), args...)
	if err != nil {
		return nil, fmt.Errorf("personRepository.List: %w", err)
	}
	defer rows.Close()

	var items []domain.PersonListItem
	var totalCount int

	for rows.Next() {
		var item domain.PersonListItem
		if err := rows.Scan(
			&item.ID, &item.FullName, &item.DocumentNumber, &item.Phone,
			&item.IsActive, &totalCount, &item.Roles,
		); err != nil {
			return nil, fmt.Errorf("personRepository.List: scan: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personRepository.List: rows: %w", err)
	}

	if items == nil {
		items = []domain.PersonListItem{}
	}

	totalPages := int(math.Ceil(float64(totalCount) / float64(filter.PerPage)))

	return &domain.PersonListResult{
		Data: items,
		Pagination: domain.Pagination{
			Page:       filter.Page,
			PerPage:    filter.PerPage,
			Total:      totalCount,
			TotalPages: totalPages,
		},
	}, nil
}

// CheckDuplicate checks for existing persons with the same document.
func (r *PersonRepository) CheckDuplicate(ctx context.Context, documentType, documentNumber string, campusID uuid.UUID) (*domain.DuplicateCheckResult, error) {
	query := `
		SELECT p.id, p.full_name, p.document_number, c.name AS campus_name
		FROM person p
		JOIN campus c ON c.id = p.campus_id
		WHERE p.document_type = $1 AND p.document_number = $2
		  AND p.is_active = TRUE`

	rows, err := r.pool.Query(ctx, query, documentType, documentNumber)
	if err != nil {
		return nil, fmt.Errorf("personRepository.CheckDuplicate: %w", err)
	}
	defer rows.Close()

	var matches []domain.DuplicateMatch
	for rows.Next() {
		var m domain.DuplicateMatch
		if err := rows.Scan(&m.ID, &m.FullName, &m.DocumentNumber, &m.Campus); err != nil {
			return nil, fmt.Errorf("personRepository.CheckDuplicate: scan: %w", err)
		}
		m.MatchType = "exact_document"
		matches = append(matches, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personRepository.CheckDuplicate: rows: %w", err)
	}

	if matches == nil {
		matches = []domain.DuplicateMatch{}
	}

	return &domain.DuplicateCheckResult{
		HasDuplicates: len(matches) > 0,
		Matches:       matches,
	}, nil
}

// UpdateAddress upserts the primary address for a person.
func (r *PersonRepository) UpdateAddress(ctx context.Context, personID uuid.UUID, address domain.Address) (*domain.Address, error) {
	query := `
		INSERT INTO address (id, person_id, street, number, complement,
		                     neighborhood, city, state, zip_code, country, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE)
		ON CONFLICT (person_id, is_primary) WHERE is_primary = TRUE
		DO UPDATE SET street = $3, number = $4, complement = $5,
		              neighborhood = $6, city = $7, state = $8,
		              zip_code = $9, country = $10, updated_at = NOW()
		RETURNING id, created_at, updated_at`

	err := r.pool.QueryRow(ctx, query,
		address.ID, personID, address.Street, address.Number,
		address.Complement, address.Neighborhood, address.City,
		address.State, address.ZipCode, address.Country,
	).Scan(&address.ID, &address.CreatedAt, &address.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("personRepository.UpdateAddress: %w", err)
	}

	return &address, nil
}

func (r *PersonRepository) findAddresses(ctx context.Context, personID uuid.UUID) ([]domain.Address, error) {
	query := `
		SELECT id, person_id, street, number, complement, neighborhood,
		       city, state, zip_code, country, is_primary, created_at, updated_at
		FROM address
		WHERE person_id = $1
		ORDER BY is_primary DESC, created_at`

	rows, err := r.pool.Query(ctx, query, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var addresses []domain.Address
	for rows.Next() {
		var a domain.Address
		if err := rows.Scan(
			&a.ID, &a.PersonID, &a.Street, &a.Number, &a.Complement,
			&a.Neighborhood, &a.City, &a.State, &a.ZipCode, &a.Country,
			&a.IsPrimary, &a.CreatedAt, &a.UpdatedAt,
		); err != nil {
			return nil, err
		}
		addresses = append(addresses, a)
	}

	if addresses == nil {
		addresses = []domain.Address{}
	}
	return addresses, rows.Err()
}

func (r *PersonRepository) findRoles(ctx context.Context, personID uuid.UUID) ([]domain.PersonRole, error) {
	query := `
		SELECT id, person_id, role_type, professional_specialty, is_active,
		       activated_at, deactivated_at, activated_by, deactivated_by, notes
		FROM person_role
		WHERE person_id = $1
		ORDER BY activated_at`

	rows, err := r.pool.Query(ctx, query, personID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var roles []domain.PersonRole
	for rows.Next() {
		var pr domain.PersonRole
		if err := rows.Scan(
			&pr.ID, &pr.PersonID, &pr.RoleType, &pr.ProfessionalSpecialty,
			&pr.IsActive, &pr.ActivatedAt, &pr.DeactivatedAt,
			&pr.ActivatedBy, &pr.DeactivatedBy, &pr.Notes,
		); err != nil {
			return nil, err
		}
		roles = append(roles, pr)
	}

	if roles == nil {
		roles = []domain.PersonRole{}
	}
	return roles, rows.Err()
}

// classifyUniqueViolation maps PostgreSQL unique constraint violations
// to specific domain errors based on the constraint name.
func classifyUniqueViolation(err error) error {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return nil
	}

	switch pgErr.ConstraintName {
	case "uq_person_email_campus":
		return domain.ErrDuplicateEmail
	case "uq_person_phone_campus":
		return domain.ErrDuplicatePhone
	default:
		return domain.ErrDuplicate
	}
}
