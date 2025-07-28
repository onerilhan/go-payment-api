package main

import (
	stdlog "log"
	"net/http"

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
	balanceService := services.NewBalanceService(balanceRepo) // ← YENİ
	transactionService := services.NewTransactionService(transactionRepo, balanceService, database)

	// Transaction Queue oluştur (3 worker, 50 buffer)
	transactionQueue := services.NewTransactionQueue(3, transactionService, 50) // ← YENİ
	transactionQueue.Start()                                                    // ← YENİ: Queue'yu başlat

	// Graceful shutdown için cleanup
	defer transactionQueue.Stop() // ← YENİ: Program kapanırken queue'yu durdur

	userHandler := handlers.NewUserHandler(userService)
	balanceHandler := handlers.NewBalanceHandler(balanceService)
	transactionHandler := handlers.NewTransactionHandler(transactionService, transactionQueue, balanceService) // ← GÜNCEL

	// HTTP routes
	http.HandleFunc("/api/v1/auth/register", userHandler.Register)
	http.HandleFunc("/api/v1/auth/login", userHandler.Login)
	http.HandleFunc("/api/v1/auth/refresh", userHandler.Refresh)
	http.HandleFunc("/api/v1/users/profile", middleware.AuthMiddleware(userHandler.GetProfile))
	http.HandleFunc("/api/v1/users/", middleware.AuthMiddleware(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			userHandler.GetUserByID(w, r)
		case http.MethodPut:
			userHandler.UpdateUser(w, r)
		case http.MethodDelete:
			userHandler.DeleteUser(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}))
	http.HandleFunc("/api/v1/users", middleware.AuthMiddleware(userHandler.GetAllUsers))
	http.HandleFunc("/api/v1/transactions/transfer", middleware.AuthMiddleware(transactionHandler.Transfer))
	http.HandleFunc("/api/v1/transactions/credit", middleware.AuthMiddleware(transactionHandler.Credit))
	http.HandleFunc("/api/v1/transactions/debit", middleware.AuthMiddleware(transactionHandler.Debit))
	http.HandleFunc("/api/v1/transactions/", middleware.AuthMiddleware(transactionHandler.GetTransactionByID))
	http.HandleFunc("/api/v1/transactions/history", middleware.AuthMiddleware(transactionHandler.GetHistory))
	http.HandleFunc("/api/v1/balances/current", middleware.AuthMiddleware(balanceHandler.GetCurrentBalance))
	http.HandleFunc("/api/v1/balances/historical", middleware.AuthMiddleware(balanceHandler.GetBalanceHistory))
	http.HandleFunc("/api/v1/balances/at-time", middleware.AuthMiddleware(balanceHandler.GetBalanceAtTime))

	// Server'ı başlat
	serverAddr := ":" + cfg.Port
	log.Info().Str("port", cfg.Port).Msg("🌐 HTTP Server başlatıldı")

	if err := http.ListenAndServe(serverAddr, nil); err != nil {
		log.Fatal().Err(err).Msg("❌ Server başlatılamadı")
	}
}
