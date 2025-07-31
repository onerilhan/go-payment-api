package main

import (
	"context"
	"encoding/json"
	stdlog "log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/onerilhan/go-payment-api/internal/config"
	"github.com/onerilhan/go-payment-api/internal/db"
	"github.com/onerilhan/go-payment-api/internal/handlers"
	"github.com/onerilhan/go-payment-api/internal/logger"
	"github.com/onerilhan/go-payment-api/internal/middleware"
	"github.com/onerilhan/go-payment-api/internal/middleware/errors"
	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/onerilhan/go-payment-api/internal/repository"
	"github.com/onerilhan/go-payment-api/internal/services"
)

func main() {
	// .env dosyasını yükle
	if err := godotenv.Load(); err != nil {
		stdlog.Println(".env dosyası bulunamadı, ortam değişkenlerinden okunacak.")
	}

	// config yükle
	cfg := config.LoadConfig()

	// logger başlat
	logger.Init(cfg.AppEnv)

	log.Info().
		Str("environment", cfg.AppEnv).
		Str("port", cfg.Port).
		Msg("Ödeme API Projesi başlatıldı")

	// Database bağlantısı
	database, err := db.Connect(cfg.GetDSN())
	if err != nil {
		log.Fatal().Err(err).Msg("Veritabanı bağlantısı başarısız")
	}
	defer func() {
		log.Info().Msg("Database bağlantısı kapatılıyor...")
		if err := database.Close(); err != nil {
			log.Error().Err(err).Msg("Database kapatma hatası")
		} else {
			log.Info().Msg("Database başarıyla kapatıldı")
		}
	}()

	// Repository, Service, Handler katmanları
	userRepo := repository.NewUserRepository(database)
	transactionRepo := repository.NewTransactionRepository(database)
	balanceRepo := repository.NewBalanceRepository(database)

	userService := services.NewUserService(userRepo)
	balanceService := services.NewBalanceService(balanceRepo)
	transactionService := services.NewTransactionService(transactionRepo, balanceService, database)

	// Transaction Queue oluştur (3 worker, 50 buffer)
	transactionQueue := services.NewTransactionQueue(3, transactionService, 50)
	transactionQueue.Start()

	userHandler := handlers.NewUserHandler(userService)
	balanceHandler := handlers.NewBalanceHandler(balanceService)
	transactionHandler := handlers.NewTransactionHandler(transactionService, transactionQueue, balanceService)

	// Gorilla Mux Router Setup
	router := setupRouter(userHandler, balanceHandler, transactionHandler, cfg.AppEnv, userService)

	// HTTP Server configuration
	serverAddr := ":" + cfg.Port
	server := &http.Server{
		Addr:         serverAddr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Graceful shutdown setup
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM, syscall.SIGINT)

	// Server'ı goroutine'de başlat
	serverErr := make(chan error, 1)
	go func() {
		log.Info().
			Str("port", cfg.Port).
			Str("addr", serverAddr).
			Int("read_timeout", 15).
			Int("write_timeout", 15).
			Int("idle_timeout", 60).
			Msg("HTTP Server (Gorilla Mux) başlatıldı")

		// Server'ı başlat
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Shutdown signal'ını veya server error'ını bekle
	select {
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("Server başlatma hatası")
	case sig := <-shutdown:
		log.Info().
			Str("signal", sig.String()).
			Msg("Shutdown signal alındı, graceful shutdown başlıyor...")

		// Graceful shutdown sequence başlat
		performGracefulShutdown(server, transactionQueue)
	}
}

// performGracefulShutdown graceful shutdown işlemlerini sırasıyla yapar
func performGracefulShutdown(server *http.Server, transactionQueue *services.TransactionQueue) {
	// Shutdown timeout context (maksimum 30 saniye bekle)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	log.Info().Msg("Graceful shutdown sırası:")
	log.Info().Msg("   1. HTTP Server'ı durdur (yeni request kabul etme)")
	log.Info().Msg("   2. Aktif HTTP request'leri bitir")
	log.Info().Msg("   3. Transaction Queue'yu durdur")
	log.Info().Msg("   4. Database bağlantılarını kapat")

	// 1. HTTP Server'ı graceful shutdown yap
	log.Info().Msg("HTTP Server graceful shutdown başlatılıyor...")

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("HTTP Server graceful shutdown hatası")
		} else {
			log.Info().Msg("HTTP Server graceful shutdown tamamlandı")
		}
	}()

	// Shutdown timeout kontrolü
	select {
	case <-done:
		// Shutdown başarılı
	case <-shutdownCtx.Done():
		log.Warn().Msg("HTTP Server shutdown timeout! Zorla kapatılıyor...")
		// Force close context
		forceCtx, forceCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer forceCancel()
		if err := server.Shutdown(forceCtx); err != nil {
			log.Error().Err(err).Msg("HTTP Server force shutdown hatası")
		}
	}

	// 2. Transaction Queue'yu durdur
	log.Info().Msg("Transaction Queue graceful shutdown başlatılıyor...")
	queueDone := make(chan struct{})
	go func() {
		defer close(queueDone)
		transactionQueue.Stop()
		log.Info().Msg("Transaction Queue graceful shutdown tamamlandı")
	}()

	// Queue shutdown timeout kontrolü (10 saniye)
	queueTimeout := time.NewTimer(10 * time.Second)
	select {
	case <-queueDone:
		queueTimeout.Stop()
	case <-queueTimeout.C:
		log.Warn().Msg("Transaction Queue shutdown timeout!")
	}

	// 3. Final log
	log.Info().Msg("Ödeme API graceful shutdown tamamlandı")
}

