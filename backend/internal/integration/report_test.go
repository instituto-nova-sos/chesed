//go:build integration

package integration_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// dashboardResponse mirrors domain.DashboardMetrics for decoding assertions.
type dashboardResponse struct {
	TotalPersons         int            `json:"total_persons"`
	AttendancesThisMonth int            `json:"attendances_this_month"`
	UpcomingScheduled    int            `json:"upcoming_scheduled"`
	ActiveCampaigns      int            `json:"active_campaigns"`
	AttendancesByStatus  map[string]int `json:"attendances_by_status"`
	RecentMonths         []struct {
		Month string `json:"month"`
		Count int    `json:"count"`
	} `json:"recent_months"`
}

// seedAttendanceAt inserts an attendance with an explicit attendance_date
// (relative SQL expression) so the dashboard time windows are exercised
// deterministically regardless of the wall clock at test time.
func seedAttendanceAt(t *testing.T, h *testHarness, campusID, personID, professionalID, serviceTypeID uuid.UUID, status, dateExpr string) {
	t.Helper()
	_, err := h.pool.Exec(context.Background(), `
		INSERT INTO attendance (person_id, campus_id, service_type_id, professional_id, status, attendance_date)
		VALUES ($1, $2, $3, $4, $5, `+dateExpr+`)
	`, personID, campusID, serviceTypeID, professionalID, status)
	require.NoError(t, err)
}

// TestDashboard_HappyPath seeds persons, attendances across statuses and time
// windows, plus an ACTIVE campaign, and asserts every dashboard KPI (S10.3).
func TestDashboard_HappyPath(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	seedCampaign(t, h, h.campusID, "Active Dashboard Campaign") // status ACTIVE
	personA := seedPerson(t, h, h.campusID, "Person A", "20120230340")
	personB := seedPerson(t, h, h.campusID, "Person B", "20120230341")
	professional := seedPerson(t, h, h.campusID, "Professional", "20120230342")

	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))

	// Current month: 2 COMPLETED, 1 IN_PROGRESS.
	seedAttendanceAt(t, h, h.campusID, personA, professional, serviceTypeID, "COMPLETED", "date_trunc('month', now()) + INTERVAL '2 days'")
	seedAttendanceAt(t, h, h.campusID, personB, professional, serviceTypeID, "COMPLETED", "date_trunc('month', now()) + INTERVAL '3 days'")
	seedAttendanceAt(t, h, h.campusID, personA, professional, serviceTypeID, "IN_PROGRESS", "date_trunc('month', now()) + INTERVAL '4 days'")
	// Upcoming SCHEDULED (tomorrow) — counts in upcoming_scheduled and this month if same month.
	seedAttendanceAt(t, h, h.campusID, personB, professional, serviceTypeID, "SCHEDULED", "current_date + INTERVAL '10 days'")
	// Past month: 1 COMPLETED (not in this-month, but in recent_months and all-time by_status).
	seedAttendanceAt(t, h, h.campusID, personA, professional, serviceTypeID, "COMPLETED", "date_trunc('month', now()) - INTERVAL '1 month'")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/dashboard", nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got dashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	assert.Equal(t, 3, got.TotalPersons, "personA, personB, professional are all active")
	assert.Equal(t, 4, got.AttendancesThisMonth, "3 seeded this-month rows + the upcoming SCHEDULED in +10 days")
	assert.Equal(t, 1, got.UpcomingScheduled, "only the future SCHEDULED row")
	assert.Equal(t, 1, got.ActiveCampaigns)
	assert.Equal(t, 3, got.AttendancesByStatus["COMPLETED"], "all-time snapshot includes the past-month row")
	assert.Equal(t, 1, got.AttendancesByStatus["IN_PROGRESS"])
	assert.Equal(t, 1, got.AttendancesByStatus["SCHEDULED"])
	require.Len(t, got.RecentMonths, 6, "last 6 calendar months, oldest first")

	// Current month is the last element; it holds this-month attendances count.
	current := got.RecentMonths[len(got.RecentMonths)-1]
	assert.Equal(t, got.AttendancesThisMonth, current.Count,
		"the final recent_months bucket equals this month's volume")
	// The immediately previous bucket holds the single past-month row.
	prev := got.RecentMonths[len(got.RecentMonths)-2]
	assert.Equal(t, 1, prev.Count)
}

