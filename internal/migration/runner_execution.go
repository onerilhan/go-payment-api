// internal/migration/runner_execution.go
package migration

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// RunUp belirtilen version'a kadar veya tüm pending migration'ları çalıştırır
func (r *Runner) RunUp(targetVersion int64) ([]MigrationResult, error) {
	if r.config.Verbose {
		if targetVersion > 0 {
			log.Info().Int64("target_version", targetVersion).Msg("Migration UP başlatılıyor - belirli version'a kadar")
		} else {
			log.Info().Msg("Migration UP başlatılıyor - tüm pending")
		}
	}

	// Tracking table
	if err := r.Initialize(); err != nil {
		return nil, fmt.Errorf("migration sistemi initialize edilemedi: %w", err)
	}

	// Migration'ları status ile yükle
	migrations, err := r.LoadMigrationsWithStatus()
	if err != nil {
		return nil, fmt.Errorf("migration'lar yüklenemedi: %w", err)
	}

	var results []MigrationResult
	executedCount := 0

	for _, migration := range migrations {
		// Zaten applied'ları atla
		if migration.Applied {
			if r.config.Debug {
				log.Debug().Int64("version", migration.Version).Msg("Migration zaten applied, atlanıyor")
			}
			continue
		}

		// Target version sınırı
		if targetVersion > 0 && migration.Version > targetVersion {
			if r.config.Debug {
				log.Debug().
					Int64("version", migration.Version).
					Int64("target", targetVersion).
					Msg("Target version aşıldı, durduruluyor")
			}
			break
		}

		// Çalıştır
		result := r.executeMigration(migration, DirectionUp)
		results = append(results, result)
		executedCount++

		if !result.Success {
			log.Error().
				Int64("version", migration.Version).
				Str("error", result.Error).
				Msg("Migration başarısız, durduruluyor")
			break
		}

		if r.config.Verbose {
			log.Info().
				Int64("version", migration.Version).
				Str("name", migration.Name).
				Dur("duration", result.ExecutionTime).
				Msg("Migration başarıyla uygulandı")
		}
	}

	if r.config.Verbose {
		log.Info().Int("executed", executedCount).Int("total_results", len(results)).Msg("Migration UP tamamlandı")
	}

	return results, nil
}

// RunDown belirtilen version'a kadar migration'ları geri alır
func (r *Runner) RunDown(targetVersion int64) ([]MigrationResult, error) {
	if r.config.Verbose {
		log.Info().Int64("target_version", targetVersion).Msg("Migration DOWN başlatılıyor")
	}

	// Migration'ları status ile yükle
	migrations, err := r.LoadMigrationsWithStatus()
	if err != nil {
		return nil, fmt.Errorf("migration'lar yüklenemedi: %w", err)
	}

	var results []MigrationResult
	executedCount := 0

	// Reverse order (yeni->eski)
	for i := len(migrations) - 1; i >= 0; i-- {
		migration := migrations[i]

		// Applied olmayanları atla
		if !migration.Applied {
			continue
		}

		// Target'a geldik mi?
		if migration.Version <= targetVersion {
			if r.config.Debug {
				log.Debug().
					Int64("version", migration.Version).
					Int64("target", targetVersion).
					Msg("Target version'a ulaşıldı, durduruluyor")
			}
			break
		}

		// DOWN dosyası yoksa karar
		if !migration.HasDownFile {
			if r.config.RequireDownFiles {
				result := MigrationResult{
					Success:   false,
					Version:   migration.Version,
					Name:      migration.Name,
					Direction: DirectionDown,
					Error:     "DOWN dosyası bulunamadı ve RequireDownFiles=true",
					StartedAt: time.Now(),
				}
				results = append(results, result)
				break
			}
			log.Warn().Int64("version", migration.Version).Msg("DOWN dosyası yok, atlanıyor")
			continue
		}

		// Geri al
		result := r.executeMigration(migration, DirectionDown)
		results = append(results, result)
		executedCount++

		if !result.Success {
			log.Error().
				Int64("version", migration.Version).
				Str("error", result.Error).
				Msg("Migration rollback başarısız, durduruluyor")
			break
		}

		if r.config.Verbose {
			log.Info().
				Int64("version", migration.Version).
				Str("name", migration.Name).
				Dur("duration", result.ExecutionTime).
				Msg("Migration başarıyla geri alındı")
		}
	}

	if r.config.Verbose {
		log.Info().Int("executed", executedCount).Int("total_results", len(results)).Msg("Migration DOWN tamamlandı")
	}

	return results, nil
}

