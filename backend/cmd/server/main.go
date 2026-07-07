package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/instituto-nova-sos/chesed/internal/config"
	"github.com/instituto-nova-sos/chesed/internal/database"
	"github.com/instituto-nova-sos/chesed/internal/handler"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/instituto-nova-sos/chesed/internal/service"
	"github.com/instituto-nova-sos/chesed/internal/storage"
	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	if err := run(); err != nil {
		slog.Error("server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("main.run: %w", err)
	}

	logger := setupLogger(cfg.LogLevel)
	slog.SetDefault(logger)

	ctx := context.Background()
	pool, err := database.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		return fmt.Errorf("main.run: %w", err)
	}
	defer pool.Close()

	// Owner (RLS-bypassing) pool for pre-campus routes that must operate outside
	// any single campus (self-register, onboarding cross-campus email lookup).
	adminPool, err := database.NewPool(ctx, cfg.AdminDatabaseURL)
	if err != nil {
		return fmt.Errorf("main.run: admin pool: %w", err)
	}
	defer adminPool.Close()

	// Warn if the app pool connects as a role that bypasses RLS (owner or
	// superuser) — that silently disables campus isolation at the DB layer.
	warnIfRLSBypassed(ctx, pool)

	issuerURL := cfg.KeycloakURL + "/realms/" + cfg.KeycloakRealm
	authMW, err := middleware.OIDCAuth(issuerURL, cfg.KeycloakClientID, cfg.OIDCSkipIssuerCheck)
	if err != nil {
		return fmt.Errorf("main.run: oidc: %w", err)
	}

	store, err := storage.NewMinioStorage(ctx, storage.Config{
		Endpoint:  cfg.S3Endpoint,
		Bucket:    cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		UseSSL:    cfg.S3UseSSL,
	})
	if err != nil {
		return fmt.Errorf("main.run: storage: %w", err)
	}

	router := setupRouter(cfg, pool, adminPool, authMW, store)

	srv := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return startServer(srv, logger)
}

// appDeps bundles the constructed handlers and the repositories the router
// middleware needs, so route registration can be split into focused helpers.
type appDeps struct {
	health       *handler.HealthHandler
	serviceType  *handler.ServiceTypeHandler
	person       *handler.PersonHandler
	selfRegister *handler.SelfRegisterHandler
	agreement    *handler.VolunteerAgreementHandler
	onboarding   *handler.OnboardingHandler
	campus       *handler.CampusHandler
	triage       *handler.TriageHandler
	attendance   *handler.AttendanceHandler
	campaign     *handler.CampaignHandler
	consent      *handler.ConsentHandler
	donation     *handler.DonationHandler
	document     *handler.DocumentHandler
	report       *handler.ReportHandler
	compliance   *handler.ComplianceReportHandler
	audit        *handler.AuditHandler
	admin        *handler.AdminHandler
	sync         *handler.SyncHandler
	public       *handler.PublicHandler

	userSvc        *service.UserService
	auditSvc       *service.AuditService
	agreementRepo  *repository.VolunteerAgreementRepository
	personRoleRepo *repository.PersonRoleRepository
	campusRepo     *repository.CampusRepository
}

