package main

import (
	"context"
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
		Msg("🚀 Ödeme API Projesi başlatıldı")

	// Database bağlantısı
	database, err := db.Connect(cfg.GetDSN())
	if err != nil {
		log.Fatal().Err(err).Msg("❌ Veritabanı bağlantısı başarısız")
	}
	defer func() {
		log.Info().Msg("🗄️  Database bağlantısı kapatılıyor...")
		if err := database.Close(); err != nil {
			log.Error().Err(err).Msg("❌ Database kapatma hatası")
		} else {
			log.Info().Msg("✅ Database başarıyla kapatıldı")
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
	router := setupRouter(userHandler, balanceHandler, transactionHandler)

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
			Msg("🌐 HTTP Server (Gorilla Mux) başlatıldı")

		// Server'ı başlat
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverErr <- err
		}
	}()

	// Shutdown signal'ını veya server error'ını bekle
	select {
	case err := <-serverErr:
		log.Fatal().Err(err).Msg("❌ Server başlatma hatası")
	case sig := <-shutdown:
		log.Info().
			Str("signal", sig.String()).
			Msg("🛑 Shutdown signal alındı, graceful shutdown başlıyor...")

		// Graceful shutdown sequence başlat
		performGracefulShutdown(server, transactionQueue)
	}
}

// performGracefulShutdown graceful shutdown işlemlerini sırasıyla yapar
func performGracefulShutdown(server *http.Server, transactionQueue *services.TransactionQueue) {
	// Shutdown timeout context (maksimum 30 saniye bekle)
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	log.Info().Msg("📋 Graceful shutdown sırası:")
	log.Info().Msg("   1️⃣  HTTP Server'ı durdur (yeni request kabul etme)")
	log.Info().Msg("   2️⃣  Aktif HTTP request'leri bitir")
	log.Info().Msg("   3️⃣  Transaction Queue'yu durdur")
	log.Info().Msg("   4️⃣  Database bağlantılarını kapat")

	// 1. HTTP Server'ı graceful shutdown yap
	log.Info().Msg("📡 HTTP Server graceful shutdown başlatılıyor...")

	done := make(chan struct{})
	go func() {
		defer close(done)
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Error().Err(err).Msg("❌ HTTP Server graceful shutdown hatası")
		} else {
			log.Info().Msg("✅ HTTP Server graceful shutdown tamamlandı")
		}
	}()

	// Shutdown timeout kontrolü
	select {
	case <-done:
		// Shutdown başarılı
	case <-shutdownCtx.Done():
		log.Warn().Msg("⚠️  HTTP Server shutdown timeout! Zorla kapatılıyor...")
		// Force close context
		forceCtx, forceCancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer forceCancel()
		if err := server.Shutdown(forceCtx); err != nil {
			log.Error().Err(err).Msg("❌ HTTP Server force shutdown hatası")
		}
	}

	// 2. Transaction Queue'yu durdur
	log.Info().Msg("🔄 Transaction Queue graceful shutdown başlatılıyor...")
	queueDone := make(chan struct{})
	go func() {
		defer close(queueDone)
		transactionQueue.Stop()
		log.Info().Msg("✅ Transaction Queue graceful shutdown tamamlandı")
	}()

	// Queue shutdown timeout kontrolü (10 saniye)
	queueTimeout := time.NewTimer(10 * time.Second)
	select {
	case <-queueDone:
		queueTimeout.Stop()
	case <-queueTimeout.C:
		log.Warn().Msg("⚠️  Transaction Queue shutdown timeout!")
	}

	// 3. Final log
	log.Info().Msg("👋 Ödeme API graceful shutdown tamamlandı")
}

// setupRouter Gorilla Mux router'ını ayarlar
func setupRouter(userHandler *handlers.UserHandler, balanceHandler *handlers.BalanceHandler, transactionHandler *handlers.TransactionHandler) *mux.Router {
	router := mux.NewRouter()

	// CORS middleware
	router.Use(middleware.CORSMiddlewareWithDefaults())
	// Logger middleware
	router.Use(middleware.RequestLoggingMiddlewareWithDefaults())
	// Security headers middleware
	router.Use(middleware.SecurityHeadersMiddlewareWithDefaults())
	// Rate limit middleware
	router.Use(middleware.RateLimitMiddlewareWithDefaults())

	// Global OPTIONS handler - tüm route'lar için otomatik OPTIONS support
	router.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// CORS middleware zaten header'ları set etti
		// Sadece 204 No Content döndür
		w.WriteHeader(http.StatusNoContent)
	})
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

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

	// User endpoints
	users := protected.PathPrefix("/users").Subrouter()
	users.HandleFunc("", userHandler.GetAllUsers).Methods("GET")
	users.HandleFunc("/profile", userHandler.GetProfile).Methods("GET")
	users.HandleFunc("/{id:[0-9]+}", userHandler.GetUserByID).Methods("GET")
	users.HandleFunc("/{id:[0-9]+}", userHandler.UpdateUser).Methods("PUT")
	users.HandleFunc("/{id:[0-9]+}", userHandler.DeleteUser).Methods("DELETE")

	// Transaction endpoints
	transactions := protected.PathPrefix("/transactions").Subrouter()
	transactions.HandleFunc("/credit", transactionHandler.Credit).Methods("POST")
	transactions.HandleFunc("/debit", transactionHandler.Debit).Methods("POST")
	transactions.HandleFunc("/transfer", transactionHandler.Transfer).Methods("POST")
	transactions.HandleFunc("/history", transactionHandler.GetHistory).Methods("GET")
	transactions.HandleFunc("/{id:[0-9]+}", transactionHandler.GetTransactionByID).Methods("GET")

	// Balance endpoints
	balances := protected.PathPrefix("/balances").Subrouter()
	balances.HandleFunc("/current", balanceHandler.GetCurrentBalance).Methods("GET")
	balances.HandleFunc("/historical", balanceHandler.GetBalanceHistory).Methods("GET")
	balances.HandleFunc("/at-time", balanceHandler.GetBalanceAtTime).Methods("GET")

	// Route listesini log'la (development için)
	router.Walk(func(route *mux.Route, router *mux.Router, ancestors []*mux.Route) error {
		pathTemplate, err := route.GetPathTemplate()
		if err == nil {
			methods, _ := route.GetMethods()
			log.Debug().
				Str("path", pathTemplate).
				Strs("methods", methods).
				Msg("📍 Route registered")
		}
		return nil
	})

	return router
}
