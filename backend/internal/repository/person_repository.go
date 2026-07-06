package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/instituto-nova-sos/chesed/internal/domain"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// PersonRepository handles person persistence.
type PersonRepository struct {
	base
}

// NewPersonRepository creates a new PersonRepository.
func NewPersonRepository(pool Querier) *PersonRepository {
	return &PersonRepository{base: base{pool: pool}}
}

// Create inserts a person and optional address in a single transaction.
func (r *PersonRepository) Create(ctx context.Context, person domain.Person, address *domain.Address) (*domain.Person, error) {
	tx, err := r.q(ctx).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("personRepository.Create: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

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

	if err := insertAddress(ctx, tx, person.ID, address); err != nil {
		return nil, fmt.Errorf("personRepository.Create: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("personRepository.Create: commit: %w", err)
	}

	return &person, nil
}

// insertAddress inserts the optional primary address within an open transaction.
// It is a no-op when address is nil.
func insertAddress(ctx context.Context, tx pgx.Tx, personID uuid.UUID, address *domain.Address) error {
	if address == nil {
		return nil
	}
	const addressQuery = `
		INSERT INTO address (id, person_id, street, number, complement,
		                     neighborhood, city, state, zip_code, country, is_primary)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`
	err := tx.QueryRow(ctx, addressQuery,
		address.ID, personID, address.Street, address.Number,
		address.Complement, address.Neighborhood, address.City,
		address.State, address.ZipCode, address.Country, address.IsPrimary,
	).Scan(&address.CreatedAt, &address.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert address: %w", err)
	}
	return nil
}

// CreateWithSync inserts a person carrying the client-supplied sync_id.
// Idempotency is enforced at the DB layer by the uq_person_sync_id partial
// unique index — a concurrent push with the same sync_id raises a unique
// violation, which the caller maps to ErrDuplicate.
func (r *PersonRepository) CreateWithSync(ctx context.Context, person domain.Person, address *domain.Address, syncID uuid.UUID) (*domain.Person, error) {
	tx, err := r.q(ctx).Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("personRepository.CreateWithSync: begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	const q = `
		INSERT INTO person (id, full_name, birth_date, document_type, document_number,
		                    gender, email, phone, photo_url, referral_source,
		                    nationality, campus_id, is_active, created_by, sync_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING created_at, updated_at`

	if err := tx.QueryRow(ctx, q,
		person.ID, person.FullName, person.BirthDate, person.DocumentType,
		person.DocumentNumber, person.Gender, person.Email, person.Phone,
		person.PhotoURL, person.ReferralSource, person.Nationality, person.CampusID,
		person.IsActive, person.CreatedBy, syncID,
	).Scan(&person.CreatedAt, &person.UpdatedAt); err != nil {
		if dupErr := classifyUniqueViolation(err); dupErr != nil {
			return nil, dupErr
		}
		return nil, fmt.Errorf("personRepository.CreateWithSync: insert: %w", err)
	}

	if err := insertAddress(ctx, tx, person.ID, address); err != nil {
		return nil, fmt.Errorf("personRepository.CreateWithSync: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("personRepository.CreateWithSync: commit: %w", err)
	}
	return &person, nil
}

// FindBySyncID returns a person by its client-supplied sync_id, scoped to campus.
// Used by the sync engine to detect already-applied push records (idempotency).
func (r *PersonRepository) FindBySyncID(ctx context.Context, syncID, campusID uuid.UUID) (*domain.Person, error) {
	const q = `
		SELECT id, full_name, birth_date, document_type, document_number,
		       gender, email, phone, photo_url, referral_source,
		       nationality, campus_id, is_active, created_at, updated_at, created_by
		FROM person
		WHERE sync_id = $1 AND campus_id = $2`

	var p domain.Person
	err := r.q(ctx).QueryRow(ctx, q, syncID, campusID).Scan(
		&p.ID, &p.FullName, &p.BirthDate, &p.DocumentType, &p.DocumentNumber,
		&p.Gender, &p.Email, &p.Phone, &p.PhotoURL, &p.ReferralSource,
		&p.Nationality, &p.CampusID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("personRepository.FindBySyncID: %w", err)
	}
	return &p, nil
}

// ListUpdatedSince returns persons modified after the cursor, ordered ASC by
// updated_at. The limit bounds the page so callers can detect has_more by
// comparing len(result) to limit.
func (r *PersonRepository) ListUpdatedSince(ctx context.Context, campusID uuid.UUID, since time.Time, limit int) ([]domain.Person, error) {
	const q = `
		SELECT id, full_name, birth_date, document_type, document_number,
		       gender, email, phone, photo_url, referral_source,
		       nationality, campus_id, is_active, created_at, updated_at, created_by
		FROM person
		WHERE campus_id = $1 AND updated_at > $2
		ORDER BY updated_at ASC
		LIMIT $3`

	rows, err := r.q(ctx).Query(ctx, q, campusID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("personRepository.ListUpdatedSince: %w", err)
	}
	defer rows.Close()

	out := []domain.Person{}
	for rows.Next() {
		var p domain.Person
		if err := rows.Scan(
			&p.ID, &p.FullName, &p.BirthDate, &p.DocumentType, &p.DocumentNumber,
			&p.Gender, &p.Email, &p.Phone, &p.PhotoURL, &p.ReferralSource,
			&p.Nationality, &p.CampusID, &p.IsActive, &p.CreatedAt, &p.UpdatedAt, &p.CreatedBy,
		); err != nil {
			return nil, fmt.Errorf("personRepository.ListUpdatedSince: scan: %w", err)
		}
		out = append(out, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("personRepository.ListUpdatedSince: rows: %w", err)
	}
	return out, nil
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
	err := r.q(ctx).QueryRow(ctx, query, id, campusID).Scan(
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
	err := r.q(ctx).QueryRow(ctx, query, email, campusID).Scan(
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

	rows, err := r.q(ctx).Query(ctx, query, email)
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

	err := r.q(ctx).QueryRow(ctx, query,
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

// Anonymize scrubs a person's PII and its address rows in place (LGPD right to
// erasure, S11.3). The row is kept for referential integrity; full_name gets a
// placeholder (the column is NOT NULL) and document_number a per-person sentinel
// so a second anonymization never trips the uq_person_document NULLS-NOT-DISTINCT
// unique index. The sentinel is 'ANON-' + the first 25 hex digits of the id (30
// chars total) to fit document_number's VARCHAR(30) while staying unique. The
// person UPDATE is not gated on is_active (an already-deactivated person can
// still hold PII). search_vector is refreshed automatically by the person
// BEFORE-UPDATE trigger. Both statements run via the request-scoped campus
// transaction, so they commit or roll back together.
func (r *PersonRepository) Anonymize(ctx context.Context, personID, campusID uuid.UUID) error {
	const personQuery = `
		UPDATE person
		SET full_name = '[ANONYMIZED]',
		    document_number = 'ANON-' || left(replace(id::text, '-', ''), 25),
		    email = NULL, phone = NULL, photo_url = NULL, referral_source = NULL,
		    birth_date = NULL, gender = NULL,
		    anonymized_at = NOW(), updated_at = NOW()
		WHERE id = $1 AND campus_id = $2`

	tag, err := r.q(ctx).Exec(ctx, personQuery, personID, campusID)
	if err != nil {
		return fmt.Errorf("personRepository.Anonymize: person: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return domain.ErrNotFound
	}

	const addressQuery = `
		UPDATE address
		SET street = NULL, number = NULL, complement = NULL, neighborhood = NULL,
		    city = NULL, state = NULL, zip_code = NULL, updated_at = NOW()
		WHERE person_id = $1`

	if _, err := r.q(ctx).Exec(ctx, addressQuery, personID); err != nil {
		return fmt.Errorf("personRepository.Anonymize: address: %w", err)
	}
	return nil
}

// List returns a paginated list of persons matching the filter.
func appendAgreementCondition(sb *strings.Builder, status string) {
	switch status {
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
}

func buildPersonListQuery(filter domain.PersonFilter, offset int) (string, []any) {
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

	appendAgreementCondition(&sb, filter.AgreementStatus)

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
	return sb.String(), args
}

func (r *PersonRepository) List(ctx context.Context, filter domain.PersonFilter) (*domain.PersonListResult, error) {
	offset := (filter.Page - 1) * filter.PerPage
	query, args := buildPersonListQuery(filter, offset)

	rows, err := r.q(ctx).Query(ctx, query, args...)
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
	// campus_id scoping is mandatory (CLAUDE.md rule #4, threat model T3): a
	// caller must only ever learn about duplicates within their own campus.
	query := `
		SELECT p.id, p.full_name, p.document_number
		FROM person p
		WHERE p.document_type = $1 AND p.document_number = $2
		  AND p.campus_id = $3
		  AND p.is_active = TRUE`

	rows, err := r.q(ctx).Query(ctx, query, documentType, documentNumber, campusID)
	if err != nil {
		return nil, fmt.Errorf("personRepository.CheckDuplicate: %w", err)
	}
	defer rows.Close()

	var matches []domain.DuplicateMatch
	for rows.Next() {
		var m domain.DuplicateMatch
		if err := rows.Scan(&m.ID, &m.FullName, &m.DocumentNumber); err != nil {
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

	err := r.q(ctx).QueryRow(ctx, query,
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

	rows, err := r.q(ctx).Query(ctx, query, personID)
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

	rows, err := r.q(ctx).Query(ctx, query, personID)
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
