// internal/migration/runner_db.go
package migration

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// AppliedMigration database'den okunan applied migration bilgisi
type AppliedMigration struct {
	Version         int64     `db:"version"`
	Name            string    `db:"name"`
	UpChecksum      string    `db:"up_checksum"`
	DownChecksum    *string   `db:"down_checksum"` // Nullable
	AppliedAt       time.Time `db:"applied_at"`
	ExecutionTimeMs int64     `db:"execution_time_ms"`
	FileSize        int64     `db:"file_size"`
	CreatedBy       string    `db:"created_by"`
}

// LoadAppliedMigrations database'den uygulanan migration'ları okur
func (r *Runner) LoadAppliedMigrations() (map[int64]AppliedMigration, error) {
	query := fmt.Sprintf(`
		SELECT 
			version, 
			name, 
			up_checksum, 
			down_checksum, 
			applied_at, 
			execution_time_ms,
			file_size,
			created_by
		FROM %s 
		ORDER BY version ASC
	`, r.config.TableName)

	rows, err := r.db.Query(query)
	if err != nil {
		// Tablo yoksa boş map döndür (ilk çalıştırma)
		if isTableNotExistError(err) {
			if r.config.Verbose {
				log.Info().Msg("Migration tablosu henüz yok, ilk çalıştırma")
			}
			return make(map[int64]AppliedMigration), nil
		}
		return nil, fmt.Errorf("applied migration'lar okunamadı: %w", err)
	}
	defer rows.Close()

	appliedMigrations := make(map[int64]AppliedMigration)

	for rows.Next() {
		var applied AppliedMigration
		err := rows.Scan(
			&applied.Version,
			&applied.Name,
			&applied.UpChecksum,
			&applied.DownChecksum,
			&applied.AppliedAt,
			&applied.ExecutionTimeMs,
			&applied.FileSize,
			&applied.CreatedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("applied migration scan hatası: %w", err)
		}

		appliedMigrations[applied.Version] = applied
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("applied migration iteration hatası: %w", err)
	}

	if r.config.Debug {
		log.Debug().
			Int("count", len(appliedMigrations)).
			Msg("Applied migration'lar database'den okundu")
	}

	return appliedMigrations, nil
}

// LoadMigrationsWithStatus dosyaları okur ve database ile karşılaştırarak status belirler
func (r *Runner) LoadMigrationsWithStatus() ([]Migration, error) {
	// Deadlock riskini azaltmak için burada global lock kullanmıyoruz

	// Dosyalardan migration'ları oku
	migrations, err := r.LoadMigrationsFromDisk()
	if err != nil {
		return nil, fmt.Errorf("dosyalardan migration okunamadı: %w", err)
	}

	// Database'den applied migration'ları oku
	appliedMigrations, err := r.LoadAppliedMigrations()
	if err != nil {
		return nil, fmt.Errorf("database'den applied migration okunamadı: %w", err)
	}

	// Migration'ları applied status ile güncelle
	for i := range migrations {
		if applied, exists := appliedMigrations[migrations[i].Version]; exists {
			// Bu migration database'de kayıtlı
			migrations[i].Applied = true
			migrations[i].AppliedAt = &applied.AppliedAt

			// Checksum kontrolü
			if r.config.ValidateChecksums {
				if err := r.validateChecksums(migrations[i], applied); err != nil {
					if r.config.AllowDirtyMigrate {
						log.Warn().
							Err(err).
							Int64("version", migrations[i].Version).
							Msg("Checksum uyumsuzluğu tespit edildi ama AllowDirtyMigrate=true")
					} else {
						return nil, fmt.Errorf("migration %d checksum hatası: %w", migrations[i].Version, err)
					}
				}
			}
		}
	}

	if r.config.Verbose {
		appliedCount := 0
		pendingCount := 0
		for _, m := range migrations {
			if m.Applied {
				appliedCount++
			} else {
				pendingCount++
			}
		}

		log.Info().
			Int("total", len(migrations)).
			Int("applied", appliedCount).
			Int("pending", pendingCount).
			Msg("Migration status güncellendi")
	}

	return migrations, nil
}

