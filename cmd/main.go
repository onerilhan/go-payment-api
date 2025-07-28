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
	defer database.Close()

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
	go func() {
		log.Info().
			Str("port", cfg.Port).
			Str("addr", serverAddr).
			Int("read_timeout", 15).
			Int("write_timeout", 15).
			Int("idle_timeout", 60).
			Msg("🌐 HTTP Server (Gorilla Mux) başlatıldı")

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("❌ Server başlatma hatası")
		}
	}()

	// Shutdown signal'ını bekle
	<-shutdown
	log.Info().Msg("🛑 Shutdown signal alındı, server kapatılıyor...")

	// Graceful shutdown sequence
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	// 1. HTTP Server'ı kapat (aktif bağlantıları bekle)
	log.Info().Msg("📡 HTTP Server kapatılıyor...")
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("❌ HTTP Server kapatma hatası")
	} else {
		log.Info().Msg("✅ HTTP Server başarıyla kapatıldı")
	}

	// 2. Transaction Queue'yu kapat
	log.Info().Msg("🔄 Transaction Queue kapatılıyor...")
	transactionQueue.Stop()
	log.Info().Msg("✅ Transaction Queue başarıyla kapatıldı")

	// 3. Database bağlantısını kapat (defer ile zaten kapatılacak)
	log.Info().Msg("🗄️  Database bağlantısı kapatılıyor...")

	log.Info().Msg("👋 Ödeme API başarıyla kapatıldı")
}

// setupRouter Gorilla Mux router'ını ayarlar
func setupRouter(userHandler *handlers.UserHandler, balanceHandler *handlers.BalanceHandler, transactionHandler *handlers.TransactionHandler) *mux.Router {
	router := mux.NewRouter()

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