// buildDeps wires repositories → services → handlers.
func buildDeps(pool *pgxpool.Pool, store service.ObjectStorage) appDeps {
	auditRepo := repository.NewAuditRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	serviceTypeRepo := repository.NewServiceTypeRepository(pool)
	personRepo := repository.NewPersonRepository(pool)
	personRoleRepo := repository.NewPersonRoleRepository(pool)
	agreementRepo := repository.NewVolunteerAgreementRepository(pool)
	campusRepo := repository.NewCampusRepository(pool)
	triageRepo := repository.NewTriageRepository(pool)
	attendanceRepo := repository.NewAttendanceRepository(pool)
	campaignRepo := repository.NewCampaignRepository(pool)
	donationRepo := repository.NewDonationRepository(pool)
	consentRepo := repository.NewConsentRepository(pool)
	documentRepo := repository.NewDocumentRepository(pool)
	reportRepo := repository.NewReportRepository(pool)
	complianceRepo := repository.NewComplianceReportRepository(pool)
	retentionRepo := repository.NewRetentionRepository(pool)

	auditSvc := service.NewAuditService(auditRepo)
	userSvc := service.NewUserService(userRepo, auditSvc)
	personSvc := service.NewPersonService(personRepo, personRoleRepo, agreementRepo, auditSvc)
	selfRegisterSvc := service.NewSelfRegisterService(personRepo, personRoleRepo, userRepo, agreementRepo, auditSvc)
	agreementSvc := service.NewVolunteerAgreementService(agreementRepo, personRoleRepo, auditSvc)
	onboardingSvc := service.NewOnboardingService(userRepo, personRepo, personRoleRepo, agreementRepo, campusRepo, auditSvc)

	uploadDir := "uploads/agreements"
	return appDeps{
		health:         handler.NewHealthHandler(pool),
		serviceType:    handler.NewServiceTypeHandler(service.NewServiceTypeService(serviceTypeRepo)),
		person:         handler.NewPersonHandler(personSvc),
		selfRegister:   handler.NewSelfRegisterHandler(selfRegisterSvc),
		agreement:      handler.NewVolunteerAgreementHandler(agreementSvc, uploadDir),
		onboarding:     handler.NewOnboardingHandler(onboardingSvc),
		campus:         handler.NewCampusHandler(service.NewCampusService(campusRepo, auditSvc)),
		triage:         handler.NewTriageHandler(service.NewTriageService(triageRepo, campaignRepo, auditSvc)),
		attendance:     handler.NewAttendanceHandler(service.NewAttendanceService(attendanceRepo, campaignRepo, auditSvc)),
		campaign:       handler.NewCampaignHandler(service.NewCampaignService(campaignRepo, personRepo, auditSvc)),
		consent:        handler.NewConsentHandler(service.NewConsentService(consentRepo, personRepo, personRepo, auditSvc)),
		donation:       handler.NewDonationHandler(service.NewDonationService(donationRepo, personRepo, campaignRepo, store, service.NewReceiptRenderer(), auditSvc)),
		document:       handler.NewDocumentHandler(service.NewDocumentService(documentRepo, personRepo, attendanceRepo, store, auditSvc)),
		report:         handler.NewReportHandler(service.NewReportService(reportRepo)),
		compliance:     handler.NewComplianceReportHandler(service.NewComplianceReportService(complianceRepo, auditSvc)),
		audit:          handler.NewAuditHandler(service.NewAuditReadService(auditRepo)),
		admin:          handler.NewAdminHandler(service.NewRetentionService(retentionRepo, personRepo, auditSvc)),
		sync:           handler.NewSyncHandler(service.NewSyncService(personRepo, triageRepo, attendanceRepo, campaignRepo, auditSvc)),
		public:         handler.NewPublicHandler(service.NewPublicService(campaignRepo, personRepo, personRoleRepo, agreementRepo, auditSvc)),
		userSvc:        userSvc,
		auditSvc:       auditSvc,
		agreementRepo:  agreementRepo,
		personRoleRepo: personRoleRepo,
		campusRepo:     campusRepo,
	}
}

// warnIfRLSBypassed logs a warning when the app's connection role bypasses RLS
// (superuser, BYPASSRLS, or table owner). Campus isolation at the DB layer
// silently disappears in that case, so a misconfiguration that reconnects the
// app as the owner role is detectable from the logs at startup.
func warnIfRLSBypassed(ctx context.Context, pool *pgxpool.Pool) {
	const q = `
		SELECT rolsuper OR rolbypassrls
		       OR pg_catalog.pg_has_role(current_user, (SELECT relowner FROM pg_class WHERE relname = 'person'), 'USAGE')
		FROM pg_roles WHERE rolname = current_user`
	var bypasses bool
	if err := pool.QueryRow(ctx, q).Scan(&bypasses); err != nil {
		slog.WarnContext(ctx, "main: could not verify RLS enforcement for the app role", "error", err.Error())
		return
	}
	if bypasses {
		slog.WarnContext(ctx, "main: app database role BYPASSES row-level security — campus isolation is NOT enforced at the DB layer; connect as a non-owner role (e.g. chesed_app)")
	} else {
		slog.InfoContext(ctx, "main: app database role is subject to row-level security (campus isolation enforced)")
	}
}