// setupRouter Gorilla Mux router'ını ayarlar
func setupRouter(userHandler *handlers.UserHandler, balanceHandler *handlers.BalanceHandler, transactionHandler *handlers.TransactionHandler, appEnv string, userService *services.UserService) *mux.Router {
	router := mux.NewRouter()

	// MIDDLEWARE CHAIN SIRASI (önemli!)
	// Request → Error → CORS → Logging → Security → RateLimit → Auth → Handler

	// 1. Error Handling Middleware (en dışta - panic recovery için)
	if appEnv == "development" {
		router.Use(middleware.ErrorHandlingMiddlewareForDevelopment())
	} else {
		router.Use(middleware.ErrorHandlingMiddlewareForProduction())
	}

	// 2. CORS middleware
	router.Use(middleware.CORSMiddlewareWithDefaults())

	// 3. Logger middleware
	router.Use(middleware.RequestLoggingMiddlewareWithDefaults())

	// 4. Security headers middleware
	router.Use(middleware.SecurityHeadersMiddlewareWithDefaults())

	// 5. Rate limit middleware
	router.Use(middleware.RateLimitMiddlewareWithDefaults())

	// Global OPTIONS handler
	router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	// Health check endpoint
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy","timestamp":"` + time.Now().Format(time.RFC3339) + `"}`))
	}).Methods("GET")

	// Development test endpoints
	if appEnv == "development" {
		router.HandleFunc("/panic", func(w http.ResponseWriter, r *http.Request) {
			panic("Test panic - Error handling middleware test")
		}).Methods("GET")

		router.HandleFunc("/error/400", func(w http.ResponseWriter, r *http.Request) {
			panic(&errors.ValidationError{
				Message:    "Bad Request Test - Invalid parameters",
				StatusCode: http.StatusBadRequest,
				Field:      "test_field",
				Value:      "invalid_value",
			})
		}).Methods("GET")

		router.HandleFunc("/error/401", func(w http.ResponseWriter, r *http.Request) {
			panic(&errors.AuthError{
				Message:    "Unauthorized Test - Token required",
				StatusCode: http.StatusUnauthorized,
			})
		}).Methods("GET")

		router.HandleFunc("/error/403", func(w http.ResponseWriter, r *http.Request) {
			panic(&errors.RBACError{
				Message:    "Forbidden Test - Access denied",
				StatusCode: http.StatusForbidden,
				Resource:   "test_resource",
				Action:     "test_action",
			})
		}).Methods("GET")

		router.HandleFunc("/error/500", func(w http.ResponseWriter, r *http.Request) {
			panic("Internal Server Error Test - Something went wrong")
		}).Methods("GET")

		// Development only: Create initial admin user
		router.HandleFunc("/dev/create-admin", func(w http.ResponseWriter, r *http.Request) {
			adminReq := &models.CreateUserRequest{
				Name:            "System Admin",
				Email:           "admin@system.com",
				Password:        "admin123456",
				ConfirmPassword: "admin123456",
				Role:            "admin",
			}

			if err := adminReq.Validate(); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			adminUser, err := userService.CreateAdminUser(adminReq)
			if err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "Admin user created successfully",
				"admin": map[string]interface{}{
					"id":    adminUser.ID,
					"name":  adminUser.Name,
					"email": adminUser.Email,
					"role":  adminUser.Role,
				},
			})
		}).Methods("POST")
	}

	// API v1 subrouter
	api := router.PathPrefix("/api/v1").Subrouter()

	// Public endpoints (Authentication)
	auth := api.PathPrefix("/auth").Subrouter()
	auth.HandleFunc("/register", userHandler.Register).Methods("POST")
	auth.HandleFunc("/login", userHandler.Login).Methods("POST")
	auth.HandleFunc("/refresh", userHandler.Refresh).Methods("POST")

	// Protected endpoints (Authentication required)
	protected := api.NewRoute().Subrouter()
	protected.Use(middleware.AuthMiddleware)

	// User endpoints with RBAC
	users := protected.PathPrefix("/users").Subrouter()
	users.Use(middleware.UserManagementRBAC())
	users.HandleFunc("", userHandler.GetAllUsers).Methods("GET")
	users.HandleFunc("/profile", userHandler.GetProfile).Methods("GET")
	users.HandleFunc("/{id:[0-9]+}", userHandler.GetUserByID).Methods("GET")
	users.HandleFunc("/{id:[0-9]+}", userHandler.UpdateUser).Methods("PUT")
	users.HandleFunc("/{id:[0-9]+}", userHandler.DeleteUser).Methods("DELETE")

	// Admin-only endpoints
	adminUsers := protected.PathPrefix("/admin/users").Subrouter()
	adminUsers.Use(middleware.RequireAdmin())
	adminUsers.HandleFunc("/{id:[0-9]+}/promote", userHandler.PromoteToMod).Methods("POST")
	adminUsers.HandleFunc("/{id:[0-9]+}/demote", userHandler.DemoteUser).Methods("POST")

	// Transaction endpoints with RBAC
	transactions := protected.PathPrefix("/transactions").Subrouter()
	transactions.Use(middleware.RequirePermission(middleware.PermMakeTransaction))
	transactions.HandleFunc("/credit", transactionHandler.Credit).Methods("POST")
	transactions.HandleFunc("/debit", transactionHandler.Debit).Methods("POST")
	transactions.HandleFunc("/transfer", transactionHandler.Transfer).Methods("POST")
	transactions.HandleFunc("/history", transactionHandler.GetHistory).Methods("GET")
	transactions.HandleFunc("/{id:[0-9]+}", transactionHandler.GetTransactionByID).Methods("GET")

	// Balance endpoints with RBAC
	balances := protected.PathPrefix("/balances").Subrouter()
	balances.Use(middleware.RequirePermission(middleware.PermViewOwnBalance))
	balances.HandleFunc("/current", balanceHandler.GetCurrentBalance).Methods("GET")
	balances.HandleFunc("/historical", balanceHandler.GetBalanceHistory).Methods("GET")
	balances.HandleFunc("/at-time", balanceHandler.GetBalanceAtTime).Methods("GET")

	// JSON NotFound ve MethodNotAllowed handlers
	router.NotFoundHandler = middleware.NotFoundJSONHandler()
	router.MethodNotAllowedHandler = middleware.MethodNotAllowedJSONHandler()

	// Route listesini log'la (development için)
	if appEnv == "development" {
		router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
			pathTemplate, err := route.GetPathTemplate()
			if err == nil {
				methods, _ := route.GetMethods()
				log.Debug().
					Str("path", pathTemplate).
					Strs("methods", methods).
					Msg("Route registered")
			}
			return nil
		})

		log.Info().Msg("Custom JSON handlers registered:")
		log.Info().Msg("  - 404 NotFound → JSON response")
		log.Info().Msg("  - 405 MethodNotAllowed → JSON response")
	}

	return router
}