// TestDashboard_CampusBoundary proves data seeded in a second campus never
// leaks into the caller's dashboard.
func TestDashboard_CampusBoundary(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	otherCampus := seedSecondCampus(t, h)
	otherPerson := seedPerson(t, h, otherCampus, "Foreign Person", "20120230350")
	otherProf := seedPerson(t, h, otherCampus, "Foreign Professional", "20120230351")
	seedCampaign(t, h, otherCampus, "Foreign Active Campaign")

	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))
	seedAttendanceAt(t, h, otherCampus, otherPerson, otherProf, serviceTypeID, "COMPLETED",
		"date_trunc('month', now()) + INTERVAL '1 day'")

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/dashboard", nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var got dashboardResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &got))

	assert.Zero(t, got.TotalPersons, "foreign-campus persons excluded")
	assert.Zero(t, got.AttendancesThisMonth, "foreign-campus attendances excluded")
	assert.Zero(t, got.ActiveCampaigns, "foreign-campus campaign excluded")
	assert.Empty(t, got.AttendancesByStatus, "no in-campus attendances")
}

// TestDashboard_RBAC proves the endpoint is Coordinator+; a VOLUNTEER is denied.
func TestDashboard_RBAC(t *testing.T) {
	h := freshHarness(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/reports/dashboard", nil)
	rec := h.doRequest(h.authedRequest(req)) // default role is VOLUNTEER
	assert.Equal(t, http.StatusForbidden, rec.Code)
}

// TestAttendanceReport_ProfessionalFilter proves the professional_id filter
// narrows the campus-scoped result set and the by_professional aggregation is
// present and ordered by count desc (S10.1).
func TestAttendanceReport_ProfessionalFilter(t *testing.T) {
	h := freshHarness(t)
	ctx := context.Background()

	person := seedPerson(t, h, h.campusID, "Reported Person", "20120230360")
	profA := seedPerson(t, h, h.campusID, "Alice Professional", "20120230361")
	profB := seedPerson(t, h, h.campusID, "Bob Professional", "20120230362")

	var serviceTypeID uuid.UUID
	require.NoError(t, h.pool.QueryRow(ctx,
		`SELECT id FROM service_type LIMIT 1`).Scan(&serviceTypeID))

	// profA gets 3 attendances, profB gets 1, all inside the report window.
	inWindow := "date_trunc('month', now()) + INTERVAL '2 days'"
	seedAttendanceAt(t, h, h.campusID, person, profA, serviceTypeID, "COMPLETED", inWindow)
	seedAttendanceAt(t, h, h.campusID, person, profA, serviceTypeID, "COMPLETED", inWindow)
	seedAttendanceAt(t, h, h.campusID, person, profA, serviceTypeID, "SCHEDULED", inWindow)
	seedAttendanceAt(t, h, h.campusID, person, profB, serviceTypeID, "COMPLETED", inWindow)

	// Window covering the current month (seeded rows land on day-of-month 3).
	// Must stay within the 366-day report range guard, so anchor on now().
	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, -1)
	start := monthStart.Format("2006-01-02")
	end := monthEnd.Format("2006-01-02")

	// Unfiltered: by_professional lists both, profA first (3 > 1).
	req := httptest.NewRequest(http.MethodGet,
		"/api/v1/reports/attendances?start="+start+"&end="+end, nil)
	rec := h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var report struct {
		TotalAttendances int `json:"total_attendances"`
		ByProfessional   []struct {
			ProfessionalID   uuid.UUID `json:"professional_id"`
			ProfessionalName string    `json:"professional_name"`
			Count            int       `json:"count"`
		} `json:"by_professional"`
	}
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, 4, report.TotalAttendances)
	require.Len(t, report.ByProfessional, 2)
	assert.Equal(t, profA, report.ByProfessional[0].ProfessionalID, "ordered by count desc")
	assert.Equal(t, 3, report.ByProfessional[0].Count)
	assert.Equal(t, "Alice Professional", report.ByProfessional[0].ProfessionalName)
	assert.Equal(t, profB, report.ByProfessional[1].ProfessionalID)
	assert.Equal(t, 1, report.ByProfessional[1].Count)

	// Filtered by profB: totals and by_professional narrow to just profB.
	req = httptest.NewRequest(http.MethodGet,
		"/api/v1/reports/attendances?start="+start+"&end="+end+"&professional_id="+profB.String(), nil)
	rec = h.doRequest(h.authedRequest(req, asCoordinator))
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &report))
	assert.Equal(t, 1, report.TotalAttendances)
	require.Len(t, report.ByProfessional, 1)
	assert.Equal(t, profB, report.ByProfessional[0].ProfessionalID)
	assert.Equal(t, 1, report.ByProfessional[0].Count)
}
