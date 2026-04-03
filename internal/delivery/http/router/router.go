package router

import (
	"log/slog"
	"net/http"

	"github.com/assessly/assessly-be/internal/delivery/http/middleware"
	"github.com/go-chi/chi/v5"
)

// Router wraps chi router and provides route registration
type Router struct {
	*chi.Mux
}

// New creates a new router instance
func New() *Router {
	return &Router{
		Mux: chi.NewRouter(),
	}
}

// SetupRoutes configures all application routes
func (r *Router) SetupRoutes(
	healthHandler http.HandlerFunc,
	metricsHandler http.Handler,
	authHandler interface {
		Register(w http.ResponseWriter, r *http.Request)
		Login(w http.ResponseWriter, r *http.Request)
		RequestPasswordReset(w http.ResponseWriter, r *http.Request)
		ResetPassword(w http.ResponseWriter, r *http.Request)
	},
	testHandler interface {
		ListTests(w http.ResponseWriter, r *http.Request)
		GetTest(w http.ResponseWriter, r *http.Request)
		CreateTest(w http.ResponseWriter, r *http.Request)
		PublishTest(w http.ResponseWriter, r *http.Request)
	},
	questionHandler interface {
		AddQuestion(w http.ResponseWriter, r *http.Request)
	},
	submissionHandler interface {
		GenerateAccessToken(w http.ResponseWriter, r *http.Request)
		GenerateAccessTokenForTest(w http.ResponseWriter, r *http.Request)
		SubmitTest(w http.ResponseWriter, r *http.Request)
		GetSubmission(w http.ResponseWriter, r *http.Request)
	},
	reviewHandler interface {
		HandleAddManualReview(w http.ResponseWriter, r *http.Request)
		HandleGetReview(w http.ResponseWriter, r *http.Request)
		HandleListSubmissions(w http.ResponseWriter, r *http.Request)
	},
	jwtMiddleware func(next http.Handler) http.Handler,
	logger *slog.Logger,
) {
	// Health check endpoint (no auth required)
	r.Get("/health", healthHandler)

	// T115: Prometheus metrics endpoint (no auth required)
	if metricsHandler != nil {
		r.Handle("/metrics", metricsHandler)
	} else {
		r.Get("/metrics", notImplementedHandler)
	}

	// API v1 routes
	r.Route("/api/v1", func(r chi.Router) {
		// Public routes (no auth required)
		r.Group(func(r chi.Router) {
			// Auth endpoints
			if authHandler != nil {
				r.Post("/auth/register", authHandler.Register)
				r.Post("/auth/login", authHandler.Login)
				r.Post("/auth/request-reset", authHandler.RequestPasswordReset)
				r.Put("/auth/reset-password", authHandler.ResetPassword)
			} else {
				r.Post("/auth/register", notImplementedHandler)
				r.Post("/auth/login", notImplementedHandler)
				r.Post("/auth/request-reset", notImplementedHandler)
				r.Put("/auth/reset-password", notImplementedHandler)
			}

			// Participant submission endpoints
			if submissionHandler != nil {
				r.Post("/submissions/access", submissionHandler.GenerateAccessToken)
				r.Post("/submissions", submissionHandler.SubmitTest)
				r.Get("/submissions/{id}", submissionHandler.GetSubmission)
			} else {
				r.Post("/submissions/access", notImplementedHandler)
				r.Post("/submissions", notImplementedHandler)
				r.Get("/submissions/{id}", notImplementedHandler)
			}
		})

		// Protected routes (auth required)
		r.Group(func(r chi.Router) {
			// Apply JWT middleware
			if jwtMiddleware != nil {
				r.Use(jwtMiddleware)
			}

			// Test creator endpoints (requires creator role)
			r.Route("/tests", func(r chi.Router) {
			// Routes accessible by both creators and reviewers (auth required)
			// GET /{testID} - View test details (with role-based access control in use case)
			if testHandler != nil {
				r.Get("/{testID}", testHandler.GetTest)
			} else {
				r.Get("/{testID}", notImplementedHandler)
			}

			// Creator-only routes
			r.Group(func(r chi.Router) {
				r.Use(middleware.RequireRole("creator", logger))

				if testHandler != nil {
					r.Get("/", testHandler.ListTests)
					r.Post("/", testHandler.CreateTest)
					r.Post("/{testID}/publish", testHandler.PublishTest)
				} else {
					r.Get("/", notImplementedHandler)
					r.Post("/", notImplementedHandler)
					r.Post("/{testID}/publish", notImplementedHandler)
				}

					// Questions
					if questionHandler != nil {
						r.Post("/{testID}/questions", questionHandler.AddQuestion)
					} else {
						r.Post("/{testID}/questions", notImplementedHandler)
					}
					r.Put("/{testID}/questions/{questionID}", notImplementedHandler)
					r.Delete("/{testID}/questions/{questionID}", notImplementedHandler)

					// Testing endpoint: Generate access token for test (development only)
					if submissionHandler != nil {
						r.Post("/{testID}/access-token", submissionHandler.GenerateAccessTokenForTest)
					}
				})

				// Reviewer-only routes
				r.Group(func(r chi.Router) {
					r.Use(middleware.RequireRole("reviewer", logger))

					// T089: Submissions list for reviewers
					if reviewHandler != nil {
						r.Get("/{testId}/submissions", reviewHandler.HandleListSubmissions)
					} else {
						r.Get("/{testId}/submissions", notImplementedHandler)
					}
				})
			})

			// T089: Review endpoints (requires reviewer role)
			r.Route("/reviews", func(r chi.Router) {
				r.Use(middleware.RequireRole("reviewer", logger))

				if reviewHandler != nil {
					r.Put("/{answerId}", reviewHandler.HandleAddManualReview)
					r.Get("/{answerId}", reviewHandler.HandleGetReview)
				} else {
					r.Put("/{answerId}", notImplementedHandler)
					r.Get("/{answerId}", notImplementedHandler)
				}
			})

			// User endpoints
			r.Get("/me", notImplementedHandler)
		})
	})
}

// notImplementedHandler returns 501 Not Implemented
func notImplementedHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNotImplemented)
	if _, err := w.Write([]byte(`{"error":"endpoint not implemented yet"}`)); err != nil {
		// Error writing response, connection may be closed
	}
}