// SQLExecutionResult SQL çalıştırma sonucu
type SQLExecutionResult struct {
	AffectedRows   int64
	StatementCount int
}

// executeMigration tek bir migration'ı çalıştırır (UP / DOWN)
func (r *Runner) executeMigration(migration Migration, direction MigrationDirection) MigrationResult {
	startTime := time.Now()
	result := MigrationResult{
		Version:       migration.Version,
		Name:          migration.Name,
		Direction:     direction,
		StartedAt:     startTime,
		ChecksumValid: true,
	}

	// SQL seç
	var sqlText string
	if direction == DirectionUp {
		sqlText = migration.UpSQL
	} else {
		sqlText = migration.DownSQL
		if strings.TrimSpace(sqlText) == "" {
			result.Error = "DOWN SQL boş"
			return result
		}
	}

	if r.config.Debug {
		log.Debug().
			Int64("version", migration.Version).
			Str("direction", string(direction)).
			Int("sql_length", len(sqlText)).
			Msg("Migration execution başlıyor")
	}

	// DryRun
	if r.config.DryRun {
		result.Success = true
		result.ExecutionTime = time.Since(startTime)
		result.CompletedAt = &result.StartedAt
		result.SqlStatements = countSQLStatements(sqlText)

		log.Info().
			Int64("version", migration.Version).
			Str("direction", string(direction)).
			Msg("DRY RUN: Migration test edildi, uygulanmadı")
		return result
	}

	// Transaction
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(r.config.TransactionTimeout)*time.Second)
	defer cancel()

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		result.Error = fmt.Sprintf("transaction başlatılamadı: %v", err)
		return result
	}
	defer tx.Rollback() // commit başarısızsa otomatik geri al

	// SQL çalıştır
	sqlResult, err := r.executeSQL(tx, sqlText)
	if err != nil {
		result.Error = fmt.Sprintf("SQL execution hatası: %v", err)
		return result
	}
	result.AffectedRows = sqlResult.AffectedRows
	result.SqlStatements = sqlResult.StatementCount

	// Tracking tablosu
	if direction == DirectionUp {
		if err := r.recordMigrationInTx(tx, migration, time.Since(startTime)); err != nil {
			result.Error = fmt.Sprintf("migration kaydı eklenemedi: %v", err)
			return result
		}
	} else {
		if err := r.deleteMigrationRecordInTx(tx, migration.Version); err != nil {
			result.Error = fmt.Sprintf("migration kaydı silinemedi: %v", err)
			return result
		}
	}

	// Commit
	if err := tx.Commit(); err != nil {
		result.Error = fmt.Sprintf("transaction commit hatası: %v", err)
		return result
	}

	// Başarılı
	result.Success = true
	result.ExecutionTime = time.Since(startTime)
	completedAt := time.Now()
	result.CompletedAt = &completedAt
	return result
}

// executeSQL SQL'i transaction içinde çalıştırır
func (r *Runner) executeSQL(tx *sql.Tx, sqlContent string) (*SQLExecutionResult, error) {
	stmts := r.splitSQLStatements(sqlContent)
	if len(stmts) == 0 {
		return nil, fmt.Errorf("hiç SQL statement bulunamadı")
	}

	var totalRows int64
	count := 0

	for i, statement := range stmts {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		if r.config.Debug {
			preview := statement
			if len(preview) > 50 {
				preview = preview[:50] + "..."
			}
			log.Debug().
				Int("statement_no", i+1).
				Int("length", len(statement)).
				Str("preview", preview).
				Msg("SQL statement çalıştırılıyor")
		}

		res, err := tx.Exec(statement)
		if err != nil {
			return nil, fmt.Errorf("statement %d çalıştırılamadı: %w", i+1, err)
		}

		if rows, err := res.RowsAffected(); err == nil {
			totalRows += rows
		}
		count++
	}

	return &SQLExecutionResult{
		AffectedRows:   totalRows,
		StatementCount: count,
	}, nil
}