func setupRouter(cfg config.Config, pool, adminPool *pgxpool.Pool, authMW func(http.Handler) http.Handler, store service.ObjectStorage) *chi.Mux {
	d := buildDeps(pool, store)

	// The dev origin is always allowed; production/public origins (e.g. the
	// WordPress site) are added via PUBLIC_CORS_ORIGINS so the public API can be
	// consumed cross-origin without another hardcoded literal.
	corsOrigins := append([]string{"http://localhost:5173"}, cfg.PublicCORSOrigins...)

	r := chi.NewRouter()
	// HSTS is flag-gated (HTTPS-only; invalid over plain HTTP), so it is off by
	// default and enabled via HSTS_ENABLED in TLS-terminated deployments.
	r.Use(middleware.SecurityHeadersWith(cfg.HSTSEnabled))
	r.Use(middleware.CORS(corsOrigins...))
	r.Get("/health", d.health.ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", d.health.ServeHTTP)
		d.registerPublicRoutes(r, pool, cfg.PublicRateLimitRPM)
		d.registerAuthOnlyRoutes(r, authMW, adminPool)
		d.registerAgreementRoutes(r, authMW)
		d.registerProtectedRoutes(r, authMW, pool)
	})

	return r
}

// registerPublicRoutes mounts the unauthenticated, internet-facing public API
// (S12.1). It runs on the NON-OWNER pool inside PublicCampusTx so RLS enforces
// campus isolation from a validated, request-supplied campus_id — a fail-closed
// safety net for a surface that has no auth token. A per-IP rate limiter sheds
// abusive traffic before any DB work.
func (d appDeps) registerPublicRoutes(r chi.Router, pool *pgxpool.Pool, rateLimitRPM int) {
	r.Route("/public", func(r chi.Router) {
		r.Use(middleware.PublicRateLimit(rateLimitRPM))
		r.Use(middleware.PublicCampusValidator(d.campusRepo))
		r.Use(middleware.PublicCampusTx(pool))
		r.Get("/campaigns", d.public.ListCampaigns)
		r.Post("/volunteer-signup", d.public.VolunteerSignup)
	})
}

// registerAuthOnlyRoutes mounts routes needing auth but no provision/RBAC.
// These run BEFORE a campus is established (onboarding cross-campus lookup,
// self-registration), so they use BypassRLS to run on the owner connection —
// they cannot be subject to per-request RLS (there is no campus GUC yet).
func (d appDeps) registerAuthOnlyRoutes(r chi.Router, authMW func(http.Handler) http.Handler, adminPool *pgxpool.Pool) {
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Use(middleware.BypassRLS(adminPool))
		r.Get("/auth/me", d.onboarding.GetStatus)
		r.Post("/self-register", d.selfRegister.Register)
		r.Get("/campuses", d.campus.ListActive)
	})
}

// registerAgreementRoutes mounts routes needing auth + provision but no agreement guard.
func (d appDeps) registerAgreementRoutes(r chi.Router, authMW func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Use(middleware.AutoProvision(d.userSvc))
		r.Get("/volunteer-agreement/text", d.agreement.GetText)
		r.Post("/volunteer-agreement/accept", d.agreement.Accept)
		r.Post("/volunteer-agreement/reject", d.agreement.Reject)
	})
}

// registerProtectedRoutes mounts the fully-guarded application routes. Every
// route here runs inside the per-request campus transaction (CampusTx) so
// PostgreSQL RLS enforces campus isolation at the database layer.
func (d appDeps) registerProtectedRoutes(r chi.Router, authMW func(http.Handler) http.Handler, pool *pgxpool.Pool) {
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Use(middleware.AutoProvision(d.userSvc))
		r.Use(middleware.CampusTx(pool))
		r.Use(middleware.RequireAgreement(d.agreementRepo, d.personRoleRepo))

		allRoles := middleware.RequireRole(d.auditSvc, "VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
		r.With(allRoles).Get("/service-types", d.serviceType.List)

		d.registerCampusRoutes(r)
		d.registerPersonRoutes(r, allRoles)
		d.registerTriageRoutes(r, allRoles)
		d.registerAttendanceRoutes(r, allRoles)
		d.registerCampaignRoutes(r, allRoles)
		d.registerConsentRoutes(r, allRoles)
		d.registerDonationRoutes(r)
		d.registerDocumentRoutes(r)
		d.registerReportRoutes(r)
		d.registerAuditRoutes(r)
		d.registerAdminRoutes(r)
		d.registerSyncRoutes(r, allRoles)
	})
}