// validateChecksums dosya ve database checksum'larını karşılaştırır
func (r *Runner) validateChecksums(fileMigration Migration, dbMigration AppliedMigration) error {
	// UP checksum kontrolü
	if fileMigration.UpChecksum != dbMigration.UpChecksum {
		return fmt.Errorf("UP dosyası değiştirilmiş - dosya: %s, db: %s",
			fileMigration.UpChecksum[:8], dbMigration.UpChecksum[:8])
	}

	// DOWN checksum kontrolü (eğer ikisinde de varsa)
	if fileMigration.HasDownFile && dbMigration.DownChecksum != nil {
		if fileMigration.DownChecksum != *dbMigration.DownChecksum {
			return fmt.Errorf("DOWN dosyası değiştirilmiş - dosya: %s, db: %s",
				fileMigration.DownChecksum[:8], (*dbMigration.DownChecksum)[:8])
		}
	}

	return nil
}

// GetStatus migration sisteminin genel durumunu döner
func (r *Runner) GetStatus() (*MigrationStatus, error) {
	// Migration'ları status ile beraber yükle
	migrations, err := r.LoadMigrationsWithStatus()
	if err != nil {
		return nil, fmt.Errorf("migration status alınamadı: %w", err)
	}

	// Status hesapla
	status := &MigrationStatus{
		Migrations:    migrations,
		TotalCount:    len(migrations),
		AppliedCount:  0,
		PendingCount:  0,
		SystemHealth:  StatusHealthy,
		ChecksumValid: true,
		ErrorCount:    0,
		WarningCount:  0,
	}

	var lastAppliedAt *time.Time
	currentVersion := int64(0)

	for _, migration := range migrations {
		if migration.Applied {
			status.AppliedCount++

			// En yüksek applied version'ı bul
			if migration.Version > currentVersion {
				currentVersion = migration.Version
			}

			// En son applied zamanı
			if migration.AppliedAt != nil {
				if lastAppliedAt == nil || migration.AppliedAt.After(*lastAppliedAt) {
					lastAppliedAt = migration.AppliedAt
				}
			}
		} else {
			status.PendingCount++
			status.WarningCount++
		}
	}

	status.CurrentVersion = currentVersion
	status.LastAppliedAt = lastAppliedAt

	// System health belirle
	if status.ErrorCount > 0 {
		status.SystemHealth = StatusError
		status.ChecksumValid = false
	} else if status.PendingCount > 0 {
		status.SystemHealth = StatusWarning
	}

	if r.config.Debug {
		log.Debug().
			Int64("current_version", status.CurrentVersion).
			Int("applied", status.AppliedCount).
			Int("pending", status.PendingCount).
			Str("health", string(status.SystemHealth)).
			Msg("Migration status hesaplandı")
	}

	return status, nil
}

// RecordMigration migration'ı database'de kayıt eder (UP migration için)
func (r *Runner) RecordMigration(migration Migration, executionTime time.Duration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			version, 
			name, 
			up_checksum, 
			down_checksum, 
			applied_at, 
			execution_time_ms, 
			file_size, 
			created_by
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, r.config.TableName)

	downChecksum := sql.NullString{}
	if migration.HasDownFile {
		downChecksum.String = migration.DownChecksum
		downChecksum.Valid = true
	}

	createdBy := "system"
	if r.config.IsCLI {
		createdBy = "cli"
	}

	_, err := r.db.Exec(
		query,
		migration.Version,
		migration.Name,
		migration.UpChecksum,
		downChecksum,
		time.Now(),
		executionTime.Milliseconds(),
		migration.UpFileSize,
		createdBy,
	)

	if err != nil {
		return fmt.Errorf("migration kaydı eklenemedi: %w", err)
	}

	if r.config.Debug {
		log.Debug().
			Int64("version", migration.Version).
			Dur("execution_time", executionTime).
			Msg("Migration database'de kaydedildi")
	}

	return nil
}

// DeleteMigrationRecord migration kaydını database'den siler (DOWN migration için)
func (r *Runner) DeleteMigrationRecord(version int64) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE version = $1`, r.config.TableName)

	result, err := r.db.Exec(query, version)
	if err != nil {
		return fmt.Errorf("migration kaydı silinemedi: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("migration silme sonucu kontrol edilemedi: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("migration kaydı bulunamadı: version %d", version)
	}

	if r.config.Debug {
		log.Debug().
			Int64("version", version).
			Msg("Migration kaydı database'den silindi")
	}

	return nil
}

// isTableNotExistError database tablosunun var olup olmadığını kontrol eder
func isTableNotExistError(err error) bool {
	if err == nil {
		return false
	}
	// lib/pq kullanıyorsan istersen burada pq.Error.Code == "42P01" kontrolü ekleyebilirsin.
	s := err.Error()
	return strings.Contains(s, "does not exist") ||
		strings.Contains(s, "no such table") ||
		strings.Contains(s, "undefined table")
}
