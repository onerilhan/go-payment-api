// internal/migration/runner.go
package migration

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/rs/zerolog/log"
)

// Runner migration işlemlerini yöneten ana yapı
type Runner struct {
	db     *sql.DB          // Database bağlantısı
	config *MigrationConfig // Migration ayarları
}

// NewRunner yeni migration runner oluşturur
func NewRunner(db *sql.DB, config *MigrationConfig) *Runner {
	if config == nil {
		config = DefaultConfig()
	}

	// Path kontrolü ve oluşturma
	if config.AutoCreatePath {
		if err := ensurePathExists(config.MigrationsPath); err != nil {
			log.Warn().
				Err(err).
				Str("path", config.MigrationsPath).
				Msg("Migration path oluşturulamadı, mevcut path kullanılacak")
		}
	}

	return &Runner{
		db:     db,
		config: config,
	}
}

// ensurePathExists klasör yoksa oluşturur
func ensurePathExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		if err := os.MkdirAll(path, 0755); err != nil {
			return fmt.Errorf("migration klasörü oluşturulamadı: %w", err)
		}
		log.Info().Str("path", path).Msg("Migration klasörü oluşturuldu")
	}
	return nil
}

// Initialize migration tracking tablosunu oluşturur
func (r *Runner) Initialize() error {
	createTableSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version BIGINT PRIMARY KEY,                    -- Migration version (timestamp)
			name VARCHAR(255) NOT NULL,                    -- Migration adı
			up_checksum VARCHAR(64) NOT NULL,              -- UP dosyası checksum
			down_checksum VARCHAR(64),                     -- DOWN dosyası checksum (nullable)
			applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP, -- Uygulandığı zaman
			execution_time_ms INTEGER DEFAULT 0,           -- Execution süresi (millisecond)
			file_size BIGINT DEFAULT 0,                    -- Dosya boyutu
			created_by VARCHAR(100) DEFAULT 'system'       -- Kim tarafından oluşturuldu
		)
	`, r.config.TableName)

	if _, err := r.db.Exec(createTableSQL); err != nil {
		return fmt.Errorf("migration tracking tablosu oluşturulamadı: %w", err)
	}

	indexSQL := fmt.Sprintf(`
		CREATE INDEX IF NOT EXISTS idx_%s_applied_at 
		ON %s (applied_at DESC)
	`, r.config.TableName, r.config.TableName)

	if _, err := r.db.Exec(indexSQL); err != nil {
		log.Warn().Err(err).Msg("Migration index oluşturulamadı")
	}

	log.Info().
		Str("table", r.config.TableName).
		Str("path", r.config.MigrationsPath).
		Msg("Migration sistemi initialize edildi")

	return nil
}

// Close runner'ı kapatır (DB bağlantısını kapatmaz)
func (r *Runner) Close() error {
	log.Debug().Msg("Migration runner kapatıldı")
	return nil
}