func (d appDeps) registerCampusRoutes(r chi.Router) {
	adminOnly := middleware.RequireRole(d.auditSvc, "ADMIN")
	r.Route("/campuses", func(r chi.Router) {
		r.With(adminOnly).Get("/all", d.campus.ListAll)
		r.With(adminOnly).Get("/{id}", d.campus.Get)
		r.With(adminOnly).Post("/", d.campus.Create)
		r.With(adminOnly).Put("/{id}", d.campus.Update)
	})
}

func (d appDeps) registerPersonRoutes(r chi.Router, allRoles func(http.Handler) http.Handler) {
	secretaryUp := middleware.RequireRole(d.auditSvc, "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
	professionalUp := middleware.RequireRole(d.auditSvc, "PROFESSIONAL", "COORDINATOR", "ADMIN")
	coordinatorUp := middleware.RequireRole(d.auditSvc, "COORDINATOR", "ADMIN")
	r.Route("/persons", func(r chi.Router) {
		r.With(allRoles).Post("/", d.person.Create)
		r.With(allRoles).Get("/", d.person.List)
		r.With(allRoles).Get("/check-duplicate", d.person.CheckDuplicate)
		r.With(allRoles).Get("/{id}", d.person.Get)
		r.With(secretaryUp).Put("/{id}", d.person.Update)
		r.With(professionalUp).Get("/{id}/history", d.person.GetHistory)
		r.With(coordinatorUp).Post("/{id}/roles", d.person.AddRole)
		r.With(coordinatorUp).Patch("/{id}/roles/{roleId}", d.person.ToggleRole)
		r.With(coordinatorUp).Get("/{id}/agreement", d.agreement.GetPersonAgreement)
		r.With(coordinatorUp).Post("/{id}/agreement/upload", d.agreement.Upload)
		r.With(coordinatorUp).Get("/{id}/agreement/document", d.agreement.DownloadDocument)
	})
}

func (d appDeps) registerTriageRoutes(r chi.Router, allRoles func(http.Handler) http.Handler) {
	triageRoles := middleware.RequireRole(d.auditSvc, "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
	r.Route("/triages", func(r chi.Router) {
		r.With(triageRoles).Post("/", d.triage.Create)
		r.With(allRoles).Get("/", d.triage.List)
		r.With(allRoles).Get("/{id}", d.triage.Get)
		r.With(triageRoles).Patch("/{id}", d.triage.Update)
	})
}

func (d appDeps) registerAttendanceRoutes(r chi.Router, allRoles func(http.Handler) http.Handler) {
	attendanceRoles := middleware.RequireRole(d.auditSvc, "PROFESSIONAL", "COORDINATOR", "ADMIN")
	r.Route("/attendances", func(r chi.Router) {
		r.With(attendanceRoles).Post("/", d.attendance.Create)
		r.With(allRoles).Get("/", d.attendance.List)
		r.With(allRoles).Get("/{id}", d.attendance.Get)
		r.With(attendanceRoles).Post("/{id}/transitions", d.attendance.Transition)
		r.With(attendanceRoles).Patch("/{id}/notes", d.attendance.UpdateNotes)
	})
}

func (d appDeps) registerCampaignRoutes(r chi.Router, allRoles func(http.Handler) http.Handler) {
	coordinatorUp := middleware.RequireRole(d.auditSvc, "COORDINATOR", "ADMIN")
	r.Route("/campaigns", func(r chi.Router) {
		r.With(coordinatorUp).Post("/", d.campaign.Create)
		r.With(allRoles).Get("/", d.campaign.List)
		r.With(allRoles).Get("/{id}", d.campaign.Get)
		r.With(coordinatorUp).Put("/{id}", d.campaign.Update)
		r.With(coordinatorUp).Post("/{id}/team", d.campaign.AddTeamMember)
		r.With(coordinatorUp).Delete("/{id}/team/{personId}", d.campaign.RemoveTeamMember)
	})
}

func (d appDeps) registerConsentRoutes(r chi.Router, allRoles func(http.Handler) http.Handler) {
	secretaryUp := middleware.RequireRole(d.auditSvc, "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
	adminOnly := middleware.RequireRole(d.auditSvc, "ADMIN")
	r.Route("/consents", func(r chi.Router) {
		r.With(allRoles).Post("/", d.consent.Create)
		r.With(adminOnly).Patch("/{id}/revoke", d.consent.Revoke)
	})
	// Sibling of the mounted /persons Route group: chi matches this explicit
	// param pattern before falling back to the /persons/* mount, so the two
	// registrations coexist without collision.
	r.With(secretaryUp).Get("/persons/{id}/consents", d.consent.ListByPerson)
}

// registerDonationRoutes mounts the donation endpoints per docs/11-api-design.md:
// writes (create/edit) are Secretary+; reads (list/detail) are Coordinator+.
func (d appDeps) registerDonationRoutes(r chi.Router) {
	secretaryUp := middleware.RequireRole(d.auditSvc, "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
	coordinatorUp := middleware.RequireRole(d.auditSvc, "COORDINATOR", "ADMIN")
	r.Route("/donations", func(r chi.Router) {
		r.With(secretaryUp).Post("/", d.donation.Create)
		r.With(coordinatorUp).Get("/", d.donation.List)
		r.With(coordinatorUp).Get("/{id}", d.donation.Get)
		r.With(secretaryUp).Put("/{id}", d.donation.Update)
		r.With(coordinatorUp).Get("/{id}/receipt", d.donation.Receipt)
	})
}

// registerDocumentRoutes mounts the document endpoints per docs/16: person
// uploads are Secretary+, attendance uploads and every read are Professional+.
func (d appDeps) registerDocumentRoutes(r chi.Router) {
	secretaryUp := middleware.RequireRole(d.auditSvc, "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
	professionalUp := middleware.RequireRole(d.auditSvc, "PROFESSIONAL", "COORDINATOR", "ADMIN")
	// Siblings of the mounted /persons and /attendances Route groups: chi
	// matches these explicit param patterns before the group mounts (same
	// approach as /persons/{id}/consents above).
	r.With(secretaryUp).Post("/persons/{id}/documents", d.document.UploadForPerson)
	r.With(professionalUp).Get("/persons/{id}/documents", d.document.ListByPerson)
	r.With(professionalUp).Post("/attendances/{id}/documents", d.document.UploadForAttendance)
	r.With(professionalUp).Get("/attendances/{id}/documents", d.document.ListByAttendance)
	r.With(professionalUp).Get("/documents/{id}/download", d.document.Download)
}

func (d appDeps) registerReportRoutes(r chi.Router) {
	reportRoles := middleware.RequireRole(d.auditSvc, "COORDINATOR", "ADMIN")
	r.Route("/reports", func(r chi.Router) {
		r.With(reportRoles).Get("/dashboard", d.report.Dashboard)
		r.With(reportRoles).Get("/attendances", d.report.AttendanceSummary)
		r.With(reportRoles).Get("/attendances/export", d.report.AttendanceExport)
		r.With(reportRoles).Get("/campaigns/{id}", d.report.CampaignMetrics)
		r.With(reportRoles).Get("/compliance", d.compliance.Report)
		r.With(reportRoles).Get("/compliance/export", d.compliance.Export)
	})
}

// registerAuditRoutes mounts the read-only audit log viewer (RF-53). ADMIN only;
// campus scoping is applied in SQL because audit_log is excluded from RLS.
func (d appDeps) registerAuditRoutes(r chi.Router) {
	adminOnly := middleware.RequireRole(d.auditSvc, "ADMIN")
	r.With(adminOnly).Get("/audit/logs", d.audit.List)
}

// registerAdminRoutes mounts ADMIN-only compliance operations (data retention).
func (d appDeps) registerAdminRoutes(r chi.Router) {
	adminOnly := middleware.RequireRole(d.auditSvc, "ADMIN")
	r.With(adminOnly).Post("/admin/retention/run", d.admin.RunRetention)
}

func (d appDeps) registerSyncRoutes(r chi.Router, allRoles func(http.Handler) http.Handler) {
	r.Route("/sync", func(r chi.Router) {
		r.With(allRoles).Post("/push", d.sync.Push)
		r.With(allRoles).Get("/pull", d.sync.Pull)
	})
}

func setupLogger(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	return slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
}

func startServer(srv *http.Server, logger *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "addr", srv.Addr)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- fmt.Errorf("main.startServer: %w", err)
		}
		close(errCh)
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-quit:
		logger.Info("shutdown signal received", "signal", sig.String())
	case err := <-errCh:
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	logger.Info("shutting down server")
	if err := srv.Shutdown(ctx); err != nil {
		return fmt.Errorf("main.startServer: shutdown: %w", err)
	}

	logger.Info("server stopped")
	return nil
}
