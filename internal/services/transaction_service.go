package services

import (
	"database/sql"
	"fmt"

	"github.com/onerilhan/go-payment-api/internal/db"
	"github.com/onerilhan/go-payment-api/internal/interfaces"
	"github.com/onerilhan/go-payment-api/internal/models"
)

// TransactionService transaction business logic'i
type TransactionService struct {
	transactionRepo interfaces.TransactionRepositoryInterface
	balanceService  interfaces.BalanceServiceInterface // DİKKAT: ARTIK BU DA ARAYÜZ
	database        *sql.DB
}

// NewTransactionService, arayüzleri kabul eder ve *pointer döner
// Bu, hem 'lock' uyarısını engeller hem de main.go'daki hatayı çözer.
func NewTransactionService(transactionRepo interfaces.TransactionRepositoryInterface,
	balanceService interfaces.BalanceServiceInterface, // Bu da arayüz
	database *sql.DB) *TransactionService {
	return &TransactionService{
		transactionRepo: transactionRepo,
		balanceService:  balanceService,
		database:        database,
	}
}

// ValidateTransactionType transaction type'ını doğrular
func (s *TransactionService) ValidateTransactionType(txType string) error {
	validTypes := map[string]bool{
		"credit":   true,
		"debit":    true,
		"transfer": true,
	}

	if !validTypes[txType] {
		return fmt.Errorf("geçersiz transaction tipi: %s. Geçerli tipler: credit, debit, transfer", txType)
	}

	return nil
}

// ValidateAmount para miktarını doğrular
func (s *TransactionService) ValidateAmount(amount float64) error {
	if amount <= 0 {
		return fmt.Errorf("miktar sıfırdan büyük olmalıdır")
	}

	if amount > 1000000 { // maksimum limit
		return fmt.Errorf("maksimum transfer limiti: 1,000,000 TL")
	}

	return nil
}

// Transfer kullanıcılar arası para transferi yapar - STATE MANAGEMENT EKLENDİ
func (s *TransactionService) Transfer(fromUserID int, req *models.TransferRequest) (*models.Transaction, error) {
	//  Request validation
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Aynı kullanıcıya transfer kontrolü
	if fromUserID == req.ToUserID {
		return nil, fmt.Errorf("kendinize para gönderemezsiniz")
	}

	//  Factory method ile transaction oluştur
	transaction := models.NewTransferTransaction(fromUserID, req.ToUserID, req.Amount, req.Description)

	//  Transaction validation
	if err := transaction.Validate(); err != nil {
		return nil, fmt.Errorf("transaction validation hatası: %w", err)
	}

	var result *models.Transaction

	// Database transaction ile rollback mechanism
	err := db.WithTransaction(s.database, func(tx *sql.Tx) error {
		txRepo := db.NewTransactionRepository(tx)

		// 1. Gönderen kullanıcının bakiyesini kontrol et ve lock et
		var fromBalance float64
		err := txRepo.QueryRow(`
			SELECT amount FROM balances WHERE user_id = $1 FOR UPDATE
		`, fromUserID).Scan(&fromBalance)

		if err == sql.ErrNoRows {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("gönderen kullanıcının bakiyesi bulunamadı")
		}
		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("gönderen bakiye sorgusu hatası: %w", err)
		}

		// 2. Yeterli bakiye kontrolü
		if fromBalance < req.Amount {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("yetersiz bakiye. Mevcut bakiye: %.2f TL", fromBalance)
		}

		// 3. Alan kullanıcının bakiyesini al ve lock et
		var toBalance float64
		err = txRepo.QueryRow(`
			SELECT amount FROM balances WHERE user_id = $1 FOR UPDATE
		`, req.ToUserID).Scan(&toBalance)

		if err == sql.ErrNoRows {
			// Alan kullanıcının bakiyesi yoksa oluştur
			_, err = txRepo.Exec(`
				INSERT INTO balances (user_id, amount) VALUES ($1, 0.00)
			`, req.ToUserID)
			if err != nil {
				transaction.SetStatus(models.StatusFailed)
				return fmt.Errorf("alan kullanıcı bakiyesi oluşturulamadı: %w", err)
			}
			toBalance = 0.00
		} else if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("alan kullanıcı bakiye sorgusu hatası: %w", err)
		}

		// 4. Transaction kaydını oluştur (PENDING status ile)
		var transactionID int
		var createdAt sql.NullTime
		err = txRepo.QueryRow(`
			INSERT INTO transactions (from_user_id, to_user_id, amount, type, status, description) 
			VALUES ($1, $2, $3, $4, $5, $6)
			RETURNING id, created_at
		`, fromUserID, req.ToUserID, req.Amount, transaction.Type, transaction.Status, req.Description).Scan(&transactionID, &createdAt)

		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("transaction kaydı oluşturulamadı: %w", err)
		}

		// 5. Bakiyeleri güncelle
		newFromBalance := fromBalance - req.Amount
		newToBalance := toBalance + req.Amount

		// Gönderen bakiyesini güncelle
		_, err = txRepo.Exec(`
			UPDATE balances SET amount = $1 WHERE user_id = $2
		`, newFromBalance, fromUserID)
		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("gönderen bakiye güncellenemedi: %w", err)
		}

		// Alan bakiyesini güncelle
		_, err = txRepo.Exec(`
			UPDATE balances SET amount = $1 WHERE user_id = $2
		`, newToBalance, req.ToUserID)
		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("alan bakiye güncellenemedi: %w", err)
		}

		//  Transaction'ı completed olarak işaretle
		if err := transaction.SetStatus(models.StatusCompleted); err != nil {
			return fmt.Errorf("transaction status güncellenemedi: %w", err)
		}

		// Status'u database'de güncelle
		_, err = txRepo.Exec(`
			UPDATE transactions SET status = $1 WHERE id = $2
		`, transaction.Status, transactionID)
		if err != nil {
			return fmt.Errorf("transaction status database'de güncellenemedi: %w", err)
		}

		// 6. Result struct'ını oluştur
		transaction.ID = transactionID
		transaction.CreatedAt = createdAt.Time
		result = transaction

		return nil // SUCCESS - transaction commit edilecek
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetUserTransactions kullanıcının transaction geçmişini getirir
func (s *TransactionService) GetUserTransactions(userID int, limit, offset int) ([]*models.Transaction, error) {
	transactions, err := s.transactionRepo.GetByUserID(userID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("transaction geçmişi alınamadı: %w", err)
	}

	return transactions, nil
}

