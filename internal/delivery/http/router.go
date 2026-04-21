package http

import (
	"net/http"
	"strings"

	appappt "amaur/api/internal/application/appointment"
	appauth "amaur/api/internal/application/auth"
	appcssession "amaur/api/internal/application/caresession"
	appcompany "amaur/api/internal/application/company"
	appcontract "amaur/api/internal/application/contract"
	appatient "amaur/api/internal/application/patient"
	appprogram "amaur/api/internal/application/program"
	appservicetype "amaur/api/internal/application/servicetype"
	appuser "amaur/api/internal/application/user"
	appvisit "amaur/api/internal/application/visit"
	appworker "amaur/api/internal/application/worker"
	"amaur/api/internal/config"
	"amaur/api/internal/delivery/http/handlers"
	"amaur/api/internal/delivery/http/middleware"
	"amaur/api/internal/infrastructure/postgres"
	jwtpkg "amaur/api/pkg/jwt"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"gorm.io/gorm"
)

func New(db *gorm.DB, cfg *config.Config, log zerolog.Logger) http.Handler {
	r := chi.NewRouter()

	// ── Global middleware ──────────────────────────────────────────────────
	r.Use(chimiddleware.RequestID)
	r.Use(chimiddleware.RealIP)
	r.Use(chimiddleware.Recoverer)
	r.Use(chimiddleware.CleanPath)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   strings.Split(cfg.CORSAllowedOrigins, ","),
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Request-ID"},
		ExposedHeaders:   []string{"X-Request-ID"},
		AllowCredentials: true,
		MaxAge:           300,
	}))

	// ── Dependencies ───────────────────────────────────────────────────────
	jwt := jwtpkg.NewManager(cfg.JWTSecret, cfg.JWTAccessExpiry, cfg.JWTRefreshExpiry)

	userRepo := postgres.NewUserRepository(db)
	patientRepo := postgres.NewPatientRepository(db)
	companyRepo := postgres.NewCompanyRepository(db)
	workerRepo := postgres.NewWorkerRepository(db)
	visitRepo := postgres.NewVisitRepository(db)
	contractRepo := postgres.NewContractRepository(db)
	programRepo := postgres.NewProgramRepository(db)
	careSessionRepo := postgres.NewCareSessionRepository(db)
	serviceTypeRepo := postgres.NewServiceTypeRepository(db)
	appointmentRepo := postgres.NewAppointmentRepository(db)

	authSvc := appauth.NewService(userRepo, jwt)
	patientSvc := appatient.NewService(patientRepo, userRepo, companyRepo)
	userSvc := appuser.NewService(userRepo, workerRepo)
	companySvc := appcompany.NewService(companyRepo, userSvc)
	workerSvc := appworker.NewService(workerRepo, userRepo, userSvc).WithAppointmentRepo(appointmentRepo).WithProgramRepo(programRepo)
	visitSvc := appvisit.NewService(visitRepo)
	contractSvc := appcontract.NewService(contractRepo)
	programSvc := appprogram.NewService(programRepo, contractRepo, careSessionRepo)
	careSessionSvc := appcssession.NewService(careSessionRepo)
	serviceTypeSvc := appservicetype.NewService(serviceTypeRepo)
	appointmentSvc := appappt.NewService(appointmentRepo).WithProgramRepo(programRepo).WithWorkerRepo(workerRepo)

	authH := handlers.NewAuthHandler(authSvc)
	patientH := handlers.NewPatientHandler(patientSvc)
	userH := handlers.NewUserHandler(userSvc)
	companyH := handlers.NewCompanyHandler(companySvc)
	workerH := handlers.NewWorkerHandler(workerSvc)
	visitH := handlers.NewVisitHandler(visitSvc)
	contractH := handlers.NewContractHandler(contractSvc)
	programH := handlers.NewProgramHandler(programSvc)
	careSessionH := handlers.NewCareSessionHandler(careSessionSvc)
	serviceTypeH := handlers.NewServiceTypeHandler(serviceTypeSvc)
	appointmentH := handlers.NewAppointmentHandler(appointmentSvc)

	// ── Health ─────────────────────────────────────────────────────────────
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"status":"ok"}`))
	})

	// ── Public routes ──────────────────────────────────────────────────────
	r.Route("/api/v1", func(r chi.Router) {
		r.Post("/auth/login", authH.Login)
		r.Post("/auth/refresh", authH.Refresh)

		// ── Protected routes ──────────────────────────────────────────────
		r.Group(func(r chi.Router) {
			r.Use(middleware.Authenticate(jwt))

			r.Post("/auth/logout", authH.Logout)
			r.Get("/auth/me", authH.Me)

			// Patients
			r.Route("/patients", func(r chi.Router) {
				r.With(middleware.RequirePermission("patients:view")).Get("/", patientH.List)
				r.With(middleware.RequirePermission("patients:create")).Post("/", patientH.Create)

				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("patients:view")).Get("/", patientH.GetByID)
					r.With(middleware.RequirePermission("patients:edit")).Put("/", patientH.Update)
					r.With(middleware.RequirePermission("patients:delete")).Delete("/", patientH.Delete)

					// Company associations
					r.With(middleware.RequirePermission("patients:view")).Get("/companies", patientH.GetCompanies)

					// Tutor/ward relationship
					// GET /patients/{id}/wards → list minor patients tutored by this patient
					r.With(middleware.RequirePermission("patients:view")).Get("/wards", patientH.GetWards)

					// Clinical record
					r.With(middleware.RequirePermission("clinical_records:view")).Get("/clinical-record", patientH.GetClinicalRecord)
					r.With(middleware.RequirePermission("clinical_records:edit")).Put("/clinical-record", patientH.UpdateClinicalRecord)

					// Portal login management
					r.With(middleware.RequirePermission("patients:edit")).Get("/login", patientH.GetLoginInfo)
					r.With(middleware.RequirePermission("patients:edit")).Post("/login", patientH.EnableLogin)
					r.With(middleware.RequirePermission("patients:edit")).Delete("/login", patientH.DisableLogin)
				})
			})

			// Companies
			r.Route("/companies", func(r chi.Router) {
				r.With(middleware.RequirePermission("companies:view")).Get("/", companyH.List)
				r.With(middleware.RequirePermission("companies:create")).Post("/", companyH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("companies:view")).Get("/", companyH.GetByID)
					r.With(middleware.RequirePermission("companies:edit")).Patch("/", companyH.Update)
					r.With(middleware.RequirePermission("companies:delete")).Delete("/", companyH.Delete)
					r.With(middleware.RequirePermission("companies:view")).Get("/branches", companyH.ListBranches)
					r.With(middleware.RequirePermission("companies:view")).Get("/patients", companyH.ListPatients)
				})
			})

			// Workers
			r.Route("/workers", func(r chi.Router) {
				r.With(middleware.RequirePermission("workers:view")).Get("/", workerH.List)
				r.With(middleware.RequirePermission("workers:create")).Post("/", workerH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("workers:view")).Get("/", workerH.GetByID)
					r.With(middleware.RequirePermission("workers:edit")).Patch("/", workerH.Update)
					r.With(middleware.RequirePermission("workers:delete")).Delete("/", workerH.Delete)
					r.With(middleware.RequirePermission("workers:edit")).Put("/availability", workerH.SetAvailabilityRules)
					r.With(middleware.RequirePermission("workers:view")).Get("/availability", workerH.GetAvailabilityRules)
					r.With(middleware.RequirePermission("workers:view")).Get("/slots", workerH.GetWorkerSlots)
					r.With(middleware.RequirePermission("workers:view")).Get("/calendar", workerH.GetWorkerCalendar)
					r.With(middleware.RequirePermission("workers:edit")).Put("/specialties", workerH.SetWorkerSpecialties)
				})
			})

			// Specialty catalog
			r.With(middleware.RequirePermission("workers:view")).Get("/specialties", workerH.ListSpecialties)
			r.With(middleware.RequirePermission("workers:create")).Post("/specialties", workerH.CreateSpecialty)
			r.With(middleware.RequirePermission("workers:delete")).Delete("/specialties/{code}", workerH.DeleteSpecialty)

			// Users
			r.Route("/users", func(r chi.Router) {
				r.With(middleware.RequirePermission("users:view")).Get("/", userH.List)
				r.With(middleware.RequirePermission("users:create")).Post("/", userH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("users:view")).Get("/", userH.GetByID)
					r.With(middleware.RequirePermission("users:edit")).Patch("/", userH.Update)
					r.With(middleware.RequirePermission("users:delete")).Delete("/", userH.Delete)
					r.With(middleware.RequirePermission("users:edit")).Put("/password", userH.ChangePassword)
					r.With(middleware.RequirePermission("users:edit")).Put("/roles", userH.AssignRoles)
				})
			})

			// Roles (read-only list for UI)
			r.With(middleware.RequirePermission("roles:view")).Get("/roles", userH.ListRoles)

			// Visits
			r.Route("/visits", func(r chi.Router) {
				r.With(middleware.RequirePermission("visits:view")).Get("/", visitH.List)
				r.With(middleware.RequirePermission("visits:create")).Post("/", visitH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("visits:view")).Get("/", visitH.GetByID)
					r.With(middleware.RequirePermission("visits:edit")).Patch("/", visitH.Update)
					r.With(middleware.RequirePermission("visits:delete")).Delete("/", visitH.Delete)
					r.With(middleware.RequirePermission("visits:view")).Get("/group-sessions", careSessionH.ListGroupSessions)
					r.With(middleware.RequirePermission("care_sessions:create")).Post("/group-sessions", careSessionH.CreateGroupSession)
				})
			})

			// Agendas (alias for visits)
			r.Route("/agendas", func(r chi.Router) {
				r.With(middleware.RequirePermission("visits:view")).Get("/", visitH.List)
				r.With(middleware.RequirePermission("visits:create")).Post("/", visitH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("visits:view")).Get("/", visitH.GetByID)
					r.With(middleware.RequirePermission("visits:edit")).Patch("/", visitH.Update)
					r.With(middleware.RequirePermission("visits:delete")).Delete("/", visitH.Delete)
					r.With(middleware.RequirePermission("visits:view")).Get("/group-sessions", careSessionH.ListGroupSessions)
					r.With(middleware.RequirePermission("care_sessions:create")).Post("/group-sessions", careSessionH.CreateGroupSession)
				})
			})

			// Care sessions (individual)
			r.Route("/care-sessions", func(r chi.Router) {
				r.With(middleware.RequirePermission("care_sessions:view")).Get("/", careSessionH.List)
				r.With(middleware.RequirePermission("care_sessions:create")).Post("/", careSessionH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("care_sessions:view")).Get("/", careSessionH.GetByID)
					r.With(middleware.RequirePermission("care_sessions:edit")).Patch("/", careSessionH.Update)
					r.With(middleware.RequirePermission("care_sessions:delete")).Delete("/", careSessionH.Delete)
				})
			})

			// Service types
			r.Route("/service-types", func(r chi.Router) {
				r.With(middleware.RequirePermission("care_sessions:view")).Get("/", serviceTypeH.List)
				r.With(middleware.RequirePermission("care_sessions:create")).Post("/", serviceTypeH.Create)
				r.With(middleware.RequirePermission("care_sessions:edit")).Patch("/{id}", serviceTypeH.Update)
			})

			// Contracts
			r.Route("/contracts", func(r chi.Router) {
				r.With(middleware.RequirePermission("contracts:view")).Get("/", contractH.List)
				r.With(middleware.RequirePermission("contracts:create")).Post("/", contractH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("contracts:view")).Get("/", contractH.GetByID)
					r.With(middleware.RequirePermission("contracts:edit")).Patch("/", contractH.Update)
					r.With(middleware.RequirePermission("contracts:delete")).Delete("/", contractH.Delete)
					r.With(middleware.RequirePermission("contracts:view")).Get("/services", contractH.ListServices)
				})
			})

			// Company programs (agenda recurrente)
			r.Route("/programs", func(r chi.Router) {
				r.With(middleware.RequirePermission("contracts:view")).Get("/", programH.List)
				r.With(middleware.RequirePermission("contracts:create")).Post("/", programH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("contracts:view")).Get("/", programH.GetByID)
					r.With(middleware.RequirePermission("contracts:edit")).Patch("/", programH.Update)
					r.With(middleware.RequirePermission("contracts:view")).Get("/agendas", programH.ListProgramAgendas)
					r.With(middleware.RequirePermission("contracts:edit")).Post("/generate-agendas", programH.GenerateAgendas)
				})
			})

			// Agenda services and participants for execution tracking
			r.With(middleware.RequirePermission("visits:view")).Get("/agendas/{agendaId}/services", programH.ListAgendaServices)
			r.With(middleware.RequirePermission("visits:edit")).Post("/agendas/{agendaId}/services", programH.CreateAgendaService)
			r.With(middleware.RequirePermission("visits:view")).Get("/agenda-services/{agendaServiceId}/participants", programH.ListParticipants)
			r.With(middleware.RequirePermission("visits:edit")).Post("/agenda-services/{agendaServiceId}/participants", programH.UpsertParticipants)
			r.With(middleware.RequirePermission("visits:edit")).Post("/agenda-services/{agendaServiceId}/complete", programH.CompleteAgendaService)

			// Individual appointments
			r.Route("/appointments", func(r chi.Router) {
				r.With(middleware.RequirePermission("appointments:view")).Get("/", appointmentH.List)
				r.With(middleware.RequirePermission("appointments:create")).Post("/", appointmentH.Create)
				r.Route("/{id}", func(r chi.Router) {
					r.With(middleware.RequirePermission("appointments:view")).Get("/", appointmentH.GetByID)
					r.With(middleware.RequirePermission("appointments:edit")).Patch("/", appointmentH.Update)
					r.With(middleware.RequirePermission("appointments:delete")).Delete("/", appointmentH.Delete)
				})
			})
		})
	})

	return r
}
