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

	issuerURL := cfg.KeycloakURL + "/realms/" + cfg.KeycloakRealm
	authMW, err := middleware.OIDCAuth(issuerURL, cfg.KeycloakClientID, cfg.OIDCSkipIssuerCheck)
	if err != nil {
		return fmt.Errorf("main.run: oidc: %w", err)
	}

	router := setupRouter(pool, authMW)

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
	report       *handler.ReportHandler
	sync         *handler.SyncHandler

	userSvc        *service.UserService
	auditSvc       *service.AuditService
	agreementRepo  *repository.VolunteerAgreementRepository
	personRoleRepo *repository.PersonRoleRepository
}

// buildDeps wires repositories → services → handlers.
func buildDeps(pool *pgxpool.Pool) appDeps {
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
	reportRepo := repository.NewReportRepository(pool)

	auditSvc := service.NewAuditService(auditRepo)
	userSvc := service.NewUserService(userRepo, auditSvc)
	personSvc := service.NewPersonService(personRepo, personRoleRepo, agreementRepo, auditSvc)
	selfRegisterSvc := service.NewSelfRegisterService(personRepo, personRoleRepo, userRepo, agreementRepo, auditSvc)
	agreementSvc := service.NewVolunteerAgreementService(agreementRepo, personRoleRepo, auditSvc)
	onboardingSvc := service.NewOnboardingService(userRepo, personRepo, personRoleRepo, agreementRepo, auditSvc)

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
		report:         handler.NewReportHandler(service.NewReportService(reportRepo)),
		sync:           handler.NewSyncHandler(service.NewSyncService(personRepo, triageRepo, attendanceRepo, auditSvc)),
		userSvc:        userSvc,
		auditSvc:       auditSvc,
		agreementRepo:  agreementRepo,
		personRoleRepo: personRoleRepo,
	}
}

func setupRouter(pool *pgxpool.Pool, authMW func(http.Handler) http.Handler) *chi.Mux {
	d := buildDeps(pool)

	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS("http://localhost:5173"))
	r.Get("/health", d.health.ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", d.health.ServeHTTP)
		d.registerAuthOnlyRoutes(r, authMW)
		d.registerAgreementRoutes(r, authMW)
		d.registerProtectedRoutes(r, authMW)
	})

	return r
}

// registerAuthOnlyRoutes mounts routes needing auth but no provision/RBAC.
func (d appDeps) registerAuthOnlyRoutes(r chi.Router, authMW func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(authMW)
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

// registerProtectedRoutes mounts the fully-guarded application routes.
func (d appDeps) registerProtectedRoutes(r chi.Router, authMW func(http.Handler) http.Handler) {
	r.Group(func(r chi.Router) {
		r.Use(authMW)
		r.Use(middleware.AutoProvision(d.userSvc))
		r.Use(middleware.RequireAgreement(d.agreementRepo, d.personRoleRepo))

		allRoles := middleware.RequireRole(d.auditSvc, "VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")
		r.With(allRoles).Get("/service-types", d.serviceType.List)

		d.registerCampusRoutes(r)
		d.registerPersonRoutes(r, allRoles)
		d.registerTriageRoutes(r, allRoles)
		d.registerAttendanceRoutes(r, allRoles)
		d.registerCampaignRoutes(r, allRoles)
		d.registerReportRoutes(r)
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

func (d appDeps) registerReportRoutes(r chi.Router) {
	reportRoles := middleware.RequireRole(d.auditSvc, "COORDINATOR", "ADMIN")
	r.Route("/reports", func(r chi.Router) {
		r.With(reportRoles).Get("/attendances", d.report.AttendanceSummary)
		r.With(reportRoles).Get("/attendances/export", d.report.AttendanceExport)
		r.With(reportRoles).Get("/campaigns/{id}", d.report.CampaignMetrics)
	})
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
