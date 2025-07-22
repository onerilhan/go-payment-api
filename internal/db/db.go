package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq" // PostgreSQL driver
	"github.com/rs/zerolog/log"
)

// Connect veritabanına bağlantı açar
func Connect(dsn string) (*sql.DB, error) {
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, fmt.Errorf("veritabanı açılırken hata: %w", err)
	}

	// Bağlantıyı test et
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("veritabanına ping atılamadı: %w", err)
	}

	log.Info().Msg("✅ PostgreSQL veritabanına başarıyla bağlandı")
	return db, nil
}
