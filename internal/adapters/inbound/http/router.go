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
	revenueCatService inbound.RevenueCatService, // ✅ ADICIONAR
) *chi.Mux {
	r := chi.NewRouter()

	r.Use(chimiddleware.Logger)
	r.Use(chimiddleware.Recoverer)

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
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-Revenuecat-Signature"}, // ✅ ADICIONAR header
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	authHandler := handler.NewAuthHandler(authService, userService)
	resumeHandler := handler.NewResumeHandler(resumeService)
	subscriptionHandler := handler.NewSubscriptionHandler(subscriptionService)
	webhookHandler := handler.NewWebhookHandler(paymentService)
	revenueCatWebhookHandler := handler.NewRevenueCatWebhookHandler(revenueCatService) // ✅ ADICIONAR
	userHandler := handler.NewUserHandler(userService)

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok","service":"applywise-api"}`))
	})

	r.Route("/api/v1", func(r chi.Router) {
		r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"status":"ok","version":"v1"}`))
		})

		// Públicas
		r.Post("/auth/signup", authHandler.SignUp)
		r.Post("/auth/signin", authHandler.SignIn)
		r.Post("/auth/refresh", authHandler.RefreshToken)
		r.Post("/auth/confirm", authHandler.ConfirmSignUp)
		r.Post("/auth/resend-code", authHandler.ResendCode)
		r.Post("/auth/forgot-password", authHandler.ForgotPassword)
		r.Post("/auth/confirm-forgot-password", authHandler.ConfirmForgotPassword)
		r.Post("/webhooks/stripe", webhookHandler.HandleStripeWebhook)
		r.Post("/webhooks/revenuecat", revenueCatWebhookHandler.HandleWebhook) // ✅ ADICIONAR

		// Protegidas
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(authService, userService))

			r.Post("/resumes", resumeHandler.UploadResume)
			r.Post("/resumes/manual", resumeHandler.CreateManualResume)
			r.Get("/resumes", resumeHandler.ListResumes)
			r.Get("/resumes/{resumeID}", resumeHandler.GetResume)
			r.Put("/resumes/manual/{resumeID}", resumeHandler.UpdateManualResume)
			r.Post("/resumes/optimize", resumeHandler.OptimizeResume)
			r.Get("/resumes/optimized", resumeHandler.ListOptimizedResumes)
			r.Get("/resumes/optimized/{optimizedID}", resumeHandler.GetOptimizedResume)
			r.Put("/resumes/optimized/{optimizedID}", resumeHandler.UpdateOptimizedResume)
			r.Delete("/resumes/{resumeID}", resumeHandler.DeleteResume)

			r.Get("/resumes/optimize/jobs/{jobID}", resumeHandler.GetOptimizationJobStatus)

			// LinkedIn optimization
			r.Post("/resumes/linkedin/optimize", resumeHandler.OptimizeForLinkedIn)

			r.Get("/subscription", subscriptionHandler.GetSubscription)
			r.Post("/subscription", subscriptionHandler.CreateSubscription)
			r.Post("/subscription/checkout", subscriptionHandler.CreateCheckout)
			r.Delete("/subscription", subscriptionHandler.CancelSubscription)
			r.Get("/subscription/credits", subscriptionHandler.GetCredits)

			r.Get("/users/me", userHandler.GetMe)
			r.Post("/users/me/fcm-token", userHandler.UpdateFCMToken)
			r.Delete("/users/me", userHandler.DeleteMe)
		})
	})

	return r
}