// Credit kullanıcının hesabına para yatırır - STATE MANAGEMENT EKLENDİ
func (s *TransactionService) Credit(userID int, req *models.CreditRequest) (*models.Transaction, error) {
	//  Request validation
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Default description
	description := req.Description
	if description == "" {
		description = "Hesaba para yatırma"
	}

	//  Factory method ile transaction oluştur
	transaction := models.NewCreditTransaction(userID, req.Amount, description)

	//  Transaction validation
	if err := transaction.Validate(); err != nil {
		return nil, fmt.Errorf("transaction validation hatası: %w", err)
	}

	var result *models.Transaction

	// Database transaction ile rollback mechanism
	err := db.WithTransaction(s.database, func(tx *sql.Tx) error {
		txRepo := db.NewTransactionRepository(tx)

		// 1. Kullanıcının mevcut bakiyesini al ve lock et
		var currentBalance float64
		err := txRepo.QueryRow(`
			SELECT amount FROM balances WHERE user_id = $1 FOR UPDATE
		`, userID).Scan(&currentBalance)

		if err == sql.ErrNoRows {
			// Bakiye yoksa oluştur
			_, err = txRepo.Exec(`
				INSERT INTO balances (user_id, amount) VALUES ($1, 0.00)
			`, userID)
			if err != nil {
				//  Transaction status güncelle
				transaction.SetStatus(models.StatusFailed)
				return fmt.Errorf("bakiye oluşturulamadı: %w", err)
			}
			currentBalance = 0.00
		} else if err != nil {
			//  Transaction status güncelle
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("bakiye sorgusu hatası: %w", err)
		}

		// 2. Transaction kaydını oluştur (PENDING status ile)
		var transactionID int
		var createdAt sql.NullTime
		err = txRepo.QueryRow(`
			INSERT INTO transactions (to_user_id, from_user_id, amount, type, status, description) 
			VALUES ($1, NULL, $2, $3, $4, $5)
			RETURNING id, created_at
		`, userID, req.Amount, transaction.Type, transaction.Status, description).Scan(&transactionID, &createdAt)

		if err != nil {
			//  Transaction status güncelle
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("transaction kaydı oluşturulamadı: %w", err)
		}

		// 3. Bakiyeyi artır
		newBalance := currentBalance + req.Amount
		_, err = txRepo.Exec(`
			UPDATE balances SET amount = $1 WHERE user_id = $2
		`, newBalance, userID)
		if err != nil {
			//  Transaction status güncelle
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("bakiye güncellenemedi: %w", err)
		}

		//  Transaction'ı completed olarak işaretle
		if err := transaction.SetStatus(models.StatusCompleted); err != nil {
			return fmt.Errorf("transaction status güncellenemedi: %w", err)
		}

		// Status'u database'de güncelle
		_, err = txRepo.Exec(`
			UPDATE transactions SET status = $1 WHERE id = $2
		`, transaction.Status, transactionID)
		if err != nil {
			return fmt.Errorf("transaction status database'de güncellenemedi: %w", err)
		}

		// 4. Result struct'ını oluştur
		transaction.ID = transactionID
		transaction.CreatedAt = createdAt.Time
		result = transaction

		return nil // SUCCESS - transaction commit edilecek
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// Debit kullanıcının hesabından para çeker - STATE MANAGEMENT EKLENDİ
func (s *TransactionService) Debit(userID int, req *models.DebitRequest) (*models.Transaction, error) {
	// Request validation
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// Default description
	description := req.Description
	if description == "" {
		description = "Hesaptan para çekme"
	}

	//  Factory method ile transaction oluştur
	transaction := models.NewDebitTransaction(userID, req.Amount, description)

	// Transaction validation
	if err := transaction.Validate(); err != nil {
		return nil, fmt.Errorf("transaction validation hatası: %w", err)
	}

	var result *models.Transaction

	// Database transaction ile rollback mechanism
	err := db.WithTransaction(s.database, func(tx *sql.Tx) error {
		txRepo := db.NewTransactionRepository(tx)

		// 1. Kullanıcının mevcut bakiyesini al ve lock et
		var currentBalance float64
		err := txRepo.QueryRow(`
			SELECT amount FROM balances WHERE user_id = $1 FOR UPDATE
		`, userID).Scan(&currentBalance)

		if err == sql.ErrNoRows {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("kullanıcının bakiyesi bulunamadı")
		}
		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("bakiye sorgusu hatası: %w", err)
		}

		// 2. Yeterli bakiye kontrolü
		if currentBalance < req.Amount {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("yetersiz bakiye. Mevcut bakiye: %.2f TL", currentBalance)
		}

		// 3. Transaction kaydını oluştur (PENDING status ile)
		var transactionID int
		var createdAt sql.NullTime
		err = txRepo.QueryRow(`
			INSERT INTO transactions (from_user_id, to_user_id, amount, type, status, description) 
			VALUES ($1, NULL, $2, $3, $4, $5)
			RETURNING id, created_at
		`, userID, req.Amount, transaction.Type, transaction.Status, description).Scan(&transactionID, &createdAt)

		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("transaction kaydı oluşturulamadı: %w", err)
		}

		// 4. Bakiyeyi azalt
		newBalance := currentBalance - req.Amount
		_, err = txRepo.Exec(`
			UPDATE balances SET amount = $1 WHERE user_id = $2
		`, newBalance, userID)
		if err != nil {
			transaction.SetStatus(models.StatusFailed)
			return fmt.Errorf("bakiye güncellenemedi: %w", err)
		}

		//  Transaction'ı completed olarak işaretle
		if err := transaction.SetStatus(models.StatusCompleted); err != nil {
			return fmt.Errorf("transaction status güncellenemedi: %w", err)
		}

		// Status'u database'de güncelle
		_, err = txRepo.Exec(`
			UPDATE transactions SET status = $1 WHERE id = $2
		`, transaction.Status, transactionID)
		if err != nil {
			return fmt.Errorf("transaction status database'de güncellenemedi: %w", err)
		}

		// 5. Result struct'ını oluştur
		transaction.ID = transactionID
		transaction.CreatedAt = createdAt.Time
		result = transaction

		return nil // SUCCESS - transaction commit edilecek
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

// GetTransactionByID ID ile transaction getirir
func (s *TransactionService) GetTransactionByID(id int) (*models.Transaction, error) {
	// ID validation
	if id <= 0 {
		return nil, fmt.Errorf("geçersiz transaction ID")
	}

	// Repository'den transaction'ı al
	transaction, err := s.transactionRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("transaction bulunamadı: %w", err)
	}

	return transaction, nil
}
