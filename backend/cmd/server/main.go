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
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/instituto-nova-sos/chesed/internal/config"
	"github.com/instituto-nova-sos/chesed/internal/database"
	"github.com/instituto-nova-sos/chesed/internal/handler"
	"github.com/instituto-nova-sos/chesed/internal/middleware"
	"github.com/instituto-nova-sos/chesed/internal/repository"
	"github.com/instituto-nova-sos/chesed/internal/service"
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

func setupRouter(
	pool *pgxpool.Pool,
	authMW func(http.Handler) http.Handler,
) *chi.Mux {
	// Repositories
	auditRepo := repository.NewAuditRepository(pool)
	userRepo := repository.NewUserRepository(pool)
	serviceTypeRepo := repository.NewServiceTypeRepository(pool)
	personRepo := repository.NewPersonRepository(pool)
	personRoleRepo := repository.NewPersonRoleRepository(pool)
	agreementRepo := repository.NewVolunteerAgreementRepository(pool)
	campusRepo := repository.NewCampusRepository(pool)

	// Services
	auditSvc := service.NewAuditService(auditRepo)
	userSvc := service.NewUserService(userRepo, auditSvc)
	serviceTypeSvc := service.NewServiceTypeService(serviceTypeRepo)
	personSvc := service.NewPersonService(personRepo, personRoleRepo, agreementRepo, auditSvc)
	selfRegisterSvc := service.NewSelfRegisterService(personRepo, personRoleRepo, userRepo, agreementRepo, auditSvc)
	agreementSvc := service.NewVolunteerAgreementService(agreementRepo, personRoleRepo, auditSvc)
	onboardingSvc := service.NewOnboardingService(userRepo, personRepo, personRoleRepo, agreementRepo, auditSvc)
	campusSvc := service.NewCampusService(campusRepo, auditSvc)

	// Handlers
	uploadDir := "uploads/agreements"
	healthH := handler.NewHealthHandler(pool)
	serviceTypeH := handler.NewServiceTypeHandler(serviceTypeSvc)
	personH := handler.NewPersonHandler(personSvc)
	selfRegisterH := handler.NewSelfRegisterHandler(selfRegisterSvc)
	agreementH := handler.NewVolunteerAgreementHandler(agreementSvc, uploadDir)
	onboardingH := handler.NewOnboardingHandler(onboardingSvc)
	campusH := handler.NewCampusHandler(campusSvc)

	// Router
	r := chi.NewRouter()
	r.Use(middleware.SecurityHeaders)
	r.Use(middleware.CORS("http://localhost:5173"))

	r.Get("/health", healthH.ServeHTTP)

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", healthH.ServeHTTP)

		// Auth-only: onboarding status, self-register, campus list (no AutoProvision, no RBAC)
		r.Group(func(r chi.Router) {
			r.Use(authMW)
			r.Get("/auth/me", onboardingH.GetStatus)
			r.Post("/self-register", selfRegisterH.Register)
			r.Get("/campuses", campusH.ListActive)
		})

		// Agreement routes: require auth + provision, but NOT agreement guard
		r.Group(func(r chi.Router) {
			r.Use(authMW)
			r.Use(middleware.AutoProvision(userSvc))
			r.Get("/volunteer-agreement/text", agreementH.GetText)
			r.Post("/volunteer-agreement/accept", agreementH.Accept)
			r.Post("/volunteer-agreement/reject", agreementH.Reject)
		})

		// Protected routes: require auth + provision + agreement
		r.Group(func(r chi.Router) {
			r.Use(authMW)
			r.Use(middleware.AutoProvision(userSvc))
			r.Use(middleware.RequireAgreement(agreementRepo, personRoleRepo))

			allRoles := middleware.RequireRole("VOLUNTEER", "SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")

			r.With(allRoles).Get("/service-types", serviceTypeH.List)

				// Campus management (ADMIN only)
				r.Route("/campuses", func(r chi.Router) {
					r.With(middleware.RequireRole("ADMIN")).Get("/all", campusH.ListAll)
					r.With(middleware.RequireRole("ADMIN")).Get("/{id}", campusH.Get)
					r.With(middleware.RequireRole("ADMIN")).Post("/", campusH.Create)
					r.With(middleware.RequireRole("ADMIN")).Put("/{id}", campusH.Update)
				})

			r.Route("/persons", func(r chi.Router) {
				r.With(allRoles).Post("/", personH.Create)
				r.With(allRoles).Get("/", personH.List)
				r.With(allRoles).Get("/check-duplicate", personH.CheckDuplicate)
				r.With(allRoles).Get("/{id}", personH.Get)
				r.With(middleware.RequireRole("SECRETARY", "PROFESSIONAL", "COORDINATOR", "ADMIN")).
					Put("/{id}", personH.Update)
				r.With(middleware.RequireRole("PROFESSIONAL", "COORDINATOR", "ADMIN")).
					Get("/{id}/history", personH.GetHistory)
				r.With(middleware.RequireRole("COORDINATOR", "ADMIN")).
					Post("/{id}/roles", personH.AddRole)
				r.With(middleware.RequireRole("COORDINATOR", "ADMIN")).
					Patch("/{id}/roles/{roleId}", personH.ToggleRole)

				// Agreement management for specific persons (COORDINATOR+)
				r.With(middleware.RequireRole("COORDINATOR", "ADMIN")).
					Get("/{id}/agreement", agreementH.GetPersonAgreement)
				r.With(middleware.RequireRole("COORDINATOR", "ADMIN")).
					Post("/{id}/agreement/upload", agreementH.Upload)
				r.With(middleware.RequireRole("COORDINATOR", "ADMIN")).
					Get("/{id}/agreement/document", agreementH.DownloadDocument)
			})
		})
	})

	return r
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
