package http

import (
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/reangeline/backend_applywise/internal/adapters/inbound/http/handler"
	"github.com/reangeline/backend_applywise/internal/adapters/inbound/http/middleware"
	"github.com/reangeline/backend_applywise/internal/core/ports/inbound"
)

func NewRouter(
	authService inbound.AuthService,
	userService inbound.UserService,
	subscriptionService inbound.SubscriptionService,
	resumeService inbound.ResumeOptimizerService,
	paymentService inbound.PaymentService,
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

	// Middleware para normalizar paths (remove barras duplas)
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "//") {
				r.URL.Path = strings.ReplaceAll(r.URL.Path, "//", "/")
			}
			next.ServeHTTP(w, r)
		})
	})

	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"https://*", "http://*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	authHandler := handler.NewAuthHandler(authService, userService)
	resumeHandler := handler.NewResumeHandler(resumeService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	webhookHandler := handler.NewWebhookHandler(paymentService)

	// Health check na raiz
	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"applywise-api"}`))
	})

	// Rotas API v1
	r.Route("/api/v1", func(r chi.Router) {
		// Health check
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","version":"v1"}`))
		})

		// Públicas
		r.Post("/auth/signup", authHandler.SignUp)
		r.Post("/auth/signin", authHandler.SignIn)
		r.Post("/auth/refresh", authHandler.RefreshToken)
		r.Post("/webhooks/stripe", webhookHandler.HandleStripeWebhook)

		// Protegidas
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authService, userService))

			r.Post("/resumes", resumeHandler.UploadResume)
			r.Get("/resumes", resumeHandler.ListResumes)
			r.Get("/resumes/{resumeID}", resumeHandler.GetResume)
			r.Post("/resumes/optimize", resumeHandler.OptimizeResume)
			r.Get("/resumes/optimized", resumeHandler.ListOptimizedResumes)

			r.Get("/subscription", subscriptionHandler.GetSubscription)
			r.Post("/subscription", subscriptionHandler.CreateSubscription)
			r.Post("/subscription/checkout", subscriptionHandler.CreateCheckout) // ← ADICIONAR
			r.Delete("/subscription", subscriptionHandler.CancelSubscription)
		})
	})

	return r
}
