package services

import (
	"fmt"
	"sync"

	"github.com/onerilhan/go-payment-api/internal/models"
	"github.com/rs/zerolog/log"
)

// TransactionJob queue'da işlenecek transaction job'ı
type TransactionJob struct {
	FromUserID int
	Request    *models.TransferRequest
	ResultChan chan TransactionResult
}

// TransactionResult job sonucu
type TransactionResult struct {
	Transaction *models.Transaction
	Error       error
}

// TransactionQueue transaction işleme queue'su
type TransactionQueue struct {
	jobChan    chan TransactionJob
	workers    int
	bufferSize int
	wg         sync.WaitGroup
	service    *TransactionService
}

// NewTransactionQueue yeni queue oluşturur
func NewTransactionQueue(workers int, service *TransactionService, bufferSize int) *TransactionQueue {
	return &TransactionQueue{
		jobChan:    make(chan TransactionJob, bufferSize),
		workers:    workers,
		bufferSize: bufferSize,
		service:    service,
	}
}

// Start worker'ları başlatır
func (q *TransactionQueue) Start() {
	log.Info().
		Int("workers", q.workers).
		Int("buffer_size", q.bufferSize).
		Msg("🔄 Transaction queue başlatıldı")

	for i := 0; i < q.workers; i++ {
		q.wg.Add(1)
		go q.worker(i)
	}
}

// Stop queue'yu durdurur
func (q *TransactionQueue) Stop() {
	close(q.jobChan)
	q.wg.Wait()
	log.Info().Msg("⏹️ Transaction queue durduruldu")
}

// worker tek bir worker'ın işlem yapması
func (q *TransactionQueue) worker(id int) {
	defer q.wg.Done()

	// Panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Error().
				Interface("recover", r).
				Int("worker_id", id).
				Msg("🚨 Worker panikledi ama toparlandı")
		}
	}()

	log.Info().Int("worker_id", id).Msg("🚀 Worker başlatıldı")

	for job := range q.jobChan {
		log.Debug().
			Int("worker_id", id).
			Int("from_user", job.FromUserID).
			Int("to_user", job.Request.ToUserID).
			Float64("amount", job.Request.Amount).
			Msg("💼 Transaction işleniyor")

		// Transaction'ı işle
		transaction, err := q.service.Transfer(job.FromUserID, job.Request)

		// Sonucu gönder ve channel'ı kapat
		job.ResultChan <- TransactionResult{
			Transaction: transaction,
			Error:       err,
		}
		close(job.ResultChan) // FIX: Channel'ı kapat

		if err != nil {
			log.Error().Err(err).Int("worker_id", id).Msg("❌ Transaction başarısız")
		} else {
			log.Info().Int("worker_id", id).Int("transaction_id", transaction.ID).Msg("✅ Transaction başarılı")
		}
	}

	log.Info().Int("worker_id", id).Msg("🛑 Worker durduruldu")
}

// AddJob queue'ya yeni job ekler
func (q *TransactionQueue) AddJob(fromUserID int, req *models.TransferRequest) <-chan TransactionResult {
	resultChan := make(chan TransactionResult, 1)

	job := TransactionJob{
		FromUserID: fromUserID,
		Request:    req,
		ResultChan: resultChan,
	}

	select {
	case q.jobChan <- job:
		log.Debug().Int("from_user", fromUserID).Msg("📤 Job queue'ya eklendi")
	default:
		// Queue dolu - channel'ı kapat
		go func() {
			resultChan <- TransactionResult{
				Transaction: nil,
				Error:       fmt.Errorf("transaction queue dolu, daha sonra tekrar deneyin"),
			}
			close(resultChan) // FIX: Channel'ı kapat
		}()
	}

	return resultChan
}