// splitSQLStatements SQL'i statement'lara böler
func (r *Runner) splitSQLStatements(sql string) []string {
	var out []string
	var buf strings.Builder

	inSingle := false       // '...'
	inDollar := false       // $tag$ ... $tag$
	dollarTag := ""         // "$tag$"
	inLineComment := false  // -- ... \n
	inBlockComment := false // /* ... */

	i := 0
	for i < len(sql) {
		c := sql[i]

		// --- yorum bitişleri ---
		if inLineComment {
			buf.WriteByte(c)
			if c == '\n' {
				inLineComment = false
			}
			i++
			continue
		}
		if inBlockComment {
			if c == '*' && i+1 < len(sql) && sql[i+1] == '/' {
				buf.WriteByte(c)
				buf.WriteByte('/')
				i += 2
				inBlockComment = false
				continue
			}
			buf.WriteByte(c)
			i++
			continue
		}

		// --- yorum başlangıçları (string/dollar içinde değilken) ---
		if !inSingle && !inDollar {
			if c == '-' && i+1 < len(sql) && sql[i+1] == '-' {
				buf.WriteString("--")
				i += 2
				inLineComment = true
				continue
			}
			if c == '/' && i+1 < len(sql) && sql[i+1] == '*' {
				buf.WriteString("/*")
				i += 2
				inBlockComment = true
				continue
			}
		}

		// --- tek tırnaklı string (escape '') ---
		if !inDollar && c == '\'' {
			buf.WriteByte(c)
			i++
			inSingle = !inSingle
			if inSingle {
				for i < len(sql) {
					buf.WriteByte(sql[i])
					if sql[i] == '\'' {
						if i+1 < len(sql) && sql[i+1] == '\'' {
							buf.WriteByte(sql[i+1])
							i += 2
							continue
						}
						i++
						inSingle = false
						break
					}
					i++
				}
			}
			continue
		}

		// --- dollar-quoted başlangıç/bitiş ---
		if !inSingle && c == '$' {
			j := i + 1
			for j < len(sql) {
				ch := sql[j]
				if (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
					(ch >= '0' && ch <= '9') || ch == '_' {
					j++
					continue
				}
				break
			}
			if j < len(sql) && sql[j] == '$' {
				tag := sql[i : j+1] // $tag$
				buf.WriteString(tag)
				i = j + 1
				if !inDollar {
					inDollar = true
					dollarTag = tag
				} else if inDollar && tag == dollarTag {
					inDollar = false
					dollarTag = ""
				}
				continue
			}
		}

		// --- statement sonu ---
		if !inSingle && !inDollar && c == ';' {
			s := strings.TrimSpace(buf.String())
			if s != "" {
				out = append(out, s)
			}
			buf.Reset()
			i++
			continue
		}

		buf.WriteByte(c)
		i++
	}

	s := strings.TrimSpace(buf.String())
	if s != "" {
		out = append(out, s)
	}
	return out
}

func countSQLStatements(sql string) int {
	r := &Runner{config: &MigrationConfig{}}
	return len(r.splitSQLStatements(sql))
}

// recordMigrationInTx migration'ı transaction içinde kaydeder
func (r *Runner) recordMigrationInTx(tx *sql.Tx, migration Migration, executionTime time.Duration) error {
	query := fmt.Sprintf(`
		INSERT INTO %s (
			version, name, up_checksum, down_checksum, applied_at, execution_time_ms, file_size, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
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

	_, err := tx.Exec(
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
	return err
}

// deleteMigrationRecordInTx migration kaydını transaction içinde siler
func (r *Runner) deleteMigrationRecordInTx(tx *sql.Tx, version int64) error {
	query := fmt.Sprintf(`DELETE FROM %s WHERE version = $1`, r.config.TableName)
	res, err := tx.Exec(query, version)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return fmt.Errorf("migration kaydı bulunamadı: version %d", version)
	}
	return nil
}

// minInt iki int'in küçüğü
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
